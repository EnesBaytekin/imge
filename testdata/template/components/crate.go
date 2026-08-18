package components

import (
	"github.com/EnesBaytekin/imge/core"
)

// CrateComponent is a breakable, pushable box. Its @Health reaches zero when the
// player attacks it; on "died" it plays a thud, pops a brief @TimedDespawn
// particle, spawns a gold coin, and destroys itself.
type CrateComponent struct {
	core.BaseComponent
	ctx *core.Context
}

// Requires declares the sibling components this crate needs to function.
func (c *CrateComponent) Requires() []string {
	return []string{"@Health", "@Collider", "@Mover"}
}

func (c *CrateComponent) Initialize() {
	c.On("died", func(data any) {
		obj, ok := data.(*core.Object)
		if !ok || obj != c.GetOwner() {
			return
		}
		c.breakOpen()
	})
}

func (c *CrateComponent) Update(ctx *core.Context) {
	c.ctx = ctx
}

func (c *CrateComponent) breakOpen() {
	owner := c.GetOwner()
	if owner == nil || owner.Scene == nil {
		return
	}
	if c.ctx != nil {
		c.ctx.Audio.PlaySound("crate.wav", 0.9, 1.0)
	}

	x := owner.Transform.Position.X
	y := owner.Transform.Position.Y
	spawnGold(owner.Scene, x+2, y-16)
	spawnPoof(owner.Scene, x+2, y+2)
	owner.Destroy()
}

// spawnGold builds a gold coin object programmatically (web-safe: no file read).
func spawnGold(scene *core.Scene, x, y float64) {
	obj := core.NewObject("gold")
	obj.AddTag("gold")
	obj.SetDepth(5)
	obj.SetPosition(x, y)

	gold := &GoldComponent{Amplitude: 5, Speed: 3}
	gold.SetName("gold")
	gold.SetKind("components/gold.go")
	_ = obj.AddComponent(gold)

	spr := &Sprite{Texture: "coin.png", Width: 20, Height: 20}
	spr.SetName("sprite")
	spr.SetKind("@Sprite")
	_ = obj.AddComponent(spr)

	spin := &Spin{Speed: 3}
	spin.SetName("spin")
	spin.SetKind("@Spin")
	_ = obj.AddComponent(spin)

	col := &Collider{Width: 20, Height: 20, Mode: ColliderTrigger, CollidesWith: []string{"player"}}
	col.SetName("hitbox")
	col.SetKind("@Collider")
	_ = obj.AddComponent(col)

	_ = scene.AddObject(obj)
}

// spawnPoof builds a short-lived spinning particle (showcases @TimedDespawn).
func spawnPoof(scene *core.Scene, x, y float64) {
	obj := core.NewObject("poof")
	obj.SetDepth(6)
	obj.SetPosition(x, y)

	spr := &Sprite{Texture: "orb.png", Width: 20, Height: 20}
	spr.SetName("sprite")
	spr.SetKind("@Sprite")
	_ = obj.AddComponent(spr)

	spin := &Spin{Speed: 8}
	spin.SetName("spin")
	spin.SetKind("@Spin")
	_ = obj.AddComponent(spin)

	td := &TimedDespawn{Lifetime: 0.35}
	td.SetName("despawn")
	td.SetKind("@TimedDespawn")
	_ = obj.AddComponent(td)

	_ = scene.AddObject(obj)
}
