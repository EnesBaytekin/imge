// Package components provides built-in game components.
package components

import (
	"github.com/EnesBaytekin/imge/core"
	"github.com/EnesBaytekin/imge/core/math"
)

// SpriteComponent renders a texture at the object's position.
type SpriteComponent struct {
	core.BaseComponent
	texturePath string
	color       math.Color
	width       float64
	height      float64
}

func (c *SpriteComponent) Initialize(args []interface{}) error {
	// Parse args map
	if len(args) > 0 {
		if argMap, ok := args[0].(map[string]interface{}); ok {
			if texture, ok := argMap["texture"].(string); ok {
				c.texturePath = texture
			}
			if colorMap, ok := argMap["color"].(map[string]interface{}); ok {
				if r, ok := colorMap["r"].(float64); ok {
					c.color.R = uint8(r)
				}
				if g, ok := colorMap["g"].(float64); ok {
					c.color.G = uint8(g)
				}
				if b, ok := colorMap["b"].(float64); ok {
					c.color.B = uint8(b)
				}
				if a, ok := colorMap["a"].(float64); ok {
					c.color.A = uint8(a)
				}
			}
			if width, ok := argMap["width"].(float64); ok {
				c.width = width
			}
			if height, ok := argMap["height"].(float64); ok {
				c.height = height
			}
		}
	}
	// Default color white
	if c.color.R == 0 && c.color.G == 0 && c.color.B == 0 && c.color.A == 0 {
		c.color = math.White
	}
	return nil
}

func (c *SpriteComponent) Update(ctx *core.ComponentContext) {
	// No update logic needed for sprite
}

func (c *SpriteComponent) Draw(renderer core.Renderer) {
	owner := c.GetOwner()
	if owner == nil {
		return
	}
	transform := owner.Transform
	position := transform.Position

	// If we have a texture, draw it (stub for now)
	// In real implementation, would use renderer.DrawTexture
	// For now, draw a colored rectangle as placeholder
	if c.width > 0 && c.height > 0 {
		rect := math.Rect{
			Position: position,
			Size:     math.NewVector2(c.width, c.height),
		}
		renderer.DrawRect(rect, c.color)
	} else {
		// Default size
		rect := math.Rect{
			Position: position,
			Size:     math.NewVector2(32, 32),
		}
		renderer.DrawRect(rect, c.color)
	}
}

func init() {
	core.RegisterComponent("@Sprite", func(args []interface{}) (core.Component, error) {
		comp := &SpriteComponent{}
		if err := comp.Initialize(args); err != nil {
			return nil, err
		}
		return comp, nil
	})
}