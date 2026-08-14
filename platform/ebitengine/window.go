package ebitengine

import "github.com/hajimehoshi/ebiten/v2"

// Window implements core.Window using Ebitengine window management.
type Window struct {
	title      string
	width      int
	height     int
	fullscreen bool
}

func newWindow() *Window {
	return &Window{}
}

// Create configures the window. The actual window is created by Ebitengine at Run.
func (w *Window) Create(title string, width, height int) error {
	w.title = title
	w.width = width
	w.height = height
	ebiten.SetWindowTitle(title)
	ebiten.SetWindowSize(width, height)
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	return nil
}

// Destroy is a no-op; Ebitengine tears the window down when the game exits.
func (w *Window) Destroy() {}

// ShouldClose reports whether the user has requested the window to close.
func (w *Window) ShouldClose() bool {
	return ebiten.IsWindowBeingClosed()
}

// GetSize returns the current window size in pixels.
func (w *Window) GetSize() (width, height int) {
	return w.width, w.height
}

// SetTitle sets the window title.
func (w *Window) SetTitle(title string) {
	w.title = title
	ebiten.SetWindowTitle(title)
}

// SetSize sets the window size.
func (w *Window) SetSize(width, height int) {
	w.width = width
	w.height = height
	ebiten.SetWindowSize(width, height)
}

// SetFullscreen toggles fullscreen mode.
func (w *Window) SetFullscreen(fullscreen bool) {
	w.fullscreen = fullscreen
	ebiten.SetFullscreen(fullscreen)
}

// PollEvents is a no-op; Ebitengine processes events in its own loop.
func (w *Window) PollEvents() {}

// syncSize updates the tracked size after a window resize (called from Layout).
func (w *Window) syncSize(width, height int) {
	w.width = width
	w.height = height
}
