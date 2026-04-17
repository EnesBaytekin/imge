// Package components provides built-in game components.
package components

import (
	"github.com/EnesBaytekin/imge/core"
)

// TransformComponent provides additional transform functionality.
// Note: Objects already have a Transform field; this component can add
// interpolation, constraints, or other transform-related behavior.
type TransformComponent struct {
	core.BaseComponent
	// Additional transform properties could go here
	speed float64
}

func (c *TransformComponent) Initialize(args []interface{}) error {
	if len(args) > 0 {
		if argMap, ok := args[0].(map[string]interface{}); ok {
			if speed, ok := argMap["speed"].(float64); ok {
				c.speed = speed
			}
		}
	}
	return nil
}

func (c *TransformComponent) Update(deltaTime float64) {
	// Example: move object based on speed
	if c.speed != 0 {
		owner := c.GetOwner()
		if owner != nil {
			// Simple movement for demonstration
			pos := owner.Transform.Position
			pos.X += c.speed * deltaTime
			owner.Transform.Position = pos
		}
	}
}

func (c *TransformComponent) Draw(renderer core.Renderer) {
	// No drawing needed for transform component
}

func init() {
	core.RegisterComponent("@Transform", func(args []interface{}) (core.Component, error) {
		comp := &TransformComponent{}
		if err := comp.Initialize(args); err != nil {
			return nil, err
		}
		return comp, nil
	})
}