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

	// pixelPerUnit is the framebuffer-pixels-per-unit scale (pixel_per_unit). The
	// render target is logicalWidth*pixelPerUnit x logicalHeight*pixelPerUnit.
	pixelPerUnit int

	// resizable reports whether the window can be drag-resized (windowed only).
	// When true, the window size drives the logical resolution, so the scene grows
	// and shrinks with the window instead of letterboxing to a fixed size.
	resizable bool

	// scale is the integer window-magnification factor (see WindowConfig.Scale).
	// For resizable windows it is folded into pixelScale so each logical unit maps
	// to pixelPerUnit*scale physical pixels; Layout then returns the window size 1:1.
	scale int
}

// New creates a new Ebitengine platform instance.
func New() (*Platform, error) {
	return &Platform{
		renderer:     newRenderer(),
		input:        newInput(),
		audio:        newAudio(),
		time:         newTime(),
		window:       newWindow(),
		filesystem:   &FileSystem{},
		pixelPerUnit: 1,
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
func (p *Platform) Init(cfg core.WindowConfig) error {
	p.logicalWidth = cfg.Width
	p.logicalHeight = cfg.Height
	p.resizable = cfg.Resizable && !cfg.Fullscreen

	ppu := cfg.PixelPerUnit
	if ppu <= 0 {
		ppu = 1
	}
	p.pixelPerUnit = ppu

	// The pixelScale maps screen pixels to logical units for both the renderer and
	// input. For a fixed-size window it is just ppu (Ebitengine magnifies the
	// render target to the window); for a resizable window the window magnification
	// is folded in (ppu*scale) so Layout can return the window size 1:1 and each
	// logical unit stays an integer number of physical pixels.
	pixelScale := ppu
	p.scale = cfg.Scale
	if p.resizable {
		if p.scale < 1 {
			p.scale = 1
		}
		pixelScale = ppu * p.scale
	}

	p.renderer.setPixelScale(float64(pixelScale))
	p.renderer.setSmoothShapes(cfg.SmoothShapes)
	p.input.setPixelScale(float64(pixelScale))

	return p.window.Create(cfg)
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

	// F11 (or Alt+Enter) toggles fullscreen. The window size is locked via
	// SetWindowSizeLimits at startup, and Ebitengine re-applies that lock when
	// leaving fullscreen, so no manual resize is needed here.
	if p.input.IsKeyJustPressed(core.KeyF11) ||
		(p.input.IsKeyPressed(core.KeyAlt) && p.input.IsKeyJustPressed(core.KeyEnter)) {
		p.window.SetFullscreen(!ebiten.IsFullscreen())
	}

	ctx := &core.Context{
		Input:    p.input,
		Audio:    p.audio,
		Time:     p.time,
		Renderer: p.renderer,
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
//
// For a resizable window it returns the outside size 1:1 instead: the window
// magnification is already folded into pixelScale, so the scene's logical size
// simply tracks the window size.
func (r *runner) Layout(outsideWidth, outsideHeight int) (int, int) {
	p := r.platform
	p.window.syncSize(outsideWidth, outsideHeight)

	if p.resizable && !ebiten.IsFullscreen() {
		// Windowed + resizable: the window size drives the logical resolution.
		// pixelScale already folds the window scale in (Init), so logical = window
		// / pixelScale and the frame is drawn 1:1 (crisp at any window size).
		if outsideWidth < 1 {
			outsideWidth = 1
		}
		if outsideHeight < 1 {
			outsideHeight = 1
		}
		ps := p.pixelPerUnit * p.scale
		if ps < 1 {
			ps = 1
		}
		lw := outsideWidth / ps
		lh := outsideHeight / ps
		if lw < 1 {
			lw = 1
		}
		if lh < 1 {
			lh = 1
		}
		p.renderer.setViewport(lw, lh)
		return outsideWidth, outsideHeight
	}

	w, h := p.logicalWidth, p.logicalHeight
	if w <= 0 || h <= 0 {
		// No configured logical size: use the window 1:1 (ppu not applied).
		p.renderer.setViewport(outsideWidth, outsideHeight)
		return outsideWidth, outsideHeight
	}
	p.renderer.setViewport(w, h)
	return w * p.pixelPerUnit, h * p.pixelPerUnit
}
