// Package ebitengine provides a pure-Go platform implementation for IMGE built on Ebitengine.
//
// It satisfies the core.Platform interface so the engine's platform-agnostic core
// runs unchanged, while Ebitengine drives the actual game loop. This means no CGO,
// no SDL, and a single `go build` produces a working game.
package ebitengine

import (
	"fmt"
	"io/fs"

	"github.com/EnesBaytekin/imge/core"
	"github.com/hajimehoshi/ebiten/v2"
)

// Platform implements core.Platform on top of Ebitengine.
type Platform struct {
	renderer   *Renderer
	input      *Input
	audio      *Audio
	time       *Time
	window     *Window
	filesystem *FileSystem

	// logicalWidth/logicalHeight is the game's fixed logical screen size (from
	// game.imge). Layout returns this size so the surface letterboxes the game
	// inside the actual window/browser, preserving the aspect ratio instead of
	// stretching to whatever the browser viewport happens to be.
	logicalWidth  int
	logicalHeight int
}

// New creates a new Ebitengine platform instance.
func New() (*Platform, error) {
	return &Platform{
		renderer:   newRenderer(),
		input:      newInput(),
		audio:      newAudio(),
		time:       newTime(),
		window:     newWindow(),
		filesystem: &FileSystem{},
	}, nil
}

// Renderer returns the renderer.
func (p *Platform) Renderer() core.Renderer { return p.renderer }

// Input returns the input handler.
func (p *Platform) Input() core.Input { return p.input }

// Audio returns the audio handler.
func (p *Platform) Audio() core.Audio { return p.audio }

// Time returns the timing handler.
func (p *Platform) Time() core.Time { return p.time }

// Window returns the window handler.
func (p *Platform) Window() core.Window { return p.window }

// FileSystem returns the filesystem handler.
func (p *Platform) FileSystem() core.FileSystem { return p.filesystem }

// SetAssetFS sets the filesystem used to load textures and audio. Web builds
// pass their embedded fs.FS here; desktop builds leave it nil so assets load
// from the OS filesystem.
func (p *Platform) SetAssetFS(fsys fs.FS) {
	p.renderer.SetAssetFS(fsys)
	p.audio.SetAssetFS(fsys)
}

// Init configures the Ebitengine window. The underlying window is created by
// Ebitengine when Run is called.
func (p *Platform) Init(title string, width, height int) error {
	p.logicalWidth = width
	p.logicalHeight = height
	return p.window.Create(title, width, height)
}

// Update is a no-op. When the platform is driven by Run (the normal path),
// Ebitengine owns the loop and the runner updates input/time directly.
func (p *Platform) Update() {}

// Run starts the Ebitengine game loop and blocks until the game exits.
// Ebitengine drives the loop, calling the core game's Update/Draw via a runner adapter.
func (p *Platform) Run(game *core.Game) error {
	if game == nil {
		return fmt.Errorf("ebitengine: nil game")
	}
	return ebiten.RunGame(&runner{platform: p, game: game})
}

// Cleanup releases audio resources.
func (p *Platform) Cleanup() {
	if p.audio != nil {
		p.audio.close()
	}
}

// runner adapts core.Game to the ebiten.Game interface so Ebitengine's
// callback-driven loop can drive the engine's pull-based core without touching it.
type runner struct {
	platform *Platform
	game     *core.Game
}

// Update implements ebiten.Game.Update.
func (r *runner) Update() error {
	p := r.platform
	p.input.Update()
	p.time.Tick()
	ctx := &core.Context{
		Input: p.input,
		Audio: p.audio,
		Time:  p.time,
	}
	r.game.Update(ctx)
	return nil
}

// Draw implements ebiten.Game.Draw. The core game clears the screen with the
// active scene's background color.
func (r *runner) Draw(screen *ebiten.Image) {
	p := r.platform
	p.renderer.begin(screen)
	r.game.Draw()
}

// Layout implements ebiten.Game.Layout. It returns the game's fixed logical
// size rather than the outside surface size, so Ebitengine scales the frame to
// fit the actual window/browser while preserving the aspect ratio and
// letterboxing the extra space. This keeps the web build at the configured
// window size instead of stretching to the full browser viewport.
func (r *runner) Layout(outsideWidth, outsideHeight int) (int, int) {
	p := r.platform
	p.window.syncSize(outsideWidth, outsideHeight)

	w, h := p.logicalWidth, p.logicalHeight
	if w <= 0 || h <= 0 {
		w, h = outsideWidth, outsideHeight
	}
	p.renderer.setViewport(w, h)
	return w, h
}
