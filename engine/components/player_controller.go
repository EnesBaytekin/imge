package components

import (
	"github.com/EnesBaytekin/imge/core"
	"github.com/EnesBaytekin/imge/core/math"
)

// PlayerController moves its owner with WASD / arrow keys. It drives the owner's
// @Mover (collision-aware) directly; without one it teleports the position. This
// is the arcade/overhead controller — for velocity-based platformer movement,
// write your own component and set @Velocity yourself.
//
// Export variables (JSON args): speed.
type PlayerController struct {
	core.BaseComponent

	Speed float64 `json:"speed"`
}

// Initialize applies defaults.
func (c *PlayerController) Initialize() {
	if c.Speed <= 0 {
		c.Speed = 200
	}
}

// Update reads input and moves the owner.
func (c *PlayerController) Update(ctx *core.Context) {
	owner := c.GetOwner()
	if owner == nil {
		return
	}

	var dx, dy float64
	if ctx.Input.IsKeyPressed(core.KeyW) || ctx.Input.IsKeyPressed(core.KeyUp) {
		dy = -1
	}
	if ctx.Input.IsKeyPressed(core.KeyS) || ctx.Input.IsKeyPressed(core.KeyDown) {
		dy = 1
	}
	if ctx.Input.IsKeyPressed(core.KeyA) || ctx.Input.IsKeyPressed(core.KeyLeft) {
		dx = -1
	}
	if ctx.Input.IsKeyPressed(core.KeyD) || ctx.Input.IsKeyPressed(core.KeyRight) {
		dx = 1
	}
	if dx == 0 && dy == 0 {
		return
	}

	dir := math.NewVector2(dx, dy).Normalize()
	dist := c.Speed * ctx.DeltaTime()

	if mover := core.GetFrom[*Mover](owner); mover != nil {
		mover.Move(dir.X*dist, dir.Y*dist)
		return
	}
	owner.Transform.Position = owner.Transform.Position.Add(dir.Multiply(dist))
}
