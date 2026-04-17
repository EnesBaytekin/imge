// Package components provides custom components for the basic sample game.
package components

import (
	"github.com/EnesBaytekin/imge/internal/core"
	"github.com/EnesBaytekin/imge/internal/core/math"
)

// PlayerMovement component allows an object to be moved with keyboard input.
type PlayerMovement struct {
	core.BaseComponent
	speed float64
}

func (c *PlayerMovement) Initialize(args []interface{}) error {
	c.speed = 200.0 // default speed (pixels per second)
	if len(args) > 0 {
		if argMap, ok := args[0].(map[string]interface{}); ok {
			if speed, ok := argMap["speed"].(float64); ok {
				c.speed = speed
			}
		}
	}
	return nil
}

func (c *PlayerMovement) Update(deltaTime float64) {
	owner := c.GetOwner()
	if owner == nil {
		return
	}

	// Get scene to access input via platform
	scene := core.GetSceneFromComponent(c)
	if scene == nil {
		return
	}

	// In a real implementation, we would get the game instance and platform input
	// For now, just move with predetermined keys (mock input will print debug)
	// We'll implement proper input handling later
	// For mock platform, we'll just move based on some logic
	// This is a placeholder - actual input handling needs platform access
	// For now, move slowly in a circle for demonstration
	pos := owner.Transform.Position
	time := 0.0 // would get from platform.Time()
	// Since we don't have time access, we'll use a simple animation
	pos.X += c.speed * deltaTime * 0.1
	pos.Y += c.speed * deltaTime * 0.05
	owner.Transform.Position = pos
}

func (c *PlayerMovement) Draw(renderer core.Renderer) {
	// No drawing needed
}

func init() {
	core.RegisterComponent("components/player.go", func(args []interface{}) (core.Component, error) {
		comp := &PlayerMovement{}
		if err := comp.Initialize(args); err != nil {
			return nil, err
		}
		return comp, nil
	})
}