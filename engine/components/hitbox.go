// Package components contains built-in game components.
package components

import (
	"github.com/EnesBaytekin/imge/core"
	"github.com/EnesBaytekin/imge/core/math"
)

// ============================================================================
// @Hitbox Component
// ============================================================================

// HitboxComponent provides rectangle-based collision detection.
// Can check collision with other @Hitbox components and emit "collision" events.
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

	// Default size
	if c.width <= 0 {
		c.width = 32
	}
	if c.height <= 0 {
		c.height = 32
	}

	return nil
}

// GetBounds returns the hitbox rectangle in world space.
// Uses the owner's transform position as the top-left corner.
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

// CheckCollision checks if this hitbox overlaps with another hitbox.
func (c *HitboxComponent) CheckCollision(other *HitboxComponent) bool {
	return c.GetBounds().Overlaps(other.GetBounds())
}

// ContainsPoint checks if a point is inside this hitbox.
func (c *HitboxComponent) ContainsPoint(point math.Vector2) bool {
	return c.GetBounds().ContainsPoint(point)
}

// Update checks for collisions with other @Hitbox components in the scene.
// When a collision is detected, emits a "collision" event with the other object as data.
func (c *HitboxComponent) Update(ctx *core.ComponentContext) {
	owner := c.GetOwner()
	if owner == nil || owner.Scene == nil {
		return
	}

	myBounds := c.GetBounds()

	for _, other := range owner.Scene.Objects {
		if other == owner || !other.Active {
			continue
		}

		otherHitboxComp := other.GetComponentByKind("@Hitbox")
		if otherHitboxComp == nil {
			continue
		}

		otherHb, ok := otherHitboxComp.(*HitboxComponent)
		if !ok {
			continue
		}

		if myBounds.Overlaps(otherHb.GetBounds()) {
			c.Ping(core.Event{
				Name: "collision",
				Data: other,
			})
		}
	}
}

// Draw renders a debug rectangle outline for the hitbox.
func (c *HitboxComponent) Draw(renderer core.Renderer) {
	bounds := c.GetBounds()
	renderer.DrawRectOutline(bounds, math.Green, 1)
}

// ============================================================================
// Registration
// ============================================================================

func init() {
	core.RegisterComponent("@Hitbox", func(args []interface{}) (core.Component, error) {
		return &HitboxComponent{}, nil
	})
}
