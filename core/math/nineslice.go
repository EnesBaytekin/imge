package math

// Border defines the 9-slice insets, in texture pixels, that split a texture into
// nine regions: four corners (drawn at natural size), four edges (stretched along
// one axis), and a center (stretched along both). A zero value means "no border":
// the whole texture is a single stretchable region.
type Border struct {
	Left   float64 `json:"left"`
	Top    float64 `json:"top"`
	Right  float64 `json:"right"`
	Bottom float64 `json:"bottom"`
}

// Slice is one of the up-to-nine regions of a 9-slice: the source rectangle in the
// texture (texture pixels) and the destination rectangle it fills (in the target's
// coordinate space, top-left origin).
type Slice struct {
	Src Rect
	Dst Rect
}

// Slice9 splits a texW×texH texture into up to nine source/destination regions that
// together fill dst. Regions with a zero or negative source or destination area are
// omitted, so a degenerate target simply produces fewer slices.
//
// Source regions always use the original texture-pixel borders. When the target is
// smaller than the border sum on an axis, that axis's borders are scaled down
// proportionally so the corners shrink to fit instead of overflowing the target.
func Slice9(texW, texH float64, border Border, dst Rect) []Slice {
	w, h := dst.Width(), dst.Height()
	if texW <= 0 || texH <= 0 || w <= 0 || h <= 0 {
		return nil
	}

	// Destination border widths, scaled down when the target is smaller than the
	// border sum so nothing overflows.
	dl, dr := border.Left, border.Right
	if s := dl + dr; s > w {
		k := w / s
		dl, dr = dl*k, dr*k
	}
	dt, db := border.Top, border.Bottom
	if s := dt + db; s > h {
		k := h / s
		dt, db = dt*k, db*k
	}

	srcCols := [3]float64{border.Left, texW - border.Left - border.Right, border.Right}
	srcRows := [3]float64{border.Top, texH - border.Top - border.Bottom, border.Bottom}
	dstCols := [3]float64{dl, w - dl - dr, dr}
	dstRows := [3]float64{dt, h - dt - db, db}

	var out []Slice
	var sy float64
	dy := dst.Y()
	for row := 0; row < 3; row++ {
		var sx float64
		dx := dst.X()
		for col := 0; col < 3; col++ {
			sw, sh := srcCols[col], srcRows[row]
			dw, dh := dstCols[col], dstRows[row]
			if sw > 0 && sh > 0 && dw > 0 && dh > 0 {
				out = append(out, Slice{
					Src: NewRect(sx, sy, sw, sh),
					Dst: NewRect(dx, dy, dw, dh),
				})
			}
			dx += dstCols[col]
			sx += srcCols[col]
		}
		dy += dstRows[row]
		sy += srcRows[row]
	}
	return out
}
