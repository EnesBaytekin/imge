package components

import (
	"testing"

	"github.com/EnesBaytekin/imge/core/math"
)

// TestHSVConversions checks the HSV↔RGB helpers against known primaries and verifies
// a lossless-enough roundtrip.
func TestHSVConversions(t *testing.T) {
	// Primaries at full saturation/value.
	if got := hsvToRGB(0, 1, 1); got != math.Red {
		t.Fatalf("hsv(0,1,1) = %v, want red", got)
	}
	if got := hsvToRGB(120, 1, 1); got != math.Green {
		t.Fatalf("hsv(120,1,1) = %v, want green", got)
	}
	if got := hsvToRGB(240, 1, 1); got != math.Blue {
		t.Fatalf("hsv(240,1,1) = %v, want blue", got)
	}

	// Grayscale has zero saturation.
	if _, s, _ := rgbToHSV(math.White); s != 0 {
		t.Fatalf("white saturation = %v, want 0", s)
	}
	if _, s, v := rgbToHSV(math.Black); s != 0 || v != 0 {
		t.Fatalf("black hsv = (%v,%v), want s=0 v=0", s, v)
	}

	// Roundtrip within ±1 per channel (uint8 quantization).
	colors := []math.Color{
		math.Red,
		math.Green,
		math.Blue,
		math.NewColor(255, 204, 51, 255),
		math.NewColor(120, 90, 200, 255),
		math.NewColor(10, 200, 140, 255),
		math.NewColor(64, 64, 64, 255),
	}
	for _, c := range colors {
		h, s, v := rgbToHSV(c)
		back := hsvToRGB(h, s, v)
		if absDiff(back.R, c.R) > 1 || absDiff(back.G, c.G) > 1 || absDiff(back.B, c.B) > 1 {
			t.Fatalf("roundtrip of %v -> hsv(%.1f,%.3f,%.3f) -> %v drifted too far", c, h, s, v, back)
		}
	}
}

func TestFormatHex(t *testing.T) {
	if got := formatHex(math.NewColor(255, 204, 51, 255)); got != "#FFCC33" {
		t.Fatalf("opaque hex = %q, want #FFCC33", got)
	}
	if got := formatHex(math.NewColor(255, 0, 0, 128)); got != "#FF000080" {
		t.Fatalf("alpha hex = %q, want #FF000080", got)
	}
}

func TestParseByte(t *testing.T) {
	cases := []struct {
		in   string
		want uint8
		ok   bool
	}{
		{"0", 0, true},
		{"255", 255, true},
		{"128", 128, true},
		{"300", 255, true},  // clamped
		{"-5", 0, true},     // clamped
		{"", 0, false},      // empty
		{"abc", 0, false},   // non-numeric
	}
	for _, tc := range cases {
		got, ok := parseByte(tc.in)
		if ok != tc.ok || (ok && got != tc.want) {
			t.Fatalf("parseByte(%q) = (%d,%v), want (%d,%v)", tc.in, got, ok, tc.want, tc.ok)
		}
	}
}

// TestColorPickerCommitCancel verifies commit applies the working color (and closes)
// while cancel discards it.
func TestColorPickerCommitCancel(t *testing.T) {
	c := &ColorPickerComponent{}
	c.Color = math.NewColor(10, 20, 30, 255)
	c.Initialize()

	c.openPanel()
	c.working = math.NewColor(200, 100, 50, 255)
	c.commit()
	if c.open {
		t.Fatal("panel should close after commit")
	}
	if got := c.GetColor(); got != (math.NewColor(200, 100, 50, 255)) {
		t.Fatalf("committed color = %v, want (200,100,50)", got)
	}

	// Cancel leaves the committed color untouched.
	c.Color = math.NewColor(1, 2, 3, 255)
	c.openPanel()
	c.working = math.NewColor(9, 9, 9, 255)
	c.cancel()
	if c.open {
		t.Fatal("panel should close after cancel")
	}
	if got := c.GetColor(); got != (math.NewColor(1, 2, 3, 255)) {
		t.Fatalf("cancelled color = %v, want (1,2,3)", got)
	}
}

