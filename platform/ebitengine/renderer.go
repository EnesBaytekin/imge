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
	// at logical resolution and upscaled, matching textures (see shapeRes()).
	smoothShapes bool

	// smoothRotation opts texture rotation into framebuffer-resolution (fine)
	// rasterization. When false (the default) rotated textures render "chunky":
	// rasterized at logical resolution and upscaled, so rotation is quantized to
	// logical pixels (pixel-perfect) instead of sampled at sub-unit precision.
	smoothRotation bool

	// smoothRes overrides the rasterization resolution (px per world unit) of the
	// smooth paths. The default 0 is the game's behavior: smooth shapes rasterize
	// directly at the current zoom resolution, while smooth rotations rasterize at
	// pixelScale (the game's pixel_per_unit), so a rotated texture's pixels are
	// position-independent. A positive value rasterizes both into a buffer at that
	// fixed resolution and upscales by zoom()/smoothRes, so their sub-unit precision
	// is fixed (the target game's pixel_per_unit) instead of growing with the camera
	// zoom — the editor's "magnifier" behavior, so the viewport shows the scene
	// exactly as the game renders it, whatever the editor's pan/zoom.
	smoothRes float64

	// Object transform (local -> world), applied in addition to the camera while a
	// non-UI object is drawing (see SetObjectTransform). objActive is false in
	// normal rendering, in which case primitives operate in raw world space.
	objActive bool
	objPos    math.Vector2
	objRot    float64
	objScale  math.Vector2

	// shapeCache caches logical-resolution rasterizations of vector shapes. A
	// shape's chunky pixels depend only on its geometry and color — not its
	// position — so the same buffer is reused every frame (like the texture cache).
	// shapeCacheOrder records insertion order so the cache can evict its oldest
	// entries once it passes shapeCacheMaxEntries (see chunkySprite).
	shapeCache      map[string]*ebiten.Image
	shapeCacheOrder []string

	// fonts caches parsed fonts and size-specific text faces for DrawText and
	// MeasureText. See font.go.
	fonts fontState

	// Clip state: while a clip is active, drawing goes to an offscreen sub-target
	// (clipTarget) instead of the frame target (rootTarget). screenPos subtracts the
	// clip's pixel origin so coordinates stay consistent, and ClearClip blits the
	// offscreen back at that origin. clipActive is tracked separately from the
	// buffer so the offscreen can be reused across frames.
	rootTarget *ebiten.Image
	clipActive bool
	clipTarget *ebiten.Image
	clipW      int
	clipH      int
	clipOX     float64
	clipOY     float64
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
		fonts:      newFontState(),
		pixelScale: 1,
	}
}

// whiteImage is a 1x1 opaque white image used as the solid source for the
// triangle-based polygon fills (rotated rects). Vertex colors carry the actual
// color, so the source is always white. It follows the same package-init pattern
// as the vector package's own white image, so it is safe to create up front.
var whiteImage = func() *ebiten.Image {
	img := ebiten.NewImage(1, 1)
	img.WritePixels([]byte{0xff, 0xff, 0xff, 0xff})
	return img
}()

