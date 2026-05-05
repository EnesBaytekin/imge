package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/EnesBaytekin/imge"
	"github.com/EnesBaytekin/imge/build"
)

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
		fmt.Printf("imge version %s\n", imge.EngineVersion)
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
	useDocker := false
	for _, arg := range os.Args[3:] {
		if arg == "--clean" {
			cleanBuild = true
		} else if arg == "--docker" {
			useDocker = true
		}
	}

	if useDocker && platform != "sdl" {
		log.Fatal("--docker is only supported for sdl platform")
	}

	// Validate platform
	validPlatforms := map[string]bool{
		"mock":    true,  // Implemented
		"sdl":     true,  // Being implemented
		"web":     false, // Not implemented yet
		"desktop": false, // Not implemented yet
	}

	if !validPlatforms[platform] {
		log.Fatalf("Invalid platform: %s. Valid platforms: mock, sdl", platform)
	}

	// Check if platform is implemented
	if platform != "mock" && platform != "sdl" {
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
		UseDocker:    useDocker,
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

	// Build first (exits on failure)
	handleBuild()

	// Determine executable path
	outputDir := fmt.Sprintf("imge_build_%s", platform)
	exeName := "game"
	if platform == "windows" || strings.Contains(platform, "win") {
		exeName = "game.exe"
	}
	exePath := filepath.Join(outputDir, exeName)

	// Verify executable exists
	if _, err := os.Stat(exePath); os.IsNotExist(err) {
		log.Fatalf("Build output not found: %s", exePath)
	}

	fmt.Printf("Running: %s\n", exePath)

	// Run the executable with the current process's environment
	cmd := exec.Command(exePath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Run(); err != nil {
		log.Fatalf("Game exited with error: %v", err)
	}
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

	// Create sprite component — colored rectangle renderer
	spriteComponent := `package components

import (
	"github.com/EnesBaytekin/imge/core"
	"github.com/EnesBaytekin/imge/core/math"
)

type SpriteComponent struct {
	core.BaseComponent
	width  float64
	height float64
	color  math.Color
}

func (c *SpriteComponent) Initialize(args []interface{}) error {
	c.width = 32
	c.height = 32
	c.color = math.White

	if len(args) > 0 {
		if argMap, ok := args[0].(map[string]interface{}); ok {
			if w, ok := argMap["width"].(float64); ok { c.width = w }
			if h, ok := argMap["height"].(float64); ok { c.height = h }
			if colorMap, ok := argMap["color"].(map[string]interface{}); ok {
				if r, ok := colorMap["r"].(float64); ok { c.color.R = uint8(r) }
				if g, ok := colorMap["g"].(float64); ok { c.color.G = uint8(g) }
				if b, ok := colorMap["b"].(float64); ok { c.color.B = uint8(b) }
				if a, ok := colorMap["a"].(float64); ok { c.color.A = uint8(a) }
			}
		}
	}
	return nil
}

func (c *SpriteComponent) Draw(renderer core.Renderer) {
	owner := c.GetOwner()
	if owner == nil { return }
	renderer.DrawRect(
		math.NewRect(owner.Transform.Position.X, owner.Transform.Position.Y, c.width, c.height),
		c.color,
	)
}

func init() {
	core.RegisterComponent("components/sprite.go", func(args []interface{}) (core.Component, error) {
		comp := &SpriteComponent{}
		if err := comp.Initialize(args); err != nil { return nil, err }
		return comp, nil
	})
}`

	if err := os.WriteFile("components/sprite.go", []byte(spriteComponent), 0644); err != nil {
		log.Fatalf("Failed to create components/sprite.go: %v", err)
	}
	fmt.Println("Created file: components/sprite.go")

	// Create player component — WASD + @Movement + enemy collision detection
	playerComponent := `package components

import (
	"github.com/EnesBaytekin/imge/core"
	"github.com/EnesBaytekin/imge/core/math"
)

type PlayerComponent struct {
	core.BaseComponent
	invincible float64
}

func (c *PlayerComponent) SubscribeEvents() {
	scene := core.GetSceneFromComponent(c)
	if scene != nil && scene.EventManager != nil {
		scene.EventManager.Subscribe(c, "blocked_collision")
	}
}

func (c *PlayerComponent) Pong(event *core.Event, ctx *core.ComponentContext) {
	if event.Name != "blocked_collision" || c.invincible > 0 { return }
	otherObj, ok := event.Data.(*core.Object)
	if !ok || otherObj == nil { return }
	if otherObj.HasTag("enemy") {
		c.invincible = 0.3
	}
}

func (c *PlayerComponent) Update(ctx *core.ComponentContext) {
	owner := c.GetOwner()
	if owner == nil { return }

	if c.invincible > 0 {
		c.invincible -= ctx.Time.DeltaTime()
	}

	dt := ctx.Time.DeltaTime()
	speed := 200.0
	var dx, dy float64

	if ctx.Input.IsKeyPressed(core.KeyW) || ctx.Input.IsKeyPressed(core.KeyUp) { dy = -speed * dt }
	if ctx.Input.IsKeyPressed(core.KeyS) || ctx.Input.IsKeyPressed(core.KeyDown) { dy = speed * dt }
	if ctx.Input.IsKeyPressed(core.KeyA) || ctx.Input.IsKeyPressed(core.KeyLeft) { dx = -speed * dt }
	if ctx.Input.IsKeyPressed(core.KeyD) || ctx.Input.IsKeyPressed(core.KeyRight) { dx = speed * dt }

	if mover := owner.GetComponentByKind("@Movement"); mover != nil {
		if m, ok := mover.(interface{ Move(dx, dy float64) bool }); ok {
			m.Move(dx, dy)
		}
	}
}

func (c *PlayerComponent) Draw(renderer core.Renderer) {
	owner := c.GetOwner()
	if owner == nil { return }
	if c.invincible > 0 && int(c.invincible*60)%4 < 2 {
		renderer.DrawRectOutline(
			math.NewRect(owner.Transform.Position.X, owner.Transform.Position.Y, 32, 32),
			math.NewColor(255, 0, 0, 255), 2,
		)
	}
}

func init() {
	core.RegisterComponent("components/player.go", func(args []interface{}) (core.Component, error) {
		return &PlayerComponent{}, nil
	})
}`

	if err := os.WriteFile("components/player.go", []byte(playerComponent), 0644); err != nil {
		log.Fatalf("Failed to create components/player.go: %v", err)
	}
	fmt.Println("Created file: components/player.go")

	// Create enemy component — chases the player
	enemyComponent := `package components

import (
	"github.com/EnesBaytekin/imge/core"
)

type EnemyComponent struct {
	core.BaseComponent
	speed float64
}

func (c *EnemyComponent) Initialize(args []interface{}) error {
	c.speed = 60
	if len(args) > 0 {
		if argMap, ok := args[0].(map[string]interface{}); ok {
			if s, ok := argMap["speed"].(float64); ok { c.speed = s }
		}
	}
	return nil
}

func (c *EnemyComponent) Update(ctx *core.ComponentContext) {
	owner := c.GetOwner()
	if owner == nil || owner.Scene == nil { return }

	players := owner.Scene.FindObjectsWithTag("player")
	if len(players) == 0 { return }
	player := players[0]

	dt := ctx.Time.DeltaTime()
	dir := player.Transform.Position.Subtract(owner.Transform.Position)
	dist := dir.Length()
	if dist < 2 { return }

	dir = dir.Divide(dist)
	moveX := dir.X * c.speed * dt
	moveY := dir.Y * c.speed * dt

	if mover := owner.GetComponentByKind("@Movement"); mover != nil {
		if m, ok := mover.(interface{ Move(dx, dy float64) bool }); ok {
			m.Move(moveX, 0)
			m.Move(0, moveY)
		}
	}
}

func init() {
	core.RegisterComponent("components/enemy.go", func(args []interface{}) (core.Component, error) {
		comp := &EnemyComponent{}
		if err := comp.Initialize(args); err != nil { return nil, err }
		return comp, nil
	})
}`

	if err := os.WriteFile("components/enemy.go", []byte(enemyComponent), 0644); err != nil {
		log.Fatalf("Failed to create components/enemy.go: %v", err)
	}
	fmt.Println("Created file: components/enemy.go")

	// Create sample scene
	sampleScene := `{
  "name": "main",
  "background_color": "#000000",
  "objects": [
    {
      "file": "objects/player.obj",
      "transform": {
        "position": { "x": 200, "y": 300 }
      }
    },
    {
      "file": "objects/enemy.obj",
      "transform": {
        "position": { "x": 500, "y": 200 }
      }
    },
    {
      "file": "objects/enemy.obj",
      "transform": {
        "position": { "x": 400, "y": 400 }
      }
    }
  ]
}`

	if err := os.WriteFile("scenes/main.scene", []byte(sampleScene), 0644); err != nil {
		log.Fatalf("Failed to create scenes/main.scene: %v", err)
	}
	fmt.Println("Created file: scenes/main.scene")

	// Create player object — uses built-in @Hitbox, @Movement + user components
	samplePlayerObj := `{
  "name": "Player",
  "depth": 1,
  "components": [
    {
      "kind": "@Hitbox",
      "name": "hitbox",
      "args": {
        "width": 32,
        "height": 32
      }
    },
    {
      "kind": "@Movement",
      "name": "movement",
      "args": {
        "speed": 200
      }
    },
    {
      "kind": "components/sprite.go",
      "name": "sprite",
      "args": {
        "width": 32,
        "height": 32,
        "color": { "r": 0, "g": 255, "b": 0, "a": 255 }
      }
    },
    {
      "kind": "components/player.go",
      "name": "player",
      "args": {}
    }
  ],
  "tags": ["player"]
}`

	if err := os.WriteFile("objects/player.obj", []byte(samplePlayerObj), 0644); err != nil {
		log.Fatalf("Failed to create objects/player.obj: %v", err)
	}
	fmt.Println("Created file: objects/player.obj")

	// Create enemy object — uses built-in @Hitbox, @Movement + user components
	sampleEnemyObj := `{
  "name": "Enemy",
  "depth": 0,
  "components": [
    {
      "kind": "@Hitbox",
      "name": "hitbox",
      "args": {
        "width": 32,
        "height": 32
      }
    },
    {
      "kind": "@Movement",
      "name": "movement",
      "args": {
        "speed": 60
      }
    },
    {
      "kind": "components/sprite.go",
      "name": "sprite",
      "args": {
        "width": 32,
        "height": 32,
        "color": { "r": 255, "g": 50, "b": 50, "a": 255 }
      }
    },
    {
      "kind": "components/enemy.go",
      "name": "enemy",
      "args": {
        "speed": 60
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
	fmt.Println("  1. Build and run: imge build sdl")
	fmt.Println("  2. Move with WASD — enemies chase you")
	fmt.Println("  3. Edit components/ to customize behavior")
	fmt.Println("  4. Edit scenes/ and objects/ to change the game world")
}

func printUsage() {
	fmt.Println("IMGE Minimal Game Engine CLI Tool")
	fmt.Println("Usage:")
	fmt.Println("  imge build <platform>    Build game for specified platform")
	fmt.Println("  imge run <platform>      Build and run game")
	fmt.Println("  imge init                Initialize new game project")
	fmt.Println("  imge version             Show version")
	fmt.Println("")
	fmt.Println("Platforms:")
	fmt.Println("  mock    - Mock platform (debug output only)")
	fmt.Println("  sdl     - SDL platform")
	fmt.Println("  web     - Web platform (not implemented)")
}
