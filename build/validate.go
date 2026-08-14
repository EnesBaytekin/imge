package build

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/EnesBaytekin/imge"
	corejson "github.com/EnesBaytekin/imge/core/json"
)

// registerKindRe matches core.RegisterComponent("kind", ...) calls, capturing the
// kind string. Kinds are always string literals in practice.
var registerKindRe = regexp.MustCompile(`RegisterComponent\s*\(\s*"([^"]+)"`)

// extractKinds returns the distinct component kinds registered by a source file.
func extractKinds(src []byte) []string {
	seen := make(map[string]bool)
	var kinds []string
	for _, m := range registerKindRe.FindAllSubmatch(src, -1) {
		k := string(m[1])
		if !seen[k] {
			seen[k] = true
			kinds = append(kinds, k)
		}
	}
	return kinds
}

// validateComponents verifies that (1) every component kind is registered by
// exactly one source file (built-in or custom) and (2) every kind referenced in
// .obj/.scene files is actually registered. It runs before compilation so users
// get a clear error instead of a silent overwrite or a runtime "not registered".
func (g *Generator) validateComponents() error {
	var problems []string

	kindToSource := make(map[string]string)

	addKinds := func(source string, kinds []string) {
		for _, k := range kinds {
			if prev, exists := kindToSource[k]; exists {
				problems = append(problems, fmt.Sprintf(
					"duplicate component kind %q registered by both %s and %s", k, prev, source))
				continue
			}
			kindToSource[k] = source
		}
	}

	// Built-in kinds, read from the embedded engine source.
	entries, err := fs.ReadDir(imge.EngineSource, "engine/components")
	if err != nil {
		return fmt.Errorf("failed to read built-in components: %w", err)
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") {
			continue
		}
		data, err := fs.ReadFile(imge.EngineSource, "engine/components/"+entry.Name())
		if err != nil {
			return err
		}
		addKinds("built-in:"+entry.Name(), extractKinds(data))
	}

	// User component kinds.
	for _, compFile := range g.Analysis.ComponentFiles {
		src, err := os.ReadFile(filepath.Join(g.Analysis.ProjectDir, compFile))
		if err != nil {
			return fmt.Errorf("failed to read component %s: %w", compFile, err)
		}
		kinds := extractKinds(src)
		if len(kinds) == 0 {
			problems = append(problems, fmt.Sprintf(
				"component file %s registers no component (missing core.RegisterComponent)", compFile))
		}
		addKinds(compFile, kinds)
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
