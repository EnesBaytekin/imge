package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"

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
	case "help", "-h", "--help":
		printUsage()
	default:
		log.Printf("Unknown command: %s", command)
		printUsage()
		os.Exit(1)
	}
}

func handleBuild() {
	fs := flag.NewFlagSet("build", flag.ExitOnError)
	linux := fs.Bool("linux", false, "build for Linux")
	windows := fs.Bool("windows", false, "build for Windows")
	macos := fs.Bool("macos", false, "build for macOS")
	amd64 := fs.Bool("amd64", false, "build for amd64 architecture")
	arm64 := fs.Bool("arm64", false, "build for arm64 architecture")
	web := fs.Bool("web", false, "build the web (WASM) bundle")
	all := fs.Bool("all", false, "build every supported target")
	fs.Usage = printUsage
	_ = fs.Parse(os.Args[2:])

	projectDir := requireProjectDir()

	// Resolve the requested desktop platforms (OS x arch) plus whether to build web.
	var requested []build.Platform
	var wantWeb bool

	anyOS := *linux || *windows || *macos
	anyArch := *amd64 || *arm64
	posWeb := len(fs.Args()) > 0 && normalizeTarget(fs.Args()[0]) == build.TargetWeb

	switch {
	case *all:
		requested = platformCombos([]string{"linux", "darwin", "windows"}, []string{"amd64", "arm64"})
		wantWeb = true
	case !anyOS && !anyArch && !*web && !posWeb:
		// Default: native desktop for this machine.
		requested = []build.Platform{build.HostPlatform()}
	case !anyOS && !anyArch && (*web || posWeb):
		wantWeb = true
	default:
		var oses, archs []string
		if anyOS {
			if *linux {
				oses = append(oses, "linux")
			}
			if *windows {
				oses = append(oses, "windows")
			}
			if *macos {
				oses = append(oses, "darwin")
			}
		} else {
			oses = []string{"linux", "darwin", "windows"}
		}
		if anyArch {
			if *amd64 {
				archs = append(archs, "amd64")
			}
			if *arm64 {
				archs = append(archs, "arm64")
			}
		} else {
			archs = []string{"amd64", "arm64"}
		}
		requested = platformCombos(oses, archs)
		wantWeb = *web || posWeb
	}

	// Build each requested target, continuing past failures so one broken target
	// (e.g. a native Linux build on a host missing X11 dev headers) doesn't abort
	// the rest of an --all run. Targets this host fundamentally can't produce are
	// "skipped"; targets we attempted but that errored are "failed".
	built, skipped, failed := 0, 0, 0
	for _, p := range requested {
		if err := build.ValidateTarget(p.GOOS, p.GOARCH); err != nil {
			fmt.Printf("Skipping %s/%s: %v\n", p.GOOS, p.GOARCH, err)
			skipped++
			continue
		}
		b := &build.Builder{ProjectDir: projectDir, Target: build.TargetDesktop, GOOS: p.GOOS, GOARCH: p.GOARCH}
		if _, err := b.Build(); err != nil {
			fmt.Printf("Failed to build %s/%s: %v\n", p.GOOS, p.GOARCH, err)
			failed++
			continue
		}
		built++
	}
	if wantWeb {
		b := &build.Builder{ProjectDir: projectDir, Target: build.TargetWeb}
		if _, err := b.Build(); err != nil {
			fmt.Printf("Failed to build web: %v\n", err)
			failed++
		} else {
			built++
		}
	}

	fmt.Printf("\nBuild summary: %d built, %d skipped, %d failed.\n", built, skipped, failed)
	if built == 0 {
		log.Fatal("Nothing was built: none of the requested targets succeeded.")
	}
}

