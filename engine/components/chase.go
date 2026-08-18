package components

import (
	"github.com/EnesBaytekin/imge/core"
	"github.com/EnesBaytekin/imge/core/math"
)

// Chase moves its owner toward the nearest object carrying a tag, stopping within
// StopDistance if set. It drives the owner's @Mover; without one it teleports.
//
// Export variables (JSON args): speed, targetTag, stopDistance.
type Chase struct {
	core.BaseComponent

	Speed        float64 `json:"speed"`
	TargetTag    string  `json:"targetTag"`
	StopDistance float64 `json:"stopDistance"`
}

// Initialize applies defaults.
func (c *Chase) Initialize() {
	if c.Speed <= 0 {
		c.Speed = 60
	}
	if c.TargetTag == "" {
		c.TargetTag = "player"
	}
}

// Update chases the nearest tagged target.
func (c *Chase) Update(ctx *core.Context) {
	owner := c.GetOwner()
	if owner == nil || owner.Scene == nil {
		return
	}

	targets := owner.Scene.FindObjectsWithTag(c.TargetTag)
	if len(targets) == 0 {
		return
	}

	target := nearest(owner.Transform.Position, targets)
	dir := target.Transform.Position.Subtract(owner.Transform.Position)
	dist := dir.Length()
	if dist < 0.001 {
		return
	}
	if c.StopDistance > 0 && dist <= c.StopDistance {
		return
	}

	dir = dir.Divide(dist)
	move := dir.Multiply(c.Speed * ctx.DeltaTime())

	if mover := core.GetFrom[*Mover](owner); mover != nil {
		mover.Move(move.X, move.Y)
		return
	}
	owner.Transform.Position = owner.Transform.Position.Add(move)
}

// nearest returns the object in objs closest to pos.
func nearest(pos math.Vector2, objs []*core.Object) *core.Object {
	var best *core.Object
	bestDist := 0.0
	for _, obj := range objs {
		if obj == nil || obj.IsDestroyed() {
			continue
		}
		d := obj.Transform.Position.DistanceSquared(pos)
		if best == nil || d < bestDist {
			best = obj
			bestDist = d
		}
	}
	return best
}