// TestColorPickerPanelPlacement verifies the panel opens below the swatch by default
// and flips above when it would overflow the bottom.
func TestColorPickerPanelPlacement(t *testing.T) {
	c := &ColorPickerComponent{}
	c.Width = 40
	c.Height = 20
	c.Initialize()

	// Unknown viewport -> below (default).
	c.viewportH = 0
	if c.openUp() {
		t.Fatal("expected down when the viewport is unknown")
	}

	// Fits below -> stays below.
	c.Offset = math.NewVector2(0, 0)
	c.viewportH = 500
	if c.openUp() {
		t.Fatal("expected down when the panel fits below")
	}

	// Overflow bottom, room above -> flips up.
	c.Offset = math.NewVector2(0, 400)
	c.viewportH = 420 // header top 400, bottom 420; below = 0 < panelHeight, above = 400
	if !c.openUp() {
		t.Fatal("expected up when the panel overflows the bottom")
	}

	// Fits neither -> opens on the side with more room.
	c.Offset = math.NewVector2(0, 100)
	c.viewportH = 200 // below = 80, above = 100 -> above has more room
	if !c.openUp() {
		t.Fatal("expected up when more room is above")
	}
}

// TestColorPickerPanelRect checks the panel abuts the swatch on the correct side.
func TestColorPickerPanelRect(t *testing.T) {
	c := &ColorPickerComponent{}
	c.Width = 40
	c.Height = 20
	c.Offset = math.NewVector2(10, 30)
	c.Initialize()

	c.viewportH = 0 // down
	pr := c.panelRect()
	if pr.Top() != c.headerRect().Bottom() {
		t.Fatalf("panel should start at the swatch bottom: got %v", pr.Top())
	}

	c.Offset = math.NewVector2(10, 500)
	c.viewportH = 520 // header top 500, bottom 520; below = 0, above = 500 -> up
	pr = c.panelRect()
	if pr.Bottom() != c.headerRect().Top() {
		t.Fatalf("panel should end at the swatch top when opened up: got bottom %v vs top %v", pr.Bottom(), c.headerRect().Top())
	}
}

// TestColorPickerSquareDrag verifies the square drag sets saturation/value while
// keeping the alpha channel.
func TestColorPickerSquareDrag(t *testing.T) {
	c := &ColorPickerComponent{}
	c.Color = math.NewColor(255, 0, 0, 200) // red, alpha 200
	c.Initialize()
	c.openPanel()

	// Drag to the top-right: full saturation, full value (pure red), alpha preserved.
	c.dragRegion = dragSquare
	sq := c.squareRect()
	c.applyDrag(math.NewVector2(sq.Right(), sq.Top()))
	if c.working.A != 200 {
		t.Fatalf("square drag changed alpha: got %d, want 200", c.working.A)
	}
	if c.working != math.NewColor(255, 0, 0, 200) {
		t.Fatalf("top-right drag = %v, want red", c.working)
	}

	// Drag to the bottom-left: zero saturation, zero value -> black.
	c.applyDrag(math.NewVector2(sq.Left(), sq.Bottom()))
	if c.working.R != 0 || c.working.G != 0 || c.working.B != 0 {
		t.Fatalf("bottom-left drag = %v, want black", c.working)
	}
}

// TestColorPickerAlphaDrag verifies the alpha bar sets opacity while preserving RGB.
func TestColorPickerAlphaDrag(t *testing.T) {
	c := &ColorPickerComponent{}
	c.Color = math.NewColor(255, 0, 0, 255)
	c.Initialize()
	c.openPanel()

	c.dragRegion = dragAlpha
	ab := c.alphaRect()
	c.applyDrag(math.NewVector2(ab.Left(), ab.Bottom())) // bottom = transparent
	if c.working.A != 0 {
		t.Fatalf("bottom alpha drag = %d, want 0", c.working.A)
	}
	if c.working.R != 255 {
		t.Fatalf("alpha drag changed RGB: got R=%d, want 255", c.working.R)
	}

	c.applyDrag(math.NewVector2(ab.Left(), ab.Top())) // top = opaque
	if c.working.A != 255 {
		t.Fatalf("top alpha drag = %d, want 255", c.working.A)
	}
}

func absDiff(a, b uint8) uint8 {
	if a > b {
		return a - b
	}
	return b - a
}