// platformCombos returns the cartesian product of the given OSes and
// architectures as desktop build targets.
func platformCombos(oses, archs []string) []build.Platform {
	var ps []build.Platform
	for _, o := range oses {
		for _, a := range archs {
			ps = append(ps, build.Platform{GOOS: o, GOARCH: a})
		}
	}
	return ps
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
	// `imge run` builds and runs natively. `imge run web` just builds the bundle.
	if len(os.Args) >= 3 && (os.Args[2] == "web" || os.Args[2] == "wasm") {
		handleBuild()
		return
	}

	projectDir := requireProjectDir()

	builder := &build.Builder{ProjectDir: projectDir, Target: build.TargetDesktop}
	outPath, err := builder.Build()
	if err != nil {
		log.Fatalf("Build failed: %v", err)
	}

	fmt.Printf("Running: %s\n", outPath)

	cmd := exec.Command(outPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Run(); err != nil {
		log.Fatalf("Game exited with error: %v", err)
	}
}

// requireProjectDir returns the current directory, failing if it isn't a project.
func requireProjectDir() string {
	projectDir, err := os.Getwd()
	if err != nil {
		log.Fatalf("Failed to get current directory: %v", err)
	}
	if _, err := os.Stat(filepath.Join(projectDir, "game.json")); os.IsNotExist(err) {
		log.Fatal("No game.json found in this directory. Run `imge init` first.")
	}
	return projectDir
}

func handleInit() {
	entries, err := os.ReadDir(".")
	if err != nil {
		log.Fatalf("Failed to read current directory: %v", err)
	}
	if len(entries) > 0 {
		fmt.Fprintln(os.Stderr, "Refusing to initialize: this directory is not empty.")
		fmt.Fprintln(os.Stderr, "Run `imge init` in an empty directory so it doesn't overwrite existing files.")
		os.Exit(1)
	}

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

// SpriteComponent draws a colored rectangle. Exported fields are "export
// variables": the object file's args inject into them by JSON key.
type SpriteComponent struct {
	core.BaseComponent
	Width  float64    ` + "`json:\"width\"`" + `
	Height float64    ` + "`json:\"height\"`" + `
	Color  math.Color ` + "`json:\"color\"`" + `
}

// Initialize applies defaults for any args that weren't provided.
func (c *SpriteComponent) Initialize() {
	if c.Width <= 0 {
		c.Width = 32
	}
	if c.Height <= 0 {
		c.Height = 32
	}
	if c.Color == (math.Color{}) {
		c.Color = math.White
	}
}

func (c *SpriteComponent) Draw(renderer core.Renderer) {
	owner := c.GetOwner()
	if owner == nil {
		return
	}
	renderer.DrawRect(
		math.NewRect(owner.Transform.Position.X, owner.Transform.Position.Y, c.Width, c.Height),
		c.Color,
	)
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

// PlayerComponent moves its owner with WASD and flashes red briefly when it
// collides with an enemy.
type PlayerComponent struct {
	core.BaseComponent
	invincible float64
}

// Initialize registers a handler for the @Movement component's "blocked_collision"
// event. Events are delivered with Emit(name, data) / On(name, handler).
func (c *PlayerComponent) Initialize() {
	c.On("blocked_collision", func(data any) {
		if c.invincible > 0 {
			return
		}
		other, ok := data.(*core.Object)
		if !ok || other == nil || !other.HasTag("enemy") {
			return
		}
		c.invincible = 0.3
	})
}

func (c *PlayerComponent) Update(ctx *core.Context) {
	if c.invincible > 0 {
		c.invincible -= ctx.DeltaTime()
	}

	speed := 200.0
	dt := ctx.DeltaTime()
	var dx, dy float64

	if ctx.Input.IsKeyPressed(core.KeyW) || ctx.Input.IsKeyPressed(core.KeyUp) { dy = -speed * dt }
	if ctx.Input.IsKeyPressed(core.KeyS) || ctx.Input.IsKeyPressed(core.KeyDown) { dy = speed * dt }
	if ctx.Input.IsKeyPressed(core.KeyA) || ctx.Input.IsKeyPressed(core.KeyLeft) { dx = -speed * dt }
	if ctx.Input.IsKeyPressed(core.KeyD) || ctx.Input.IsKeyPressed(core.KeyRight) { dx = speed * dt }

	if m := core.Get[*MovementComponent](c); m != nil {
		m.Move(dx, dy)
	}
}

func (c *PlayerComponent) Draw(renderer core.Renderer) {
	owner := c.GetOwner()
	if owner == nil {
		return
	}
	if c.invincible > 0 && int(c.invincible*60)%4 < 2 {
		renderer.DrawRectOutline(
			math.NewRect(owner.Transform.Position.X, owner.Transform.Position.Y, 32, 32),
			math.NewColor(255, 0, 0, 255), 2,
		)
	}
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

// EnemyComponent chases the nearest object tagged "player".
type EnemyComponent struct {
	core.BaseComponent
	Speed float64 ` + "`json:\"speed\"`" + `
}

// Initialize applies defaults for any args that weren't provided.
func (c *EnemyComponent) Initialize() {
	if c.Speed <= 0 {
		c.Speed = 60
	}
}

func (c *EnemyComponent) Update(ctx *core.Context) {
	owner := c.GetOwner()
	if owner == nil || owner.Scene == nil {
		return
	}

	players := owner.Scene.FindObjectsWithTag("player")
	if len(players) == 0 {
		return
	}
	player := players[0]

	dir := player.Transform.Position.Subtract(owner.Transform.Position)
	dist := dir.Length()
	if dist < 2 {
		return
	}

	dir = dir.Divide(dist)
	dt := ctx.DeltaTime()
	moveX := dir.X * c.Speed * dt
	moveY := dir.Y * c.Speed * dt

	if m := core.Get[*MovementComponent](c); m != nil {
		m.Move(moveX, 0)
		m.Move(0, moveY)
	}
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
	fmt.Printf("IMGE Minimal Game Engine CLI Tool (version %s)\n", imge.EngineVersion)
	fmt.Println("Usage:")
	fmt.Println("  imge init                 Initialize a new game project (empty directory only)")
	fmt.Println("  imge build [flags]        Build the game")
	fmt.Println("  imge run                  Build and run natively (desktop)")
	fmt.Println("  imge version              Show engine version")
	fmt.Println("  imge help                 Show this help")
	fmt.Println("")
	fmt.Println("Build flags (combine freely):")
	fmt.Println("  --linux --windows --macos   target OS (omit with --arch to target every OS)")
	fmt.Println("  --amd64 --arm64             target architecture (omit to build both)")
	fmt.Println("  --web                       build the web (WASM) bundle")
	fmt.Println("  --all                       build every supported target")
	fmt.Println("")
	fmt.Println("Examples:")
	fmt.Println("  imge build                    native desktop for this machine")
	fmt.Println("  imge build --windows          Windows amd64 + arm64")
	fmt.Println("  imge build --windows --amd64  Windows amd64 only")
	fmt.Println("  imge build --amd64            amd64 for every OS (linux, macos, windows)")
	fmt.Println("  imge build --web              web bundle only")
	fmt.Println("  imge build --windows --web    Windows (both archs) + web")
	fmt.Println("  imge build --all              every buildable target (skips what it can't)")
	fmt.Println("")
	fmt.Println("Output goes to imge_build/<name>_<os>-<arch> (web to imge_build/web/).")
	fmt.Println("Cross-compilation:")
	fmt.Println("  Windows (amd64/arm64) works from any host (pure Go).")
	fmt.Println("  macOS and non-native Linux can't be cross-compiled (Ebitengine uses Cgo there);")
	fmt.Println("  build those natively or via CI (GitHub Actions has macOS + ARM runners).")
}
