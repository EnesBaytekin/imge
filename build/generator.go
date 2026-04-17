package build

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

// Generator creates build files in a temporary directory
type Generator struct {
	BuildDir     string
	Analysis     *ProjectAnalysis
	Platform     string
	EngineSource string // Path to engine source code
}

// Generate creates all necessary build files
func (g *Generator) Generate() error {
	// Create build directory structure
	if err := g.createDirectories(); err != nil {
		return fmt.Errorf("failed to create directories: %v", err)
	}

	// Copy engine source code to build directory
	if err := g.copyEngineCode(); err != nil {
		return fmt.Errorf("failed to copy engine code: %v", err)
	}

	// Copy user components to build directory
	if err := g.copyUserComponents(); err != nil {
		return fmt.Errorf("failed to copy user components: %v", err)
	}

	// Copy project assets (assets/, scenes/, objects/)
	if err := g.copyProjectAssets(); err != nil {
		return fmt.Errorf("failed to copy project assets: %v", err)
	}

	// Generate main.go
	if err := g.generateMainGo(); err != nil {
		return fmt.Errorf("failed to generate main.go: %v", err)
	}

	// Generate go.mod
	if err := g.generateGoMod(); err != nil {
		return fmt.Errorf("failed to generate go.mod: %v", err)
	}

	return nil
}

