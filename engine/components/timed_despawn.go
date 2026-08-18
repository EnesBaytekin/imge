package components

import (
	"github.com/EnesBaytekin/imge/core"
)

// TimedDespawn destroys its owner after a fixed lifetime.
//
// Export variables (JSON args): lifetime (seconds).
type TimedDespawn struct {
	core.BaseComponent

	Lifetime float64 `json:"lifetime"`
	elapsed  float64
}

// Initialize applies defaults.
func (c *TimedDespawn) Initialize() {
	if c.Lifetime <= 0 {
		c.Lifetime = 5
	}
}

// Update counts down and destroys the owner.
func (c *TimedDespawn) Update(ctx *core.Context) {
	owner := c.GetOwner()
	if owner == nil {
		return
	}
	c.elapsed += ctx.DeltaTime()
	if c.elapsed >= c.Lifetime {
		owner.Destroy()
	}
}
