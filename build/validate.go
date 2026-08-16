package build

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	corejson "github.com/EnesBaytekin/imge/core/json"
)

// validateComponents verifies that (1) every component kind maps to exactly one
// source file (built-in or custom) and (2) every kind referenced in .obj/.scene
// files is actually registered. Kinds are derived from each file's component
// struct (see componentKinds), matching the codegen that produces registry.go, so
// validation and registration can never disagree.
func (g *Generator) validateComponents(kinds []componentKind) error {
	var problems []string

	kindToSource := make(map[string]string)
	for _, k := range kinds {
		if prev, exists := kindToSource[k.kind]; exists {
			problems = append(problems, fmt.Sprintf(
				"duplicate component kind %q registered by both %s and %s", k.kind, prev, k.source))
			continue
		}
		kindToSource[k.kind] = k.source
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
			return err
		}
		cfg, err := corejson.ParseObjectConfig(data)
		if err != nil {
			return fmt.Errorf("failed to parse %s: %w", objFile, err)
		}
		for _, comp := range cfg.Components {
			collect(objFile, comp.Kind)
		}
	}

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
			for _, comp := range obj.Components {
				collect(sceneFile, comp.Kind)
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
		return fmt.Errorf("component validation failed:\n  - %s", strings.Join(problems, "\n  - "))
	}
	return nil
}
