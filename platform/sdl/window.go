package sdl

import (
	"fmt"
	"log"

	"github.com/veandco/go-sdl2/sdl"
)

// SDLWindow implements core.Window interface using SDL2.
type SDLWindow struct {
	window     *sdl.Window
	renderer   *sdl.Renderer // Optional, might be owned by SDLRenderer
	title      string
	width      int
	height     int
	fullscreen bool
	shouldClose bool
}

// Create creates a new SDL window with the given title and size.
func (w *SDLWindow) Create(title string, width, height int) error {
	// Initialize SDL if not already done (should be done by SDLPlatform)

	// Create window
	window, err := sdl.CreateWindow(
		title,
		sdl.WINDOWPOS_UNDEFINED, sdl.WINDOWPOS_UNDEFINED,
		int32(width), int32(height),
		sdl.WINDOW_SHOWN|sdl.WINDOW_RESIZABLE,
	)
	if err != nil {
		return fmt.Errorf("failed to create SDL window: %v", err)
	}

	w.window = window
	w.title = title
	w.width = width
	w.height = height
	w.fullscreen = false
	w.shouldClose = false

	log.Printf("SDL window created: %s (%dx%d)", title, width, height)
	return nil
}

// Destroy closes and cleans up the SDL window.
func (w *SDLWindow) Destroy() {
	if w.window != nil {
		w.window.Destroy()
		w.window = nil
	}
	log.Println("SDL window destroyed")
}

// ShouldClose returns true if the window should close (e.g., user clicked X).
// Also processes SDL events to detect window close.
func (w *SDLWindow) ShouldClose() bool {
	// Process pending events to update shouldClose state
	w.PollEvents()
	return w.shouldClose
}

// GetSize returns the current window size in pixels.
func (w *SDLWindow) GetSize() (width, height int) {
	if w.window != nil {
		width32, height32 := w.window.GetSize()
		w.width, w.height = int(width32), int(height32)
	}
	return w.width, w.height
}

// SetTitle sets the window title.
func (w *SDLWindow) SetTitle(title string) {
	if w.window != nil {
		w.window.SetTitle(title)
	}
	w.title = title
}

// SetSize sets the window size.
func (w *SDLWindow) SetSize(width, height int) {
	if w.window != nil {
		w.window.SetSize(int32(width), int32(height))
	}
	w.width = width
	w.height = height
}

// SetFullscreen toggles fullscreen mode.
func (w *SDLWindow) SetFullscreen(fullscreen bool) {
	if w.window == nil {
		return
	}

	var flags uint32
	if fullscreen {
		flags = sdl.WINDOW_FULLSCREEN_DESKTOP
	} else {
		flags = 0
	}

	if err := w.window.SetFullscreen(flags); err != nil {
		log.Printf("Failed to set fullscreen: %v", err)
		return
	}

	w.fullscreen = fullscreen
}

// PollEvents processes SDL window events (should be called each frame).
// This is called by SDLPlatform.Update().
func (w *SDLWindow) PollEvents() {
	// Event polling is handled in SDLInput.Update() to consolidate
	// all input processing in one place.
	// This method is kept for interface compliance.
}

// SetShouldClose sets the window close flag.
// Called by SDLInput when a quit event is received.
func (w *SDLWindow) SetShouldClose(shouldClose bool) {
	w.shouldClose = shouldClose
}

// GetSDLWindow returns the underlying SDL_Window pointer.
// Used by SDLRenderer to create renderer context.
func (w *SDLWindow) GetSDLWindow() *sdl.Window {
	return w.window
}

// Cleanup releases SDL window resources.
// Called by SDLPlatform.Cleanup().
func (w *SDLWindow) Cleanup() {
	w.Destroy()
}