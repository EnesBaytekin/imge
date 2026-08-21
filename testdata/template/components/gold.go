package components

import (
	stdmath "math"

	"github.com/EnesBaytekin/imge/core"
)

// GoldComponent makes its owner bob up and down on a sine wave and collects
// itself when the player overlaps its @Trigger. The @Spin sibling provides the
// visible rotation.
type GoldComponent struct {
	core.BaseComponent
	Amplitude float64 `json:"amplitude"`
	Speed     float64 `json:"speed"`

	baseY float64
	t     float64
	ctx   *core.Context
}

// Requires declares the sibling components this gold needs to function.
func (c *GoldComponent) Requires() []string {
	return []string{"@Trigger", "@Sprite", "@Spin"}
}

func (c *GoldComponent) Initialize() {
	if c.Amplitude <= 0 {
		c.Amplitude = 5
	}
	if c.Speed <= 0 {
		c.Speed = 3
	}
	if owner := c.GetOwner(); owner != nil {
		c.baseY = owner.Transform.Position.Y
	}
}

func (c *GoldComponent) Update(ctx *core.Context) {
	c.ctx = ctx
	owner := c.GetOwner()
	if owner == nil {
		return
	}

	// Bob up and down on a sine wave.
	c.t += ctx.DeltaTime()
	owner.Transform.Position.Y = c.baseY + stdmath.Sin(c.t*c.Speed)*c.Amplitude

	// Collect only when the player overlaps THIS coin's trigger. We read the
	// trigger's own overlap set rather than the scene-global "trigger_enter"
	// event, because that event is broadcast to every listener — every coin would
	// collect at once the moment any one of them is touched.
	trg := core.GetFrom[*Trigger](owner)
	if trg == nil {
		return
	}
	for _, other := range trg.GetOverlaps() {
		if other.HasTag("player") {
			c.collect()
			return
		}
	}
}

func (c *GoldComponent) collect() {
	if c.ctx != nil {
		c.ctx.Audio.PlaySound("assets/pickup.wav", 0.8, 1.0)
	}
	c.Emit("gold_collected", c.GetOwner())
	if owner := c.GetOwner(); owner != nil {
		owner.Destroy()
	}
}