func (g *Generator) createDirectories() error {
	dirs := []string{
		filepath.Join(g.BuildDir, "core"),
		filepath.Join(g.BuildDir, "engine", "components"),
		filepath.Join(g.BuildDir, "core", "math"),
		filepath.Join(g.BuildDir, "platform", g.Platform),
		filepath.Join(g.BuildDir, "components"),
		filepath.Join(g.BuildDir, "assets"),
		filepath.Join(g.BuildDir, "scenes"),
		filepath.Join(g.BuildDir, "objects"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}

	return nil
}

func (g *Generator) copyEngineCode() error {
	// Copy core directory
	coreSrc := filepath.Join(g.EngineSource, "core")
	coreDst := filepath.Join(g.BuildDir, "core")
	if err := copyDir(coreSrc, coreDst); err != nil {
		return fmt.Errorf("failed to copy core: %v", err)
	}

	// Copy engine/components directory to components (built-in components)
	componentsSrc := filepath.Join(g.EngineSource, "engine", "components")
	componentsDst := filepath.Join(g.BuildDir, "components")
	if err := copyDir(componentsSrc, componentsDst); err != nil {
		return fmt.Errorf("failed to copy components: %v", err)
	}

	// Copy core/math directory
	mathSrc := filepath.Join(g.EngineSource, "core", "math")
	if _, err := os.Stat(mathSrc); !os.IsNotExist(err) {
		mathDst := filepath.Join(g.BuildDir, "core", "math")
		if err := copyDir(mathSrc, mathDst); err != nil {
			return fmt.Errorf("failed to copy math: %v", err)
		}
	}

	// Copy platform directory
	platformSrc := filepath.Join(g.EngineSource, "platform", g.Platform)
	platformDst := filepath.Join(g.BuildDir, "platform", g.Platform)
	if err := copyDir(platformSrc, platformDst); err != nil {
		return fmt.Errorf("failed to copy platform: %v", err)
	}

	// Copy other platform directories that might be needed
	// (some platforms might depend on shared code in platform/common for example)
	commonPlatformSrc := filepath.Join(g.EngineSource, "platform", "common")
	if _, err := os.Stat(commonPlatformSrc); !os.IsNotExist(err) {
		commonPlatformDst := filepath.Join(g.BuildDir, "platform", "common")
		if err := copyDir(commonPlatformSrc, commonPlatformDst); err != nil {
			return fmt.Errorf("failed to copy common platform: %v", err)
		}
	}

	return nil
}

func (g *Generator) copyProjectAssets() error {
	projectDir := g.Analysis.ProjectDir

	// Copy assets directory if it exists
	assetsSrc := filepath.Join(projectDir, "assets")
	if _, err := os.Stat(assetsSrc); !os.IsNotExist(err) {
		assetsDst := filepath.Join(g.BuildDir, "assets")
		if err := copyDir(assetsSrc, assetsDst); err != nil {
			return fmt.Errorf("failed to copy assets: %v", err)
		}
	}

	// Copy scenes directory if it exists
	scenesSrc := filepath.Join(projectDir, "scenes")
	if _, err := os.Stat(scenesSrc); !os.IsNotExist(err) {
		scenesDst := filepath.Join(g.BuildDir, "scenes")
		if err := copyDir(scenesSrc, scenesDst); err != nil {
			return fmt.Errorf("failed to copy scenes: %v", err)
		}
	}

	// Copy objects directory if it exists
	objectsSrc := filepath.Join(projectDir, "objects")
	if _, err := os.Stat(objectsSrc); !os.IsNotExist(err) {
		objectsDst := filepath.Join(g.BuildDir, "objects")
		if err := copyDir(objectsSrc, objectsDst); err != nil {
			return fmt.Errorf("failed to copy objects: %v", err)
		}
	}

	return nil
}

func (g *Generator) copyUserComponents() error {
	for _, compFile := range g.Analysis.ComponentFiles {
		srcPath := filepath.Join(g.Analysis.ProjectDir, compFile)
		dstPath := filepath.Join(g.BuildDir, compFile)

		// Create destination directory
		dstDir := filepath.Dir(dstPath)
		if err := os.MkdirAll(dstDir, 0755); err != nil {
			return fmt.Errorf("failed to create component directory %s: %v", dstDir, err)
		}

		// Copy the file
		if err := copyFile(srcPath, dstPath); err != nil {
			return fmt.Errorf("failed to copy component %s: %v", compFile, err)
		}
	}

	return nil
}

func (g *Generator) generateMainGo() error {
	mainTemplate := `// GENERATED CODE - DO NOT EDIT
package main

import (
	"log"
	"os"
	"path/filepath"

	_ "github.com/EnesBaytekin/imge/components"
	"github.com/EnesBaytekin/imge/core"
	"github.com/EnesBaytekin/imge/platform/{{.Platform}}"
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
	game := core.NewGameWithConfig(config)
	game.SetPlatform(platform)

	// Initialize the game
	if err := game.Init(); err != nil {
		log.Fatalf("Failed to initialize game: %v", err)
	}

	// Load initial scene if specified
	if config.InitialScene != "" {
		scenePath := filepath.Join(exeDir, "scenes", config.InitialScene + ".scene")
		scene := core.NewScene(config.InitialScene)
		if err := scene.LoadFromFile(scenePath); err != nil {
			log.Printf("Warning: Could not load initial scene: %v", err)
			log.Println("Starting with empty scene")
		}
		game.AddScene(scene)
		game.SetActiveScene(config.InitialScene)
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

	mainPath := filepath.Join(g.BuildDir, "main.go")
	file, err := os.Create(mainPath)
	if err != nil {
		return err
	}
	defer file.Close()

	return tmpl.Execute(file, data)
}

func (g *Generator) generateGoMod() error {
	// Create go.mod with replace directive to use local code
	modContent := `module github.com/EnesBaytekin/imge

go 1.24

require github.com/EnesBaytekin/imge v0.1.0

replace github.com/EnesBaytekin/imge => .
`

	dstModPath := filepath.Join(g.BuildDir, "go.mod")
	return os.WriteFile(dstModPath, []byte(modContent), 0644)
}

// Helper functions for file copying
func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories for now, we'll create them as needed
		if info.IsDir() {
			return nil
		}

		// Skip test files
		if strings.HasSuffix(info.Name(), "_test.go") {
			return nil
		}

		// Calculate relative path
		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}

		dstPath := filepath.Join(dst, relPath)

		// Create destination directory
		dstDir := filepath.Dir(dstPath)
		if err := os.MkdirAll(dstDir, 0755); err != nil {
			return err
		}

		// Copy file
		return copyFile(path, dstPath)
	})
}

func copyFile(src, dst string) error {
	// Get source file info for permissions
	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		return err
	}

	// Preserve source file permissions
	return os.Chmod(dst, srcInfo.Mode())
}