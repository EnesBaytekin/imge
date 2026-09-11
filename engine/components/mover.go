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
// shapes, movement is swept along each axis: the owner advances only up to the
// first contact and rests flush against it, so it never leaves a sub-pixel gap
// and never tunnels through a thin obstacle. Solids block; pushables are pushed
// ahead and the mover follows, so they stay flush. Without a collider, the mover
// teleports freely.
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
	owner := m.GetOwner()
	if owner == nil {
		return MoveResult{}
	}

	// Compute the body and candidate obstacle colliders once and share them across
	// both axes, instead of re-scanning the scene (and re-allocating) per axis.
	own := core.GetAllFrom[*Collider](owner)
	obstacles := m.obstacleColliders(owner, unionCollidesWith(own))
	return MoveResult{
		X: m.moveAxisPrepared(0, dx, 0, own, obstacles),
		Y: m.moveAxisPrepared(1, dy, 0, own, obstacles),
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

// IsGrounded reports whether a collider sits directly below the owner's body,
// within a small tolerance — the ground probe platformer jump logic uses to decide
// whether a jump may start. It is a pure query (no movement), so it is independent
// of component update order. Solid and pushable colliders both count as ground.
func (m *Mover) IsGrounded() bool {
	const probe = 1.0
	owner := m.GetOwner()
	if owner == nil {
		return false
	}
	own := core.GetAllFrom[*Collider](owner)
	if len(own) == 0 {
		return false
	}

	others := shapeCandidates(owner, unionCollidesWith(own))
	for _, other := range others {
		for _, oc := range core.GetAllFrom[*Collider](other) {
			for _, c := range own {
				if contactBetween(1, 1, c, oc, probe) < probe {
					return true
				}
			}
		}
	}
	return false
}

// moveAxis resolves movement along one axis (0 = X, 1 = Y), computing the owner's
// body and candidate obstacles fresh. It is the entry point for push recursion,
// where the mover is a pushed obstacle and may have a different body than the
// original mover.
func (m *Mover) moveAxis(axis int, delta float64, depth int) bool {
	owner := m.GetOwner()
	if owner == nil {
		return false
	}
	own := core.GetAllFrom[*Collider](owner)
	obstacles := m.obstacleColliders(owner, unionCollidesWith(own))
	return m.moveAxisPrepared(axis, delta, depth, own, obstacles)
}

// obstacleColliders collects every collider the owner may hit into one flat list,
// so the two movement axes share a single scene scan instead of re-running
// GetAllFrom per axis. Shapes are still derived per axis inside moveAxisPrepared:
// a pushed obstacle may have moved between the X and Y passes, so its corners and
// bounds must be read fresh there.
func (m *Mover) obstacleColliders(owner *core.Object, collidesWith []string) []*Collider {
	others := shapeCandidates(owner, collidesWith)
	if len(others) == 0 {
		return nil
	}
	out := make([]*Collider, 0, len(others))
	for _, other := range others {
		out = append(out, core.GetAllFrom[*Collider](other)...)
	}
	return out
}

// moveAxisPrepared performs the swept, single-axis movement using a precomputed
// body and candidate list. delta is signed; depth caps push recursion. Returns
// true when the full requested distance was applied.
func (m *Mover) moveAxisPrepared(axis int, delta float64, depth int, own []*Collider, obstacles []*Collider) bool {
	owner := m.GetOwner()
	if delta == 0 {
		return true
	}
	if owner == nil {
		return false
	}

	if len(own) == 0 {
		// No body: teleport freely along this axis.
		m.shift(owner, axis, delta)
		return true
	}

	dir := 1.0
	if delta < 0 {
		dir = -1.0
	}
	dist := delta
	if dist < 0 {
		dist = -dist
	}

	// Precompute the body's shapes and push factor once, outside the obstacle loop.
	// A shape keeps its world corners (for the rotated/OBB path), its axis-aligned
	// bounds (for the fast unrotated path), and whether its owner is rotated.
	ownShapes := make([]colliderShape, len(own))
	ownPushFactor := 0.0
	for i, c := range own {
		ownShapes[i] = colliderShape{
			corners: c.corners(),
			bounds:  c.GetBounds(),
			rotated: c.isRotated(),
		}
		if c.PushFactor > ownPushFactor {
			ownPushFactor = c.PushFactor
		}
	}

	// Earliest contact distance along the axis, and the collider that imposes it.
	limit := dist
	var hitObj *core.Object
	var hitCollider *Collider

	for _, oc := range obstacles {
		for _, os := range ownShapes {
			var d float64
			if os.rotated || oc.isRotated() {
				d = obbContactAlong(axis, dir, os.corners, oc.corners(), limit)
			} else {
				d = contactAlong(axis, dir, os.bounds, oc.GetBounds(), limit)
			}
			if d < limit {
				limit = d
				hitObj = oc.GetOwner()
				hitCollider = oc
			}
		}
	}

	if hitObj == nil {
		// No contact: the whole distance is clear.
		m.shift(owner, axis, delta)
		return true
	}

	if hitCollider.PushFactor <= 0 || depth >= maxPushDepth {
		// Solid (or push chain too deep): rest flush against the contact and stop.
		m.shift(owner, axis, dir*limit)
		m.Emit("blocked_collision", hitObj)
		return false
	}

	otherMover := core.GetFrom[*Mover](hitObj)
	if otherMover == nil {
		m.shift(owner, axis, dir*limit)
		m.Emit("blocked_collision", hitObj)
		return false
	}

	// Push the obstacle the remaining distance, scaled by the combined push factor,
	// then follow by however far it actually went so the two stay flush (whether it
	// moved fully, partway, or not at all).
	push := (dist - limit) * (1 - ownPushFactor) * hitCollider.PushFactor
	before := hitObj.Transform.Position
	pushed := otherMover.moveAxis(axis, dir*push, depth+1)

	var actual float64
	if axis == 0 {
		actual = hitObj.Transform.Position.X - before.X
	} else {
		actual = hitObj.Transform.Position.Y - before.Y
	}
	if actual < 0 {
		actual = -actual
	}

	m.shift(owner, axis, dir*(limit+actual))
	if !pushed {
		m.Emit("blocked_collision", hitObj)
	}
	return pushed
}

// shift moves the owner by delta along one axis (0 = X, 1 = Y).
func (m *Mover) shift(owner *core.Object, axis int, delta float64) {
	if axis == 0 {
		owner.Transform.Position.X += delta
	} else {
		owner.Transform.Position.Y += delta
	}
}

// contactAlong returns how far a rect may travel along one axis in direction dir
// before first touching the obstacle rect, capped at maxDist. The two rects must
// overlap on the perpendicular axis for a collision to be possible; otherwise the
// obstacle is irrelevant and maxDist is returned. A rect already overlapping the
// obstacle returns 0 (already in contact, so it can't move further in this
// direction).
func contactAlong(axis int, dir float64, mover, obstacle math.Rect, maxDist float64) float64 {
	if axis == 0 {
		// Moving in X: the Y intervals must overlap.
		if mover.Top() >= obstacle.Bottom() || mover.Bottom() <= obstacle.Top() {
			return maxDist
		}
		if dir > 0 {
			if obstacle.Right() <= mover.Left() {
				return maxDist // obstacle is behind
			}
			if obstacle.Left() < mover.Right() {
				return 0 // already overlapping
			}
			if d := obstacle.Left() - mover.Right(); d < maxDist {
				return d
			}
			return maxDist
		}
		if obstacle.Left() >= mover.Right() {
			return maxDist // obstacle is ahead
		}
		if obstacle.Right() > mover.Left() {
			return 0 // already overlapping
		}
		if d := mover.Left() - obstacle.Right(); d < maxDist {
			return d
		}
		return maxDist
	}

	// Moving in Y: the X intervals must overlap.
	if mover.Left() >= obstacle.Right() || mover.Right() <= obstacle.Left() {
		return maxDist
	}
	if dir > 0 {
		if obstacle.Bottom() <= mover.Top() {
			return maxDist // obstacle is above
		}
		if obstacle.Top() < mover.Bottom() {
			return 0 // already overlapping
		}
		if d := obstacle.Top() - mover.Bottom(); d < maxDist {
			return d
		}
		return maxDist
	}
	if obstacle.Top() >= mover.Bottom() {
		return maxDist // obstacle is below
	}
	if obstacle.Bottom() > mover.Top() {
		return 0 // already overlapping
	}
	if d := mover.Top() - obstacle.Bottom(); d < maxDist {
		return d
	}
	return maxDist
}

// colliderShape is a precomputed view of a collider for one movement axis: its world
// corners (the rotated quad), its axis-aligned bounds (the unrotated fast path), and
// whether its owner is rotated.
type colliderShape struct {
	corners [4]math.Vector2
	bounds  math.Rect
	rotated bool
}

// contactBetween returns how far own may travel along one axis in direction dir before
// first touching obstacle, capped at maxDist. It uses the exact axis-aligned path when
// neither shape is rotated, and the OBB path otherwise, so an angled collider resolves
// against its true rotated shape.
func contactBetween(axis int, dir float64, own, obstacle *Collider, maxDist float64) float64 {
	if own.isRotated() || obstacle.isRotated() {
		return obbContactAlong(axis, dir, own.corners(), obstacle.corners(), maxDist)
	}
	return contactAlong(axis, dir, own.GetBounds(), obstacle.GetBounds(), maxDist)
}
