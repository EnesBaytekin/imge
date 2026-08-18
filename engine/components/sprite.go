package components

import (
	"github.com/EnesBaytekin/imge/core"
	"github.com/EnesBaytekin/imge/core/math"
)

// Sprite draws a texture (optionally a region of it) at the owner's transform.
// It owns no animation state — an @Animator drives the source region via
// SetSourceRect when needed.
//
// Export variables (JSON args): texture, width, height, tint, flipX, flipY.
type Sprite struct {
	core.BaseComponent

	Texture string     `json:"texture"`
	Width   float64    `json:"width"`  // display width (0 = natural)
	Height  float64    `json:"height"` // display height (0 = natural)
	Tint    math.Color `json:"tint"`
	FlipX   bool       `json:"flipX"`
	FlipY   bool       `json:"flipY"`

	// sourceRect is the region of the texture to draw (zero = entire texture).
	sourceRect math.Rect
}

// Initialize applies defaults.
func (s *Sprite) Initialize() {
	if s.Tint == (math.Color{}) {
		s.Tint = math.White
	}
}

// Draw renders the current source region at the owner's transform.
func (s *Sprite) Draw(renderer core.Renderer) {
	owner := s.GetOwner()
	if owner == nil || s.Texture == "" {
		return
	}

	src := s.sourceRect
	var naturalW, naturalH float64
	if src.Size.X > 0 || src.Size.Y > 0 {
		naturalW, naturalH = src.Size.X, src.Size.Y
	} else {
		naturalW, naturalH = renderer.GetTextureSize(s.Texture)
	}

	// Scale the source region up to the requested display size (if any), then
	// apply the owner's own scale and flips on top.
	scale := owner.Transform.Scale
	if s.Width > 0 && naturalW > 0 {
		scale.X *= s.Width / naturalW
	}
	if s.Height > 0 && naturalH > 0 {
		scale.Y *= s.Height / naturalH
	}
	if s.FlipX {
		scale.X *= -1
	}
	if s.FlipY {
		scale.Y *= -1
	}

	renderer.DrawTexture(s.Texture, src, owner.Transform.Position, scale, owner.Transform.Rotation, s.Tint)
}

// SetSourceRect sets the texture region to draw (zero Rect = entire texture).
func (s *Sprite) SetSourceRect(rect math.Rect) {
	s.sourceRect = rect
}

// SetTexture sets the texture path to render.
func (s *Sprite) SetTexture(path string) {
	s.Texture = path
}

// SetTint sets the color tint for rendering.
func (s *Sprite) SetTint(tint math.Color) {
	s.Tint = tint
}
