package build

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

// Generator creates build files in a temporary directory
type Generator struct {
	BuildDir string
	Analysis *ProjectAnalysis
	Platform string
}

// Generate creates all necessary build files
func (g *Generator) Generate() error {
	// Create build directory structure
	if err := g.createDirectories(); err != nil {
		return fmt.Errorf("failed to create directories: %v", err)
	}

	// Generate go.mod
	if err := g.generateGoMod(); err != nil {
		return fmt.Errorf("failed to generate go.mod: %v", err)
	}

	// Generate components registry
	if err := g.generateComponentsRegistry(); err != nil {
		return fmt.Errorf("failed to generate components registry: %v", err)
	}

	// Generate main.go
	if err := g.generateMainGo(); err != nil {
		return fmt.Errorf("failed to generate main.go: %v", err)
	}

	// Copy or symlink assets
	if err := g.handleAssets(); err != nil {
		return fmt.Errorf("failed to handle assets: %v", err)
	}

	return nil
}

func (g *Generator) createDirectories() error {
	dirs := []string{
		filepath.Join(g.BuildDir, "generated"),
		filepath.Join(g.BuildDir, "assets"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}

	return nil
}

func (g *Generator) generateGoMod() error {
	modContent := `module game_build

go 1.24

require github.com/EnesBaytekin/imge v0.1.0

// For local development, uncomment the line below:
// replace github.com/EnesBaytekin/imge => ../imge
`

	modPath := filepath.Join(g.BuildDir, "go.mod")
	return os.WriteFile(modPath, []byte(modContent), 0644)
}

func (g *Generator) generateComponentsRegistry() error {
	// Create import statements for user components
	imports := []string{}
	for _, compFile := range g.Analysis.ComponentFiles {
		// Convert file path to import path
		// Remove .go extension and convert to import path
		importPath := strings.TrimSuffix(compFile, ".go")
		imports = append(imports, fmt.Sprintf(`_ "%s"`, importPath))
	}

	// Create the registry file
	registryTemplate := `// GENERATED CODE - DO NOT EDIT
package main

import (
	// User component imports
{{range .Imports}}	{{.}}
{{end}}
	// Engine imports (components will auto-register via init() functions)
)

// Note: All components are automatically registered via their init() functions
// when the packages are imported above.
`

	tmpl, err := template.New("registry").Parse(registryTemplate)
	if err != nil {
		return err
	}

	data := struct {
		Imports []string
	}{
		Imports: imports,
	}

	registryPath := filepath.Join(g.BuildDir, "generated", "components_registry.go")
	file, err := os.Create(registryPath)
	if err != nil {
		return err
	}
	defer file.Close()

	return tmpl.Execute(file, data)
}

func (g *Generator) generateMainGo() error {
	mainTemplate := `// GENERATED CODE - DO NOT EDIT
package main

import (
	"log"
	"os"
	"path/filepath"

	"github.com/EnesBaytekin/imge/internal/core"
	"github.com/EnesBaytekin/imge/internal/platform/{{.Platform}}"
)

func main() {
	// Get the executable path to locate assets
	exePath, err := os.Executable()
	if err != nil {
		log.Fatalf("Failed to get executable path: %v", err)
	}
	exeDir := filepath.Dir(exePath)

	// Create platform
	platform := {{.Platform}}.New()

	// Create game configuration
	config := core.Config{
		WindowWidth:  {{.WindowWidth}},
		WindowHeight: {{.WindowHeight}},
		WindowTitle:  "{{.WindowTitle}}",
		TargetFPS:    {{.TargetFPS}},
		FixedUpdate:  false,
		InitialScene: "{{.InitialScene}}",
	}

	// Create game
	game := core.NewGameWithConfig(config, platform)

	// Load initial scene if specified
	if config.InitialScene != "" {
		scenePath := filepath.Join(exeDir, "scenes", config.InitialScene + ".scene")
		if err := game.LoadScene(scenePath); err != nil {
			log.Printf("Warning: Could not load initial scene: %v", err)
			log.Println("Starting with empty scene")
		}
	}

	// Run the game
	if err := game.Run(); err != nil {
		log.Fatalf("Game error: %v", err)
	}
}
`

	tmpl, err := template.New("main").Parse(mainTemplate)
	if err != nil {
		return err
	}

	data := struct {
		Platform      string
		GameName      string
		WindowTitle   string
		WindowWidth   int
		WindowHeight  int
		TargetFPS     int
		InitialScene  string
	}{
		Platform:      g.Platform,
		GameName:      g.Analysis.GameConfig.Name,
		WindowTitle:   g.Analysis.GameConfig.Window.Title,
		WindowWidth:   g.Analysis.GameConfig.Window.Width,
		WindowHeight:  g.Analysis.GameConfig.Window.Height,
		TargetFPS:     g.Analysis.GameConfig.Game.TargetFPS,
		InitialScene:  g.Analysis.GameConfig.Game.InitialScene,
	}

	mainPath := filepath.Join(g.BuildDir, "generated", "main.go")
	file, err := os.Create(mainPath)
	if err != nil {
		return err
	}
	defer file.Close()

	return tmpl.Execute(file, data)
}

func (g *Generator) handleAssets() error {
	// For now, we'll just create a README explaining asset location
	// In a real implementation, we might embed assets or copy them
	readmeContent := `Assets Directory

This directory should contain game assets (images, sounds, etc.).
For the mock platform, assets are not used.
For other platforms, assets should be placed here or embedded in the binary.

To embed assets, use Go's embed directive in your components.
`

	readmePath := filepath.Join(g.BuildDir, "assets", "README.md")
	return os.WriteFile(readmePath, []byte(readmeContent), 0644)
}