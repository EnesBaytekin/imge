package components

import (
	"github.com/EnesBaytekin/imge/core"
)

// Damage applies damage to overlapping objects that have a @Health. TargetTags
// restricts which tagged objects take damage (empty = any). Cooldown spaces out
// repeated ticks (seconds; 0 = every frame).
//
// Export variables (JSON args): amount, targetTags, cooldown.
type Damage struct {
	core.BaseComponent

	Amount     int      `json:"amount"`
	TargetTags []string `json:"target_tags"`
	Cooldown   float64  `json:"cooldown"`

	timer float64
}

// Requires declares the component this one needs to function.
func (d *Damage) Requires() []string { return []string{"@Collider"} }

// Initialize applies defaults.
func (d *Damage) Initialize() {
	if d.Amount <= 0 {
		d.Amount = 1
	}
}

// Update applies damage to overlapping targets on the cooldown.
func (d *Damage) Update(ctx *core.Context) {
	d.timer -= ctx.DeltaTime()
	if d.timer > 0 {
		return
	}

	owner := d.GetOwner()
	if owner == nil {
		return
	}
	collider := core.GetFrom[*Collider](owner)
	if collider == nil {
		return
	}

	applied := false
	for _, other := range collider.GetOverlaps() {
		if !matchesAnyTag(other, d.TargetTags) {
			continue
		}
		if health := core.GetFrom[*Health](other); health != nil {
			health.Damage(d.Amount)
			applied = true
		}
	}

	if applied && d.Cooldown > 0 {
		d.timer = d.Cooldown
	}
}

// matchesAnyTag reports whether obj has any of tags (empty = always true).
func matchesAnyTag(obj *core.Object, tags []string) bool {
	if obj == nil {
		return false
	}
	if len(tags) == 0 {
		return true
	}
	for _, tag := range tags {
		if obj.HasTag(tag) {
			return true
		}
	}
	return false
}
