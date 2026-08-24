// Command renderdemo generates the comparison images used by docs/rendering.md.
// It reproduces the two rasterization models the engine applies to vector shapes
// ("chunky" = rasterize at logical resolution, upscale by pixel_per_unit, blit at
// a fractional position; "fine" = rasterize directly at framebuffer resolution)
// with a small software rasterizer, so the pictures illustrate the real geometry
// without needing a display. These images are illustrative diagrams, not
// pixel-exact engine dumps (the engine rasterizes through Ebitengine's vector
// package).
//
// Run from the repository root:  go run ./cmd/renderdemo
package main

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/gif"
	"image/png"
	"log"
	"math"
	"os"
	"path/filepath"
)

// Palette used across the images.
var (
	bg   = color.RGBA{20, 20, 48, 255}    // #141430 dark navy
	grid = color.RGBA{46, 46, 84, 255}    // #2e2e54 faint grid
	fg   = color.RGBA{102, 255, 209, 255} // #66ffd1 mint (vector shapes)
	red  = color.RGBA{255, 90, 90, 255}   // #ff5a5a sprite
)

// gifPalette is the fixed palette the animated GIFs are quantized against. Every
// frame uses only bg/grid/fg, so these three entries cover it exactly.
var gifPalette = color.Palette{bg, grid, fg, color.RGBA{255, 255, 255, 255}, color.RGBA{0, 0, 0, 255}}

func main() {
	outDir := "docs/assets"
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		log.Fatalf("mkdir %s: %v", outDir, err)
	}

	// smooth_shapes x pixel_per_unit comparison cells.
	for _, ppu := range []int{1, 2, 4, 8} {
		for _, s := range []struct {
			smooth bool
			name   string
		}{{false, "chunky"}, {true, "fine"}} {
			img := makeCell(ppu, s.smooth)
			path := filepath.Join(outDir, fmt.Sprintf("rendering-%s-ppu%d.png", s.name, ppu))
			writePNG(path, img)
		}
	}

	// Sub-pixel motion filmstrips (default chunky), one per pixel_per_unit.
	for _, ppu := range []int{1, 2, 4, 8} {
		path := filepath.Join(outDir, fmt.Sprintf("rendering-motion-ppu%d.gif", ppu))
		writeGIF(path, makeMotion(ppu))
	}

	// Sprites are always chunky, even when smooth_shapes makes vector shapes fine.
	writePNG(filepath.Join(outDir, "rendering-sprite.png"), makeSpriteCompare())
}

// newCanvas returns a width x height RGBA filled with c.
func newCanvas(w, h int, c color.RGBA) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.Draw(img, img.Bounds(), &image.Uniform{c}, image.Point{}, draw.Src)
	return img
}

// filledCircle draws a hard (non-antialiased) filled disk centered at (px, py)
// with radius r, in pixels. Only pixels whose centers fall inside the disk are
// set; the rest are left transparent.
func filledCircle(img *image.RGBA, px, py, r float64, c color.RGBA) {
	for y := int(math.Floor(py - r - 1)); y <= int(math.Ceil(py+r+1)); y++ {
		for x := int(math.Floor(px - r - 1)); x <= int(math.Ceil(px+r+1)); x++ {
			dx := float64(x) + 0.5 - px
			dy := float64(y) + 0.5 - py
			if dx*dx+dy*dy <= r*r && image.Pt(x, y).In(img.Rect) {
				img.SetRGBA(x, y, c)
			}
		}
	}
}

// nearestUpscale scales src by an integer factor k using nearest-neighbor, so
// each source pixel becomes a k x k block.
func nearestUpscale(src *image.RGBA, k int) *image.RGBA {
	if k <= 1 {
		out := image.NewRGBA(src.Bounds())
		draw.Draw(out, out.Bounds(), src, image.Point{}, draw.Src)
		return out
	}
	b := src.Bounds()
	out := image.NewRGBA(image.Rect(0, 0, b.Dx()*k, b.Dy()*k))
	for y := 0; y < b.Dy(); y++ {
		for x := 0; x < b.Dx(); x++ {
			c := src.RGBAAt(b.Min.X+x, b.Min.Y+y)
			for dy := 0; dy < k; dy++ {
				for dx := 0; dx < k; dx++ {
					out.SetRGBA(x*k+dx, y*k+dy, c)
				}
			}
		}
	}
	return out
}

