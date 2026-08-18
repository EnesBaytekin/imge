package components

import (
	"github.com/EnesBaytekin/imge/core"
	"github.com/EnesBaytekin/imge/core/math"
)

// DoorComponent is the exit. It stays locked until the "all_gold" event fires,
// at which point it plays its "open" @Animator clip and its @Sound. When the
// player comes within OpenDistance while unlocked, it emits "win". While locked
// and the player is within WarnDistance, it draws a "!" warning above the door.
type DoorComponent struct {
	core.BaseComponent
	OpenDistance float64 `json:"openDistance"`
	WarnDistance float64 `json:"warnDistance"`

	animator *Animator
	sound    *Sound
	unlocked bool
	near     bool
	done     bool
	ctx      *core.Context
}

// Requires declares the sibling components this door needs to function.
func (c *DoorComponent) Requires() []string {
	return []string{"@Animator", "@Sprite", "@Sound"}
}

func (c *DoorComponent) Initialize() {
	if c.OpenDistance <= 0 {
		c.OpenDistance = 40
	}
	if c.WarnDistance <= 0 {
		c.WarnDistance = 100
	}
	c.animator = core.GetFrom[*Animator](c.GetOwner())
	c.sound = core.GetFrom[*Sound](c.GetOwner())

	c.On("all_gold", func(data any) {
		c.unlock()
	})
}

func (c *DoorComponent) unlock() {
	if c.unlocked {
		return
	}
	c.unlocked = true
	if c.animator != nil {
		c.animator.Play("open")
	}
	if c.sound != nil && c.ctx != nil {
		c.sound.Play(c.ctx)
	}
}

func (c *DoorComponent) Update(ctx *core.Context) {
	c.ctx = ctx
	owner := c.GetOwner()
	if owner == nil || owner.Scene == nil || c.done {
		return
	}

	players := owner.Scene.FindObjectsWithTag("player")
	if len(players) == 0 {
		return
	}
	player := players[0]

	// Compare centers rather than top-left corners.
	doorCenter := owner.Transform.Position.Add(math.NewVector2(16, 24))
	playerCenter := player.Transform.Position.Add(math.NewVector2(14, 16))
	dist := doorCenter.Distance(playerCenter)

	if c.unlocked && dist < c.OpenDistance {
		c.done = true
		c.Emit("win", nil)
		return
	}
	c.near = !c.unlocked && dist < c.WarnDistance
}

func (c *DoorComponent) Draw(renderer core.Renderer) {
	if !c.near {
		return
	}
	owner := c.GetOwner()
	if owner == nil {
		return
	}
	x := owner.Transform.Position.X + 16
	y := owner.Transform.Position.Y - 26
	yellow := math.NewColor(255, 220, 60, 255)
	renderer.DrawRect(math.NewRect(x-3, y, 6, 14), yellow)    // stem
	renderer.DrawRect(math.NewRect(x-3, y+18, 6, 6), yellow)  // dot
}
