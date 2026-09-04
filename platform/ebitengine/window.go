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
	resizable  bool
	scale      int
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
	w.resizable = cfg.Resizable && !cfg.Fullscreen
	w.scale = cfg.Scale

	ebiten.SetWindowTitle(cfg.Title)

	// Nearest-neighbor scaling so pixel art stays crisp when the logical
	// resolution is scaled up to fit the window/screen.
	ebiten.SetScreenFilterEnabled(false)

	// The render resolution is logical * pixel_per_unit; the window magnifies it
	// by an integer scale factor (auto-fit, or an explicit Scale when Resizable).
	ppu := cfg.PixelPerUnit
	if ppu <= 0 {
		ppu = 1
	}
	renderW, renderH := cfg.Width*ppu, cfg.Height*ppu

	// Resolve the integer window scale:
	//   - fullscreen ignores scale (the surface stretches to the display);
	//   - resizable uses a fixed scale (auto -> 1, so the window size drives the
	//     logical resolution rather than the reverse);
	//   - fixed-size uses the explicit scale, or at 0 the largest integer factor
	//     of the render resolution that still fits the monitor.
	scale := cfg.Scale
	if cfg.Fullscreen {
		scale = 1
		ebiten.SetWindowResizingMode(ebiten.WindowResizingModeDisabled)
	} else if cfg.Resizable {
		ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
		if scale < 1 {
			scale = 1
		}
	} else {
		ebiten.SetWindowResizingMode(ebiten.WindowResizingModeDisabled)
		if scale <= 0 {
			scale = autoFitScale(renderW, renderH)
		}
	}

	ww, wh := renderW*scale, renderH*scale
	ebiten.SetWindowSize(ww, wh)

	if cfg.Fullscreen {
		ebiten.SetFullscreen(true)
	} else if cfg.Resizable {
		// Allow shrinking down to roughly one logical unit, with no maximum — the
		// Layout path recomputes the logical size from the window every frame, so
		// the scene grows and shrinks with the window. A negative value means "no
		// limit" to Ebitengine/GLFW; 0 is an invalid maximum that GLFW rejects.
		ebiten.SetWindowSizeLimits(ppu*scale, ppu*scale, -1, -1)
	} else {
		// Lock the window to its resolved size with matching min/max limits.
		// Ebitengine disables the limits during fullscreen and re-applies them on
		// exit, so leaving fullscreen deterministically clamps back to this size
		// instead of drifting (which would otherwise letterbox the game left/right).
		ebiten.SetWindowSizeLimits(ww, wh, ww, wh)
	}
	return nil
}

// autoFitScale returns the largest integer factor by which the given render
// resolution still fits the primary monitor, leaving a margin for window
// decorations. It falls back to 1 when no monitor is reported.
func autoFitScale(width, height int) int {
	if width <= 0 || height <= 0 {
		return 1
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
	return scale
}

// Destroy is a no-op; Ebitengine tears the window down when the game exits.
func (w *Window) Destroy() {}

// ShouldClose reports whether the user has requested the window to close.
func (w *Window) ShouldClose() bool {
	return ebiten.IsWindowBeingClosed()
}

// SetClosingHandled enables or disables close interception. With it enabled, the OS
// close button no longer terminates the process; ShouldClose reports the request for
// one frame and the game is expected to terminate itself when ready.
func (w *Window) SetClosingHandled(handled bool) {
	ebiten.SetWindowClosingHandled(handled)
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
