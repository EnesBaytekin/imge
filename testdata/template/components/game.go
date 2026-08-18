package components

import (
	"github.com/EnesBaytekin/imge/core"
	"github.com/EnesBaytekin/imge/core/math"
)

// GameComponent is the HUD and win state. It counts gold via "gold_collected",
// unlocks the door by emitting "all_gold" once the target is reached, and shows a
// "you win" overlay on the "win" event. It draws the gold counter and the player's
// hearts with plain shapes (the renderer has no text).
type GameComponent struct {
	core.BaseComponent
	Total int `json:"total"`

	collected int
	state     string // "playing" | "won"
}

func (c *GameComponent) Initialize() {
	if c.Total <= 0 {
		c.Total = 5
	}
	c.collected = 0
	c.state = "playing"

	c.On("gold_collected", func(data any) {
		if c.state != "playing" {
			return
		}
		c.collected++
		if c.collected >= c.Total {
			c.Emit("all_gold", nil)
		}
	})

	c.On("win", func(data any) {
		c.state = "won"
	})
}

func (c *GameComponent) Draw(renderer core.Renderer) {
	w, h := renderer.GetViewportSize()
	if w <= 0 || h <= 0 {
		return
	}

	// Coin progress icons (top-left): filled = collected, outline = remaining.
	gold := math.NewColor(255, 205, 50, 255)
	dark := math.NewColor(40, 40, 50, 255)
	for i := 0; i < c.Total; i++ {
		cx := 22.0 + float64(i)*22
		cy := 20.0
		if i < c.collected {
			renderer.DrawCircle(math.NewVector2(cx, cy), 8, gold)
			renderer.DrawCircleOutline(math.NewVector2(cx, cy), 8, math.White, 2)
		} else {
			renderer.DrawCircleOutline(math.NewVector2(cx, cy), 8, dark, 2)
		}
	}

	// Hearts = the player's current HP.
	if scene := c.Scene(); scene != nil {
		if players := scene.FindObjectsWithTag("player"); len(players) > 0 {
			if hp := core.GetFrom[*Health](players[0]); hp != nil {
				red := math.NewColor(255, 80, 90, 255)
				empty := math.NewColor(60, 60, 70, 255)
				for i := 0; i < hp.Max; i++ {
					cx := 22.0 + float64(i)*30
					cy := 46.0
					col := red
					if i >= hp.Current() {
						col = empty
					}
					renderer.DrawCircle(math.NewVector2(cx, cy), 9, col)
				}
			}
		}
	}

	if c.state != "won" {
		return
	}

	// Win overlay: dim the world and draw a big check mark.
	renderer.DrawRect(math.NewRect(0, 0, float64(w), float64(h)), math.NewColor(0, 0, 0, 150))
	cx := float64(w) / 2
	cy := float64(h) / 2
	green := math.NewColor(90, 230, 120, 255)
	white := math.White
	renderer.DrawCircle(math.NewVector2(cx, cy), 64, green)
	renderer.DrawCircleOutline(math.NewVector2(cx, cy), 64, white, 4)
	renderer.DrawCircleOutline(math.NewVector2(cx, cy), 82, white, 2)
	renderer.DrawLine(math.NewVector2(cx-30, cy+2), math.NewVector2(cx-8, cy+24), white, 7)
	renderer.DrawLine(math.NewVector2(cx-8, cy+24), math.NewVector2(cx+34, cy-28), white, 7)
}
