package ebitengine

import (
	"github.com/EnesBaytekin/imge/core"
	"github.com/hajimehoshi/ebiten/v2"
)

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
func (w *Window) Create(cfg core.WindowConfig) error {
	w.title = cfg.Title
	w.width = cfg.Width
	w.height = cfg.Height
	w.fullscreen = cfg.Fullscreen

	ebiten.SetWindowTitle(cfg.Title)

	// Nearest-neighbor scaling so pixel art stays crisp when the logical
	// resolution is scaled up to fit the window/screen.
	ebiten.SetScreenFilterEnabled(false)

	// The window is fixed-size: it can't be drag-resized, only toggled between
	// windowed and fullscreen.
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeDisabled)

	// Open the window at the largest integer scale of the RENDER resolution
	// (logical * pixel_per_unit) that fits the monitor. Basing it on the render
	// resolution rather than the logical one keeps the final upscale an integer,
	// so pixel art stays crisp when pixel_per_unit > 1 and the extra sub-unit
	// resolution is actually shown (web ignores the window size — the browser owns it).
	ppu := cfg.PixelPerUnit
	if ppu <= 0 {
		ppu = 1
	}
	ww, wh := fitWindowSize(cfg.Width*ppu, cfg.Height*ppu)
	ebiten.SetWindowSize(ww, wh)

	// Lock the window to the auto-fit size with matching min/max limits. Ebitengine
	// disables the limits during fullscreen and re-applies them on exit, so leaving
	// fullscreen deterministically clamps back to this size instead of drifting
	// (which would otherwise letterbox the game left/right).
	ebiten.SetWindowSizeLimits(ww, wh, ww, wh)

	if cfg.Fullscreen {
		ebiten.SetFullscreen(true)
	}
	return nil
}

// fitWindowSize returns the logical resolution scaled up by the largest integer
// factor that still fits the primary monitor, leaving a margin for window
// decorations. It falls back to the logical size when no monitor is reported.
func fitWindowSize(width, height int) (int, int) {
	if width <= 0 || height <= 0 {
		return width, height
	}
	scale := 1
	if m := ebiten.Monitor(); m != nil {
		mw, mh := m.Size()
		if mw > 0 && mh > 0 {
			// Leave room for the title bar and borders.
			const margin = 80
			sx := (mw - margin) / width
			sy := (mh - margin) / height
			scale = sx
			if sy < scale {
				scale = sy
			}
			if scale < 1 {
				scale = 1
			}
		}
	}
	return width * scale, height * scale
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
