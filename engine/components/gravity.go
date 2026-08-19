package components

import (
	"github.com/EnesBaytekin/imge/core"
	"github.com/EnesBaytekin/imge/core/math"
)

// Gravity applies a constant acceleration to the owner's @Velocity each frame,
// optionally clamped to a maximum speed.
//
// Export variables (JSON args): acceleration {x,y}, maxSpeed.
type Gravity struct {
	core.BaseComponent

	Acceleration math.Vector2 `json:"acceleration"`
	MaxSpeed     float64      `json:"max_speed"`

	velocity *Velocity
}

// Requires declares the component this one needs to function.
func (g *Gravity) Requires() []string { return []string{"@Velocity"} }

// Initialize applies defaults (downward, ~980 px/s^2).
func (g *Gravity) Initialize() {
	if g.Acceleration == (math.Vector2{}) {
		g.Acceleration = math.NewVector2(0, 980)
	}
	g.velocity = core.GetFrom[*Velocity](g.GetOwner())
}

// Update integrates acceleration into the velocity, clamped to MaxSpeed.
func (g *Gravity) Update(ctx *core.Context) {
	if g.velocity == nil {
		g.velocity = core.GetFrom[*Velocity](g.GetOwner())
	}
	if g.velocity == nil {
		return
	}

	dt := ctx.DeltaTime()
	vx, vy := g.velocity.Velocity()
	vx += g.Acceleration.X * dt
	vy += g.Acceleration.Y * dt

	if g.MaxSpeed > 0 {
		speed := math.NewVector2(vx, vy).Length()
		if speed > g.MaxSpeed {
			scale := g.MaxSpeed / speed
			vx *= scale
			vy *= scale
		}
	}

	g.velocity.SetVelocity(vx, vy)
}

// SetAcceleration sets the acceleration vector.
func (g *Gravity) SetAcceleration(x, y float64) {
	g.Acceleration = math.NewVector2(x, y)
}

// SetMaxSpeed sets the speed cap (<= 0 disables the cap).
func (g *Gravity) SetMaxSpeed(maxSpeed float64) {
	g.MaxSpeed = maxSpeed
}
