package components

import (
	"github.com/EnesBaytekin/imge/core"
	"github.com/EnesBaytekin/imge/core/math"
)

// MoveResult reports whether movement was applied on each axis.
type MoveResult struct {
	X bool // X-axis movement was applied
	Y bool // Y-axis movement was applied
}

// Moved reports whether any movement was applied.
func (r MoveResult) Moved() bool { return r.X || r.Y }

// maxPushDepth caps push chains (a pushable pushing another pushable) so a
// circular layout can't recurse forever.
const maxPushDepth = 4

// Mover moves its owner with collision resolution. It holds no speed state:
// callers pass an explicit displacement in pixels. When the owner has a @Collider,
// movement that would overlap another object is resolved axis-by-axis (so a
// diagonal move slides along walls): solids block, pushables are pushed, triggers
// are ignored. Without a collider, the mover teleports freely.
type Mover struct {
	core.BaseComponent
}

// Teleport instantly places the owner at (x, y) with no collision check.
func (m *Mover) Teleport(x, y float64) {
	owner := m.GetOwner()
	if owner == nil {
		return
	}
	owner.Transform.Position = math.NewVector2(x, y)
}

// Move attempts to move the owner by (dx, dy), resolving collisions per axis.
// When a move is blocked, a "blocked_collision" event is emitted with the
// blocking object as Data. Returns which axes were applied.
func (m *Mover) Move(dx, dy float64) MoveResult {
	return MoveResult{
		X: m.moveAxis(0, dx, 0),
		Y: m.moveAxis(1, dy, 0),
	}
}

// MoveTowards moves the owner up to `distance` pixels toward a target, with the
// same collision checks as Move. Returns true if any movement was applied.
func (m *Mover) MoveTowards(target math.Vector2, distance float64) bool {
	owner := m.GetOwner()
	if owner == nil || distance <= 0 {
		return false
	}

	dir := target.Subtract(owner.Transform.Position)
	dist := dir.Length()
	if dist <= 0 {
		return false
	}
	if distance > dist {
		distance = dist
	}

	dir = dir.Divide(dist)
	return m.Move(dir.X*distance, dir.Y*distance).Moved()
}

// moveAxis attempts to move the owner by `delta` along one axis (0 = X, 1 = Y),
// resolving collisions. depth limits push recursion. Returns true if applied.
func (m *Mover) moveAxis(axis int, delta float64, depth int) bool {
	owner := m.GetOwner()
	if owner == nil || delta == 0 {
		return delta == 0
	}

	pos := owner.Transform.Position
	candidate := pos
	if axis == 0 {
		candidate.X += delta
	} else {
		candidate.Y += delta
	}

	collider := core.GetFrom[*Collider](owner)
	if collider == nil {
		owner.Transform.Position = candidate
		return true
	}

	bounds := collider.GetBounds()
	bounds.Position = candidate

	for _, other := range collider.candidates() {
		otherCollider := core.GetFrom[*Collider](other)
		if otherCollider == nil || !bounds.Overlaps(otherCollider.GetBounds()) {
			continue
		}

		switch otherCollider.Mode {
		case ColliderTrigger:
			continue
		case ColliderSolid:
			m.Emit("blocked_collision", other)
			return false
		case ColliderPushable:
			if depth >= maxPushDepth || !m.push(other, otherCollider, axis, bounds, depth+1) {
				m.Emit("blocked_collision", other)
				return false
			}
		}
	}

	owner.Transform.Position = candidate
	return true
}

// push tries to slide a pushable collider out of the way along `axis`, returning
// true if the push cleared the overlap. pushFactor scales the distance.
func (m *Mover) push(other *core.Object, otherCollider *Collider, axis int, myBounds math.Rect, depth int) bool {
	otherMover := core.GetFrom[*Mover](other)
	if otherMover == nil {
		return false
	}
	otherBounds := otherCollider.GetBounds()

	// Overlap length along the axis and push direction (away from the mover).
	var overlap, dir float64
	if axis == 0 {
		overlap = fmin(myBounds.Right(), otherBounds.Right()) - fmax(myBounds.Left(), otherBounds.Left())
		if otherBounds.Center().X >= myBounds.Center().X {
			dir = 1
		} else {
			dir = -1
		}
	} else {
		overlap = fmin(myBounds.Bottom(), otherBounds.Bottom()) - fmax(myBounds.Top(), otherBounds.Top())
		if otherBounds.Center().Y >= myBounds.Center().Y {
			dir = 1
		} else {
			dir = -1
		}
	}

	if overlap <= 0 {
		return false
	}
	amount := overlap * otherCollider.PushFactor * dir
	if amount == 0 {
		return false
	}
	return otherMover.moveAxis(axis, amount, depth)
}
