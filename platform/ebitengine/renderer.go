package ebitengine

import (
	"fmt"
	"image"
	"image/color"
	_ "image/jpeg" // register JPEG decoder
	_ "image/png"  // register PNG decoder
	"io/fs"
	"log"
	stdmath "math"

	"github.com/EnesBaytekin/imge/core/math"
	"github.com/EnesBaytekin/imge/platform/ebitengine/assetfs"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/colorm"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// Renderer implements core.Renderer by drawing onto an *ebiten.Image target.
// Shapes are drawn without anti-aliasing so pixel art stays crisp.
type Renderer struct {
	target         *ebiten.Image
	textures       map[string]textureEntry
	missing        map[string]bool // textures we already warned about
	viewportWidth  int
	viewportHeight int
	camX           float64 // camera view top-left corner (world coords)
	camY           float64
	camZoom        float64
	camActive      bool
	assetFS        fs.FS // embedded assets (web); nil means use the OS filesystem

	// pixelScale is the framebuffer-pixels-per-unit scale (pixel_per_unit). Every
	// world/logical coordinate is multiplied by it when mapped to the render
	// target, so a value > 1 gives sub-unit rasterization precision.
	pixelScale float64

	// smoothShapes opts vector shapes into framebuffer-resolution (fine)
	// rasterization. When false (the default) shapes render "chunky": rasterized
	// at logical resolution and upscaled, matching textures (see chunky()).
	smoothShapes bool

	// shapeCache caches logical-resolution rasterizations of vector shapes. A
	// shape's chunky pixels depend only on its geometry and color — not its
	// position — so the same buffer is reused every frame (like the texture cache).
	shapeCache map[string]*ebiten.Image
}

// textureEntry caches a decoded texture together with its dominant hue, computed
// once at load time and used by the hue_to transform.
type textureEntry struct {
	img *ebiten.Image
	hue float64
}

func newRenderer() *Renderer {
	return &Renderer{
		textures:   make(map[string]textureEntry),
		missing:    make(map[string]bool),
		shapeCache: make(map[string]*ebiten.Image),
		pixelScale: 1,
	}
}

// begin sets the frame's draw target (called once per frame by the runner).
func (r *Renderer) begin(target *ebiten.Image) {
	r.target = target
}

func (r *Renderer) setViewport(w, h int) {
	r.viewportWidth, r.viewportHeight = w, h
}

// setPixelScale sets the framebuffer-pixels-per-unit scale applied to every draw.
func (r *Renderer) setPixelScale(ppu float64) {
	if ppu <= 0 {
		ppu = 1
	}
	r.pixelScale = ppu
}

// setSmoothShapes opts vector shapes into fine (framebuffer-resolution)
// rasterization. The default is false: shapes render chunky.
func (r *Renderer) setSmoothShapes(smooth bool) {
	r.smoothShapes = smooth
}

// chunky reports whether vector shapes should rasterize at logical resolution
// (integer-anchored, deterministic) and then be upscaled and positioned, instead
// of rasterizing directly at framebuffer resolution. smoothShapes opts into the
// fine path. Quantizing the anchor keeps a shape's pixel pattern stable as it
// moves fractionally — including at pixelScale 1, where the fine path would
// re-rasterize at the fractional position and wobble every frame.
func (r *Renderer) chunky() bool {
	return !r.smoothShapes
}

// chunkySprite returns a cached logical-resolution rasterization of a shape,
// creating and rasterizing it on first use. The buffer's pixel (0,0) is the
// shape's world-space top-left, which callers position via blitChunky.
func (r *Renderer) chunkySprite(key string, w, h int, rasterize func(*ebiten.Image)) *ebiten.Image {
	if img, ok := r.shapeCache[key]; ok {
		return img
	}
	img := ebiten.NewImage(w, h)
	rasterize(img)
	r.shapeCache[key] = img
	return img
}

// blitChunky draws a logical-resolution sprite upscaled by zoom() at the fractional
// screen position of worldMin+frac. The sprite's pixels stay chunky (zoom() x zoom())
// while its position moves fractionally — the same model textures use.
func (r *Renderer) blitChunky(img *ebiten.Image, worldMin, frac math.Vector2) {
	pos := r.screenPos(math.NewVector2(worldMin.X+frac.X, worldMin.Y+frac.Y))
	z := r.zoom()
	var geoM ebiten.GeoM
	geoM.Scale(z, z)
	geoM.Translate(pos.X, pos.Y)
	r.target.DrawImage(img, &ebiten.DrawImageOptions{GeoM: geoM})
}

// colorKey encodes a color into a stable, compact cache-key suffix.
func colorKey(c math.Color) string {
	return fmt.Sprintf("%02x%02x%02x%02x", c.R, c.G, c.B, c.A)
}

// SetCamera applies a world-to-screen camera transform to subsequent draw calls.
// (cx, cy) is the view's top-left corner in world coordinates and zoom is the
// scale factor. A zoom <= 0 disables the transform (raw screen space).
func (r *Renderer) SetCamera(cx, cy, zoom float64) {
	if zoom <= 0 {
		r.camActive = false
		r.camZoom = 1
		return
	}
	r.camActive = true
	r.camX = cx
	r.camY = cy
	r.camZoom = zoom
}

// screenPos maps a world point to screen coordinates under the current camera.
func (r *Renderer) screenPos(p math.Vector2) math.Vector2 {
	if !r.camActive {
		return math.NewVector2(p.X*r.pixelScale, p.Y*r.pixelScale)
	}
	return math.NewVector2(
		(p.X-r.camX)*r.camZoom*r.pixelScale,
		(p.Y-r.camY)*r.camZoom*r.pixelScale,
	)
}

// zoom returns the current camera zoom (1 when no camera is active).
func (r *Renderer) zoom() float64 {
	if r.camActive {
		return r.camZoom * r.pixelScale
	}
	return r.pixelScale
}

// SetAssetFS sets the filesystem textures are loaded from. Web builds pass their
// embedded fs.FS here; desktop builds leave it nil so assets load from the OS.
func (r *Renderer) SetAssetFS(fsys fs.FS) {
	r.assetFS = fsys
}

// toRGBA converts a math.Color to the standard library color.RGBA. The values are
// straight (non-premultiplied) alpha, which is what Ebitengine's Fill and vector
// shape functions expect.
func toRGBA(c math.Color) color.RGBA {
	return color.RGBA{R: c.R, G: c.G, B: c.B, A: c.A}
}

// toColorm converts a core math.ColorMatrix into the Ebitengine color matrix used
// by colorm.DrawImage. Both operate on straight-alpha colors, so the elements map
// directly (i = output channel, j = input channel, j == 4 is the constant term).
func toColorm(m math.ColorMatrix) colorm.ColorM {
	var c colorm.ColorM
	for i := 0; i < 4; i++ {
		for j := 0; j < 4; j++ {
			c.SetElement(i, j, m.M[i][j])
		}
		c.SetElement(i, 4, m.T[i])
	}
	return c
}

// Clear fills the entire target with the given color.
func (r *Renderer) Clear(c math.Color) {
	if r.target != nil {
		r.target.Fill(toRGBA(c))
	}
}

// DrawRect draws a filled rectangle.
func (r *Renderer) DrawRect(rect math.Rect, c math.Color) {
	if r.target == nil {
		return
	}
	if r.chunky() {
		r.drawRectChunky(rect, c)
		return
	}
	p := r.screenPos(rect.Position)
	z := r.zoom()
	vector.DrawFilledRect(r.target,
		float32(p.X), float32(p.Y),
		float32(rect.Width()*z), float32(rect.Height()*z),
		toRGBA(c), false)
}

// drawRectChunky rasterizes the rect at logical resolution and blits it upscaled.
func (r *Renderer) drawRectChunky(rect math.Rect, c math.Color) {
	w, h := rect.Width(), rect.Height()
	if w <= 0 || h <= 0 {
		return
	}
	qx := stdmath.Round(rect.Position.X)
	qy := stdmath.Round(rect.Position.Y)
	bw, bh := int(stdmath.Ceil(w)), int(stdmath.Ceil(h))
	key := fmt.Sprintf("rect:%g:%g:%s", w, h, colorKey(c))
	img := r.chunkySprite(key, bw, bh, func(dst *ebiten.Image) {
		vector.DrawFilledRect(dst, 0, 0, float32(w), float32(h), toRGBA(c), false)
	})
	r.blitChunky(img, math.NewVector2(qx, qy), math.NewVector2(rect.Position.X-qx, rect.Position.Y-qy))
}

// DrawRectOutline draws a rectangle outline (border only).
func (r *Renderer) DrawRectOutline(rect math.Rect, c math.Color, thickness float64) {
	if r.target == nil {
		return
	}
	if r.chunky() {
		r.drawRectOutlineChunky(rect, c, thickness)
		return
	}
	p := r.screenPos(rect.Position)
	z := r.zoom()
	vector.StrokeRect(r.target,
		float32(p.X), float32(p.Y),
		float32(rect.Width()*z), float32(rect.Height()*z),
		float32(thickness*z), toRGBA(c), false)
}

// drawRectOutlineChunky rasterizes the outline at logical resolution and blits it
// upscaled. The stroke is centered on the rect perimeter, so it extends half the
// thickness outside the rect.
func (r *Renderer) drawRectOutlineChunky(rect math.Rect, c math.Color, thickness float64) {
	if thickness <= 0 {
		return
	}
	w, h := rect.Width(), rect.Height()
	qx := stdmath.Round(rect.Position.X)
	qy := stdmath.Round(rect.Position.Y)
	ox := stdmath.Ceil(thickness / 2)
	oy := stdmath.Ceil(thickness / 2)
	bw := int(stdmath.Ceil(w) + 2*ox)
	bh := int(stdmath.Ceil(h) + 2*oy)
	key := fmt.Sprintf("rectoutline:%g:%g:%g:%s", w, h, thickness, colorKey(c))
	img := r.chunkySprite(key, bw, bh, func(dst *ebiten.Image) {
		vector.StrokeRect(dst, float32(ox), float32(oy), float32(w), float32(h), float32(thickness), toRGBA(c), false)
	})
	r.blitChunky(img, math.NewVector2(qx-ox, qy-oy), math.NewVector2(rect.Position.X-qx, rect.Position.Y-qy))
}

// DrawCircle draws a filled circle.
func (r *Renderer) DrawCircle(center math.Vector2, radius float64, c math.Color) {
	if r.target == nil {
		return
	}
	if r.chunky() {
		r.drawCircleChunky(center, radius, c)
		return
	}
	p := r.screenPos(center)
	z := r.zoom()
	vector.DrawFilledCircle(r.target,
		float32(p.X), float32(p.Y), float32(radius*z),
		toRGBA(c), false)
}

// drawCircleChunky rasterizes the circle at logical resolution and blits it
// upscaled. The center is quantized to a whole unit so the circle's edge snaps to
// the unit grid; the sub-unit remainder is applied as a sub-pixel blit offset.
func (r *Renderer) drawCircleChunky(center math.Vector2, radius float64, c math.Color) {
	if radius <= 0 {
		return
	}
	qx := stdmath.Round(center.X)
	qy := stdmath.Round(center.Y)
	pad := int(stdmath.Ceil(radius)) + 1 // +1 keeps the edge from clipping
	key := fmt.Sprintf("circle:%g:%s", radius, colorKey(c))
	img := r.chunkySprite(key, 2*pad, 2*pad, func(dst *ebiten.Image) {
		vector.DrawFilledCircle(dst, float32(pad), float32(pad), float32(radius), toRGBA(c), false)
	})
	r.blitChunky(img,
		math.NewVector2(qx-float64(pad), qy-float64(pad)),
		math.NewVector2(center.X-qx, center.Y-qy))
}

// DrawCircleOutline draws a circle outline.
func (r *Renderer) DrawCircleOutline(center math.Vector2, radius float64, c math.Color, thickness float64) {
	if r.target == nil {
		return
	}
	if r.chunky() {
		r.drawCircleOutlineChunky(center, radius, c, thickness)
		return
	}
	p := r.screenPos(center)
	z := r.zoom()
	vector.StrokeCircle(r.target,
		float32(p.X), float32(p.Y), float32(radius*z),
		float32(thickness*z), toRGBA(c), false)
}

// drawCircleOutlineChunky rasterizes the outline at logical resolution and blits it
// upscaled. The stroke is centered on the circle of the given radius, so it extends
// half the thickness beyond it.
func (r *Renderer) drawCircleOutlineChunky(center math.Vector2, radius float64, c math.Color, thickness float64) {
	if thickness <= 0 {
		return
	}
	qx := stdmath.Round(center.X)
	qy := stdmath.Round(center.Y)
	pad := int(stdmath.Ceil(radius+thickness/2)) + 1
	key := fmt.Sprintf("circleoutline:%g:%g:%s", radius, thickness, colorKey(c))
	img := r.chunkySprite(key, 2*pad, 2*pad, func(dst *ebiten.Image) {
		vector.StrokeCircle(dst, float32(pad), float32(pad), float32(radius), float32(thickness), toRGBA(c), false)
	})
	r.blitChunky(img,
		math.NewVector2(qx-float64(pad), qy-float64(pad)),
		math.NewVector2(center.X-qx, center.Y-qy))
}

// DrawLine draws a line between two points.
func (r *Renderer) DrawLine(start, end math.Vector2, c math.Color, thickness float64) {
	if r.target == nil {
		return
	}
	if r.chunky() {
		r.drawLineChunky(start, end, c, thickness)
		return
	}
	s := r.screenPos(start)
	e := r.screenPos(end)
	z := r.zoom()
	vector.StrokeLine(r.target,
		float32(s.X), float32(s.Y),
		float32(e.X), float32(e.Y),
		float32(thickness*z), toRGBA(c), false)
}

// drawLineChunky rasterizes the line at logical resolution and blits it upscaled.
// Both endpoints snap to whole units (a line has no single anchor to keep
// fractional), and the stroke extends half the thickness around the line.
func (r *Renderer) drawLineChunky(start, end math.Vector2, c math.Color, thickness float64) {
	if thickness <= 0 {
		return
	}
	x0 := stdmath.Round(start.X)
	y0 := stdmath.Round(start.Y)
	x1 := stdmath.Round(end.X)
	y1 := stdmath.Round(end.Y)
	minX := stdmath.Min(x0, x1)
	minY := stdmath.Min(y0, y1)
	maxX := stdmath.Max(x0, x1)
	maxY := stdmath.Max(y0, y1)
	pad := stdmath.Ceil(thickness / 2)
	bw := int(maxX - minX + 2*pad)
	bh := int(maxY - minY + 2*pad)
	key := fmt.Sprintf("line:%g:%g:%g:%s", x1-x0, y1-y0, thickness, colorKey(c))
	img := r.chunkySprite(key, bw, bh, func(dst *ebiten.Image) {
		vector.StrokeLine(dst,
			float32(x0-minX+pad), float32(y0-minY+pad),
			float32(x1-minX+pad), float32(y1-minY+pad),
			float32(thickness), toRGBA(c), false)
	})
	// No fractional offset: both endpoints are snapped to the grid.
	r.blitChunky(img, math.NewVector2(minX-pad, minY-pad), math.NewVector2(0, 0))
}

// DrawTexture draws a texture (or a sub-region of it) at the given position with
// scale, rotation, and a color transform. Textures are loaded lazily on first use
// and cached by ID.
func (r *Renderer) DrawTexture(textureID string, src math.Rect, position math.Vector2, scale math.Vector2, rotation float64, transform math.ColorTransform) {
	if r.target == nil {
		return
	}

	img, hue := r.loadTexture(textureID)
	if img == nil {
		return
	}

	var drawImg *ebiten.Image
	var w, h float64

	if src.Width() > 0 && src.Height() > 0 {
		x, y := int(src.X()), int(src.Y())
		iw, ih := int(src.Width()), int(src.Height())
		drawImg = img.SubImage(image.Rect(x, y, x+iw, y+ih)).(*ebiten.Image)
		w, h = float64(iw), float64(ih)
	} else {
		b := img.Bounds()
		drawImg = img
		w, h = float64(b.Dx()), float64(b.Dy())
	}

	// Apply the camera: world position -> screen position, world scale -> screen
	// scale. Rotation is unchanged (uniform zoom preserves angles).
	pos := r.screenPos(position)
	z := r.zoom()
	sx := scale.X * z
	sy := scale.Y * z

	cx := w / 2
	cy := h / 2

	// Negative scale is used for flips. Mirror around the drawn image's center so a
	// flipped sprite keeps the same bounding box (top-left at `pos`) instead of
	// flipping around its top-left corner, which would shift it by one full drawn
	// width/height.
	px := pos.X
	py := pos.Y
	if sx < 0 {
		px -= sx * w // sx is negative, so this adds the drawn width
	}
	if sy < 0 {
		py -= sy * h
	}

	// Anchor the (sub-)image at its center, apply scale and rotation, then place
	// its top-left corner at `pos`.
	var geoM ebiten.GeoM
	geoM.Translate(-cx, -cy)
	geoM.Scale(sx, sy)
	geoM.Rotate(rotation)
	geoM.Translate(px+cx*sx, py+cy*sy)

	// The common case is no color transform, so keep the plain texture shader path.
	if transform.IsIdentity() {
		r.target.DrawImage(drawImg, &ebiten.DrawImageOptions{GeoM: geoM})
		return
	}

	cm := toColorm(transform.Matrix(hue))
	colorm.DrawImage(r.target, drawImg, cm, &colorm.DrawImageOptions{GeoM: geoM})
}

// GetTextureSize returns the natural pixel size of a texture, loading it if
// needed. Returns (0, 0) when the texture cannot be loaded.
func (r *Renderer) GetTextureSize(textureID string) (float64, float64) {
	img, _ := r.loadTexture(textureID)
	if img == nil {
		return 0, 0
	}
	b := img.Bounds()
	return float64(b.Dx()), float64(b.Dy())
}

// Present is a no-op; Ebitengine presents the frame automatically.
func (r *Renderer) Present() {}

// SetViewport sets the rendering viewport size.
func (r *Renderer) SetViewport(width, height int) {
	r.setViewport(width, height)
}

// GetViewportSize returns the current viewport size.
func (r *Renderer) GetViewportSize() (width, height int) {
	return r.viewportWidth, r.viewportHeight
}

// loadTexture loads and caches a texture by ID. The ID is treated as a
// project-root-relative file path, resolved exactly as written (no assets/
// fallback).
func (r *Renderer) loadTexture(textureID string) (*ebiten.Image, float64) {
	if e, ok := r.textures[textureID]; ok {
		return e.img, e.hue
	}

	f, err := assetfs.Open(r.assetFS, textureID)
	if err != nil {
		if !r.missing[textureID] {
			log.Printf("ebitengine: texture not found: %q", textureID)
			r.missing[textureID] = true
		}
		return nil, 0
	}
	defer f.Close()

	src, _, err := image.Decode(f)
	if err != nil {
		if !r.missing[textureID] {
			log.Printf("ebitengine: failed to decode texture %q: %v", textureID, err)
			r.missing[textureID] = true
		}
		return nil, 0
	}

	// NewImageFromImage defaults to FilterNearest, which keeps pixel art crisp.
	// DominantHue is computed once here (from the decoded source) and cached, so
	// hue_to does not re-scan the texture every frame.
	hue := math.DominantHue(src)
	img := ebiten.NewImageFromImage(src)
	r.textures[textureID] = textureEntry{img: img, hue: hue}
	return img, hue
}
