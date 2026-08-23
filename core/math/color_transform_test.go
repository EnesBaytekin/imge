package math

import (
	"image"
	"image/color"
	"testing"
)

func almost(a, b float64) bool {
	d := a - b
	if d < 0 {
		d = -d
	}
	return d < 1e-3
}

func apply(t ColorTransform, r, g, b, a float64) (float64, float64, float64, float64) {
	return t.Matrix(0).Apply(r, g, b, a)
}

func TestHueOf(t *testing.T) {
	cases := []struct {
		name       string
		r, g, b    float64
		want       float64
	}{
		{"red", 1, 0, 0, 0},
		{"yellow", 1, 1, 0, 60},
		{"green", 0, 1, 0, 120},
		{"cyan", 0, 1, 1, 180},
		{"blue", 0, 0, 1, 240},
		{"magenta", 1, 0, 1, 300},
		{"gray has no hue", 0.5, 0.5, 0.5, 0},
		{"black has no hue", 0, 0, 0, 0},
		{"white has no hue", 1, 1, 1, 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := HueOf(c.r, c.g, c.b); !almost(got, c.want) {
				t.Fatalf("HueOf(%v,%v,%v) = %v, want %v", c.r, c.g, c.b, got, c.want)
			}
		})
	}
}

func solidImage(r, g, b, a uint8) image.Image {
	img := image.NewNRGBA(image.Rect(0, 0, 2, 2))
	for y := 0; y < 2; y++ {
		for x := 0; x < 2; x++ {
			img.SetNRGBA(x, y, color.NRGBA{R: r, G: g, B: b, A: a})
		}
	}
	return img
}

func TestDominantHue(t *testing.T) {
	cases := []struct {
		name string
		img  image.Image
		want float64
	}{
		{"red", solidImage(255, 0, 0, 255), 0},
		{"green", solidImage(0, 255, 0, 255), 120},
		{"blue", solidImage(0, 0, 255, 255), 240},
		{"transparent ignored", solidImage(0, 255, 0, 0), 0},
		{"grayscale ignored", solidImage(128, 128, 128, 255), 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := DominantHue(c.img); !almost(got, c.want) {
				t.Fatalf("DominantHue = %v, want %v", got, c.want)
			}
		})
	}
}

func TestDominantHueCircularMean(t *testing.T) {
	// Half red (0) and half blue (240) pixels: the circular mean is 300 (or -60),
	// not the naive 120. Verifies hues wrap correctly.
	img := image.NewNRGBA(image.Rect(0, 0, 2, 1))
	img.SetNRGBA(0, 0, color.NRGBA{R: 255, G: 0, B: 0, A: 255})
	img.SetNRGBA(1, 0, color.NRGBA{R: 0, G: 0, B: 255, A: 255})
	if got := DominantHue(img); !almost(got, 300) {
		t.Fatalf("DominantHue(red+blue) = %v, want 300", got)
	}
}

func TestMatrixIdentity(t *testing.T) {
	m := IdentityColorMatrix()
	r, g, b, a := m.Apply(0.25, 0.5, 0.75, 1)
	if r != 0.25 || g != 0.5 || b != 0.75 || a != 1 {
		t.Fatalf("identity Apply = (%v,%v,%v,%v)", r, g, b, a)
	}
}

func TestTransformIsIdentity(t *testing.T) {
	if !(ColorTransform{}).IsIdentity() {
		t.Fatal("zero transform should be identity")
	}
	if !(ColorTransform{Tint: White}).IsIdentity() {
		t.Fatal("white tint should be identity")
	}
	if (ColorTransform{Tint: Red}).IsIdentity() {
		t.Fatal("red tint should not be identity")
	}
	if (ColorTransform{Hue: 120}).IsIdentity() {
		t.Fatal("hue should not be identity")
	}
	if (ColorTransform{Grayscale: true}).IsIdentity() {
		t.Fatal("grayscale should not be identity")
	}
	if (ColorTransform{Solid: true}).IsIdentity() {
		t.Fatal("solid should not be identity")
	}
}

func TestMatrixGrayscale(t *testing.T) {
	// Pure red collapses to its luma (0.299), replicated across RGB.
	r, g, b, a := apply(ColorTransform{Grayscale: true}, 1, 0, 0, 1)
	if !almost(r, 0.2990) || !almost(g, 0.2990) || !almost(b, 0.2990) || a != 1 {
		t.Fatalf("grayscale(red) = (%v,%v,%v,%v), want luma", r, g, b, a)
	}

	// White and black are unchanged (up to float rounding).
	r, g, b, a = apply(ColorTransform{Grayscale: true}, 1, 1, 1, 1)
	if !almost(r, 1) || !almost(g, 1) || !almost(b, 1) || a != 1 {
		t.Fatalf("grayscale(white) = (%v,%v,%v,%v)", r, g, b, a)
	}
	r, g, b, _ = apply(ColorTransform{Grayscale: true}, 0, 0, 0, 1)
	if r != 0 || g != 0 || b != 0 {
		t.Fatalf("grayscale(black) = (%v,%v,%v)", r, g, b)
	}
}

