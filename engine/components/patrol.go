package components

import (
	"github.com/EnesBaytekin/imge/core"
	"github.com/EnesBaytekin/imge/core/math"
)

// Patrol moves its owner between a list of waypoints, looping (or ping-ponging)
// through them. It drives the owner's @Mover; without one it teleports.
//
// Export variables (JSON args): points, speed, pingPong.
type Patrol struct {
	core.BaseComponent

	Points   []math.Vector2 `json:"points"`
	Speed    float64        `json:"speed"`
	PingPong bool           `json:"pingPong"`

	index int
	dir   int
}

// Initialize applies defaults.
func (c *Patrol) Initialize() {
	if c.Speed <= 0 {
		c.Speed = 60
	}
	c.dir = 1
}

// Update advances toward the current waypoint.
func (c *Patrol) Update(ctx *core.Context) {
	owner := c.GetOwner()
	if owner == nil || len(c.Points) < 2 {
		return
	}

	target := c.Points[c.index]
	delta := target.Subtract(owner.Transform.Position)
	dist := delta.Length()
	step := c.Speed * ctx.DeltaTime()

	if dist <= step {
		owner.Transform.Position = target
		c.advance()
		return
	}

	dir := delta.Divide(dist)
	if mover := core.GetFrom[*Mover](owner); mover != nil {
		mover.Move(dir.X*step, dir.Y*step)
		return
	}
	owner.Transform.Position = owner.Transform.Position.Add(dir.Multiply(step))
}

// advance moves to the next waypoint (loop or ping-pong).
func (c *Patrol) advance() {
	if c.PingPong {
		next := c.index + c.dir
		if next < 0 || next >= len(c.Points) {
			c.dir = -c.dir
			next = c.index + c.dir
		}
		c.index = next
		return
	}
	c.index = (c.index + 1) % len(c.Points)
}
