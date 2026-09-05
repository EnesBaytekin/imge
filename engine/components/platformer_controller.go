package components

import (
	"github.com/EnesBaytekin/imge/core"
)

// PlatformerController is a velocity-based platformer controller. Unlike
// @PlayerController (the overhead/arcade 4-way mover), it drives the owner's
// @Velocity instead of moving the position directly, so it composes with @Gravity
// (the fall) and @Mover/@Collider (collision + ground detection) into a proper
// platformer body.
//
//   - Left/right sets horizontal velocity to ±move_speed (snappy; no momentum).
//   - Jump (Space, W, or Up) fires a vertical impulse of jump_speed when grounded,
//     with a short coyote window (coyote_time) so a jump pressed just after walking
//     off a ledge still registers.
//   - Releasing the jump key while rising shortens the jump (variable height).
//
// Export variables (JSON args): move_speed, jump_speed, coyote_time.
type PlatformerController struct {
	core.BaseComponent

	MoveSpeed  float64 `json:"move_speed"`
	JumpSpeed  float64 `json:"jump_speed"`
	CoyoteTime float64 `json:"coyote_time"`

	velocity *Velocity
	coyote   float64 // seconds remaining to still jump after leaving the ground
}

// Requires declares the component this one needs to function.
func (c *PlatformerController) Requires() []string { return []string{"@Velocity"} }

// Initialize applies defaults and resolves the velocity dependency.
func (c *PlatformerController) Initialize() {
	if c.MoveSpeed <= 0 {
		c.MoveSpeed = 160
	}
	if c.JumpSpeed <= 0 {
		c.JumpSpeed = 420
	}
	if c.CoyoteTime <= 0 {
		c.CoyoteTime = 0.1
	}
	c.velocity = core.GetFrom[*Velocity](c.GetOwner())
}

// Update reads input and writes the owner's velocity.
func (c *PlatformerController) Update(ctx *core.Context) {
	if c.velocity == nil {
		c.velocity = core.GetFrom[*Velocity](c.GetOwner())
	}
	if c.velocity == nil || ctx == nil || ctx.Input == nil {
		return
	}

	// Horizontal: set vx directly (instant acceleration and stop). Vertical stays
	// owned by @Gravity and the jump impulse below.
	var dir float64
	if ctx.Input.IsKeyPressed(core.KeyA) || ctx.Input.IsKeyPressed(core.KeyLeft) {
		dir = -1
	} else if ctx.Input.IsKeyPressed(core.KeyD) || ctx.Input.IsKeyPressed(core.KeyRight) {
		dir = 1
	}
	vx := dir * c.MoveSpeed
	_, vy := c.velocity.Velocity()

	// Grounded is a query (order-independent), not derived from the last move.
	if c.isGrounded() {
		c.coyote = c.CoyoteTime
	} else if c.coyote > 0 {
		c.coyote -= ctx.DeltaTime()
		if c.coyote < 0 {
			c.coyote = 0
		}
	}

	if c.jumpPressed(ctx) && c.coyote > 0 {
		vy = -c.JumpSpeed
		c.coyote = 0 // consume the window so a held key doesn't re-trigger
	}

	// Variable jump height: releasing the key mid-ascent shortens the jump.
	if vy < 0 && c.jumpReleased(ctx) {
		vy *= 0.5
	}

	c.velocity.SetVelocity(vx, vy)
}

// isGrounded resolves the owner's @Mover ground probe. Without a mover there is no
// collision, so the controller reports airborne (and jump does nothing).
func (c *PlatformerController) isGrounded() bool {
	mover := core.GetFrom[*Mover](c.GetOwner())
	return mover != nil && mover.IsGrounded()
}

func (c *PlatformerController) jumpPressed(ctx *core.Context) bool {
	return ctx.Input.IsKeyJustPressed(core.KeySpace) ||
		ctx.Input.IsKeyJustPressed(core.KeyW) ||
		ctx.Input.IsKeyJustPressed(core.KeyUp)
}

func (c *PlatformerController) jumpReleased(ctx *core.Context) bool {
	return ctx.Input.IsKeyJustReleased(core.KeySpace) ||
		ctx.Input.IsKeyJustReleased(core.KeyW) ||
		ctx.Input.IsKeyJustReleased(core.KeyUp)
}

// SetMoveSpeed sets the horizontal speed.
func (c *PlatformerController) SetMoveSpeed(speed float64) { c.MoveSpeed = speed }

// SetJumpSpeed sets the jump impulse speed.
func (c *PlatformerController) SetJumpSpeed(speed float64) { c.JumpSpeed = speed }

// SetCoyoteTime sets the post-ledge jump grace window.
func (c *PlatformerController) SetCoyoteTime(seconds float64) { c.CoyoteTime = seconds }
