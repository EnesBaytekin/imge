package components

import (
	"github.com/EnesBaytekin/imge/core"
	"github.com/EnesBaytekin/imge/core/math"
)

// Velocity holds per-axis speed and integrates it through a @Mover each frame.
// It is the state other components (Gravity, Friction, PlayerController) read and
// write. When the @Mover blocks an axis, that axis's velocity is zeroed.
//
// Export variables (JSON args): vx, vy (initial velocity).
type Velocity struct {
	core.BaseComponent

	VX float64 `json:"vx"`
	VY float64 `json:"vy"`

	vx float64
	vy float64
}

// Initialize copies the configured initial velocity into runtime state.
func (v *Velocity) Initialize() {
	v.vx = v.VX
	v.vy = v.VY
}

// Update integrates the current velocity through the owner's @Mover (or, without
// one, moves the owner directly).
func (v *Velocity) Update(ctx *core.Context) {
	owner := v.GetOwner()
	if owner == nil {
		return
	}

	dt := ctx.DeltaTime()
	dx := v.vx * dt
	dy := v.vy * dt

	mover := core.GetFrom[*Mover](owner)
	if mover == nil {
		owner.Transform.Position = owner.Transform.Position.Add(math.NewVector2(dx, dy))
		return
	}

	res := mover.Move(dx, dy)
	if !res.X {
		v.vx = 0
	}
	if !res.Y {
		v.vy = 0
	}
}

// SetVelocity sets the current velocity.
func (v *Velocity) SetVelocity(x, y float64) {
	v.vx = x
	v.vy = y
}

// Velocity returns the current velocity.
func (v *Velocity) Velocity() (x, y float64) {
	return v.vx, v.vy
}

// AddVelocity adds to the current velocity (e.g. an acceleration * dt).
func (v *Velocity) AddVelocity(x, y float64) {
	v.vx += x
	v.vy += y
}
