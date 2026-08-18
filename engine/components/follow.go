package components

import (
	"github.com/EnesBaytekin/imge/core"
	"github.com/EnesBaytekin/imge/core/math"
)

// Follow smoothly lerps its owner toward the first object carrying a tag, plus a
// fixed offset. It moves the position directly (no collision).
//
// Export variables (JSON args): targetTag, lerp, offset.
type Follow struct {
	core.BaseComponent

	TargetTag string      `json:"targetTag"`
	Lerp      float64     `json:"lerp"` // smoothing factor per frame (0..1)
	Offset    math.Vector2 `json:"offset"`
}

// Initialize applies defaults.
func (c *Follow) Initialize() {
	if c.TargetTag == "" {
		c.TargetTag = "player"
	}
	if c.Lerp <= 0 {
		c.Lerp = 0.2
	}
}

// Update lerps the owner toward the target + offset.
func (c *Follow) Update(ctx *core.Context) {
	owner := c.GetOwner()
	if owner == nil || owner.Scene == nil {
		return
	}

	targets := owner.Scene.FindObjectsWithTag(c.TargetTag)
	if len(targets) == 0 {
		return
	}

	goal := targets[0].Transform.Position.Add(c.Offset)
	owner.Transform.Position = owner.Transform.Position.Lerp(goal, c.Lerp)
}
