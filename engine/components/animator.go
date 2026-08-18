package components

import (
	"github.com/EnesBaytekin/imge/core"
	"github.com/EnesBaytekin/imge/core/math"
)

// Animation is a named clip: a sequence of frames into the sprite's texture. A
// clip either lists explicit frame rects, or describes a horizontal strip via
// frameWidth/frameHeight/frameCount starting at the texture origin.
type Animation struct {
	Name       string       `json:"name"`
	FrameW     float64      `json:"frameWidth"`
	FrameH     float64      `json:"frameHeight"`
	FrameCount int          `json:"frameCount"`
	FPS        float64      `json:"fps"`
	Loop       bool         `json:"loop"`
	Frames     []math.Rect  `json:"frames"` // explicit frames (override the strip)
}

// Animator plays named animation clips by setting the @Sprite's source region
// each frame. It requires the owner to have a @Sprite to draw into.
//
// Export variables (JSON args): clips, default.
type Animator struct {
	core.BaseComponent

	Clips   []Animation `json:"clips"`
	Default string      `json:"default"` // clip name to play on start

	clips   map[string]Animation
	current *Animation
	frame   int
	timer   float64
	playing bool
	sprite  *Sprite
}

// Requires declares the component this one needs to function.
func (a *Animator) Requires() []string { return []string{"@Sprite"} }

// Initialize normalizes clips, resolves the sprite, and starts the default clip.
func (a *Animator) Initialize() {
	a.clips = make(map[string]Animation, len(a.Clips))
	for _, clip := range a.Clips {
		if clip.FPS <= 0 {
			clip.FPS = 12
		}
		if len(clip.Frames) > 0 {
			clip.FrameCount = len(clip.Frames)
		}
		a.clips[clip.Name] = clip
	}

	a.sprite = core.GetFrom[*Sprite](a.GetOwner())

	if a.Default != "" {
		a.Play(a.Default)
	} else if len(a.Clips) > 0 {
		a.Play(a.Clips[0].Name)
	}
}

// Update advances the current clip and updates the sprite's source region.
func (a *Animator) Update(ctx *core.Context) {
	if !a.playing || a.current == nil {
		return
	}
	if a.current.FPS <= 0 || a.current.FrameCount <= 1 {
		return
	}

	a.timer += ctx.DeltaTime()
	frameDuration := 1.0 / a.current.FPS

	for a.timer >= frameDuration {
		a.timer -= frameDuration
		a.frame++

		if a.frame >= a.current.FrameCount {
			if a.current.Loop {
				a.frame = 0
			} else {
				a.frame = a.current.FrameCount - 1
				a.playing = false
				break
			}
		}
	}

	a.applyFrame()
}

// applyFrame writes the current frame's source rect into the sprite.
func (a *Animator) applyFrame() {
	if a.sprite == nil || a.current == nil {
		return
	}

	if len(a.current.Frames) > 0 {
		a.sprite.SetSourceRect(a.current.Frames[a.frame])
		return
	}

	col := a.frame
	a.sprite.SetSourceRect(math.NewRect(
		float64(col)*a.current.FrameW, 0,
		a.current.FrameW, a.current.FrameH,
	))
}

// Play starts the named clip from its first frame.
func (a *Animator) Play(name string) {
	clip, ok := a.clips[name]
	if !ok {
		return
	}
	cp := clip
	a.current = &cp
	a.frame = 0
	a.timer = 0
	a.playing = true
	a.applyFrame()
}

// Stop halts playback and resets to the clip's first frame.
func (a *Animator) Stop() {
	a.playing = false
	a.frame = 0
	a.timer = 0
	a.applyFrame()
}

// IsPlaying reports whether a clip is currently playing.
func (a *Animator) IsPlaying() bool { return a.playing }
