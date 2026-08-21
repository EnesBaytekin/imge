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
// callers pass an explicit displacement in pixels. When the owner has @Collider
// shapes, movement that would overlap another object's colliders is resolved
// axis-by-axis (so a diagonal move slides along walls): solids block, pushables
// are pushed — mover and obstacle advance together at a reduced speed, so they
// never interpenetrate. Without a collider, the mover teleports freely.
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

	ownColliders := core.GetAllFrom[*Collider](owner)
	if len(ownColliders) == 0 {
		// No body: teleport freely along this axis.
		m.shift(owner, axis, delta)
		return true
	}

	// The body's own push factor (0 = solid, the default). It scales how hard this
	// mover pushes: a pushable mover (itself > 0) pushes less. Compound bodies have
	// a uniform factor in practice; take the max so any pushable part registers.
	ownPushFactor := 0.0
	for _, c := range ownColliders {
		if c.PushFactor > ownPushFactor {
			ownPushFactor = c.PushFactor
		}
	}

	for _, other := range shapeCandidates(owner, unionCollidesWith(ownColliders)) {
		otherColliders := core.GetAllFrom[*Collider](other)
		if len(otherColliders) == 0 {
			continue
		}

		// Test each of the mover's colliders (translated by delta) against each of
		// the obstacle's colliders. Any overlap triggers resolution.
		for _, own := range ownColliders {
			bounds := own.GetBounds()
			if axis == 0 {
				bounds.Position.X += delta
			} else {
				bounds.Position.Y += delta
			}

			for _, oc := range otherColliders {
				if !bounds.Overlaps(oc.GetBounds()) {
					continue
				}

				if oc.PushFactor <= 0 {
					// Solid: blocks outright.
					m.Emit("blocked_collision", other)
					return false
				}

				if depth >= maxPushDepth {
					m.Emit("blocked_collision", other)
					return false
				}

				otherMover := core.GetFrom[*Mover](other)
				if otherMover == nil {
					m.Emit("blocked_collision", other)
					return false
				}

				// Combined push factor: how hard the mover pushes (1 - p_a) scaled by
				// how easily the obstacle gives (p_b). Both advance the SAME distance
				// so they stay flush and never interpenetrate.
				f := (1 - ownPushFactor) * oc.PushFactor
				amount := delta * f

				before := other.Transform.Position
				if !otherMover.moveAxis(axis, amount, depth+1) {
					// The obstacle couldn't move (e.g. a wall behind it), so neither
					// does the mover.
					m.Emit("blocked_collision", other)
					return false
				}

				// Move the mover by however far the obstacle actually went, keeping
				// the two flush even when the obstacle only moved partway.
				var actual float64
				if axis == 0 {
					actual = other.Transform.Position.X - before.X
				} else {
					actual = other.Transform.Position.Y - before.Y
				}
				m.shift(owner, axis, actual)
				return true
			}
		}
	}

	m.shift(owner, axis, delta)
	return true
}

// shift moves the owner by delta along one axis (0 = X, 1 = Y).
func (m *Mover) shift(owner *core.Object, axis int, delta float64) {
	if axis == 0 {
		owner.Transform.Position.X += delta
	} else {
		owner.Transform.Position.Y += delta
	}
}
