package components

import (
	"github.com/EnesBaytekin/imge/core"
	"github.com/EnesBaytekin/imge/core/math"
)

// Sprite draws a texture (optionally a sub-frame of a sprite sheet) at the owner's
// transform. It owns no animation state — an @Animator drives the current frame via
// SetFrame when needed.
//
// Frame slicing: frame_width/frame_height are the size of one frame in the source
// image (pixels). frame_width = 0 means the whole texture is a single frame.
// frame_width > 0 with frame_height = 0 means a single horizontal strip — frames
// are cut left-to-right using the texture's full height. Both set means a grid,
// cut row-by-row. The total frame count is always derived from the texture size.
//
// width/height are the on-screen display size in pixels (0 = natural frame size).
// offset is added to the owner's position before drawing.
//
// Export variables (JSON args): texture, frame_width, frame_height, frame, width,
// height, flip_x, flip_y, color {tint, hue, hue_to, grayscale, solid}, offset
// {x,y}, visible, draw_layer.
type Sprite struct {
	core.BaseComponent

	Texture     string              `json:"texture"`
	FrameWidth  float64             `json:"frame_width"`
	FrameHeight float64             `json:"frame_height"`
	Frame       int                 `json:"frame"`
	Width       float64             `json:"width"`
	Height      float64             `json:"height"`
	FlipX       bool                `json:"flip_x"`
	FlipY       bool                `json:"flip_y"`
	Color       math.ColorTransform `json:"color"`
	Offset      math.Vector2        `json:"offset"`

	// Visible controls whether the sprite draws. It is a *bool so the JSON arg can
	// default to true (nil = not specified). An @Animator overrides this for the
	// sprites it manages.
	Visible *bool `json:"visible"`

	// natural size of the texture, cached on first draw (textures load lazily).
	textureW float64
	textureH float64
	sized    bool
}

// Initialize applies defaults. The color transform's zero value is identity, so
// no default is needed there.
func (s *Sprite) Initialize() {
	if s.Visible == nil {
		v := true
		s.Visible = &v
	}
}

// Draw renders the current frame at the owner's transform.
func (s *Sprite) Draw(renderer core.Renderer) {
	if !s.IsVisible() {
		return
	}
	owner := s.GetOwner()
	if owner == nil || s.Texture == "" {
		return
	}

	if !s.sized {
		s.textureW, s.textureH = renderer.GetTextureSize(s.Texture)
		s.sized = true
	}

	src := s.frameRect()

	// Natural size of the region being drawn (the frame cell, or whole texture).
	natW, natH := s.textureW, s.textureH
	if src.Width() > 0 {
		natW = src.Width()
	}
	if src.Height() > 0 {
		natH = src.Height()
	}

	// Scale the source region up to the requested display size (if any), then apply
	// the owner's scale and flips on top.
	scale := owner.Transform.Scale
	if s.Width > 0 && natW > 0 {
		scale.X *= s.Width / natW
	}
	if s.Height > 0 && natH > 0 {
		scale.Y *= s.Height / natH
	}
	if s.FlipX {
		scale.X *= -1
	}
	if s.FlipY {
		scale.Y *= -1
	}

	pos := owner.Transform.Position.Add(s.Offset)
	renderer.DrawTexture(s.Texture, src, pos, scale, owner.Transform.Rotation, s.Color)
}

// frameRect returns the source region for the current frame (zero Rect = whole
// texture). The frame index is clamped to the valid range.
func (s *Sprite) frameRect() math.Rect {
	if s.FrameWidth <= 0 {
		return math.Rect{}
	}

	cols := s.columns()
	if cols < 1 {
		cols = 1
	}

	f := s.Frame
	if f < 0 {
		f = 0
	}
	if count := s.FrameCount(); count > 0 && f >= count {
		f = count - 1
	}

	fw := s.FrameWidth
	fh := s.FrameHeight
	if fh <= 0 {
		fh = s.textureH
	}

	col := f % cols
	row := f / cols
	return math.NewRect(float64(col)*fw, float64(row)*fh, fw, fh)
}

// columns returns the number of frame columns across the texture (1 when unknown).
func (s *Sprite) columns() int {
	if s.textureW <= 0 || s.FrameWidth <= 0 {
		return 1
	}
	cols := int(s.textureW / s.FrameWidth)
	if cols < 1 {
		cols = 1
	}
	return cols
}

// FrameCount returns the total number of frames derived from the texture size and
// frame dimensions. Returns 1 until the texture size is known (cached on first
// Draw), which is safe: an animator's first frame boundary is only reached after
// at least one frame has drawn.
func (s *Sprite) FrameCount() int {
	if s.FrameWidth <= 0 {
		return 1
	}
	if s.textureW <= 0 {
		return 1
	}
	cols := s.columns()
	if s.FrameHeight <= 0 {
		return cols
	}
	rows := 1
	if s.textureH > 0 && s.FrameHeight > 0 {
		rows = int(s.textureH / s.FrameHeight)
	}
	if rows < 1 {
		rows = 1
	}
	return cols * rows
}

// IsVisible reports whether the sprite draws.
func (s *Sprite) IsVisible() bool {
	return s.Visible == nil || *s.Visible
}

// SetVisible sets whether the sprite draws.
func (s *Sprite) SetVisible(v bool) {
	b := v
	s.Visible = &b
}

// SetFrame sets the current frame index (0-based); it is clamped on draw.
func (s *Sprite) SetFrame(frame int) {
	s.Frame = frame
}

// SetOffset sets the offset added to the owner's position before drawing.
func (s *Sprite) SetOffset(x, y float64) {
	s.Offset = math.NewVector2(x, y)
}

// SetTexture sets the texture path to render.
func (s *Sprite) SetTexture(path string) {
	s.Texture = path
}

// SetColor sets the color transform for rendering.
func (s *Sprite) SetColor(transform math.ColorTransform) {
	s.Color = transform
}

// SetFlipX sets horizontal flipping.
func (s *Sprite) SetFlipX(flip bool) {
	s.FlipX = flip
}

// SetFlipY sets vertical flipping.
func (s *Sprite) SetFlipY(flip bool) {
	s.FlipY = flip
}
