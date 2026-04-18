package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/EnesBaytekin/imge/build"
)

const version = "0.1.0"

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "build":
		handleBuild()
	case "run":
		handleRun()
	case "init":
		handleInit()
	case "version":
		fmt.Printf("imge version %s\n", version)
	default:
		log.Printf("Unknown command: %s", command)
		printUsage()
		os.Exit(1)
	}
}

func handleBuild() {
	if len(os.Args) < 3 {
		log.Fatal("Usage: imge build <platform> [--clean]")
	}
	platform := os.Args[2]

	// Parse flags
	cleanBuild := false
	for _, arg := range os.Args[3:] {
		if arg == "--clean" {
			cleanBuild = true
		}
	}

	// Validate platform
	validPlatforms := map[string]bool{
		"mock":    true,  // Implemented
		"sdl":     false, // Not implemented yet
		"web":     false, // Not implemented yet
		"desktop": false, // Not implemented yet
	}

	if !validPlatforms[platform] {
		log.Fatalf("Invalid platform: %s. Valid platforms: mock", platform)
	}

	// Check if platform is implemented
	if platform != "mock" {
		log.Fatalf("Platform %s is not implemented yet", platform)
	}

	fmt.Printf("Building for platform: %s\n", platform)
	if cleanBuild {
		fmt.Println("Clean build enabled (cache will be cleared)")
	}

	// Get current directory as project directory
	projectDir, err := os.Getwd()
	if err != nil {
		log.Fatalf("Failed to get current directory: %v", err)
	}

	// Create build directory
	buildDir := filepath.Join(projectDir, ".imge_build")
	if cleanBuild {
		// Clean build directory before starting
		if err := os.RemoveAll(buildDir); err != nil {
			log.Fatalf("Failed to clean build directory: %v", err)
		}
	}
	if err := os.MkdirAll(buildDir, 0755); err != nil {
		log.Fatalf("Failed to create build directory: %v", err)
	}
	// Clean up build directory after build if cleanBuild is enabled
	if cleanBuild {
		defer func() {
			// Clean up build directory after build
			if err := os.RemoveAll(buildDir); err != nil {
				log.Printf("Warning: Failed to clean build directory: %v", err)
			}
		}()
	}

	// Determine output name (game or game.exe)
	outputName := "game"
	if platform == "windows" || strings.Contains(platform, "win") {
		outputName = "game.exe"
	}

	// Engine code will be fetched from GitHub via go modules
	// No local engine source needed
	engineSource := ""

	// Create and execute builder
	builder := &build.Builder{
		ProjectDir:   projectDir,
		BuildDir:     buildDir,
		Platform:     platform,
		OutputName:   outputName,
		EngineSource: engineSource,
	}

	if err := builder.Build(); err != nil {
		log.Fatalf("Build failed: %v", err)
	}

	fmt.Println("Build completed successfully!")
}

func handleRun() {
	if len(os.Args) < 3 {
		log.Fatal("Usage: imge run <platform>")
	}
	platform := os.Args[2]

	fmt.Printf("Building and running for platform: %s\n", platform)

	// For now, just call handleBuild
	handleBuild()
	// TODO: After build, run the executable
	log.Println("Run command not implemented yet")
}

