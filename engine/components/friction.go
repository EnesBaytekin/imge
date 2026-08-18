package components

import (
	"github.com/EnesBaytekin/imge/core"
)

// Friction damps the owner's @Velocity toward zero. Amount is the per-second
// speed reduction in px/s; Axes selects which axis ("x", "y", or "both").
//
// Export variables (JSON args): amount, axes.
type Friction struct {
	core.BaseComponent

	Amount float64 `json:"amount"`
	Axes   string  `json:"axes"`

	velocity *Velocity
}

// Requires declares the component this one needs to function.
func (f *Friction) Requires() []string { return []string{"@Velocity"} }

// Initialize applies defaults and resolves the velocity dependency.
func (f *Friction) Initialize() {
	if f.Axes == "" {
		f.Axes = "both"
	}
	f.velocity = core.GetFrom[*Velocity](f.GetOwner())
}

// Update damps the selected velocity components.
func (f *Friction) Update(ctx *core.Context) {
	if f.velocity == nil {
		f.velocity = core.GetFrom[*Velocity](f.GetOwner())
	}
	if f.velocity == nil || f.Amount <= 0 {
		return
	}

	dt := ctx.DeltaTime()
	vx, vy := f.velocity.Velocity()

	if f.Axes == "both" || f.Axes == "x" {
		vx = dampAxis(vx, f.Amount*dt)
	}
	if f.Axes == "both" || f.Axes == "y" {
		vy = dampAxis(vy, f.Amount*dt)
	}

	f.velocity.SetVelocity(vx, vy)
}

// dampAxis reduces v toward zero by step without overshooting.
func dampAxis(v, step float64) float64 {
	if v == 0 || step <= 0 {
		return v
	}
	if v > 0 {
		v -= step
		if v < 0 {
			return 0
		}
		return v
	}
	v += step
	if v > 0 {
		return 0
	}
	return v
}

// SetAmount sets the per-second speed reduction.
func (f *Friction) SetAmount(amount float64) {
	f.Amount = amount
}

// SetAxes sets which axes are damped ("x", "y", or "both").
func (f *Friction) SetAxes(axes string) {
	f.Axes = axes
}
