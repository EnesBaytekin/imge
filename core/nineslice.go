package core

import "github.com/EnesBaytekin/imge/core/math"

// DrawNineSlice draws textureID as a nine-sliced image filling dst, using border
// (in texture pixels) to split it into corner/edge/center regions so the corners
// keep their natural size while the center and edges stretch. A zero border means
// the whole texture stretches to fill dst.
//
// dst is in the current draw space (world or screen), so call it after setting the
// camera appropriately. This is the shared 9-slice routine used by UI components
// (@Panel, @Button, @TextInput) and available to any custom component or world
// object that wants a sliceable image.
func DrawNineSlice(r Renderer, textureID string, border math.Border, dst math.Rect) {
	DrawNineSliceTransform(r, textureID, border, dst, math.ColorTransform{})
}

// DrawNineSliceTransform is DrawNineSlice with a color transform applied to every
// slice, so callers can tint the whole nine-slice (e.g. a button state tint) in a
// single call. An identity transform is a plain draw.
func DrawNineSliceTransform(r Renderer, textureID string, border math.Border, dst math.Rect, transform math.ColorTransform) {
	tw, th := r.GetTextureSize(textureID)
	if tw <= 0 || th <= 0 {
		return
	}
	for _, s := range math.Slice9(tw, th, border, dst) {
		sx := s.Dst.Width() / s.Src.Width()
		sy := s.Dst.Height() / s.Src.Height()
		r.DrawTexture(textureID, s.Src, s.Dst.Position, math.NewVector2(sx, sy), 0, transform)
	}
}