// drawGrid draws 1px grid lines every `spacing` pixels (one logical unit on the
// display scale), so the reader can see where the unit grid falls.
func drawGrid(img *image.RGBA, spacing int, c color.RGBA) {
	b := img.Bounds()
	for x := 0; x < b.Dx(); x += spacing {
		for y := 0; y < b.Dy(); y++ {
			img.SetRGBA(x, y, c)
		}
	}
	for y := 0; y < b.Dy(); y += spacing {
		for x := 0; x < b.Dx(); x++ {
			img.SetRGBA(x, y, c)
		}
	}
}

// renderChunky reproduces the engine's chunky path: rasterize the shape at
// logical resolution with its anchor quantized to whole units, upscale each
// logical pixel into a pixel_per_unit-sized block, then blit it at the fractional
// position (which quantizes to 1/ppu of a logical unit on the framebuffer).
func renderChunky(cx, cy, radius float64, W, H, ppu int) *image.RGBA {
	fb := newCanvas(W*ppu, H*ppu, bg)
	pad := int(math.Ceil(radius)) + 1
	size := 2 * pad
	logical := image.NewRGBA(image.Rect(0, 0, size, size))
	filledCircle(logical, float64(pad), float64(pad), radius, fg)
	block := nearestUpscale(logical, ppu)
	blitX := int(math.Round((cx - float64(pad)) * float64(ppu)))
	blitY := int(math.Round((cy - float64(pad)) * float64(ppu)))
	draw.Draw(fb, image.Rect(blitX, blitY, blitX+size*ppu, blitY+size*ppu), block, image.Point{}, draw.Over)
	return fb
}

// renderFine reproduces the engine's fine path: rasterize the shape directly at
// framebuffer resolution at its fractional position, giving 1px edges.
func renderFine(cx, cy, radius float64, W, H, ppu int) *image.RGBA {
	fb := newCanvas(W*ppu, H*ppu, bg)
	filledCircle(fb, cx*float64(ppu), cy*float64(ppu), radius*float64(ppu), fg)
	return fb
}

// makeCell renders one smooth_shapes x pixel_per_unit cell: a filled circle at a
// fractional position, on a 16x16 logical-unit region, scaled to a fixed display
// size so the three ppu rows are visually comparable.
func makeCell(ppu int, smooth bool) *image.RGBA {
	const logical = 16
	const display = 256
	up := display / (logical * ppu)

	var fb *image.RGBA
	if smooth {
		fb = renderFine(8.3, 8.3, 3.0, logical, logical, ppu)
	} else {
		fb = renderChunky(8.3, 8.3, 3.0, logical, logical, ppu)
	}

	out := nearestUpscale(fb, up)
	drawGrid(out, ppu*up, grid)
	return out
}

// triangle returns a 0..1..0 triangle wave over n frames (ping-pong), so an
// animated motion loop reverses at the edges instead of snapping back.
func triangle(i, n int) float64 {
	half := n / 2
	p := i % n
	if p <= half {
		return float64(p) / float64(half)
	}
	return float64(n-p) / float64(half)
}

// makeMotion builds a looping GIF of a circle gliding horizontally at a
// sub-pixel speed, rendered chunky at the given pixel_per_unit. The same logical
// motion is shown at each ppu; the ppu only changes how finely the position is
// quantized on screen (1, 1/2, or 1/4 logical unit).
func makeMotion(ppu int) *gif.GIF {
	const W, H = 20, 12
	const displayW = 320
	up := displayW / (W * ppu)

	const radius = 2.5
	const cy = 6.0
	const xmin, xmax = 4.0, 14.0
	const frames = 40

	var out []*image.Paletted
	var delays []int
	for i := 0; i < frames; i++ {
		cx := xmin + triangle(i, frames)*(xmax-xmin)
		fb := renderChunky(cx, cy, radius, W, H, ppu)
		disp := nearestUpscale(fb, up)
		drawGrid(disp, ppu*up, grid)
		out = append(out, quantize(disp))
		delays = append(delays, 8)
	}
	return &gif.GIF{Image: out, Delay: delays, LoopCount: 0}
}

