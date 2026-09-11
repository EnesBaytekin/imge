// Package components contains IMGE's built-in components. At build time, custom
// component files are merged into this same package, so built-ins and customs can
// call each other's methods directly (no capability interfaces needed).
package components

import (
	stdmath "math"

	"github.com/EnesBaytekin/imge/core"
	"github.com/EnesBaytekin/imge/core/math"
)

// Collider is a rectangle shape that participates in movement resolution. It is
// pure physics: it answers overlap queries and — via its PushFactor — tells movers
// whether it blocks or gets pushed. It does NOT track overlaps or emit events;
// detection lives on @Trigger instead.
//
// Multiple colliders on one object form a single compound body: a mover tests the
// whole union against obstacles and treats them as one physical object.
//
// Export variables (JSON args): width, height, offset {x,y}, push_factor,
// collides_with.
type Collider struct {
	core.BaseComponent

	Width  float64 `json:"width"`
	Height float64 `json:"height"`

	// Offset shifts the collider rectangle relative to the owner's position
	// (top-left corner). Use it when the sprite and hitbox have different
	// origins — e.g. the sprite is centered but the collider should be.
	Offset math.Vector2 `json:"offset"`

	// PushFactor sets how this collider responds when a mover collides with it:
	// 0 (the default) is solid — it blocks outright. A value in (0, 1] makes it
	// pushable, where a higher value means lighter (easier to push; 1 = weightless).
	PushFactor float64 `json:"push_factor"`

	// CollidesWith lists object tags this collider interacts with. Empty means it
	// interacts with every object.
	CollidesWith []string `json:"collides_with"`
}

// Initialize applies defaults.
func (c *Collider) Initialize() {
	if c.Width <= 0 {
		c.Width = 32
	}
	if c.Height <= 0 {
		c.Height = 32
	}
}

// GetBounds returns the collider rectangle in world space, anchored at the
// owner's position plus Offset (top-left corner).
func (c *Collider) GetBounds() math.Rect {
	return shapeBounds(c.GetOwner(), c.Width, c.Height, c.Offset)
}

// SetSize sets the collider dimensions.
func (c *Collider) SetSize(width, height float64) {
	c.Width = width
	c.Height = height
}

// SetOffset sets the offset added to the owner's position before computing the
// collider bounds.
func (c *Collider) SetOffset(x, y float64) {
	c.Offset = math.NewVector2(x, y)
}

// GetSize returns the collider dimensions.
func (c *Collider) GetSize() (width, height float64) {
	return c.Width, c.Height
}

// CheckOverlap reports whether this collider overlaps another, testing the two rotated
// shapes directly (separating axis theorem) rather than their axis-aligned bounds, so an
// angled collider collides with its true quad.
func (c *Collider) CheckOverlap(other *Collider) bool {
	if other == nil {
		return false
	}
	return quadOverlap(c.corners(), other.corners())
}

// ContainsPoint reports whether a point is inside this collider's world-space shape — the
// rotated/scaled quad for a transformed owner, not its enclosing AABB.
func (c *Collider) ContainsPoint(point math.Vector2) bool {
	return shapeContainsPoint(c.GetOwner(), c.Width, c.Height, c.Offset, point)
}

// corners returns the collider's four world-space corners (top-left, top-right,
// bottom-right, bottom-left) of its offset×width×height shape under the owner's transform.
func (c *Collider) corners() [4]math.Vector2 {
	return shapeCorners(c.GetOwner(), c.Width, c.Height, c.Offset)
}

// isRotated reports whether the collider's owner transform includes a rotation, making its
// world shape an angled quad rather than an axis-aligned rect.
func (c *Collider) isRotated() bool {
	owner := c.GetOwner()
	return owner != nil && !owner.UI && owner.Transform.Rotation != 0
}

// DrawDebug draws the collider's hitbox as an outline when the scene has debug
// drawing enabled (imge build --debug). The outline is translucent green; the
// current selection is drawn brighter so it stands out.
func (c *Collider) DrawDebug(r core.Renderer, info core.DebugInfo) {
	color := math.NewColor(0, 220, 90, 170)
	if info.Selected {
		color = math.NewColor(130, 255, 170, 255)
	}
	// Draw the hitbox in screen space with a constant on-screen thickness, like the
	// editor's selection outline: the rect is mapped through the object transform (so a
	// rotated/scaled collider hugs the true on-screen quad) and stroked at full screen
	// resolution, thin and crisp at any zoom instead of the blocky world-space outline.
	owner := c.GetOwner()
	rect := math.NewRect(c.Offset.X, c.Offset.Y, c.Width, c.Height)
	if owner == nil || owner.UI {
		rect = c.GetBounds()
	}
	r.DrawRectOutlineScreen(rect, color, 1)
}

// LocalBounds returns the collider's local-space rectangle (Offset × Width×Height) —
// the rect the owner transform then scales and rotates about the object origin.
// Transforming its four corners through the owner transform yields the collider's actual
// on-screen quad, which the editor uses to draw a rotated selection outline that hugs the
// shape instead of its axis-aligned enclosing box. For a UI object it returns the
// screen-space rect (same as GetBounds).
func (c *Collider) LocalBounds() math.Rect {
	owner := c.GetOwner()
	if owner != nil && !owner.UI {
		return math.NewRect(c.Offset.X, c.Offset.Y, c.Width, c.Height)
	}
	return c.GetBounds()
}

