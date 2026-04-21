// Package mock provides a mock platform implementation for testing and debugging.
// All methods print debug messages to stdout.
package mock

import (
	"fmt"
	"time"

	"github.com/EnesBaytekin/imge/core"
)

// MockPlatform implements core.Platform interface with debug prints.
type MockPlatform struct {
	renderer   *MockRenderer
	input      *MockInput
	audio      *MockAudio
	time       *MockTime
	window     *MockWindow
	filesystem *MockFileSystem
}

// New creates a new MockPlatform instance.
func New() *MockPlatform {
	return &MockPlatform{
		renderer:   &MockRenderer{},
		input:      &MockInput{},
		audio:      &MockAudio{},
		time:       &MockTime{start: time.Now()},
		window:     &MockWindow{},
		filesystem: &MockFileSystem{},
	}
}

// Renderer returns the mock renderer.
func (p *MockPlatform) Renderer() core.Renderer {
	fmt.Println("[MockPlatform] Renderer() called")
	return p.renderer
}

// Input returns the mock input handler.
func (p *MockPlatform) Input() core.Input {
	fmt.Println("[MockPlatform] Input() called")
	return p.input
}

// Audio returns the mock audio handler.
func (p *MockPlatform) Audio() core.Audio {
	fmt.Println("[MockPlatform] Audio() called")
	return p.audio
}

// Time returns the mock time handler.
func (p *MockPlatform) Time() core.Time {
	fmt.Println("[MockPlatform] Time() called")
	return p.time
}

// Window returns the mock window handler.
func (p *MockPlatform) Window() core.Window {
	fmt.Println("[MockPlatform] Window() called")
	return p.window
}

// FileSystem returns the mock filesystem handler.
func (p *MockPlatform) FileSystem() core.FileSystem {
	fmt.Println("[MockPlatform] FileSystem() called")
	return p.filesystem
}

// Init initializes the platform with the given window configuration.
func (p *MockPlatform) Init(title string, width, height int) error {
	fmt.Printf("[MockPlatform] Init(title=%s, width=%d, height=%d)\n", title, width, height)
	// Create the window
	return p.window.Create(title, width, height)
}

// Update is called each frame to update platform state.
func (p *MockPlatform) Update() {
	fmt.Println("[MockPlatform] Update() called")
}