func handleInit() {
	fmt.Println("Initializing new IMGE game project...")

	// Create directory structure
	dirs := []string{
		"components",
		"assets",
		"scenes",
		"objects",
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			log.Fatalf("Failed to create directory %s: %v", dir, err)
		}
		fmt.Printf("Created directory: %s/\n", dir)
	}

	// Create game.json
	gameJSON := `{
  "name": "My Game",
  "version": "1.0.0",
  "window": {
    "title": "My IMGE Game",
    "width": 800,
    "height": 600
  },
  "game": {
    "target_fps": 60,
    "initial_scene": "main"
  }
}`

	if err := os.WriteFile("game.json", []byte(gameJSON), 0644); err != nil {
		log.Fatalf("Failed to create game.json: %v", err)
	}
	fmt.Println("Created file: game.json")

	// Create sample component
	sampleComponent := `package components

import "github.com/EnesBaytekin/imge/core"

// PlayerComponent is a sample component
type PlayerComponent struct {
	core.BaseComponent
	speed float64
}

func (c *PlayerComponent) Initialize(args []interface{}) error {
	c.speed = 200.0 // default speed
	if len(args) > 0 {
		if argMap, ok := args[0].(map[string]interface{}); ok {
			if speed, ok := argMap["speed"].(float64); ok {
				c.speed = speed
			}
		}
	}
	return nil
}

func (c *PlayerComponent) Update(deltaTime float64) {
	// TODO: Implement player movement
}

func (c *PlayerComponent) Draw(renderer core.Renderer) {
	// No drawing needed for this component
}

func init() {
	core.RegisterComponent("components/player.go", func(args []interface{}) (core.Component, error) {
		comp := &PlayerComponent{}
		if err := comp.Initialize(args); err != nil {
			return nil, err
		}
		return comp, nil
	})
}`

	if err := os.WriteFile("components/player.go", []byte(sampleComponent), 0644); err != nil {
		log.Fatalf("Failed to create components/player.go: %v", err)
	}
	fmt.Println("Created file: components/player.go")

	// Create sample scene
	sampleScene := `{
  "name": "main",
  "background_color": "#000000",
  "objects": [
    {
      "file": "objects/player.obj",
      "transform": {
        "position": { "x": 100, "y": 100 }
      }
    },
    {
      "file": "objects/enemy.obj",
      "transform": {
        "position": { "x": 400, "y": 300 }
      }
    }
  ]
}`
	if err := os.WriteFile("scenes/main.scene", []byte(sampleScene), 0644); err != nil {
		log.Fatalf("Failed to create scenes/main.scene: %v", err)
	}
	fmt.Println("Created file: scenes/main.scene")

	// Create sample player object
	samplePlayerObj := `{
  "name": "Player",
  "depth": 1,
  "components": [
    {
      "kind": "@Sprite",
      "name": "sprite",
      "args": {
        "width": 40,
        "height": 40,
        "color": { "r": 0, "g": 255, "b": 0, "a": 255 }
      }
    },
    {
      "kind": "components/player.go",
      "name": "movement",
      "args": {
        "speed": 250
      }
    }
  ],
  "tags": ["player"]
}`
	if err := os.WriteFile("objects/player.obj", []byte(samplePlayerObj), 0644); err != nil {
		log.Fatalf("Failed to create objects/player.obj: %v", err)
	}
	fmt.Println("Created file: objects/player.obj")

	// Create sample enemy object
	sampleEnemyObj := `{
  "name": "Enemy",
  "depth": 0,
  "components": [
    {
      "kind": "@Sprite",
      "name": "sprite",
      "args": {
        "width": 30,
        "height": 30,
        "color": { "r": 255, "g": 0, "b": 0, "a": 255 }
      }
    }
  ],
  "tags": ["enemy"]
}`
	if err := os.WriteFile("objects/enemy.obj", []byte(sampleEnemyObj), 0644); err != nil {
		log.Fatalf("Failed to create objects/enemy.obj: %v", err)
	}
	fmt.Println("Created file: objects/enemy.obj")

	fmt.Println("\nProject initialized successfully!")
	fmt.Println("Next steps:")
	fmt.Println("  1. Edit game.json to configure your game")
	fmt.Println("  2. Add components in the components/ directory")
	fmt.Println("  3. Add assets in the assets/ directory")
	fmt.Println("  4. Run: imge build mock")
}

func printUsage() {
	fmt.Println("IMGE Game Engine CLI Tool")
	fmt.Println("Usage:")
	fmt.Println("  imge build <platform>    Build game for specified platform")
	fmt.Println("  imge run <platform>      Build and run game")
	fmt.Println("  imge init                Initialize new game project")
	fmt.Println("  imge version             Show version")
	fmt.Println("")
	fmt.Println("Platforms:")
	fmt.Println("  mock    - Mock platform (debug output only)")
	fmt.Println("  sdl     - SDL platform (not implemented)")
	fmt.Println("  web     - Web platform (not implemented)")
	fmt.Println("  desktop - Desktop platform (not implemented)")
}