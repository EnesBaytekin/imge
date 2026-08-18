package components

import (
	"github.com/EnesBaytekin/imge/core"
	"github.com/EnesBaytekin/imge/core/math"
)

// Bounce moves its owner at a constant velocity and reflects that velocity when a
// @Mover blocks an axis (e.g. hitting a wall). It emits "bounce" with a surface
// normal Vector2 when it reflects. Without a @Mover it moves freely.
//
// Export variables (JSON args): vx, vy.
type Bounce struct {
	core.BaseComponent

	VX float64 `json:"vx"`
	VY float64 `json:"vy"`

	vx float64
	vy float64
}

// Initialize copies the configured velocity.
func (c *Bounce) Initialize() {
	c.vx = c.VX
	c.vy = c.VY
	if c.vx == 0 && c.vy == 0 {
		c.vx = 100
	}
}

// Update integrates the velocity and reflects on collision.
func (c *Bounce) Update(ctx *core.Context) {
	owner := c.GetOwner()
	if owner == nil {
		return
	}

	dt := ctx.DeltaTime()
	mover := core.GetFrom[*Mover](owner)
	if mover == nil {
		owner.Transform.Position = owner.Transform.Position.Add(math.NewVector2(c.vx*dt, c.vy*dt))
		return
	}

	res := mover.Move(c.vx*dt, c.vy*dt)
	if !res.X && c.vx != 0 {
		c.vx = -c.vx
		c.Emit("bounce", math.NewVector2(-1, 0))
	}
	if !res.Y && c.vy != 0 {
		c.vy = -c.vy
		c.Emit("bounce", math.NewVector2(0, -1))
	}
}
