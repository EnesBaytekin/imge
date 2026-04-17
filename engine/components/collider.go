// Package components provides built-in game components.
package components

import (
	"github.com/EnesBaytekin/imge/internal/core"
	"github.com/EnesBaytekin/imge/internal/core/math"
)

// ColliderComponent provides collision detection for an object.
type ColliderComponent struct {
	core.BaseComponent
	colliderType string  // "rectangle", "circle"
	width        float64 // for rectangle
	height       float64 // for rectangle
	radius       float64 // for circle
}

func (c *ColliderComponent) Initialize(args []interface{}) error {
	// Default values
	c.colliderType = "rectangle"
	c.width = 32
	c.height = 32

	if len(args) > 0 {
		if argMap, ok := args[0].(map[string]interface{}); ok {
			if colliderType, ok := argMap["type"].(string); ok {
				c.colliderType = colliderType
			}
			if width, ok := argMap["width"].(float64); ok {
				c.width = width
			}
			if height, ok := argMap["height"].(float64); ok {
				c.height = height
			}
			if radius, ok := argMap["radius"].(float64); ok {
				c.radius = radius
			}
		}
	}
	return nil
}

func (c *ColliderComponent) Update(deltaTime float64) {
	// Collision detection would happen here in a real implementation
}

func (c *ColliderComponent) Draw(renderer core.Renderer) {
	// Optionally draw collider bounds for debugging
	owner := c.GetOwner()
	if owner == nil {
		return
	}
	position := owner.Transform.Position

	// Draw collider outline (debug visualization)
	if c.colliderType == "rectangle" {
		rect := math.Rect{
			Position: position,
			Width:    c.width,
			Height:   c.height,
		}
		// Draw red outline
		renderer.DrawRectOutline(rect, math.Red, 1.0)
	} else if c.colliderType == "circle" {
		center := math.Vector2{
			X: position.X + c.radius,
			Y: position.Y + c.radius,
		}
		renderer.DrawCircleOutline(center, c.radius, math.Red, 1.0)
	}
}

func init() {
	core.RegisterComponent("@Collider", func(args []interface{}) (core.Component, error) {
		comp := &ColliderComponent{}
		if err := comp.Initialize(args); err != nil {
			return nil, err
		}
		return comp, nil
	})
}