// Package sdl provides an SDL2-based platform implementation.
package sdl

import (
	"fmt"
	"log"

	"github.com/EnesBaytekin/imge/core"
	"github.com/veandco/go-sdl2/sdl"
	"github.com/veandco/go-sdl2/mix"
)

// SDLPlatform implements core.Platform interface using SDL2.
type SDLPlatform struct {
	renderer   *SDLRenderer
	input      *SDLInput
	audio      *SDLAudio
	time       *SDLTime
	window     *SDLWindow
	filesystem *SDLFileSystem
}

// New creates a new SDLPlatform instance.
// Initializes SDL subsystems and creates window/renderer.
func New() (*SDLPlatform, error) {
	// Initialize SDL
	if err := sdl.Init(sdl.INIT_VIDEO | sdl.INIT_AUDIO | sdl.INIT_EVENTS); err != nil {
		return nil, fmt.Errorf("SDL initialization failed: %v", err)
	}

	// Initialize SDL_mixer for audio
	if err := mix.Init(mix.INIT_MP3 | mix.INIT_OGG); err != nil {
		log.Printf("Warning: SDL_mixer initialization failed: %v", err)
		// Continue without audio support
	}

	// Initialize SDL_ttf if needed (for fonts)
	// ttf.Init()

	// Create window first
	window := &SDLWindow{}

	// Create input with window reference
	input := NewSDLInput(window)

	// Create other subsystems
	platform := &SDLPlatform{
		renderer:   &SDLRenderer{},
		input:      input,
		audio:      NewSDLAudio(),
		time:       &SDLTime{},
		window:     window,
		filesystem: &SDLFileSystem{},
	}

	return platform, nil
}

// Cleanup releases SDL resources.
func (p *SDLPlatform) Cleanup() {
	// Cleanup in reverse order of initialization
	if p.audio != nil {
		p.audio.Cleanup()
	}
	if p.renderer != nil {
		p.renderer.Cleanup()
	}
	if p.window != nil {
		p.window.Cleanup()
	}
	mix.Quit()
	sdl.Quit()
}

// Renderer returns the SDL renderer.
func (p *SDLPlatform) Renderer() core.Renderer {
	return p.renderer
}

// Input returns the SDL input handler.
func (p *SDLPlatform) Input() core.Input {
	return p.input
}

// Audio returns the SDL audio handler.
func (p *SDLPlatform) Audio() core.Audio {
	return p.audio
}

// Time returns the SDL time handler.
func (p *SDLPlatform) Time() core.Time {
	return p.time
}

// Window returns the SDL window handler.
func (p *SDLPlatform) Window() core.Window {
	return p.window
}

// FileSystem returns the SDL filesystem handler.
func (p *SDLPlatform) FileSystem() core.FileSystem {
	return p.filesystem
}

// Init initializes the platform with the given window configuration.
// Creates the window, initializes renderer, opens audio device, etc.
func (p *SDLPlatform) Init(title string, width, height int) error {
	// Create window
	if err := p.window.Create(title, width, height); err != nil {
		return fmt.Errorf("failed to create window: %v", err)
	}

	// Initialize renderer with window
	if err := p.renderer.Init(p.window); err != nil {
		return fmt.Errorf("failed to initialize renderer: %v", err)
	}

	// Open audio device for SDL_mixer
	// Default audio format: 22050 Hz, signed 16-bit, stereo, 4096 byte buffer
	if err := mix.OpenAudio(22050, uint16(mix.DEFAULT_FORMAT), 2, 4096); err != nil {
		log.Printf("Warning: Failed to open audio device: %v", err)
		// Continue without audio
	} else {
		log.Println("Audio device opened successfully")
	}

	// Initialize time subsystem (reset timers)
	// SDLTime doesn't need explicit initialization

	log.Printf("SDL platform initialized: %s (%dx%d)", title, width, height)
	return nil
}

// Update is called each frame to update platform state.
// Polls window events and updates platform state.
func (p *SDLPlatform) Update() {
	// Poll window events (SDL event loop)
	p.window.PollEvents()
}