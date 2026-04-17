package mock

import (
	"fmt"
)

// MockWindow implements core.Window interface with debug prints.
type MockWindow struct {
	title      string
	width      int
	height     int
	fullscreen bool
	shouldClose bool
}

// Create creates a new window with the given title and size.
func (w *MockWindow) Create(title string, width, height int) error {
	fmt.Printf("[MockWindow] Create(title=%s, width=%d, height=%d)\n", title, width, height)
	w.title = title
	w.width = width
	w.height = height
	w.shouldClose = false
	return nil
}

// Destroy closes and cleans up the window.
func (w *MockWindow) Destroy() {
	fmt.Println("[MockWindow] Destroy()")
}

// ShouldClose returns true if the window should close (e.g., user clicked X).
func (w *MockWindow) ShouldClose() bool {
	fmt.Println("[MockWindow] ShouldClose() called")
	return w.shouldClose
}

// GetSize returns the current window size in pixels.
func (w *MockWindow) GetSize() (width, height int) {
	fmt.Printf("[MockWindow] GetSize() = (%d, %d)\n", w.width, w.height)
	return w.width, w.height
}

// SetTitle sets the window title.
func (w *MockWindow) SetTitle(title string) {
	fmt.Printf("[MockWindow] SetTitle(title=%s)\n", title)
	w.title = title
}

// SetSize sets the window size.
func (w *MockWindow) SetSize(width, height int) {
	fmt.Printf("[MockWindow] SetSize(width=%d, height=%d)\n", width, height)
	w.width = width
	w.height = height
}

// SetFullscreen toggles fullscreen mode.
func (w *MockWindow) SetFullscreen(fullscreen bool) {
	fmt.Printf("[MockWindow] SetFullscreen(fullscreen=%v)\n", fullscreen)
	w.fullscreen = fullscreen
}

// PollEvents processes window events (should be called each frame).
func (w *MockWindow) PollEvents() {
	fmt.Println("[MockWindow] PollEvents() called")
	// Simulate window close after 1000 frames or so
	// This is just for testing - in real use, this would check actual events
}