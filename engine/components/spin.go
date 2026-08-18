package components

import (
	"github.com/EnesBaytekin/imge/core"
)

// Spin rotates its owner over time.
//
// Export variables (JSON args): speed (radians per second).
type Spin struct {
	core.BaseComponent

	Speed float64 `json:"speed"`
}

// Initialize applies defaults.
func (c *Spin) Initialize() {
	if c.Speed == 0 {
		c.Speed = 3
	}
}

// Update advances the owner's rotation.
func (c *Spin) Update(ctx *core.Context) {
	owner := c.GetOwner()
	if owner == nil {
		return
	}
	owner.Transform.Rotation += c.Speed * ctx.DeltaTime()
}
