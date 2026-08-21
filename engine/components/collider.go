// Package components contains IMGE's built-in components. At build time, custom
// component files are merged into this same package, so built-ins and customs can
// call each other's methods directly (no capability interfaces needed).
package components

import (
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

// CheckOverlap reports whether this collider overlaps another.
func (c *Collider) CheckOverlap(other *Collider) bool {
	if other == nil {
		return false
	}
	return c.GetBounds().Overlaps(other.GetBounds())
}

// ContainsPoint reports whether a point is inside this collider.
func (c *Collider) ContainsPoint(point math.Vector2) bool {
	return c.GetBounds().ContainsPoint(point)
}

// shapeBounds returns the world-space rectangle occupied by a width×height shape
// anchored at the owner's position plus offset (top-left corner). The owner may be
// nil (the rectangle is then anchored at the offset alone), which keeps the helper
// usable before an object is in a scene.
func shapeBounds(owner *core.Object, width, height float64, offset math.Vector2) math.Rect {
	if owner == nil {
		return math.NewRect(offset.X, offset.Y, width, height)
	}
	return math.NewRect(
		owner.Transform.Position.X+offset.X,
		owner.Transform.Position.Y+offset.Y,
		width,
		height,
	)
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
