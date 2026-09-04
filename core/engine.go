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
	Window WindowConfig

	// Game settings
	TargetFPS   int
	FixedUpdate bool

	// Scene settings
	InitialScene string
}

// DefaultConfig returns a default configuration.
func DefaultConfig() Config {
	return Config{
		Window: WindowConfig{
			Title:  "IMGE Game",
			Width:  640,
			Height: 360,
			// PixelPerUnit 1 = pixel-perfect: one framebuffer pixel per unit.
			PixelPerUnit: 1,
			// SmoothShapes defaults to false: vector shapes render chunky.
			SmoothShapes: false,
			// Fullscreen defaults to false: the window opens at the largest
			// integer scale fitting the screen and is locked there, toggling only
			// between windowed and fullscreen. Resizable defaults to false (fixed
			// size); Scale 0 = auto-fit.
			Fullscreen: false,
			Resizable:  false,
			Scale:      0,
		},
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

	// pendingScene queues a scene switch to apply at the start of the next frame
	// (see SwitchScene), so a scene's objects never draw before their Initialize
	// has run.
	pendingScene string

	// Game state
	running     bool
	initialized bool

	// closeHandler, when set, intercepts the OS window-close button: on a close
	// request it returns true to quit or false to keep running (e.g. an editor
	// prompting to save). nil means "close immediately".
	closeHandler func() bool

	// terminate is set when the game should end; the platform loop polls
	// ShouldTerminate and exits (returning a regular termination).
	terminate bool
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

// SwitchScene queues a scene change to take effect at the start of the next
// frame. Deferring avoids drawing the new scene before its objects' Initialize()
// has run (which is what would happen with an immediate SetActiveScene from
// inside a component's Update). Components reach this via ctx.Game.SwitchScene.
// Returns false if the scene doesn't exist.
func (g *Game) SwitchScene(name string) bool {
	if _, exists := g.scenes[name]; !exists {
		return false
	}
	g.pendingScene = name
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
	if err := g.platform.Init(g.config.Window); err != nil {
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
		ctx := &Context{
			Input:    g.platform.Input(),
			Audio:    g.platform.Audio(),
			Time:     g.platform.Time(),
			Renderer: g.platform.Renderer(),
		}

		// Update game logic
		g.Update(ctx)

		// Draw game (clears with the active scene's background color)
		g.Draw()

		// Present rendered frame
		g.platform.Renderer().Present()

		// Tick time (advance frame)
		g.platform.Time().Tick()
	}

	return g.Shutdown()
}

// Update updates game logic for the current frame.
func (g *Game) Update(ctx *Context) {
	// Expose the game to components so they can switch scenes.
	ctx.Game = g

	// Handle an OS window-close request first. When one is pending and the game is
	// about to quit, skip this frame's scene update.
	if g.handleWindowClose() {
		return
	}

	// Apply a queued scene switch before this frame's update so the new scene's
	// objects initialize before they are ever drawn.
	if g.pendingScene != "" {
		if scene, ok := g.scenes[g.pendingScene]; ok {
			g.activeScene = scene
		}
		g.pendingScene = ""
	}

	if g.activeScene != nil {
		g.activeScene.Update(ctx)
	}
}

// handleWindowClose intercepts an OS window-close request. It returns true when the
// game is terminating this frame (the caller should skip the scene update). Without a
// close handler the default is to quit immediately; with one, the handler decides —
// true quits, false keeps the window open (the close request is reported for a single
// frame, so returning false cleanly cancels it).
func (g *Game) handleWindowClose() bool {
	if g.platform == nil || g.platform.Window() == nil {
		return false
	}
	if !g.platform.Window().ShouldClose() {
		return false
	}
	if g.closeHandler == nil || g.closeHandler() {
		g.terminate = true
		return true
	}
	return false
}

// SetCloseHandler registers a close interceptor and switches the window into
// close-handled mode. Pass nil to restore the default (close immediately). The
// handler runs on each OS close request; return true to quit, false to keep running.
func (g *Game) SetCloseHandler(h func() bool) {
	g.closeHandler = h
	if w := g.platform.Window(); w != nil {
		w.SetClosingHandled(h != nil)
	}
}

// Terminate requests the game loop to stop at the end of the current frame.
func (g *Game) Terminate() {
	g.terminate = true
}

// ShouldTerminate reports whether Terminate has been called and the platform loop
// should exit now.
func (g *Game) ShouldTerminate() bool {
	return g.terminate
}

// Draw renders the game for the current frame. It clears the screen with the
// active scene's background color, then draws that scene.
func (g *Game) Draw() {
	renderer := g.platform.Renderer()
	if g.activeScene == nil {
		renderer.Clear(math.Black)
		return
	}
	renderer.Clear(g.activeScene.BackgroundColor)
	g.activeScene.Draw(renderer)
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
