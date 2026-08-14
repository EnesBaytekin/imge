package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

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
	// Target is optional and defaults to a native desktop build.
	target := build.TargetDesktop
	if len(os.Args) >= 3 {
		target = normalizeTarget(os.Args[2])
	}

	projectDir, err := os.Getwd()
	if err != nil {
		log.Fatalf("Failed to get current directory: %v", err)
	}

	// Building requires an existing project with a game.json.
	if _, err := os.Stat(filepath.Join(projectDir, "game.json")); os.IsNotExist(err) {
		log.Fatal("No game.json found in this directory. Run `imge init` first.")
	}

	outputName := "game"
	if runtime.GOOS == "windows" {
		outputName = "game.exe"
	}

	if target == build.TargetWeb {
		fmt.Println("Building for web (WebAssembly)...")
	} else {
		fmt.Printf("Building with Ebitengine (single executable: %s)...\n", outputName)
	}

	builder := &build.Builder{
		ProjectDir: projectDir,
		Target:     target,
		OutputName: outputName,
	}
	if err := builder.Build(); err != nil {
		log.Fatalf("Build failed: %v", err)
	}

	fmt.Println("Build completed successfully!")
}

// normalizeTarget maps a user-provided target string to a build target.
func normalizeTarget(s string) string {
	switch s {
	case "web", "wasm":
		return build.TargetWeb
	case "desktop", "ebitengine":
		return build.TargetDesktop
	default:
		log.Fatalf("Invalid target %q. Supported: desktop (default), web", s)
		return ""
	}
}

func handleRun() {
	// Determine target (defaults to desktop).
	target := build.TargetDesktop
	if len(os.Args) >= 3 {
		target = normalizeTarget(os.Args[2])
	}

	// Web builds can't be launched directly — just build and print the serve hint.
	if target == build.TargetWeb {
		handleBuild()
		return
	}

	// Build first (exits on failure).
	handleBuild()

	outputName := "game"
	if runtime.GOOS == "windows" {
		outputName = "game.exe"
	}

	projectDir, err := os.Getwd()
	if err != nil {
		log.Fatalf("Failed to get current directory: %v", err)
	}
	exePath := filepath.Join(projectDir, outputName)

	// Verify the executable was produced.
	if _, err := os.Stat(exePath); os.IsNotExist(err) {
		log.Fatalf("Build output not found: %s", exePath)
	}

	fmt.Printf("Running: %s\n", exePath)

	// Run the executable with the current process's environment.
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

	if m, ok := owner.GetComponentByKind("@Movement").(*MovementComponent); ok {
		m.Move(dx, dy)
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

	if m, ok := owner.GetComponentByKind("@Movement").(*MovementComponent); ok {
		m.Move(moveX, 0)
		m.Move(0, moveY)
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
      "args": {}
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
      "args": {}
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
	fmt.Println("  1. Build and run: imge run")
	fmt.Println("  2. Move with WASD — enemies chase you")
	fmt.Println("  3. Edit components/ to customize behavior")
	fmt.Println("  4. Edit scenes/ and objects/ to change the game world")
}

func printUsage() {
	fmt.Println("IMGE Minimal Game Engine CLI Tool")
	fmt.Println("Usage:")
	fmt.Println("  imge init                 Initialize a new game project")
	fmt.Println("  imge build [target]       Build the game")
	fmt.Println("  imge run [target]         Build and run the game (desktop only)")
	fmt.Println("  imge version              Show version")
	fmt.Println("")
	fmt.Println("Targets:")
	fmt.Println("  desktop     - Native executable (default)")
	fmt.Println("  web         - WebAssembly bundle (web/)")
}