// heart is a 12x10 pixel-art heart used as the "sprite" in the sprite comparison.
var heart = []string{
	".XX......XX.",
	"XXXX....XXXX",
	"XXXXXXXXXXXX",
	"XXXXXXXXXXXX",
	"XXXXXXXXXXXX",
	".XXXXXXXXXX.",
	"..XXXXXXXX..",
	"...XXXXXX...",
	"....XXXX....",
	".....XX.....",
}

// makeSpriteCompare renders two panels at pixel_per_unit 4: on the left a sprite
// (which the engine always draws chunky), on the right a vector circle drawn fine
// (smooth_shapes: true). Same ppu, same fractional positions — the sprite keeps
// ppu-sized blocks while the vector shape gets 1px edges.
func makeSpriteCompare() *image.RGBA {
	const logical = 16
	const display = 192
	const ppu = 4
	up := display / (logical * ppu)

	// Left: sprite. A texture is always nearest-neighbor upscaled by ppu.
	native := image.NewRGBA(image.Rect(0, 0, 12, 10))
	for y, row := range heart {
		for x, ch := range row {
			if ch == 'X' {
				native.SetRGBA(x, y, red)
			}
		}
	}
	left := newCanvas(logical*ppu, logical*ppu, bg)
	block := nearestUpscale(native, ppu)
	bx := int(math.Round(2.3 * float64(ppu)))
	by := int(math.Round(3.4 * float64(ppu)))
	draw.Draw(left, image.Rect(bx, by, bx+block.Bounds().Dx(), by+block.Bounds().Dy()), block, image.Point{}, draw.Over)
	left = nearestUpscale(left, up)
	drawGrid(left, ppu*up, grid)

	// Right: vector circle, fine.
	right := renderFine(8.3, 8.3, 4.0, logical, logical, ppu)
	right = nearestUpscale(right, up)
	drawGrid(right, ppu*up, grid)

	const gap = 8
	panel := logical * ppu * up
	combined := newCanvas(2*panel+gap, panel, bg)
	draw.Draw(combined, image.Rect(0, 0, panel, panel), left, image.Point{}, draw.Src)
	draw.Draw(combined, image.Rect(panel+gap, 0, 2*panel+gap, panel), right, image.Point{}, draw.Src)
	return combined
}

// quantize converts an RGBA image to a Paletted image using the fixed GIF palette.
func quantize(img *image.RGBA) *image.Paletted {
	pm := image.NewPaletted(img.Bounds(), gifPalette)
	for y := 0; y < img.Bounds().Dy(); y++ {
		for x := 0; x < img.Bounds().Dx(); x++ {
			pm.Set(x, y, img.At(x, y))
		}
	}
	return pm
}

func writePNG(path string, img *image.RGBA) {
	f, err := os.Create(path)
	if err != nil {
		log.Fatalf("create %s: %v", path, err)
	}
	if err := png.Encode(f, img); err != nil {
		log.Fatalf("encode %s: %v", path, err)
	}
	if err := f.Close(); err != nil {
		log.Fatalf("close %s: %v", path, err)
	}
	log.Printf("wrote %s", path)
}

func writeGIF(path string, g *gif.GIF) {
	f, err := os.Create(path)
	if err != nil {
		log.Fatalf("create %s: %v", path, err)
	}
	if err := gif.EncodeAll(f, g); err != nil {
		log.Fatalf("encode %s: %v", path, err)
	}
	if err := f.Close(); err != nil {
		log.Fatalf("close %s: %v", path, err)
	}
	log.Printf("wrote %s", path)
}
