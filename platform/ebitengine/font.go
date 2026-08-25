package ebitengine

import (
	"image/color"
	stdmath "math"

	"github.com/EnesBaytekin/imge/core/math"
	"github.com/EnesBaytekin/imge/platform/ebitengine/fonts"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text"
)

// fontState holds the renderer's font library plus a reusable rasterization
// buffer. Font parsing and measurement live in the fonts package (so they can be
// unit-tested without a display); only the Ebitengine rasterization stays here.
type fontState struct {
	lib *fonts.Library

	// scratch is a reusable logical-resolution buffer text is rasterized into
	// before being upscaled. It grows to the largest text drawn and never shrinks,
	// so dynamic text (scores, timers) doesn't allocate a texture every frame.
	scratch *ebiten.Image
}

func newFontState() fontState {
	return fontState{lib: fonts.NewLibrary()}
}

// DrawText draws a single line of text with its top-left corner at position, at
// the given size (logical units) and color. See core.Renderer.DrawText.
//
// Text is rasterized at logical resolution (glyphs drawn at `size` pixels) and
// then upscaled by pixel_per_unit × camera zoom with nearest-neighbor, exactly
// like a texture or sprite. A pixel font therefore stays crisp at any
// pixel_per_unit as long as `size` is an integer multiple of the font's design
// size — the same integer-scaling rule that applies to pixel art.
func (r *Renderer) DrawText(str string, fontID string, size float64, position math.Vector2, clr math.Color) {
	if r.target == nil || str == "" {
		return
	}

	face := r.fonts.lib.Face(r.assetFS, fontID, size)
	if face == nil {
		return
	}
	w, h := r.fonts.lib.Measure(r.assetFS, fontID, size, str)
	if w <= 0 || h <= 0 {
		return
	}
	// +1 keeps antialiased glyph edges from clipping against the buffer.
	bw := int(stdmath.Ceil(w)) + 1
	bh := int(stdmath.Ceil(h)) + 1

	scratch := r.fonts.scratch
	if scratch == nil || scratch.Bounds().Dx() < bw || scratch.Bounds().Dy() < bh {
		scratch = ebiten.NewImage(bw, bh)
		r.fonts.scratch = scratch
	}
	scratch.Clear()

	// text.Draw positions the *baseline* at (x, y), so draw the baseline `ascent`
	// pixels down to keep the glyphs inside the buffer (Ceil so a fractional
	// ascent never clips the top). NRGBA is straight alpha, which ScaleWithColor
	// premultiplies correctly at any alpha.
	baseline := int(stdmath.Ceil(float64(face.Metrics().Ascent) / 64))
	text.Draw(scratch, str, face, 0, baseline, color.NRGBA{R: clr.R, G: clr.G, B: clr.B, A: clr.A})

	// Snap the anchor to whole units and apply the remainder as a sub-pixel offset,
	// the same way shapes do, so the text stays on the unit grid as it moves.
	qx := stdmath.Round(position.X)
	qy := stdmath.Round(position.Y)
	r.blitChunky(scratch, math.NewVector2(qx, qy), math.NewVector2(position.X-qx, position.Y-qy))
}

// MeasureText returns the width and height (in logical units) of the given text at
// the given size — the same box DrawText places. See core.Renderer.MeasureText.
func (r *Renderer) MeasureText(str string, fontID string, size float64) (float64, float64) {
	return r.fonts.lib.Measure(r.assetFS, fontID, size, str)
}
