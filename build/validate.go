package build

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	corejson "github.com/EnesBaytekin/imge/core/json"
)

// fileReferenceArgs are component `args` keys whose string value is a
// project-root-relative file path that must exist at build time. The built-ins
// that reference files are @Sprite (`texture`) and @Sound (`sound`). Scene objects
// additionally reference a template file via `file`, handled separately.
var fileReferenceArgs = map[string]bool{
	"texture": true,
	"sound":   true,
}

// validateComponents verifies that (1) every component kind maps to exactly one
// source file, (2) no two components share a Go type name (which would collide
// in the merged components package), and (3) every kind referenced in .obj/.scene
// files is actually registered. It also collects dependency warnings: an object
// that uses a component without the components it declares it needs. Kinds are
// derived from each file's component struct (see componentKinds), matching the
// codegen that produces registry.go, so validation and registration can never
// disagree.
func (g *Generator) validateComponents(kinds []componentKind) ([]string, error) {
	var problems []string
	var warnings []string

	kindToSource := make(map[string]string)
	typeToSource := make(map[string]string)
	requiresByKind := make(map[string][]string)

	for _, k := range kinds {
		if prev, exists := kindToSource[k.kind]; exists {
			problems = append(problems, fmt.Sprintf(
				"duplicate component kind %q registered by both %s and %s", k.kind, prev, k.source))
			continue
		}
		kindToSource[k.kind] = k.source

		if prev, exists := typeToSource[k.typeName]; exists {
			problems = append(problems, fmt.Sprintf(
				"duplicate component type name %q in both %s and %s (types collide in the merged components package)",
				k.typeName, prev, k.source))
			continue
		}
		typeToSource[k.typeName] = k.source

		if len(k.requires) > 0 {
			requiresByKind[k.kind] = k.requires
		}
	}

	// Kinds referenced from .obj and .scene files.
	referenced := make(map[string]string) // kind -> first file referencing it
	collect := func(file, kind string) {
		if _, exists := referenced[kind]; !exists {
			referenced[kind] = file
		}
	}

	for _, objFile := range g.Analysis.ObjectFiles {
		data, err := os.ReadFile(filepath.Join(g.Analysis.ProjectDir, objFile))
		if err != nil {
			return warnings, err
		}
		cfg, err := corejson.ParseObjectConfig(data)
		if err != nil {
			return warnings, fmt.Errorf("failed to parse %s: %w", objFile, err)
		}
		for _, comp := range cfg.Components {
			collect(objFile, comp.Kind)
		}
		warnings = append(warnings, checkObjectDeps(cfg.Components, requiresByKind, objFile)...)
	}

	for _, sceneFile := range g.Analysis.SceneFiles {
		data, err := os.ReadFile(filepath.Join(g.Analysis.ProjectDir, sceneFile))
		if err != nil {
			return warnings, err
		}
		cfg, err := corejson.ParseSceneConfig(data)
		if err != nil {
			return warnings, fmt.Errorf("failed to parse %s: %w", sceneFile, err)
		}
		for _, obj := range cfg.Objects {
			for _, comp := range obj.Components {
				collect(sceneFile, comp.Kind)
			}
			if len(obj.Components) > 0 {
				warnings = append(warnings, checkObjectDeps(obj.Components, requiresByKind,
					filepath.Join(sceneFile, obj.Name))...)
			}
		}
	}

	for kind, file := range referenced {
		if _, ok := kindToSource[kind]; !ok {
			problems = append(problems, fmt.Sprintf(
				"unknown component kind %q referenced in %s (not registered by any component)", kind, file))
		}
	}

	if len(problems) > 0 {
		return warnings, fmt.Errorf("component validation failed:\n  - %s", strings.Join(problems, "\n  - "))
	}
	return warnings, nil
}

// validateFileReferences verifies that every file referenced from scene/object
// JSON — a scene object's `file` template, or an @Sprite `texture` / @Sound
// `sound` arg — actually exists in the project. A missing file (e.g. a typo in an
// asset path) should fail the build loudly instead of embedding a game that
// silently warns at runtime.
func (g *Generator) validateFileReferences() error {
	// The set of every file the build will embed: assets, scenes, and templates.
	// File references are project-root-relative, matching these paths.
	known := make(map[string]bool)
	add := func(p string) {
		known[path.Clean(filepath.ToSlash(p))] = true
	}
	for _, f := range g.Analysis.AssetFiles {
		add(f)
	}
	for _, f := range g.Analysis.SceneFiles {
		add(f)
	}
	for _, f := range g.Analysis.ObjectFiles {
		add(f)
	}

	var problems []string
	check := func(key, ref, where string) {
		if ref == "" {
			return
		}
		if !known[path.Clean(filepath.ToSlash(ref))] {
			problems = append(problems, fmt.Sprintf(
				"%s: %s %q not found in the project (add the file or fix the path)", where, key, ref))
		}
	}

	// Object templates (.obj) — validate each component's file-reference args.
	for _, objFile := range g.Analysis.ObjectFiles {
		data, err := os.ReadFile(filepath.Join(g.Analysis.ProjectDir, objFile))
		if err != nil {
			return err
		}
		cfg, err := corejson.ParseObjectConfig(data)
		if err != nil {
			return fmt.Errorf("failed to parse %s: %w", objFile, err)
		}
		for _, comp := range cfg.Components {
			checkComponentFileRefs(comp, objFile, check)
		}
	}

	// Scenes (.scene) — validate object template `file` references and component
	// file-reference args, both inline and on referenced templates.
	for _, sceneFile := range g.Analysis.SceneFiles {
		data, err := os.ReadFile(filepath.Join(g.Analysis.ProjectDir, sceneFile))
		if err != nil {
			return err
		}
		cfg, err := corejson.ParseSceneConfig(data)
		if err != nil {
			return fmt.Errorf("failed to parse %s: %w", sceneFile, err)
		}
		for _, obj := range cfg.Objects {
			where := sceneFile
			if obj.Name != "" {
				where = filepath.Join(sceneFile, obj.Name)
			}
			if obj.File != "" {
				check("file", obj.File, where)
			}
			for _, comp := range obj.Components {
				checkComponentFileRefs(comp, where, check)
			}
		}
	}

	if len(problems) > 0 {
		return fmt.Errorf("file validation failed:\n  - %s", strings.Join(problems, "\n  - "))
	}
	return nil
}

// checkComponentFileRefs calls check for each file-reference arg on a component.
func checkComponentFileRefs(comp corejson.ComponentInstanceConfig, where string, check func(key, ref, where string)) {
	for key, val := range comp.Args {
		if !fileReferenceArgs[key] {
			continue
		}
		ref, ok := val.(string)
		if !ok {
			continue
		}
		check(key, ref, where)
	}
}

// checkObjectDeps warns when an object's component list includes a component that
// declares a Requires() dependency not present on the same object.
func checkObjectDeps(comps []corejson.ComponentInstanceConfig, requiresByKind map[string][]string, where string) []string {
	present := make(map[string]bool, len(comps))
	for _, comp := range comps {
		present[comp.Kind] = true
	}

	var warnings []string
	for _, comp := range comps {
		for _, req := range requiresByKind[comp.Kind] {
			if !present[req] {
				warnings = append(warnings, fmt.Sprintf(
					"%s: component %q requires %q, but the object has no %q component",
					where, comp.Kind, req, req))
			}
		}
	}
	return warnings
}
