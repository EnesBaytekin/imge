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
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// Renderer implements core.Renderer by drawing onto an *ebiten.Image target.
// Shapes are drawn without anti-aliasing so pixel art stays crisp.
type Renderer struct {
	target         *ebiten.Image
	textures       map[string]*ebiten.Image
	missing        map[string]bool // textures we already warned about
	viewportWidth  int
	viewportHeight int
	assetFS        fs.FS // embedded assets (web); nil means use the OS filesystem
}

func newRenderer() *Renderer {
	return &Renderer{
		textures: make(map[string]*ebiten.Image),
		missing:  make(map[string]bool),
	}
}

// begin sets the frame's draw target (called once per frame by the runner).
func (r *Renderer) begin(target *ebiten.Image) {
	r.target = target
}

func (r *Renderer) setViewport(w, h int) {
	r.viewportWidth, r.viewportHeight = w, h
}

// SetAssetFS sets the filesystem textures are loaded from. Web builds pass their
// embedded fs.FS here; desktop builds leave it nil so assets load from the OS.
func (r *Renderer) SetAssetFS(fsys fs.FS) {
	r.assetFS = fsys
}

// toRGBA converts a math.Color to the standard library color.RGBA.
func toRGBA(c math.Color) color.RGBA {
	return color.RGBA{R: c.R, G: c.G, B: c.B, A: c.A}
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
	vector.DrawFilledRect(r.target,
		float32(rect.X()), float32(rect.Y()),
		float32(rect.Width()), float32(rect.Height()),
		toRGBA(c), false)
}

// DrawRectOutline draws a rectangle outline (border only).
func (r *Renderer) DrawRectOutline(rect math.Rect, c math.Color, thickness float64) {
	if r.target == nil {
		return
	}
	vector.StrokeRect(r.target,
		float32(rect.X()), float32(rect.Y()),
		float32(rect.Width()), float32(rect.Height()),
		float32(thickness), toRGBA(c), false)
}

// DrawCircle draws a filled circle.
func (r *Renderer) DrawCircle(center math.Vector2, radius float64, c math.Color) {
	if r.target == nil {
		return
	}
	vector.DrawFilledCircle(r.target,
		float32(center.X), float32(center.Y), float32(radius),
		toRGBA(c), false)
}

// DrawCircleOutline draws a circle outline.
func (r *Renderer) DrawCircleOutline(center math.Vector2, radius float64, c math.Color, thickness float64) {
	if r.target == nil {
		return
	}
	vector.StrokeCircle(r.target,
		float32(center.X), float32(center.Y), float32(radius),
		float32(thickness), toRGBA(c), false)
}

// DrawLine draws a line between two points.
func (r *Renderer) DrawLine(start, end math.Vector2, c math.Color, thickness float64) {
	if r.target == nil {
		return
	}
	vector.StrokeLine(r.target,
		float32(start.X), float32(start.Y),
		float32(end.X), float32(end.Y),
		float32(thickness), toRGBA(c), false)
}

// DrawTexture draws a texture at the given position with scale, rotation, and tint.
// Textures are loaded lazily on first use and cached by ID.
func (r *Renderer) DrawTexture(textureID string, position math.Vector2, scale math.Vector2, rotation float64, tint math.Color) {
	if r.target == nil {
		return
	}

	img := r.loadTexture(textureID)
	if img == nil {
		return
	}

	w := float64(img.Bounds().Dx())
	h := float64(img.Bounds().Dy())
	cx := w / 2
	cy := h / 2

	opts := &ebiten.DrawImageOptions{}
	// Anchor the image at its center, apply scale and rotation, then place its
	// top-left corner at `position`.
	opts.GeoM.Translate(-cx, -cy)
	opts.GeoM.Scale(scale.X, scale.Y)
	opts.GeoM.Rotate(rotation)
	opts.GeoM.Translate(position.X+cx*scale.X, position.Y+cy*scale.Y)
	opts.ColorScale.ScaleWithColor(toRGBA(tint))

	r.target.DrawImage(img, opts)
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

// loadTexture loads and caches a texture by ID. The ID is treated as a file path,
// resolved relative to the current working directory or the assets/ directory.
func (r *Renderer) loadTexture(textureID string) *ebiten.Image {
	if img, ok := r.textures[textureID]; ok {
		return img
	}

	f, err := assetfs.Open(r.assetFS, textureID)
	if err != nil {
		if !r.missing[textureID] {
			log.Printf("ebitengine: texture not found: %q", textureID)
			r.missing[textureID] = true
		}
		return nil
	}
	defer f.Close()

	src, _, err := image.Decode(f)
	if err != nil {
		if !r.missing[textureID] {
			log.Printf("ebitengine: failed to decode texture %q: %v", textureID, err)
			r.missing[textureID] = true
		}
		return nil
	}

	// NewImageFromImage defaults to FilterNearest, which keeps pixel art crisp.
	img := ebiten.NewImageFromImage(src)
	r.textures[textureID] = img
	return img
}
