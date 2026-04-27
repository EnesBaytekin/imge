// Package components contains built-in game components.
package components

import (
	"github.com/EnesBaytekin/imge/core"
	"github.com/EnesBaytekin/imge/core/math"
)

// ============================================================================
// @Movement Component
// ============================================================================

// MovementComponent provides basic 2D movement with optional collision checking.
// When the owner also has an @Hitbox component, Move() and MoveTowards()
// will check for collisions and block movement that would overlap other hitboxes.
type MovementComponent struct {
	core.BaseComponent
	speed float64
}

// Initialize parses component configuration from JSON args.
// Supported args: speed (default: 100).
func (c *MovementComponent) Initialize(args []interface{}) error {
	if len(args) > 0 {
		if argMap, ok := args[0].(map[string]interface{}); ok {
			if s, ok := argMap["speed"].(float64); ok {
				c.speed = s
			}
		}
	}

	if c.speed <= 0 {
		c.speed = 100
	}

	return nil
}

// Move attempts to move the owner by (dx, dy).
// If the owner has an @Hitbox, checks collision with other objects' hitboxes.
// When a collision blocks the movement, emits a "collision" event via Ping
// with the blocking object as Data, so subscribers can react.
// Returns true if the movement was applied successfully.
func (c *MovementComponent) Move(dx, dy float64) bool {
	owner := c.GetOwner()
	if owner == nil {
		return false
	}

	newPos := owner.Transform.Position.Add(math.Vector2{X: dx, Y: dy})

	// Collision check if owner has @Hitbox
	if collisionObj := c.checkCollisionAt(newPos); collisionObj != nil {
		c.Ping(core.Event{
			Name: "blocked_collision",
			Data: collisionObj,
		})
		return false
	}

	owner.Transform.Position = newPos
	return true
}

// MoveTowards moves the owner towards a target position at the given speed.
// Returns true if the movement was applied (no collision).
func (c *MovementComponent) MoveTowards(target math.Vector2, speed float64) bool {
	owner := c.GetOwner()
	if owner == nil {
		return false
	}

	// Calculate direction to target
	direction := target.Subtract(owner.Transform.Position)
	dist := direction.Length()
	if dist <= 0 {
		return false
	}

	// Normalize and scale by speed
	direction = direction.Divide(dist)
	movement := direction.Multiply(speed)

	return c.Move(movement.X, movement.Y)
}

// SetSpeed sets the movement speed.
func (c *MovementComponent) SetSpeed(speed float64) {
	c.speed = speed
}

// GetSpeed returns the current movement speed.
func (c *MovementComponent) GetSpeed() float64 {
	return c.speed
}

// checkCollisionAt checks if moving the owner to newPos would cause a collision.
// Only checks if the owner has an @Hitbox component.
func (c *MovementComponent) checkCollisionAt(newPos math.Vector2) bool {
	owner := c.GetOwner()
	if owner == nil || owner.Scene == nil {
		return false
	}

	hitboxComp := owner.GetComponent("@Hitbox")
	if hitboxComp == nil {
		return false
	}

	hb, ok := hitboxComp.(*HitboxComponent)
	if !ok {
		return false
	}

	// Simulate hitbox at the new position
	bounds := hb.GetBounds()
	bounds.Position = newPos

	for _, other := range owner.Scene.Objects {
		if other == owner || !other.Active {
			continue
		}

		otherHitboxComp := other.GetComponent("@Hitbox")
		if otherHitboxComp == nil {
			continue
		}

		otherHb, ok := otherHitboxComp.(*HitboxComponent)
		if !ok {
			continue
		}

		if bounds.Overlaps(otherHb.GetBounds()) {
			return true // collision detected
		}
	}

	return false
}

// ============================================================================
// Registration
// ============================================================================

func init() {
	core.RegisterComponent("@Movement", func(args []interface{}) (core.Component, error) {
		return &MovementComponent{}, nil
	})
}
