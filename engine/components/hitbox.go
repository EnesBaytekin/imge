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
type HitboxComponent struct {
	core.BaseComponent
	width  float64
	height float64
}

// Initialize parses component configuration from JSON args.
// Supported args: width, height (default: 32x32).
func (c *HitboxComponent) Initialize(args []interface{}) error {
	if len(args) > 0 {
		if argMap, ok := args[0].(map[string]interface{}); ok {
			if w, ok := argMap["width"].(float64); ok {
				c.width = w
			}
			if h, ok := argMap["height"].(float64); ok {
				c.height = h
			}
		}
	}

	if c.width <= 0 {
		c.width = 32
	}
	if c.height <= 0 {
		c.height = 32
	}

	return nil
}

// GetBounds returns the hitbox rectangle in world space, anchored at the owner's
// position (top-left corner).
func (c *HitboxComponent) GetBounds() math.Rect {
	owner := c.GetOwner()
	if owner == nil {
		return math.NewRect(0, 0, c.width, c.height)
	}
	return math.NewRect(
		owner.Transform.Position.X,
		owner.Transform.Position.Y,
		c.width,
		c.height,
	)
}

// SetSize sets the hitbox dimensions.
func (c *HitboxComponent) SetSize(width, height float64) {
	c.width = width
	c.height = height
}

// GetSize returns the hitbox dimensions.
func (c *HitboxComponent) GetSize() (width, height float64) {
	return c.width, c.height
}

// CheckCollision reports whether this hitbox overlaps another.
func (c *HitboxComponent) CheckCollision(other *HitboxComponent) bool {
	return c.GetBounds().Overlaps(other.GetBounds())
}

// ContainsPoint reports whether a point is inside this hitbox.
func (c *HitboxComponent) ContainsPoint(point math.Vector2) bool {
	return c.GetBounds().ContainsPoint(point)
}

func init() {
	core.RegisterComponent("@Hitbox", func(args []interface{}) (core.Component, error) {
		return &HitboxComponent{}, nil
	})
}
