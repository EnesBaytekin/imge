// Package components contains IMGE's built-in components. At build time, custom
// component files are merged into this same package, so built-ins and customs can
// call each other's methods directly (no capability interfaces needed).
package components

import (
	"sort"

	"github.com/EnesBaytekin/imge/core"
	"github.com/EnesBaytekin/imge/core/math"
)

// ColliderMode describes how a collider participates in movement resolution.
type ColliderMode string

const (
	// ColliderSolid blocks movers outright (walls, ground).
	ColliderSolid ColliderMode = "solid"
	// ColliderPushable is pushed out of the way by a mover; PushFactor scales how
	// far it slides.
	ColliderPushable ColliderMode = "pushable"
	// ColliderTrigger never blocks or is pushed; it only detects overlaps.
	ColliderTrigger ColliderMode = "trigger"
)

// Collider is a rectangle shape. It answers overlap queries, tracks which objects
// currently overlap it (emitting "collision_enter"/"collision_exit"), and — via
// its Mode — tells movers how to resolve against it.
//
// Export variables (JSON args): width, height, mode, pushFactor, collidesWith.
type Collider struct {
	core.BaseComponent

	Width  float64 `json:"width"`
	Height float64 `json:"height"`

	// Mode controls how movement resolves against this collider.
	Mode ColliderMode `json:"mode"`

	// PushFactor scales how far a pushable collider slides when pushed
	// (0 = immovable, 1 = full; default 1).
	PushFactor float64 `json:"pushFactor"`

	// CollidesWith lists object tags this collider interacts with. Empty means it
	// interacts with every object.
	CollidesWith []string `json:"collidesWith"`

	// overlaps tracks the objects currently overlapping this collider.
	overlaps map[uint64]*core.Object
}

// Initialize applies defaults.
func (c *Collider) Initialize() {
	if c.Width <= 0 {
		c.Width = 32
	}
	if c.Height <= 0 {
		c.Height = 32
	}
	if c.Mode == "" {
		c.Mode = ColliderSolid
	}
	if c.PushFactor == 0 {
		c.PushFactor = 1.0
	}
	c.overlaps = make(map[uint64]*core.Object)
}

// Update refreshes the overlap set and emits enter/exit events.
func (c *Collider) Update(ctx *core.Context) {
	owner := c.GetOwner()
	if owner == nil || owner.Scene == nil {
		return
	}

	bounds := c.GetBounds()
	current := make(map[uint64]*core.Object)

	for _, other := range c.candidates() {
		otherCollider := core.GetFrom[*Collider](other)
		if otherCollider == nil || !bounds.Overlaps(otherCollider.GetBounds()) {
			continue
		}
		current[other.ID] = other
		if _, seen := c.overlaps[other.ID]; !seen {
			c.Emit("collision_enter", other)
		}
	}

	for id, other := range c.overlaps {
		if _, still := current[id]; !still {
			c.Emit("collision_exit", other)
		}
	}

	c.overlaps = current
}

// GetBounds returns the collider rectangle in world space, anchored at the
// owner's position (top-left corner).
func (c *Collider) GetBounds() math.Rect {
	owner := c.GetOwner()
	if owner == nil {
		return math.NewRect(0, 0, c.Width, c.Height)
	}
	return math.NewRect(
		owner.Transform.Position.X,
		owner.Transform.Position.Y,
		c.Width,
		c.Height,
	)
}

// SetSize sets the collider dimensions.
func (c *Collider) SetSize(width, height float64) {
	c.Width = width
	c.Height = height
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

// GetOverlaps returns the objects currently overlapping this collider, ordered by
// object ID for determinism.
func (c *Collider) GetOverlaps() []*core.Object {
	result := make([]*core.Object, 0, len(c.overlaps))
	for _, obj := range c.overlaps {
		result = append(result, obj)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result
}

// candidates returns the objects this collider may interact with, filtered by
// CollidesWith. Empty CollidesWith means every active object; otherwise the scene
// tag index is used for O(1) lookup per tag.
func (c *Collider) candidates() []*core.Object {
	owner := c.GetOwner()
	if owner == nil || owner.Scene == nil {
		return nil
	}
	scene := owner.Scene

	if len(c.CollidesWith) == 0 {
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
	for _, tag := range c.CollidesWith {
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

// fmin/fmax are small float helpers so this package doesn't import the standard
// library math (which would collide with core/math).
func fmin(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func fmax(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}
