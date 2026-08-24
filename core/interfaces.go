// Package core contains platform-agnostic game engine logic.
// This file defines the interfaces that platform-specific implementations must satisfy.
package core

import (
	"github.com/EnesBaytekin/imge/core/math"
)

// ============================================================================
// Renderer Interface
// ============================================================================

// Renderer handles all 2D drawing operations.
// Platform implementations will provide actual rendering (OpenGL, DirectX, software, etc.).
type Renderer interface {
	// Clear clears the entire screen with the specified color.
	Clear(color math.Color)

	// DrawRect draws a filled rectangle.
	DrawRect(rect math.Rect, color math.Color)

	// DrawRectOutline draws a rectangle outline (border only).
	DrawRectOutline(rect math.Rect, color math.Color, thickness float64)

	// DrawCircle draws a filled circle.
	DrawCircle(center math.Vector2, radius float64, color math.Color)

	// DrawCircleOutline draws a circle outline.
	DrawCircleOutline(center math.Vector2, radius float64, color math.Color, thickness float64)

	// DrawLine draws a line between two points.
	DrawLine(start, end math.Vector2, color math.Color, thickness float64)

	// DrawTexture draws a texture (or a region of it) at the specified position
	// with transformations. textureID identifies a previously loaded texture.
	// src is the source region in the texture; a zero Rect means the entire texture.
	// transform is the color transform applied to the texture (an identity
	// transform is a plain draw).
	DrawTexture(textureID string, src math.Rect, position math.Vector2, scale math.Vector2, rotation float64, transform math.ColorTransform)

	// GetTextureSize returns the natural pixel size of a loaded texture.
	// Returns (0, 0) if the texture cannot be loaded.
	GetTextureSize(textureID string) (width, height float64)

	// SetCamera applies a world-to-screen camera transform to subsequent draw
	// calls. (cx, cy) is the view center in world coordinates and zoom is the scale
	// factor (1 = 1:1). A zoom <= 0 disables the transform (raw screen space).
	SetCamera(cx, cy, zoom float64)

	// Present presents the rendered frame to the screen (swap buffers).
	Present()

	// SetViewport sets the rendering viewport size.
	SetViewport(width, height int)

	// GetViewportSize returns the current viewport size.
	GetViewportSize() (width, height int)
}

// ============================================================================
// Input Interface
// ============================================================================

// KeyCode represents a keyboard key.
// Platform implementations will map physical keys to these codes.
type KeyCode int

// Common keyboard keys (partial list, can be extended).
const (
	KeyUnknown KeyCode = iota
	KeyA
	KeyB
	KeyC
	KeyD
	KeyE
	KeyF
	KeyG
	KeyH
	KeyI
	KeyJ
	KeyK
	KeyL
	KeyM
	KeyN
	KeyO
	KeyP
	KeyQ
	KeyR
	KeyS
	KeyT
	KeyU
	KeyV
	KeyW
	KeyX
	KeyY
	KeyZ
	Key0
	Key1
	Key2
	Key3
	Key4
	Key5
	Key6
	Key7
	Key8
	Key9
	KeySpace
	KeyEnter
	KeyEscape
	KeyBackspace
	KeyTab
	KeyShift
	KeyControl
	KeyAlt
	KeyLeft
	KeyRight
	KeyUp
	KeyDown
	KeyF1
	KeyF2
	KeyF3
	KeyF4
	KeyF5
	KeyF6
	KeyF7
	KeyF8
	KeyF9
	KeyF10
	KeyF11
	KeyF12
)

// MouseButton represents a mouse button.
type MouseButton int

const (
	MouseButtonLeft MouseButton = iota
	MouseButtonRight
	MouseButtonMiddle
	MouseButton4
	MouseButton5
)

// Input handles user input from keyboard and mouse.
type Input interface {
	// IsKeyPressed checks if a key is currently pressed.
	IsKeyPressed(key KeyCode) bool

	// IsKeyJustPressed checks if a key was pressed this frame (not held).
	IsKeyJustPressed(key KeyCode) bool

	// IsKeyJustReleased checks if a key was released this frame.
	IsKeyJustReleased(key KeyCode) bool

	// IsMouseButtonPressed checks if a mouse button is currently pressed.
	IsMouseButtonPressed(button MouseButton) bool

	// IsMouseButtonJustPressed checks if a mouse button was pressed this frame.
	IsMouseButtonJustPressed(button MouseButton) bool

	// IsMouseButtonJustReleased checks if a mouse button was released this frame.
	IsMouseButtonJustReleased(button MouseButton) bool

	// GetMousePosition returns the current mouse position in screen coordinates.
	GetMousePosition() math.Vector2

	// GetMouseDelta returns the mouse movement since last frame.
	GetMouseDelta() math.Vector2

	// GetMouseScroll returns the mouse wheel scroll delta.
	GetMouseScroll() math.Vector2

	// Update should be called once per frame to update input state.
	Update()
}

// ============================================================================
// Audio Interface
// ============================================================================

