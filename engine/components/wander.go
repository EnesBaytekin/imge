package components

import (
	stdmath "math"
	"math/rand"

	"github.com/EnesBaytekin/imge/core"
	"github.com/EnesBaytekin/imge/core/math"
)

// Wander moves its owner in a random direction, re-rolling the direction every
// ChangeInterval seconds. It drives the owner's @Mover; without one it teleports.
//
// Export variables (JSON args): speed, changeInterval.
type Wander struct {
	core.BaseComponent

	Speed          float64 `json:"speed"`
	ChangeInterval float64 `json:"changeInterval"`

	direction math.Vector2
	timer     float64
}

// Initialize applies defaults and picks a starting direction.
func (c *Wander) Initialize() {
	if c.Speed <= 0 {
		c.Speed = 40
	}
	if c.ChangeInterval <= 0 {
		c.ChangeInterval = 1.5
	}
	c.direction = randomDirection()
}

// Update re-rolls direction on the interval and moves.
func (c *Wander) Update(ctx *core.Context) {
	owner := c.GetOwner()
	if owner == nil {
		return
	}

	c.timer -= ctx.DeltaTime()
	if c.timer <= 0 {
		c.direction = randomDirection()
		c.timer = c.ChangeInterval
	}

	move := c.direction.Multiply(c.Speed * ctx.DeltaTime())
	if mover := core.GetFrom[*Mover](owner); mover != nil {
		mover.Move(move.X, move.Y)
		return
	}
	owner.Transform.Position = owner.Transform.Position.Add(move)
}

// randomDirection returns a random unit vector.
func randomDirection() math.Vector2 {
	angle := rand.Float64() * 2 * stdmath.Pi
	return math.NewVector2(stdmath.Cos(angle), stdmath.Sin(angle))
}
