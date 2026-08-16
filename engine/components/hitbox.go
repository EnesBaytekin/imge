// Package components contains IMGE's built-in components. At build time, custom
// component files are merged into this same package, so built-ins and customs can
// call each other's methods directly (no capability interfaces needed).
package components

import (
	"github.com/EnesBaytekin/imge/core"
	"github.com/EnesBaytekin/imge/core/math"
)

// HitboxComponent is a pure rectangle collider. It stores dimensions and answers
// collision queries; it does not draw or scan the scene (movement components scan
// when they move).
//
// Export variables (JSON args): width, height.
type HitboxComponent struct {
	core.BaseComponent
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
}

// Initialize applies defaults.
func (c *HitboxComponent) Initialize() {
	if c.Width <= 0 {
		c.Width = 32
	}
	if c.Height <= 0 {
		c.Height = 32
	}
}

// GetBounds returns the hitbox rectangle in world space, anchored at the owner's
// position (top-left corner).
func (c *HitboxComponent) GetBounds() math.Rect {
	owner := c.GetOwner()
	if owner == nil {
		return math.NewRect(0, 0, c.Width, c.Height)
	}
	return math.NewRect(
		owner.Transform.Position.X,
		owner.Transform.Position.Y,
		c.Width,
		c.Height,
	)
}

// SetSize sets the hitbox dimensions.
func (c *HitboxComponent) SetSize(width, height float64) {
	c.Width = width
	c.Height = height
}

// GetSize returns the hitbox dimensions.
func (c *HitboxComponent) GetSize() (width, height float64) {
	return c.Width, c.Height
}

// CheckCollision reports whether this hitbox overlaps another.
func (c *HitboxComponent) CheckCollision(other *HitboxComponent) bool {
	return c.GetBounds().Overlaps(other.GetBounds())
}

// ContainsPoint reports whether a point is inside this hitbox.
func (c *HitboxComponent) ContainsPoint(point math.Vector2) bool {
	return c.GetBounds().ContainsPoint(point)
}
