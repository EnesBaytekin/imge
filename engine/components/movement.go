package components

import (
	"github.com/EnesBaytekin/imge/core"
	"github.com/EnesBaytekin/imge/core/math"
)

// MovementComponent moves its owner and, when the owner has an @Hitbox, blocks
// movement that would overlap another object's hitbox. It holds no speed state:
// callers pass an explicit displacement in pixels each frame.
type MovementComponent struct {
	core.BaseComponent
}

// Move attempts to move the owner by (dx, dy). If the owner has an @Hitbox and
// the destination overlaps another object, the move is blocked and a
// "blocked_collision" event is emitted with the blocking object as Data.
// Returns true if the movement was applied.
func (c *MovementComponent) Move(dx, dy float64) bool {
	owner := c.GetOwner()
	if owner == nil {
		return false
	}

	newPos := owner.Transform.Position.Add(math.NewVector2(dx, dy))

	if collisionObj := c.checkCollisionAt(newPos); collisionObj != nil {
		c.Emit("blocked_collision", collisionObj)
		return false
	}

	owner.Transform.Position = newPos
	return true
}

// MoveTowards moves the owner up to `distance` pixels toward a target, with the
// same collision checks as Move. Returns true if any movement was applied.
func (c *MovementComponent) MoveTowards(target math.Vector2, distance float64) bool {
	owner := c.GetOwner()
	if owner == nil || distance <= 0 {
		return false
	}

	direction := target.Subtract(owner.Transform.Position)
	dist := direction.Length()
	if dist <= 0 {
		return false
	}

	// Don't overshoot the target.
	if distance > dist {
		distance = dist
	}

	direction = direction.Divide(dist)
	movement := direction.Multiply(distance)

	return c.Move(movement.X, movement.Y)
}

// checkCollisionAt returns the object that would block moving the owner to
// newPos, or nil if the move is clear. Only checks when the owner has an @Hitbox.
func (c *MovementComponent) checkCollisionAt(newPos math.Vector2) *core.Object {
	owner := c.GetOwner()
	if owner == nil || owner.Scene == nil {
		return nil
	}

	hb, ok := owner.GetComponentByKind("@Hitbox").(*HitboxComponent)
	if !ok {
		return nil
	}

	bounds := hb.GetBounds()
	bounds.Position = newPos

	for _, other := range owner.Scene.Objects {
		if other == owner || !other.Active || other.IsDestroyed() {
			continue
		}

		otherHb, ok := other.GetComponentByKind("@Hitbox").(*HitboxComponent)
		if !ok {
			continue
		}

		if bounds.Overlaps(otherHb.GetBounds()) {
			return other
		}
	}

	return nil
}