// DebugBounds returns the collider's world-space box for editor hit-testing, the
// same rectangle DrawDebug outlines.
func (c *Collider) DebugBounds() math.Rect { return c.GetBounds() }

// shapeBounds returns the world-space axis-aligned bounding box occupied by a
// width×height shape anchored at the owner's position plus offset (top-left corner),
// scaled and rotated by the owner's transform about the object origin. The owner may
// be nil (the rectangle is then anchored at the offset alone, un-scaled), which keeps
// the helper usable before an object is in a scene.
func shapeBounds(owner *core.Object, width, height float64, offset math.Vector2) math.Rect {
	if owner == nil {
		return math.NewRect(offset.X, offset.Y, width, height)
	}
	local := math.NewRect(offset.X, offset.Y, width, height)
	if owner.UI {
		// Screen space: no object transform; anchor at the owner position + offset.
		return math.NewRect(owner.Transform.Position.X+offset.X, owner.Transform.Position.Y+offset.Y, width, height)
	}
	return owner.Transform.RectBounds(local)
}

// transformBounds returns the axis-aligned bounding box of a width×height rectangle
// whose top-left corner sits at pos, scaled by (scaleX, scaleY) and rotated by
// rotation about the rectangle's center. The center pivot mirrors the one Sprite.Draw
// uses, so a collider and its sprite stay aligned under scale and rotation. A zero
// rotation and unit scale produce the plain pos/size rectangle.
func transformBounds(pos math.Vector2, width, height, rotation, scaleX, scaleY float64) math.Rect {
	hw := width / 2 * scaleX
	hh := height / 2 * scaleY
	cx := pos.X + hw
	cy := pos.Y + hh

	aw := stdmath.Abs(hw)
	ah := stdmath.Abs(hh)
	cos := stdmath.Abs(stdmath.Cos(rotation))
	sin := stdmath.Abs(stdmath.Sin(rotation))
	extentX := aw*cos + ah*sin
	extentY := aw*sin + ah*cos

	return math.NewRect(cx-extentX, cy-extentY, 2*extentX, 2*extentY)
}

// shapeCandidates returns the objects a shape may interact with, filtered by
// collidesWith. Empty collidesWith means every active object; otherwise the scene
// tag index is used for O(1) lookup per tag. The owner is always excluded.
func shapeCandidates(owner *core.Object, collidesWith []string) []*core.Object {
	if owner == nil || owner.Scene == nil {
		return nil
	}
	scene := owner.Scene

	if len(collidesWith) == 0 {
		objs := make([]*core.Object, 0, len(scene.Objects))
		for _, obj := range scene.Objects {
			if obj != owner && obj.Active && !obj.IsDestroyed() {
				objs = append(objs, obj)
			}
		}
		return objs
	}

	seen := make(map[uint64]bool)
	var objs []*core.Object
	for _, tag := range collidesWith {
		for _, obj := range scene.FindObjectsWithTag(tag) {
			if obj == owner || !obj.Active || obj.IsDestroyed() || seen[obj.ID] {
				continue
			}
			seen[obj.ID] = true
			objs = append(objs, obj)
		}
	}
	return objs
}

// unionCollidesWith merges the collidesWith tag lists of a compound body into one
// filter. An empty list means "interact with everything", which dominates: if any
// collider in the body has no filter, the body interacts with every object (nil).
func unionCollidesWith(colliders []*Collider) []string {
	unfiltered := false
	seen := make(map[string]bool)
	var tags []string
	for _, c := range colliders {
		if len(c.CollidesWith) == 0 {
			unfiltered = true
			continue
		}
		for _, tag := range c.CollidesWith {
			if !seen[tag] {
				seen[tag] = true
				tags = append(tags, tag)
			}
		}
	}
	if unfiltered {
		return nil
	}
	return tags
}

// shapeCorners returns the four world-space corners (top-left, top-right, bottom-right,
// bottom-left) of a width×height shape anchored at offset, under the owner's transform
// (scale -> rotate -> translate about the object origin). For a UI object the shape is
// anchored at the owner position plus offset with no scale or rotation; a nil owner keeps
// the rect at offset un-scaled.
func shapeCorners(owner *core.Object, width, height float64, offset math.Vector2) [4]math.Vector2 {
	corners := [4]math.Vector2{
		math.NewVector2(offset.X, offset.Y),
		math.NewVector2(offset.X+width, offset.Y),
		math.NewVector2(offset.X+width, offset.Y+height),
		math.NewVector2(offset.X, offset.Y+height),
	}
	if owner == nil {
		return corners
	}
	if owner.UI {
		for i := range corners {
			corners[i] = corners[i].Add(owner.Transform.Position)
		}
		return corners
	}
	for i := range corners {
		corners[i] = owner.Transform.LocalToWorld(corners[i])
	}
	return corners
}