// begin sets the frame's draw target (called once per frame by the runner).
func (r *Renderer) begin(target *ebiten.Image) {
	r.target = target
	r.rootTarget = target
	r.clipActive = false
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

// SetSmoothShapes opts vector shapes into fine (framebuffer-resolution)
// rasterization. The default is false: shapes render chunky.
func (r *Renderer) SetSmoothShapes(smooth bool) {
	r.smoothShapes = smooth
}

// SetSmoothRotation opts texture rotation into fine (framebuffer-resolution)
// rasterization. The default is false: rotated textures render chunky.
func (r *Renderer) SetSmoothRotation(smooth bool) {
	r.smoothRotation = smooth
}

// SetSmoothResolution overrides the rasterization resolution (px per world unit) of
// the smooth paths. res <= 0 restores the game's default (smooth shapes directly at
// the current zoom, smooth rotations at pixelScale); res > 0 rasterizes both into a
// buffer at that fixed resolution and upscales, so sub-unit precision is decoupled
// from camera zoom. The editor sets this to the target game's pixel_per_unit while
// drawing its scene, and restores it to 0 for its own UI.
func (r *Renderer) SetSmoothResolution(res float64) {
	r.smoothRes = res
}

// shapeRes returns the rasterization resolution (px per world unit) for vector
// shapes: 1 = chunky (rasterize at logical resolution and upscale), 0 = smooth at
// the current zoom (direct framebuffer rasterization), N>0 = smooth at a fixed N
// px/unit (rasterize into a buffer and upscale by zoom()/N). Quantizing the anchor
// to the raster grid keeps a shape's pixel pattern stable as it moves fractionally.
func (r *Renderer) shapeRes() float64 {
	if !r.smoothShapes {
		return 1
	}
	return r.smoothRes
}

// textureRes returns the rasterization resolution (px per world unit) for rotated
// textures: 1 = chunky (rasterize at logical resolution and upscale), N>0 = smooth at
// N px/unit (rasterize into a buffer and upscale by zoom()/N). For smooth rotation N
// is the editor's fixed smoothRes when set, else the renderer's pixelScale (the game's
// pixel_per_unit). Rasterizing into a buffer keeps a rotated texture's pixels
// position-independent — the image is rotated once at a fixed resolution regardless of
// where it sits, then blitted into place — so a fractional object position only shifts
// the whole image by a sub-pixel amount instead of re-sampling the rotation.
func (r *Renderer) textureRes() float64 {
	if !r.smoothRotation {
		return 1
	}
	if r.smoothRes > 0 {
		return r.smoothRes
	}
	return r.pixelScale
}

// shapeCacheMaxEntries bounds the shape cache so a long session can't exhaust
// memory. Every entry is a rasterized image; a shape whose size varies each frame
// (a slider fill, an outline under a smooth zoom) would otherwise mint a new entry
// indefinitely. Beyond the cap the oldest entries are disposed — their chunky
// pixels are cheap to re-rasterize on next use, so the cache only needs to hold
// the shapes drawn in the last few frames, not every shape ever drawn.
const shapeCacheMaxEntries = 4096

// chunkySprite returns a cached logical-resolution rasterization of a shape,
// creating and rasterizing it on first use. The buffer's pixel (0,0) is the
// shape's world-space top-left, which callers position via blitRes.
func (r *Renderer) chunkySprite(key string, w, h int, rasterize func(*ebiten.Image)) *ebiten.Image {
	if img, ok := r.shapeCache[key]; ok {
		return img
	}
	img := ebiten.NewImage(w, h)
	rasterize(img)
	r.shapeCache[key] = img
	r.shapeCacheOrder = append(r.shapeCacheOrder, key)

	// Evict the oldest entries past the cap. Dispose releases the image's atlas
	// region immediately; a bare delete would keep GPU memory held until GC.
	for len(r.shapeCache) > shapeCacheMaxEntries {
		oldest := r.shapeCacheOrder[0]
		r.shapeCacheOrder = r.shapeCacheOrder[1:]
		if evicted, ok := r.shapeCache[oldest]; ok {
			evicted.Dispose()
			delete(r.shapeCache, oldest)
		}
	}
	return img
}

// blitRes draws a buffer rasterized at `res` px per world unit onto the target,
// upscaled by zoom()/res so its res pixels land at zoom() screen pixels — the same
// on-screen size whatever res is, so a shape rasterized at a higher resolution
// (smoothRes) still covers the same screen area as its chunky (res=1) counterpart.
func (r *Renderer) blitRes(img *ebiten.Image, worldMin, frac math.Vector2, res float64) {
	pos := r.screenPos(math.NewVector2(worldMin.X+frac.X, worldMin.Y+frac.Y))
	z := r.zoom() / res
	var geoM ebiten.GeoM
	geoM.Scale(z, z)
	geoM.Translate(pos.X, pos.Y)
	r.target.DrawImage(img, &ebiten.DrawImageOptions{GeoM: geoM})
}

// blitChunky draws a logical-resolution sprite upscaled by zoom() at the fractional
// screen position of worldMin+frac — the blitRes case with res=1. Text uses it, since
// text is always rendered chunky (rasterized at logical resolution and upscaled).
func (r *Renderer) blitChunky(img *ebiten.Image, worldMin, frac math.Vector2) {
	r.blitRes(img, worldMin, frac, 1)
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

// SetObjectTransform applies a world-space object transform to subsequent draw
// calls (see core.Renderer). Object.Draw sets it before drawing a non-UI object's
// components; ClearObjectTransform removes it.
func (r *Renderer) SetObjectTransform(pos math.Vector2, rotation float64, scale math.Vector2) {
	r.objActive = true
	r.objPos = pos
	r.objRot = rotation
	r.objScale = scale
}

// ClearObjectTransform removes any object transform set by SetObjectTransform.
func (r *Renderer) ClearObjectTransform() {
	r.objActive = false
}

// objectToWorld maps a local-space point to world space under the active object
// transform (scale -> rotate -> translate), matching math.Transform.LocalToWorld.
// It returns the point unchanged when no object transform is active.
func (r *Renderer) objectToWorld(p math.Vector2) math.Vector2 {
	if !r.objActive {
		return p
	}
	// Fast path for the common unrotated, unit-scale object: a plain translate, no
	// trig. The object transform is applied to every non-UI object every frame, so
	// this keeps that path cheap.
	if r.objScale.X == 1 && r.objScale.Y == 1 && r.objRot == 0 {
		return math.NewVector2(p.X+r.objPos.X, p.Y+r.objPos.Y)
	}
	x := p.X * r.objScale.X
	y := p.Y * r.objScale.Y
	cos, sin := stdmath.Cos(r.objRot), stdmath.Sin(r.objRot)
	return math.NewVector2(
		r.objPos.X+x*cos-y*sin,
		r.objPos.Y+x*sin+y*cos,
	)
}

// objectHasRotation reports whether the active object transform includes a
// rotation that would turn axis-aligned local geometry into a rotated shape.
func (r *Renderer) objectHasRotation() bool {
	return r.objActive && r.objRot != 0
}

// objectRectCorners returns the four world-space corners (top-left, top-right,
// bottom-right, bottom-left) of a local-space rectangle under the active object
// transform. It is used to draw a rect as a rotated polygon when the object has
// rotation, since the axis-aligned local rect becomes a rotated quad in world space.
func (r *Renderer) objectRectCorners(rect math.Rect) [4]math.Vector2 {
	x, y := rect.X(), rect.Y()
	w, h := rect.Width(), rect.Height()
	return [4]math.Vector2{
		r.objectToWorld(math.NewVector2(x, y)),
		r.objectToWorld(math.NewVector2(x+w, y)),
		r.objectToWorld(math.NewVector2(x+w, y+h)),
		r.objectToWorld(math.NewVector2(x, y+h)),
	}
}

// rectToWorld maps a local-space rectangle to its world-space axis-aligned bounding
// box under the active object transform. It is only correct when the transform has
// no rotation (a scale+translate keeps the rect axis-aligned); callers branch on
// objectHasRotation first. Corner AABB handles negative object scale (mirror).
func (r *Renderer) rectToWorld(rect math.Rect) math.Rect {
	c := r.objectRectCorners(rect)
	minX := stdmath.Min(stdmath.Min(c[0].X, c[1].X), stdmath.Min(c[2].X, c[3].X))
	minY := stdmath.Min(stdmath.Min(c[0].Y, c[1].Y), stdmath.Min(c[2].Y, c[3].Y))
	maxX := stdmath.Max(stdmath.Max(c[0].X, c[1].X), stdmath.Max(c[2].X, c[3].X))
	maxY := stdmath.Max(stdmath.Max(c[0].Y, c[1].Y), stdmath.Max(c[2].Y, c[3].Y))
	return math.NewRect(minX, minY, maxX-minX, maxY-minY)
}

// objectRadiusScale returns the factor to scale a circle's radius by under the active
// object transform. A circle stays a circle under rotation, so only scale affects it;
// non-uniform scale is approximated by the geometric mean of the axis scales (exact
// for uniform scale, which is the common case).
func (r *Renderer) objectRadiusScale() float64 {
	if !r.objActive {
		return 1
	}
	return stdmath.Sqrt(stdmath.Abs(r.objScale.X * r.objScale.Y))
}

// fillPolygonScreen rasterizes a filled convex polygon given its screen-space
// vertices, triangulated as a fan from the first vertex. Vertex colors carry the
// fill color (premultiplied), matching the vector package's FillCircle path.
func (r *Renderer) fillPolygonScreen(pts []math.Vector2, c math.Color) {
	if r.target == nil || len(pts) < 3 {
		return
	}
	cr, cg, cb, ca := toRGBA(c).RGBA()
	crf := float32(cr) / 0xffff
	cgf := float32(cg) / 0xffff
	cbf := float32(cb) / 0xffff
	caf := float32(ca) / 0xffff
	vs := make([]ebiten.Vertex, 0, len(pts))
	for _, p := range pts {
		vs = append(vs, ebiten.Vertex{
			DstX:   float32(p.X),
			DstY:   float32(p.Y),
			SrcX:   0,
			SrcY:   0,
			ColorR: crf,
			ColorG: cgf,
			ColorB: cbf,
			ColorA: caf,
		})
	}
	is := make([]uint16, 0, 3*(len(pts)-2))
	for i := 2; i < len(pts); i++ {
		is = append(is, 0, uint16(i-1), uint16(i))
	}
	op := &ebiten.DrawTrianglesOptions{}
	op.ColorScaleMode = ebiten.ColorScaleModePremultipliedAlpha
	r.target.DrawTriangles(vs, is, whiteImage, op)
}

// drawFilledPolygonWorld draws a filled convex quadrilateral (world-space corners)
// with the given color, honoring the smooth/chunky shape setting. The smooth path
// rasterizes directly at framebuffer resolution; the chunky path rasterizes at
// logical resolution and blits upscaled, so the polygon's pixel pattern stays
// stable and pixel-perfect.
func (r *Renderer) drawFilledPolygonWorld(corners [4]math.Vector2, c math.Color) {
	if res := r.shapeRes(); res > 0 {
		r.drawFilledPolygonRes(corners, c, res)
		return
	}
	pts := make([]math.Vector2, 4)
	for i := range corners {
		pts[i] = r.screenPos(corners[i])
	}
	r.fillPolygonScreen(pts, c)
}

// drawFilledPolygonRes rasterizes a filled polygon at `res` px/unit (its corners
// quantized to the 1/res grid, so its pixel pattern is deterministic regardless of
// fractional motion below a game pixel) and blits it upscaled by zoom()/res — the
// generalized analog of the chunky path for rotated shapes. res=1 is the chunky
// case; res>1 is the editor's ppu-based smooth case. The cache key encodes the full
// quantized shape, since a rotated rect's pattern depends on more than just its
// bounding-box size.
func (r *Renderer) drawFilledPolygonRes(corners [4]math.Vector2, c math.Color, res float64) {
	var q [4]math.Vector2
	for i := range corners {
		q[i] = math.NewVector2(stdmath.Round(corners[i].X*res)/res, stdmath.Round(corners[i].Y*res)/res)
	}
	minX, minY := q[0].X, q[0].Y
	maxX, maxY := q[0].X, q[0].Y
	for _, p := range q[1:] {
		minX = stdmath.Min(minX, p.X)
		minY = stdmath.Min(minY, p.Y)
		maxX = stdmath.Max(maxX, p.X)
		maxY = stdmath.Max(maxY, p.Y)
	}
	bw := int(stdmath.Round((maxX - minX) * res))
	bh := int(stdmath.Round((maxY - minY) * res))
	if bw <= 0 || bh <= 0 {
		return
	}
	key := fmt.Sprintf("poly:%d:%d:%d:%d:%d:%d:%d:%d:%d:%d:%d:%s",
		int(res), bw, bh,
		int((q[0].X-minX)*res), int((q[0].Y-minY)*res),
		int((q[1].X-minX)*res), int((q[1].Y-minY)*res),
		int((q[2].X-minX)*res), int((q[2].Y-minY)*res),
		int((q[3].X-minX)*res), int((q[3].Y-minY)*res),
		colorKey(c))
	img := r.chunkySprite(key, bw, bh, func(dst *ebiten.Image) {
		pts := []math.Vector2{
			math.NewVector2((q[0].X-minX)*res, (q[0].Y-minY)*res),
			math.NewVector2((q[1].X-minX)*res, (q[1].Y-minY)*res),
			math.NewVector2((q[2].X-minX)*res, (q[2].Y-minY)*res),
			math.NewVector2((q[3].X-minX)*res, (q[3].Y-minY)*res),
		}
		cr, cg, cb, ca := toRGBA(c).RGBA()
		crf := float32(cr) / 0xffff
		cgf := float32(cg) / 0xffff
		cbf := float32(cb) / 0xffff
		caf := float32(ca) / 0xffff
		vs := make([]ebiten.Vertex, 0, 4)
		for _, p := range pts {
			vs = append(vs, ebiten.Vertex{
				DstX:   float32(p.X),
				DstY:   float32(p.Y),
				SrcX:   0,
				SrcY:   0,
				ColorR: crf,
				ColorG: cgf,
				ColorB: cbf,
				ColorA: caf,
			})
		}
		op := &ebiten.DrawTrianglesOptions{}
		op.ColorScaleMode = ebiten.ColorScaleModePremultipliedAlpha
		dst.DrawTriangles(vs, []uint16{0, 1, 2, 0, 2, 3}, whiteImage, op)
	})
	r.blitRes(img, math.NewVector2(minX, minY), math.NewVector2(0, 0), res)
}

// screenPos maps a world point to screen coordinates under the current camera,
// shifted into the clip offscreen's coordinate space when a clip is active.
func (r *Renderer) screenPos(p math.Vector2) math.Vector2 {
	var x, y float64
	if !r.camActive {
		x = p.X * r.pixelScale
		y = p.Y * r.pixelScale
	} else {
		x = (p.X - r.camX) * r.camZoom * r.pixelScale
		y = (p.Y - r.camY) * r.camZoom * r.pixelScale
	}
	if r.clipActive {
		x -= r.clipOX
		y -= r.clipOY
	}
	return math.NewVector2(x, y)
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
	if r.objActive {
		if r.objectHasRotation() {
			r.drawFilledPolygonWorld(r.objectRectCorners(rect), c)
			return
		}
		rect = r.rectToWorld(rect)
	}
	if res := r.shapeRes(); res > 0 {
		r.drawRectRes(rect, c, res)
		return
	}
	p := r.screenPos(rect.Position)
	z := r.zoom()
	vector.DrawFilledRect(r.target,
		float32(p.X), float32(p.Y),
		float32(rect.Width()*z), float32(rect.Height()*z),
		toRGBA(c), false)
}

// drawRectRes rasterizes the rect at `res` px/unit and blits it upscaled by
// zoom()/res. res=1 is the chunky path; res>1 is the editor's ppu-based smooth path.
func (r *Renderer) drawRectRes(rect math.Rect, c math.Color, res float64) {
	// Snap width/height to the res grid (matching the line path, which snaps its
	// endpoints) so a rect whose size changes fractionally each frame — a slider
	// fill, a scrollbar thumb — reuses one of a few cached buffers instead of
	// minting a new image per sub-pixel change.
	w := stdmath.Round(rect.Width() * res)
	h := stdmath.Round(rect.Height() * res)
	if w <= 0 || h <= 0 {
		return
	}
	qx := stdmath.Round(rect.Position.X*res) / res
	qy := stdmath.Round(rect.Position.Y*res) / res
	bw, bh := int(w), int(h)
	key := fmt.Sprintf("rect:%d:%d:%d:%s", int(res), bw, bh, colorKey(c))
	img := r.chunkySprite(key, bw, bh, func(dst *ebiten.Image) {
		vector.DrawFilledRect(dst, 0, 0, float32(w), float32(h), toRGBA(c), false)
	})
	r.blitRes(img, math.NewVector2(qx, qy), math.NewVector2(rect.Position.X-qx, rect.Position.Y-qy), res)
}

// DrawRectOutline draws a rectangle outline (border only).
func (r *Renderer) DrawRectOutline(rect math.Rect, c math.Color, thickness float64) {
	if r.target == nil {
		return
	}
	if r.objActive {
		if r.objectHasRotation() {
			// A rotated rect outline is just its four rotated edges.
			corners := r.objectRectCorners(rect)
			r.drawLineWorld(corners[0], corners[1], c, thickness)
			r.drawLineWorld(corners[1], corners[2], c, thickness)
			r.drawLineWorld(corners[2], corners[3], c, thickness)
			r.drawLineWorld(corners[3], corners[0], c, thickness)
			return
		}
		rect = r.rectToWorld(rect)
	}
	if res := r.shapeRes(); res > 0 {
		r.drawRectOutlineRes(rect, c, thickness, res)
		return
	}
	p := r.screenPos(rect.Position)
	z := r.zoom()
	vector.StrokeRect(r.target,
		float32(p.X), float32(p.Y),
		float32(rect.Width()*z), float32(rect.Height()*z),
		float32(thickness*z), toRGBA(c), false)
}

// drawRectOutlineRes rasterizes the outline at `res` px/unit and blits it upscaled by
// zoom()/res, aligned to the filled rect's grid. res=1 is the chunky path; res>1 is
// the editor's ppu-based smooth path. It draws four crisp edge strips inside the
// bounds, sharing the fill's (0,0) anchor and fractional width/height, so the
// outline's outer edge lands exactly on the filled rect's edge at every zoom. The old
// centered StrokeRect straddled the boundary by half a pixel, which read as a one-pixel
// shift at high zoom.
func (r *Renderer) drawRectOutlineRes(rect math.Rect, c math.Color, thickness float64, res float64) {
	t := stdmath.Round(thickness * res)
	if t <= 0 {
		return
	}
	w := stdmath.Round(rect.Width() * res)
	h := stdmath.Round(rect.Height() * res)
	if w <= 0 || h <= 0 {
		return
	}
	qx := stdmath.Round(rect.Position.X*res) / res
	qy := stdmath.Round(rect.Position.Y*res) / res
	bw, bh := int(w), int(h)
	key := fmt.Sprintf("rectoutline:%d:%d:%d:%d:%s", int(res), bw, bh, int(t), colorKey(c))
	img := r.chunkySprite(key, bw, bh, func(dst *ebiten.Image) {
		fw, fh := float32(w), float32(h)
		ft := float32(t)
		// Top and bottom strips.
		vector.DrawFilledRect(dst, 0, 0, fw, ft, toRGBA(c), false)
		vector.DrawFilledRect(dst, 0, fh-ft, fw, ft, toRGBA(c), false)
		// Left and right strips, minus the corners already covered.
		inner := fh - 2*ft
		if inner < 0 {
			inner = 0
		}
		vector.DrawFilledRect(dst, 0, ft, ft, inner, toRGBA(c), false)
		vector.DrawFilledRect(dst, fw-ft, ft, ft, inner, toRGBA(c), false)
	})
	r.blitRes(img, math.NewVector2(qx, qy), math.NewVector2(rect.Position.X-qx, rect.Position.Y-qy), res)
}

// DrawRectOutlineScreen draws a rectangle outline in screen space with a constant
// on-screen thickness, independent of camera zoom. The rect's four corners are mapped
// through the active object transform (if any) and then the camera and pixel scale, and
// the edges are stroked directly at framebuffer resolution. This is the debug-overlay
// counterpart to DrawRectOutline: a rotated/scaled world rect lands as its true on-screen
// quad, thin and crisp at any zoom, rather than rasterizing at world resolution and
// scaling with the camera (which reads blocky). It is used for editor-style overlays such
// as a collider hitbox.
func (r *Renderer) DrawRectOutlineScreen(rect math.Rect, c math.Color, thickness float64) {
	if r.target == nil {
		return
	}
	corners := r.objectRectCorners(rect)
	p0 := r.screenPos(corners[0])
	p1 := r.screenPos(corners[1])
	p2 := r.screenPos(corners[2])
	p3 := r.screenPos(corners[3])
	r.drawLineScreen(p0, p1, c, thickness)
	r.drawLineScreen(p1, p2, c, thickness)
	r.drawLineScreen(p2, p3, c, thickness)
	r.drawLineScreen(p3, p0, c, thickness)
}

// DrawCircle draws a filled circle.
func (r *Renderer) DrawCircle(center math.Vector2, radius float64, c math.Color) {
	if r.target == nil {
		return
	}
	if r.objActive {
		center = r.objectToWorld(center)
		radius *= r.objectRadiusScale()
	}
	if res := r.shapeRes(); res > 0 {
		r.drawCircleRes(center, radius, c, res)
		return
	}
	p := r.screenPos(center)
	z := r.zoom()
	vector.DrawFilledCircle(r.target,
		float32(p.X), float32(p.Y), float32(radius*z),
		toRGBA(c), false)
}

// drawCircleRes rasterizes the circle at `res` px/unit and blits it upscaled by
// zoom()/res. res=1 is the chunky path; res>1 is the editor's ppu-based smooth path.
// The center is quantized to the 1/res grid so the circle's edge snaps to the game
// pixel grid; the sub-grid remainder is applied as a sub-pixel blit offset.
func (r *Renderer) drawCircleRes(center math.Vector2, radius float64, c math.Color, res float64) {
	rr := stdmath.Round(radius * res) // radius in res-pixels
	if rr <= 0 {
		return
	}
	qx := stdmath.Round(center.X*res) / res
	qy := stdmath.Round(center.Y*res) / res
	pad := int(rr) + 1 // +1 keeps the edge from clipping
	key := fmt.Sprintf("circle:%d:%d:%s", int(res), int(rr), colorKey(c))
	img := r.chunkySprite(key, 2*pad, 2*pad, func(dst *ebiten.Image) {
		vector.DrawFilledCircle(dst, float32(pad), float32(pad), float32(rr), toRGBA(c), false)
	})
	r.blitRes(img,
		math.NewVector2(qx-float64(pad)/res, qy-float64(pad)/res),
		math.NewVector2(center.X-qx, center.Y-qy), res)
}

// DrawCircleOutline draws a circle outline.
func (r *Renderer) DrawCircleOutline(center math.Vector2, radius float64, c math.Color, thickness float64) {
	if r.target == nil {
		return
	}
	if r.objActive {
		center = r.objectToWorld(center)
		radius *= r.objectRadiusScale()
	}
	if res := r.shapeRes(); res > 0 {
		r.drawCircleOutlineRes(center, radius, c, thickness, res)
		return
	}
	p := r.screenPos(center)
	z := r.zoom()
	vector.StrokeCircle(r.target,
		float32(p.X), float32(p.Y), float32(radius*z),
		float32(thickness*z), toRGBA(c), false)
}

// drawCircleOutlineRes rasterizes the outline at `res` px/unit and blits it upscaled
// by zoom()/res. res=1 is the chunky path; res>1 is the editor's ppu-based smooth path.
// The stroke is centered on the circle of the given radius, so it extends half the
// thickness beyond it.
func (r *Renderer) drawCircleOutlineRes(center math.Vector2, radius float64, c math.Color, thickness float64, res float64) {
	t := stdmath.Round(thickness * res)
	if t <= 0 {
		return
	}
	rr := stdmath.Round(radius * res)
	if rr <= 0 {
		return
	}
	qx := stdmath.Round(center.X*res) / res
	qy := stdmath.Round(center.Y*res) / res
	pad := int(rr+t/2) + 1
	key := fmt.Sprintf("circleoutline:%d:%d:%d:%s", int(res), int(rr), int(t), colorKey(c))
	img := r.chunkySprite(key, 2*pad, 2*pad, func(dst *ebiten.Image) {
		vector.StrokeCircle(dst, float32(pad), float32(pad), float32(rr), float32(t), toRGBA(c), false)
	})
	r.blitRes(img,
		math.NewVector2(qx-float64(pad)/res, qy-float64(pad)/res),
		math.NewVector2(center.X-qx, center.Y-qy), res)
}

// DrawLine draws a line between two points.
func (r *Renderer) DrawLine(start, end math.Vector2, c math.Color, thickness float64) {
	if r.objActive {
		start = r.objectToWorld(start)
		end = r.objectToWorld(end)
	}
	r.drawLineWorld(start, end, c, thickness)
}

// drawLineWorld draws a line between two world-space points (any object transform
// has already been applied by DrawLine, or the points are already world-space as in
// a rotated rect outline).
func (r *Renderer) drawLineWorld(start, end math.Vector2, c math.Color, thickness float64) {
	if r.target == nil {
		return
	}
	if res := r.shapeRes(); res > 0 {
		r.drawLineRes(start, end, c, thickness, res)
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

// drawLineScreen strokes a line between two screen-space points (already mapped through
// the camera and pixel scale) at a constant on-screen thickness in logical units. Unlike
// drawLineWorld, the thickness does NOT scale with camera zoom — the line stays thin and
// crisp at any zoom, which is what debug overlays (a collider hitbox, a selection outline)
// want. It bypasses the chunky path and strokes directly at framebuffer resolution.
func (r *Renderer) drawLineScreen(start, end math.Vector2, c math.Color, thickness float64) {
	if r.target == nil {
		return
	}
	t := thickness * r.pixelScale
	if t <= 0 {
		return
	}
	vector.StrokeLine(r.target,
		float32(start.X), float32(start.Y),
		float32(end.X), float32(end.Y),
		float32(t), toRGBA(c), false)
}

// drawLineRes rasterizes the line at `res` px/unit and blits it upscaled by
// zoom()/res. res=1 is the chunky path; res>1 is the editor's ppu-based smooth path.
// Both endpoints snap to the 1/res grid (a line has no single anchor to keep
// fractional), and the stroke extends half the thickness around the line.
func (r *Renderer) drawLineRes(start, end math.Vector2, c math.Color, thickness float64, res float64) {
	t := stdmath.Round(thickness * res)
	if t <= 0 {
		return
	}
	x0 := stdmath.Round(start.X*res) / res
	y0 := stdmath.Round(start.Y*res) / res
	x1 := stdmath.Round(end.X*res) / res
	y1 := stdmath.Round(end.Y*res) / res
	minX := stdmath.Min(x0, x1)
	minY := stdmath.Min(y0, y1)
	maxX := stdmath.Max(x0, x1)
	maxY := stdmath.Max(y0, y1)
	pad := stdmath.Ceil(t / 2)
	bw := int(stdmath.Round((maxX-minX)*res) + 2*pad)
	bh := int(stdmath.Round((maxY-minY)*res) + 2*pad)
	key := fmt.Sprintf("line:%d:%d:%d:%d:%s", int(res), int((x1-x0)*res), int((y1-y0)*res), int(t), colorKey(c))
	img := r.chunkySprite(key, bw, bh, func(dst *ebiten.Image) {
		vector.StrokeLine(dst,
			float32((x0-minX)*res+float64(pad)), float32((y0-minY)*res+float64(pad)),
			float32((x1-minX)*res+float64(pad)), float32((y1-minY)*res+float64(pad)),
			float32(t), toRGBA(c), false)
	})
	// No fractional offset: both endpoints are snapped to the grid.
	r.blitRes(img, math.NewVector2(minX-float64(pad)/res, minY-float64(pad)/res), math.NewVector2(0, 0), res)
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

	cx := w / 2
	cy := h / 2

	// The image is drawn in local space: its top-left is at `position`, it is scaled
	// by `scale`, and — when an object transform is active — the object's scale and
	// rotation are folded in before the result is placed at the object's position.
	// The local-space image center is the pivot, so rotation happens about the
	// image's own origin (its center), independent of the object's position, and a
	// negative scale mirrors around that center instead of shifting by a full drawn
	// size.
	sx := scale.X
	sy := scale.Y
	totalRot := rotation
	if r.objActive {
		sx *= r.objScale.X
		sy *= r.objScale.Y
		totalRot += r.objRot
	}
	lsx, lsy := sx, sy // logical (pre-zoom) scale, for the chunky path

	// centerLocal is the image center in local space. Absolute scale keeps a flipped
	// sprite's top-left at `position`: a negative scale still spans |scale|*w, so the
	// center is always the corner plus half the drawn size.
	centerLocal := math.NewVector2(
		position.X+cx*stdmath.Abs(scale.X),
		position.Y+cy*stdmath.Abs(scale.Y),
	)
	centerWorld := r.objectToWorld(centerLocal)
	centerScreen := r.screenPos(centerWorld)
	z := r.zoom()
	sx *= z
	sy *= z

	// Rotated textures are rasterized into a buffer at a fixed resolution — logical
	// resolution for chunky rotation, the renderer's pixelScale (the game's ppu) for
	// smooth rotation, or the editor's target ppu — and blitted upscaled into place.
	// The buffer is rotated independent of the object's position, so a fractional
	// position shifts the whole image rather than re-sampling the rotation.
	if totalRot != 0 {
		r.drawTextureRes(drawImg, cx, cy, lsx, lsy, totalRot, centerWorld, transform, hue, r.textureRes())
		return
	}

	// Anchor the (sub-)image at its center, apply scale and rotation, then place its
	// center at the (object-transformed) center screen position.
	var geoM ebiten.GeoM
	geoM.Translate(-cx, -cy)
	geoM.Scale(sx, sy)
	geoM.Rotate(totalRot)
	geoM.Translate(centerScreen.X, centerScreen.Y)

	// The common case is no color transform, so keep the plain texture shader path.
	if transform.IsIdentity() {
		r.target.DrawImage(drawImg, &ebiten.DrawImageOptions{GeoM: geoM})
		return
	}

	cm := toColorm(transform.Matrix(hue))
	colorm.DrawImage(r.target, drawImg, cm, &colorm.DrawImageOptions{GeoM: geoM})
}

// drawTextureRes rasterizes a rotated texture into a buffer at `res` px/unit sized to
// its rotated AABB and blits it upscaled by zoom()/res, so its pixels stay snapped to
// the rasterization grid (pixel-perfect rotation). res=1 is the chunky rotation path;
// res>1 is smooth rotation (the game's pixelScale or the editor's target ppu). The
// buffer is rotated about its own center, independent of the object's position, so a
// fractional position only shifts the blit — the rotated pixels themselves are stable.
// The buffer is minted per frame (rotation angles vary, so there is little to cache).
func (r *Renderer) drawTextureRes(drawImg *ebiten.Image, cx, cy, lsx, lsy, totalRot float64, centerWorld math.Vector2, transform math.ColorTransform, hue float64, res float64) {
	cos := stdmath.Abs(stdmath.Cos(totalRot))
	sin := stdmath.Abs(stdmath.Sin(totalRot))
	extX := stdmath.Abs(lsx)*cx*cos + stdmath.Abs(lsy)*cy*sin
	extY := stdmath.Abs(lsx)*cx*sin + stdmath.Abs(lsy)*cy*cos
	bw := int(stdmath.Ceil(2 * extX * res))
	bh := int(stdmath.Ceil(2 * extY * res))
	if bw <= 0 || bh <= 0 {
		return
	}

	buf := ebiten.NewImage(bw, bh)
	var geoM ebiten.GeoM
	geoM.Translate(-cx, -cy)
	geoM.Scale(lsx*res, lsy*res)
	geoM.Rotate(totalRot)
	geoM.Translate(float64(bw)/2, float64(bh)/2)
	if transform.IsIdentity() {
		buf.DrawImage(drawImg, &ebiten.DrawImageOptions{GeoM: geoM})
	} else {
		cm := toColorm(transform.Matrix(hue))
		colorm.DrawImage(buf, drawImg, cm, &colorm.DrawImageOptions{GeoM: geoM})
	}

	minX := centerWorld.X - extX
	minY := centerWorld.Y - extY
	worldMin := math.NewVector2(stdmath.Floor(minX*res)/res, stdmath.Floor(minY*res)/res)
	frac := math.NewVector2(minX-worldMin.X, minY-worldMin.Y)
	r.blitRes(buf, worldMin, frac, res)
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

// SetClipRect restricts subsequent drawing to the given screen-space rectangle
// (logical units); content outside it is discarded. A zero/negative rect clears
// any active clip. The clip is non-nesting: a new SetClipRect flushes the current
// one (blitting it back) rather than stacking. Must be balanced with ClearClip.
func (r *Renderer) SetClipRect(rect math.Rect) {
	if rect.Width() <= 0 || rect.Height() <= 0 || r.rootTarget == nil {
		r.ClearClip()
		return
	}

	// Flush any active clip so its content lands on the root target before we
	// repoint at a new offscreen.
	r.ClearClip()

	x0 := int(stdmath.Floor(rect.X() * r.pixelScale))
	y0 := int(stdmath.Floor(rect.Y() * r.pixelScale))
	w := int(stdmath.Ceil(rect.Width() * r.pixelScale))
	h := int(stdmath.Ceil(rect.Height() * r.pixelScale))
	if w <= 0 || h <= 0 {
		return
	}

	// Reuse the offscreen across frames and clip regions; reallocate only on a
	// size change, and clear it (it holds the previous frame's pixels). The old
	// image MUST be disposed before it is replaced: ebiten.Image pins GPU memory
	// until Dispose (a bare drop waits for GC, which can't keep pace with a
	// per-frame size change), so replacing it without disposing leaks one
	// framebuffer-sized texture every time the clip size changes.
	if r.clipTarget == nil || r.clipW != w || r.clipH != h {
		if r.clipTarget != nil {
			r.clipTarget.Dispose()
		}
		r.clipTarget = ebiten.NewImage(w, h)
		r.clipW, r.clipH = w, h
	} else {
		r.clipTarget.Clear()
	}
	r.clipOX = float64(x0)
	r.clipOY = float64(y0)
	r.clipActive = true
	r.target = r.clipTarget
}

// ClearClip ends the active clip region, compositing the clipped content back onto
// the root target at its origin and restoring full-target drawing.
func (r *Renderer) ClearClip() {
	if !r.clipActive {
		return
	}
	off := r.clipTarget
	r.clipActive = false
	r.target = r.rootTarget
	if r.rootTarget != nil {
		var geoM ebiten.GeoM
		geoM.Translate(r.clipOX, r.clipOY)
		r.rootTarget.DrawImage(off, &ebiten.DrawImageOptions{GeoM: geoM})
	}
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
