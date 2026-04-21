// Package core contains platform-agnostic game engine logic.
// This file defines the main Game engine and its lifecycle.
package core

import "github.com/EnesBaytekin/imge/core/math"

// ============================================================================
// Configuration
// ============================================================================

// Config holds game configuration settings.
type Config struct {
	// Window settings
	WindowWidth  int
	WindowHeight int
	WindowTitle  string

	// Game settings
	TargetFPS   int
	FixedUpdate bool

	// Scene settings
	InitialScene string
}

// DefaultConfig returns a default configuration.
func DefaultConfig() Config {
	return Config{
		WindowWidth:  800,
		WindowHeight: 600,
		WindowTitle:  "IMGE Game",
		TargetFPS:    60,
		FixedUpdate:  false,
		InitialScene: "",
	}
}

// ============================================================================
// Game Engine
// ============================================================================

// Game is the main game engine struct.
type Game struct {
	// Platform implementations (injected via dependency injection)
	platform Platform

	// Configuration
	config Config

	// Scene management
	scenes      map[string]*Scene
	activeScene *Scene

	// Game state
	running    bool
	initialized bool
}

// NewGame creates a new game instance with default configuration.
func NewGame() *Game {
	return &Game{
		platform:    nil,
		config:      DefaultConfig(),
		scenes:      make(map[string]*Scene),
		activeScene: nil,
		running:     false,
		initialized: false,
	}
}

// NewGameWithConfig creates a new game instance with custom configuration.
func NewGameWithConfig(config Config) *Game {
	return &Game{
		platform:    nil,
		config:      config,
		scenes:      make(map[string]*Scene),
		activeScene: nil,
		running:     false,
		initialized: false,
	}
}

// ============================================================================
// Platform Dependency Injection
// ============================================================================

// SetPlatform sets the platform implementations for the game.
// Must be called before Init().
func (g *Game) SetPlatform(platform Platform) {
	g.platform = platform
}

// ============================================================================
// Scene Management
// ============================================================================

// AddScene adds a scene to the game.
func (g *Game) AddScene(scene *Scene) {
	g.scenes[scene.Name] = scene

	// If this is the first scene or matches initial scene config, set as active
	if g.activeScene == nil || scene.Name == g.config.InitialScene {
		g.activeScene = scene
	}
}

// GetScene returns a scene by name, or nil if not found.
func (g *Game) GetScene(name string) *Scene {
	return g.scenes[name]
}

// SetActiveScene sets the active scene by name.
// Returns false if the scene doesn't exist.
func (g *Game) SetActiveScene(name string) bool {
	scene, exists := g.scenes[name]
	if !exists {
		return false
	}

	g.activeScene = scene
	return true
}

// GetActiveScene returns the currently active scene.
func (g *Game) GetActiveScene() *Scene {
	return g.activeScene
}

// ============================================================================
// Lifecycle Methods
// ============================================================================

// Init initializes the game engine.
// Must be called after SetPlatform() and before Run().
func (g *Game) Init() error {
	if g.platform == nil {
		return &GameError{Stage: "Init", Reason: "platform not set"}
	}

	// Initialize platform (creates window, renderer, audio, etc.)
	if err := g.platform.Init(g.config.WindowTitle, g.config.WindowWidth, g.config.WindowHeight); err != nil {
		return &GameError{Stage: "Init", Reason: "platform initialization failed: " + err.Error()}
	}

	g.initialized = true
	return nil
}

// Run starts the main game loop.
// Blocks until the game exits.
func (g *Game) Run() error {
	if !g.initialized {
		return &GameError{Stage: "Run", Reason: "game not initialized"}
	}

	g.running = true

	// Main game loop
	for g.running {
		// Handle window events
		if g.platform.Window().ShouldClose() {
			g.running = false
			break
		}

		// Update input state
		g.platform.Input().Update()

		// Update platform state (e.g., poll events)
		g.platform.Update()

		// Create component context with engine services
		ctx := &ComponentContext{
			Input: g.platform.Input(),
			Audio: g.platform.Audio(),
			Time:  g.platform.Time(),
		}

		// Update game logic
		g.Update(ctx)

		// Begin rendering
		g.platform.Renderer().Clear(math.Black)

		// Draw game
		g.Draw()

		// Present rendered frame
		g.platform.Renderer().Present()

		// Tick time (advance frame)
		g.platform.Time().Tick()
	}

	return g.Shutdown()
}

// Update updates game logic for the current frame.
func (g *Game) Update(ctx *ComponentContext) {
	if g.activeScene != nil {
		g.activeScene.Update(ctx)
	}
}

// Draw renders the game for the current frame.
func (g *Game) Draw() {
	if g.activeScene != nil {
		g.activeScene.Draw(g.platform.Renderer())
	}
}

// Shutdown cleans up resources and shuts down the game.
func (g *Game) Shutdown() error {
	if g.platform != nil && g.platform.Window() != nil {
		g.platform.Window().Destroy()
	}

	g.running = false
	g.initialized = false

	return nil
}

// ============================================================================
// Utility Methods
// ============================================================================

// IsRunning returns true if the game is currently running.
func (g *Game) IsRunning() bool {
	return g.running
}

// Stop gracefully stops the game loop.
func (g *Game) Stop() {
	g.running = false
}

// ============================================================================
// Error Handling
// ============================================================================

// GameError represents an error that occurred during game operation.
type GameError struct {
	Stage  string
	Reason string
}

func (e *GameError) Error() string {
	return "game error [" + e.Stage + "]: " + e.Reason
}