// shapeContainsPoint reports whether point lies inside the world-space quad the shape
// occupies. For a rotated/scaled owner it inverse-transforms the point into local space and
// tests against the local rect, so the hit-test follows the true angled shape rather than
// its enclosing AABB.
func shapeContainsPoint(owner *core.Object, width, height float64, offset math.Vector2, point math.Vector2) bool {
	if owner != nil && !owner.UI {
		point = owner.Transform.WorldToLocal(point)
	} else if owner != nil {
		point = point.Subtract(owner.Transform.Position)
	}
	return point.X >= offset.X && point.X <= offset.X+width &&
		point.Y >= offset.Y && point.Y <= offset.Y+height
}

// projectQuad projects a quad's vertices onto an axis, returning the min and max.
func projectQuad(quad [4]math.Vector2, axis math.Vector2) (min, max float64) {
	d := quad[0].Dot(axis)
	min, max = d, d
	for _, p := range quad[1:] {
		d = p.Dot(axis)
		if d < min {
			min = d
		}
		if d > max {
			max = d
		}
	}
	return min, max
}

// quadOverlap reports whether two convex quads overlap, using the separating axis theorem.
// Edge normals from both quads are tested; if any axis separates the projections the quads
// are disjoint. Touching edges count as non-overlapping (matching Rect.Overlaps).
func quadOverlap(a, b [4]math.Vector2) bool {
	var axes [8]math.Vector2
	n := 0
	for _, quad := range [2][4]math.Vector2{a, b} {
		for i := 0; i < 4; i++ {
			edge := quad[(i+1)%4].Subtract(quad[i])
			axes[n] = math.NewVector2(-edge.Y, edge.X)
			n++
		}
	}
	for i := 0; i < n; i++ {
		amin, amax := projectQuad(a, axes[i])
		bmin, bmax := projectQuad(b, axes[i])
		if amax <= bmin || bmax <= amin {
			return false
		}
	}
	return true
}

// translateQuad returns a copy of quad shifted by dist along one axis (0 = X, 1 = Y).
func translateQuad(quad [4]math.Vector2, axis int, dist float64) [4]math.Vector2 {
	out := quad
	if axis == 0 {
		for i := range out {
			out[i].X += dist
		}
	} else {
		for i := range out {
			out[i].Y += dist
		}
	}
	return out
}

// overlapInterval returns the closed interval of translation distances [tLo, tHi] along unit
// direction d for which the mover quad's projection on axis overlaps (touching counts) the
// obstacle quad's projection. When the axis is perpendicular to the motion (k == 0) the
// projection is translation-independent: it returns an infinite interval if the projections
// already overlap, or an empty interval (tLo > tHi) if they are separated forever.
func overlapInterval(mover, obstacle [4]math.Vector2, axis, d math.Vector2) (tLo, tHi float64) {
	m0, m1 := projectQuad(mover, axis)
	o0, o1 := projectQuad(obstacle, axis)
	k := axis.Dot(d)
	if k == 0 {
		if m1 < o0 || m0 > o1 {
			return 1, -1 // separated forever on this axis
		}
		return stdmath.Inf(-1), stdmath.Inf(1)
	}
	// mover projection at t is [m0 + k*t, m1 + k*t]; overlap iff
	// m1 + k*t >= o0 and m0 + k*t <= o1. Solve for t, mindful of the sign of k.
	if k > 0 {
		return (o0 - m1) / k, (o1 - m0) / k
	}
	return (o1 - m0) / k, (o0 - m1) / k
}

// obbContactAlong returns how far a quad may travel along one axis in direction dir before
// first touching an obstacle quad, capped at maxDist. It is the rotated-shape counterpart to
// contactAlong: instead of the axis-aligned gap it solves the exact first contact of the two
// quads via the separating axis theorem over time, so an angled rect resolves against its
// true shape. A quad already overlapping the obstacle returns 0; no contact within maxDist
// returns maxDist.
func obbContactAlong(axis int, dir float64, mover, obstacle [4]math.Vector2, maxDist float64) float64 {
	if quadOverlap(mover, obstacle) {
		return 0
	}
	var d math.Vector2
	if axis == 0 {
		d = math.NewVector2(dir, 0)
	} else {
		d = math.NewVector2(0, dir)
	}
	var axes [8]math.Vector2
	n := 0
	for _, quad := range [2][4]math.Vector2{mover, obstacle} {
		for i := 0; i < 4; i++ {
			edge := quad[(i+1)%4].Subtract(quad[i])
			axes[n] = math.NewVector2(-edge.Y, edge.X)
			n++
		}
	}
	tLo, tHi := stdmath.Inf(-1), stdmath.Inf(1)
	for i := 0; i < n; i++ {
		a, b := overlapInterval(mover, obstacle, axes[i], d)
		if a > tLo {
			tLo = a
		}
		if b < tHi {
			tHi = b
		}
	}
	// Contact occurs only within [tLo, tHi]. Clamp to [0, maxDist].
	if tLo >= tHi || tHi <= 0 || tLo >= maxDist {
		return maxDist
	}
	if tLo <= 0 {
		return 0
	}
	return tLo
}