func TestMatrixHueRotate(t *testing.T) {
	// Neutral pixels are untouched by hue rotation (up to float rounding).
	r, g, b, _ := apply(ColorTransform{Hue: 120}, 1, 1, 1, 1)
	if !almost(r, 1) || !almost(g, 1) || !almost(b, 1) {
		t.Fatalf("hue(white) = (%v,%v,%v), want unchanged", r, g, b)
	}
	r, g, b, _ = apply(ColorTransform{Hue: 120}, 0, 0, 0, 1)
	if r != 0 || g != 0 || b != 0 {
		t.Fatalf("hue(black) = (%v,%v,%v), want unchanged", r, g, b)
	}

	// Red rotated forward 120 becomes green-hued.
	r, g, b, _ = apply(ColorTransform{Hue: 120}, 1, 0, 0, 1)
	if h := HueOf(r, g, b); !almost(h, 120) {
		t.Fatalf("hue 120 of red -> hue %v (rgb %v,%v,%v), want 120", h, r, g, b)
	}

	// Red rotated forward 240 lands in the blue family. The YCbCr rotation is a
	// linear approximation, so the hue is ~237 (blue clamped to 1), not exactly 240.
	r, g, b, _ = apply(ColorTransform{Hue: 240}, 1, 0, 0, 1)
	h := HueOf(r, g, b)
	if b != 1 || r > 0.3 || g > 0.3 || h < 200 || h > 260 {
		t.Fatalf("hue 240 of red -> hue %v (rgb %v,%v,%v), want blue family", h, r, g, b)
	}
}

func TestMatrixTintMultiply(t *testing.T) {
	// Red tint on white: green/blue channels are multiplied to zero.
	r, g, b, a := apply(ColorTransform{Tint: Red}, 1, 1, 1, 1)
	if r != 1 || g != 0 || b != 0 || a != 1 {
		t.Fatalf("tint red on white = (%v,%v,%v,%v), want (1,0,0,1)", r, g, b, a)
	}

	// Tint alpha fades the source alpha.
	_, _, _, a = apply(ColorTransform{Tint: NewColor(255, 255, 255, 128)}, 1, 1, 1, 1)
	if !almost(a, 128.0/255.0) {
		t.Fatalf("tint half-alpha = %v, want %v", a, 128.0/255.0)
	}
}

func TestMatrixSolid(t *testing.T) {
	// Solid red fills any opaque source with red, regardless of source color.
	r, g, b, a := apply(ColorTransform{Solid: true, Tint: Red}, 0, 1, 0.5, 1)
	if r != 1 || g != 0 || b != 0 || a != 1 {
		t.Fatalf("solid red = (%v,%v,%v,%v), want (1,0,0,1)", r, g, b, a)
	}

	// Solid preserves the source alpha (the sprite's shape).
	_, _, _, a = apply(ColorTransform{Solid: true, Tint: Red}, 0, 1, 0.5, 0.5)
	if !almost(a, 0.5) {
		t.Fatalf("solid translucent source alpha = %v, want 0.5", a)
	}

	// A semi-transparent tint fades the silhouette.
	_, _, _, a = apply(ColorTransform{Solid: true, Tint: NewColor(255, 0, 0, 128)}, 0, 1, 0.5, 1)
	if !almost(a, 128.0/255.0) {
		t.Fatalf("solid half-alpha tint alpha = %v, want %v", a, 128.0/255.0)
	}
}

func TestMatrixHueTo(t *testing.T) {
	// Source dominant hue is red (0); hue_to blue (240) rotates forward 240. The
	// YCbCr hue rotation is a linear approximation, so the result lands in the blue
	// family (hue ~237, blue channel clamped to 1) rather than exactly 240.
	tr := ColorTransform{HueTo: &Blue}
	r, g, b, _ := tr.Matrix(0).Apply(1, 0, 0, 1)
	h := HueOf(r, g, b)
	if b != 1 || r > 0.3 || g > 0.3 || h < 200 || h > 260 {
		t.Fatalf("hue_to blue of red -> hue %v (rgb %v,%v,%v), want blue family", h, r, g, b)
	}

	// Manual hue adds on top of the hue_to rotation: a full 360 degrees returns red.
	tr2 := ColorTransform{HueTo: &Blue, Hue: 120} // 240 - 0 + 120 = 360 -> back to red
	r, g, b, _ = tr2.Matrix(0).Apply(1, 0, 0, 1)
	if !almost(r, 1) || !almost(g, 0) || !almost(b, 0) {
		t.Fatalf("hue_to blue + hue 120 of red -> (%v,%v,%v), want red", r, g, b)
	}
}
