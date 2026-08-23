// Command colordemo generates the side-by-side comparison images used by
// docs/color-effects.md. It renders a sample sprite through the same pure color
// math (core/math.ColorTransform) the engine uses at draw time, so the generated
// images match what the engine actually produces.
//
// Run from the repository root:  go run ./cmd/colordemo
package main

import (
	"image"
	"image/color"
	"image/png"
	"log"
	"os"
	"path/filepath"

	"github.com/EnesBaytekin/imge/core/math"
)

// scale is the nearest-neighbor upscale applied to every output image so the
// pixel art reads clearly in documentation.
const scale = 4

func main() {
	outDir := "docs/assets"
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		log.Fatalf("mkdir %s: %v", outDir, err)
	}

	src := sampleGem()
	dominant := math.DominantHue(src)

	effects := []struct {
		name      string
		transform math.ColorTransform
	}{
		{"original", math.ColorTransform{}},
		{"tint", math.ColorTransform{Tint: math.Red}},
		{"hue", math.ColorTransform{Hue: 120}},
		{"hue_to", math.ColorTransform{HueTo: &math.Green}},
		{"grayscale", math.ColorTransform{Grayscale: true}},
		{"solid", math.ColorTransform{Solid: true, Tint: math.Red}},
		{"combo", math.ColorTransform{Grayscale: true, Tint: math.Blue}},
	}

	for _, e := range effects {
		out := applyTransform(src, e.transform.Matrix(dominant))
		out = upscale(out, scale)
		path := filepath.Join(outDir, "color-"+e.name+".png")
		if err := writePNG(path, out); err != nil {
			log.Fatalf("write %s: %v", path, err)
		}
		log.Printf("wrote %s", path)
	}
}

// applyTransform applies a color matrix to every pixel of a straight-alpha image,
// returning a new image.
func applyTransform(src image.Image, m math.ColorMatrix) *image.NRGBA {
	b := src.Bounds()
	out := image.NewNRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			c := color.NRGBAModel.Convert(src.At(x, y)).(color.NRGBA)
			r, g, bl, a := m.Apply(
				float64(c.R)/255,
				float64(c.G)/255,
				float64(c.B)/255,
				float64(c.A)/255,
			)
			out.SetNRGBA(x-b.Min.X, y-b.Min.Y, color.NRGBA{
				R: f2u8(r),
				G: f2u8(g),
				B: f2u8(bl),
				A: f2u8(a),
			})
		}
	}
	return out
}

// upscale enlarges an image by a whole factor using nearest-neighbor sampling.
func upscale(src *image.NRGBA, k int) *image.NRGBA {
	b := src.Bounds()
	out := image.NewNRGBA(image.Rect(0, 0, b.Dx()*k, b.Dy()*k))
	for y := 0; y < b.Dy(); y++ {
		for x := 0; x < b.Dx(); x++ {
			c := src.NRGBAAt(b.Min.X+x, b.Min.Y+y)
			for dy := 0; dy < k; dy++ {
				for dx := 0; dx < k; dx++ {
					out.SetNRGBA(x*k+dx, y*k+dy, c)
				}
			}
		}
	}
	return out
}

func writePNG(path string, img image.Image) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}

func f2u8(v float64) uint8 {
	if v <= 0 {
		return 0
	}
	if v >= 1 {
		return 255
	}
	return uint8(v*255 + 0.5)
}

// sampleGem builds a small 32x32 pixel-art gem with a blue-dominant body, a yellow
// sash, and a white highlight. Its mix of hues and shading makes each color effect
// (tint, hue, hue_to, grayscale, solid) clearly visible.
func sampleGem() *image.NRGBA {
	const size = 32
	img := image.NewNRGBA(image.Rect(0, 0, size, size))

	var (
		lightBlue = color.NRGBA{120, 180, 255, 255}
		blue      = color.NRGBA{60, 120, 255, 255}
		darkBlue  = color.NRGBA{24, 44, 150, 255}
		yellow    = color.NRGBA{255, 218, 60, 255}
		white     = color.NRGBA{240, 246, 255, 255}
	)

	const (
		cx = 15.5
		cy = 15.5
		r  = 11.0
	)

	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			dx := float64(x) - cx
			dy := float64(y) - cy
			if dx < 0 {
				dx = -dx
			}
			if dy < 0 {
				dy = -dy
			}
			if dx+dy > r {
				continue // transparent background
			}

			t := (cy - float64(y)) / r // +1 at top, -1 at bottom
			var c color.NRGBA
			switch {
			case t > 0.35:
				c = lightBlue // top cap
			case t > 0:
				c = blue // upper body
			case t > -0.18:
				c = yellow // central sash
			case t > -0.55:
				c = blue // lower body
			default:
				c = darkBlue // base
			}
			img.SetNRGBA(x, y, c)
		}
	}

	// A small white glint on the upper-left facet.
	for _, p := range [][2]int{{13, 8}, {14, 8}, {13, 9}} {
		img.SetNRGBA(p[0], p[1], white)
	}

	return img
}
