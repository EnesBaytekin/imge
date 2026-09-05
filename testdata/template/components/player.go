package components

import (
	"github.com/EnesBaytekin/imge/core"
)

// PlayerComponent is the platformer brain. It reads keyboard input, writes the
// owner's @Velocity (horizontal run + jump impulse), picks the right @Animator
// clip and flips the @Sprite, detects when it is standing on the ground, attacks
// nearby crates (damaging their @Health), and respawns at its spawn point when its
// own @Health reaches zero.
type PlayerComponent struct {
	core.BaseComponent
	Speed     float64 `json:"speed"`
	JumpSpeed float64 `json:"jumpSpeed"`

	velocity *Velocity
	mover    *Mover
	animator *Animator
	sprite   *Sprite
	health   *Health

	spawnX  float64
	spawnY  float64
	facing  float64 // +1 right, -1 left
	invuln  float64
	curClip string
}

// Requires declares the sibling components this player needs to function.
func (c *PlayerComponent) Requires() []string {
	return []string{"@Velocity", "@Gravity", "@Mover", "@Collider", "@Health", "@Sprite", "@Animator"}
}

func (c *PlayerComponent) Initialize() {
	if c.Speed <= 0 {
		c.Speed = 180
	}
	if c.JumpSpeed <= 0 {
		c.JumpSpeed = 520
	}

	owner := c.GetOwner()
	c.velocity = core.GetFrom[*Velocity](owner)
	c.mover = core.GetFrom[*Mover](owner)
	c.animator = core.GetFrom[*Animator](owner)
	c.sprite = core.GetFrom[*Sprite](owner)
	c.health = core.GetFrom[*Health](owner)
	c.facing = 1

	if owner != nil {
		c.spawnX = owner.Transform.Position.X
		c.spawnY = owner.Transform.Position.Y
	}

	// Respawn when this object's own @Health hits zero.
	c.On("died", func(data any) {
		obj, ok := data.(*core.Object)
		if !ok || obj != c.GetOwner() {
			return
		}
		c.respawn()
	})
}

func (c *PlayerComponent) respawn() {
	if c.mover != nil {
		c.mover.Teleport(c.spawnX, c.spawnY)
	}
	if c.velocity != nil {
		c.velocity.SetVelocity(0, 0)
	}
	if c.health != nil {
		c.health.Heal(9999)
	}
	c.invuln = 1.0
}

func (c *PlayerComponent) Update(ctx *core.Context) {
	owner := c.GetOwner()
	if owner == nil {
		return
	}
	if c.velocity == nil {
		c.velocity = core.GetFrom[*Velocity](owner)
	}
	if c.velocity == nil {
		return
	}

	if c.invuln > 0 {
		c.invuln -= ctx.DeltaTime()
	}

	// Horizontal input.
	dir := 0.0
	if ctx.Input.IsKeyPressed(core.KeyA) || ctx.Input.IsKeyPressed(core.KeyLeft) {
		dir = -1
	}
	if ctx.Input.IsKeyPressed(core.KeyD) || ctx.Input.IsKeyPressed(core.KeyRight) {
		dir = 1
	}

	vx, vy := c.velocity.Velocity()
	if dir != 0 {
		vx = dir * c.Speed
	}

	// Jump only when standing on something.
	jump := ctx.Input.IsKeyJustPressed(core.KeySpace) ||
		ctx.Input.IsKeyJustPressed(core.KeyW) ||
		ctx.Input.IsKeyJustPressed(core.KeyUp)
	if jump && c.grounded() {
		vy = -c.JumpSpeed
	}

	c.velocity.SetVelocity(vx, vy)

	// Face and animate.
	if dir > 0 {
		c.facing = 1
	} else if dir < 0 {
		c.facing = -1
	}
	if c.sprite != nil {
		c.sprite.FlipX = c.facing < 0
	}
	if c.animator != nil {
		clip := "idle"
		if !c.grounded() {
			clip = "jump"
		} else if dir != 0 {
			clip = "run"
		}
		if clip != c.curClip {
			c.curClip = clip
			c.animator.Play(clip)
		}
	}

	// Attack nearby crates.
	if ctx.Input.IsKeyJustPressed(core.KeyE) || ctx.Input.IsKeyJustPressed(core.KeyX) {
		c.attack()
	}
}

// grounded reports whether the owner is standing on the ground, via the @Mover's
// built-in ground probe (respects collidesWith tags and is independent of update
// order).
func (c *PlayerComponent) grounded() bool {
	if c.mover == nil {
		return false
	}
	return c.mover.IsGrounded()
}

// attack damages the nearest crate within reach.
func (c *PlayerComponent) attack() {
	owner := c.GetOwner()
	if owner == nil || owner.Scene == nil {
		return
	}

	me := owner.Transform.Position
	var target *core.Object
	best := 0.0

	for _, other := range owner.Scene.Objects {
		if other == owner || other.IsDestroyed() || !other.Active || !other.HasTag("crate") {
			continue
		}
		d := other.Transform.Position.DistanceSquared(me)
		if target == nil || d < best {
			target = other
			best = d
		}
	}

	if target == nil || best > 52*52 {
		return
	}
	if h := core.GetFrom[*Health](target); h != nil {
		h.Damage(1)
	}
}
