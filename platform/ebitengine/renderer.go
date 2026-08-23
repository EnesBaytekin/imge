package ebitengine

import (
	"image"
	"image/color"
	_ "image/jpeg" // register JPEG decoder
	_ "image/png"  // register PNG decoder
	"io/fs"
	"log"

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
	p := r.screenPos(rect.Position)
	z := r.zoom()
	vector.DrawFilledRect(r.target,
		float32(p.X), float32(p.Y),
		float32(rect.Width()*z), float32(rect.Height()*z),
		toRGBA(c), false)
}

// DrawRectOutline draws a rectangle outline (border only).
func (r *Renderer) DrawRectOutline(rect math.Rect, c math.Color, thickness float64) {
	if r.target == nil {
		return
	}
	p := r.screenPos(rect.Position)
	z := r.zoom()
	vector.StrokeRect(r.target,
		float32(p.X), float32(p.Y),
		float32(rect.Width()*z), float32(rect.Height()*z),
		float32(thickness*z), toRGBA(c), false)
}

// DrawCircle draws a filled circle.
func (r *Renderer) DrawCircle(center math.Vector2, radius float64, c math.Color) {
	if r.target == nil {
		return
	}
	p := r.screenPos(center)
	z := r.zoom()
	vector.DrawFilledCircle(r.target,
		float32(p.X), float32(p.Y), float32(radius*z),
		toRGBA(c), false)
}

// DrawCircleOutline draws a circle outline.
func (r *Renderer) DrawCircleOutline(center math.Vector2, radius float64, c math.Color, thickness float64) {
	if r.target == nil {
		return
	}
	p := r.screenPos(center)
	z := r.zoom()
	vector.StrokeCircle(r.target,
		float32(p.X), float32(p.Y), float32(radius*z),
		float32(thickness*z), toRGBA(c), false)
}

// DrawLine draws a line between two points.
func (r *Renderer) DrawLine(start, end math.Vector2, c math.Color, thickness float64) {
	if r.target == nil {
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
