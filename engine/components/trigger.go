package components

import (
	"sort"

	"github.com/EnesBaytekin/imge/core"
	"github.com/EnesBaytekin/imge/core/math"
)

// Trigger is a sensor region. It never blocks or is pushed — it only detects when
// another object's @Collider shapes overlap it, tracking the overlap set and
// emitting "trigger_enter" / "trigger_exit" events.
//
// Multiple triggers on one object are independent sensors, each with its own area.
// Unlike @Collider (pure physics), a trigger carries no PushFactor and takes no
// part in movement resolution.
//
// Export variables (JSON args): width, height, offset {x,y}, collides_with.
type Trigger struct {
	core.BaseComponent

	Width  float64 `json:"width"`
	Height float64 `json:"height"`

	// Offset shifts the trigger rectangle relative to the owner's position
	// (top-left corner).
	Offset math.Vector2 `json:"offset"`

	// CollidesWith lists object tags this trigger detects. Empty means it detects
	// every object.
	CollidesWith []string `json:"collides_with"`

	// overlaps tracks the objects currently overlapping this trigger.
	overlaps map[uint64]*core.Object
}

// Initialize applies defaults.
func (t *Trigger) Initialize() {
	if t.Width <= 0 {
		t.Width = 32
	}
	if t.Height <= 0 {
		t.Height = 32
	}
	t.overlaps = make(map[uint64]*core.Object)
}

// Update refreshes the overlap set and emits enter/exit events. It is compound-
// aware: an object is considered overlapping when ANY of its @Collider shapes
// overlaps this trigger, not just the first.
func (t *Trigger) Update(ctx *core.Context) {
	owner := t.GetOwner()
	if owner == nil || owner.Scene == nil {
		return
	}

	bounds := t.GetBounds()
	current := make(map[uint64]*core.Object)

	for _, other := range shapeCandidates(owner, t.CollidesWith) {
		if !overlapsAnyCollider(other, bounds) {
			continue
		}
		current[other.ID] = other
		if _, seen := t.overlaps[other.ID]; !seen {
			t.Emit("trigger_enter", other)
		}
	}

	for id, other := range t.overlaps {
		if _, still := current[id]; !still {
			t.Emit("trigger_exit", other)
		}
	}

	t.overlaps = current
}

// GetBounds returns the trigger rectangle in world space, anchored at the owner's
// position plus Offset (top-left corner).
func (t *Trigger) GetBounds() math.Rect {
	return shapeBounds(t.GetOwner(), t.Width, t.Height, t.Offset)
}

// SetSize sets the trigger dimensions.
func (t *Trigger) SetSize(width, height float64) {
	t.Width = width
	t.Height = height
}

// SetOffset sets the offset added to the owner's position before computing the
// trigger bounds.
func (t *Trigger) SetOffset(x, y float64) {
	t.Offset = math.NewVector2(x, y)
}

// GetSize returns the trigger dimensions.
func (t *Trigger) GetSize() (width, height float64) {
	return t.Width, t.Height
}

// CheckOverlap reports whether this trigger overlaps another trigger.
func (t *Trigger) CheckOverlap(other *Trigger) bool {
	if other == nil {
		return false
	}
	return t.GetBounds().Overlaps(other.GetBounds())
}

// ContainsPoint reports whether a point is inside this trigger.
func (t *Trigger) ContainsPoint(point math.Vector2) bool {
	return t.GetBounds().ContainsPoint(point)
}

// GetOverlaps returns the objects currently overlapping this trigger, ordered by
// object ID for determinism.
func (t *Trigger) GetOverlaps() []*core.Object {
	result := make([]*core.Object, 0, len(t.overlaps))
	for _, obj := range t.overlaps {
		result = append(result, obj)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result
}

// overlapsAnyCollider reports whether any of obj's @Collider shapes overlaps r.
func overlapsAnyCollider(obj *core.Object, r math.Rect) bool {
	for _, collider := range core.GetAllFrom[*Collider](obj) {
		if r.Overlaps(collider.GetBounds()) {
			return true
		}
	}
	return false
}