// Audio handles sound and music playback.
type Audio interface {
	// PlaySound plays a sound effect once.
	// soundID identifies a previously loaded sound.
	// volume ranges from 0.0 (silent) to 1.0 (full volume).
	// pitch ranges from 0.5 (half speed) to 2.0 (double speed).
	PlaySound(soundID string, volume, pitch float64)

	// PlayMusic starts playing background music.
	// musicID identifies a previously loaded music track.
	// loop determines if the music should repeat.
	PlayMusic(musicID string, loop bool)

	// StopMusic stops any currently playing music.
	StopMusic()

	// PauseMusic pauses the current music (can be resumed with ResumeMusic).
	PauseMusic()

	// ResumeMusic resumes paused music.
	ResumeMusic()

	// SetMasterVolume sets the overall volume (0.0 to 1.0).
	SetMasterVolume(volume float64)

	// SetSoundVolume sets the volume for sound effects.
	SetSoundVolume(volume float64)

	// SetMusicVolume sets the volume for music.
	SetMusicVolume(volume float64)
}

// ============================================================================
// Time Interface
// ============================================================================

// Time provides timing information for game loop and animations.
type Time interface {
	// DeltaTime returns the time elapsed since the last frame in seconds.
	DeltaTime() float64

	// TotalTime returns the total time elapsed since the game started in seconds.
	TotalTime() float64

	// FPS returns the current frames per second.
	FPS() float64

	// Tick should be called once per frame to update timing.
	Tick()

	// Sleep pauses execution for the specified number of seconds.
	Sleep(seconds float64)
}

// ============================================================================
// Window Interface
// ============================================================================

// WindowConfig describes how the window should be created: its logical (game)
// resolution and whether it starts fullscreen. The logical resolution is what the
// renderer draws at; the platform scales it to fit the actual window/browser,
// letterboxing to preserve the aspect ratio.
type WindowConfig struct {
	Title      string
	Width      int // logical (game) resolution, in game units
	Height     int // logical (game) resolution, in game units
	Fullscreen bool
	// PixelPerUnit is the number of framebuffer pixels per logical unit along one
	// axis (>= 1). The render target is Width*PixelPerUnit x Height*PixelPerUnit,
	// so a value > 1 lets the rasterizer show sub-unit (fractional) positions — the
	// world still moves in whole units, but the extra resolution makes that motion
	// smooth instead of snapping to whole pixels. 1 (the default) is pixel-perfect.
	PixelPerUnit int
	// SmoothShapes opts vector shapes into framebuffer-resolution rasterization
	// (fine, 1px edges). The default (false) renders them "chunky": each shape is
	// rasterized at logical resolution (its anchor quantized to whole units, so
	// its pixel pattern is deterministic) and upscaled by PixelPerUnit, matching
	// how textures already render. Applies at any PixelPerUnit — at 1 it still
	// keeps shapes pixel-perfect and stable instead of re-rasterizing at
	// fractional positions (which wobbles as the shape moves).
	SmoothShapes bool
}

// Window handles window management and events.
type Window interface {
	// Create creates a new window with the given configuration.
	Create(cfg WindowConfig) error

	// Destroy closes and cleans up the window.
	Destroy()

	// ShouldClose returns true if the window should close (e.g., user clicked X).
	ShouldClose() bool

	// GetSize returns the current window size in pixels.
	GetSize() (width, height int)

	// SetTitle sets the window title.
	SetTitle(title string)

	// SetSize sets the window size.
	SetSize(width, height int)

	// SetFullscreen toggles fullscreen mode.
	SetFullscreen(fullscreen bool)

	// PollEvents processes window events (should be called each frame).
	PollEvents()
}

// ============================================================================
// FileSystem Interface
// ============================================================================

// FileSystem handles file operations and asset loading.
type FileSystem interface {
	// ReadFile reads the entire contents of a file.
	ReadFile(path string) ([]byte, error)

	// WriteFile writes data to a file.
	WriteFile(path string, data []byte) error

	// FileExists checks if a file exists.
	FileExists(path string) bool

	// ListFiles lists all files in a directory (non-recursive).
	ListFiles(dir string) ([]string, error)

	// ListFilesRecursive lists all files in a directory recursively.
	ListFilesRecursive(dir string) ([]string, error)

	// CreateDirectory creates a new directory.
	CreateDirectory(path string) error

	// Delete deletes a file or empty directory.
	Delete(path string) error
}

// ============================================================================
// Component Context
// ============================================================================

// Context provides access to engine services from within components.
// Passed to Component.Update() method each frame. Scene is set to the active
// scene before updates run; Renderer is passed separately to Draw().
type Context struct {
	Input Input
	Audio Audio
	Time  Time
	Scene *Scene
	Game  *Game
}

// DeltaTime returns the seconds elapsed since the last frame.
// Returns 0 if no Time service is available.
func (c *Context) DeltaTime() float64 {
	if c.Time == nil {
		return 0
	}
	return c.Time.DeltaTime()
}

// ============================================================================
// Platform Interface (optional but useful)
// ============================================================================

// Platform is a convenience interface that groups all platform interfaces.
// Implementations can choose to implement this or individual interfaces.
type Platform interface {
	// Renderer returns the renderer interface for drawing operations.
	Renderer() Renderer

	// Input returns the input interface for user input handling.
	Input() Input

	// Audio returns the audio interface for sound and music playback.
	Audio() Audio

	// Time returns the time interface for timing information.
	Time() Time

	// Window returns the window interface for window management.
	Window() Window

	// FileSystem returns the filesystem interface for file operations.
	FileSystem() FileSystem

	// Init initializes the platform with the given window configuration.
	// This should create the window, initialize subsystems, etc.
	Init(cfg WindowConfig) error

	// Update is called each frame to update platform state.
	Update()
}
