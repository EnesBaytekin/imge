package components

import (
	"github.com/EnesBaytekin/imge/core"
)

// Clip describes an animation clip: which sprite it draws into, at what frame
// rate, and whether it loops. The clip's identifier is the target sprite's
// component name (the "sprite" field), so clips are keyed directly by the sprite
// they animate.
type Clip struct {
	Sprite string  `json:"sprite"`
	FPS    float64 `json:"fps"`
	Loop   bool    `json:"loop"`
}

// Animator plays named clips by driving the frame of the sprite each clip targets.
// It owns the visibility and flip of every sprite it manages (the sprites named in
// its clips): exactly one managed sprite is visible at a time, and all managed
// sprites mirror the animator's flip_x/flip_y. Sprites not named in any clip are
// left untouched.
//
// When a non-looping clip finishes, it emits "animation_finished" with the sprite
// name as Data.
//
// Export variables (JSON args): clips [{sprite, fps, loop}], default, flip_x,
// flip_y.
type Animator struct {
	core.BaseComponent

	Clips   []Clip `json:"clips"`
	Default string `json:"default"`
	FlipX   bool   `json:"flip_x"`
	FlipY   bool   `json:"flip_y"`

	clips   map[string]Clip
	sprites map[string]*Sprite
	current string
	frame   int
	timer   float64
	playing bool
}

// Requires declares the component this one needs to function.
func (a *Animator) Requires() []string { return []string{"@Sprite"} }

// Initialize resolves the target sprites and auto-plays the default clip.
func (a *Animator) Initialize() {
	owner := a.GetOwner()

	a.clips = make(map[string]Clip, len(a.Clips))
	a.sprites = make(map[string]*Sprite, len(a.Clips))
	for _, clip := range a.Clips {
		if clip.FPS <= 0 {
			clip.FPS = 12
		}
		a.clips[clip.Sprite] = clip
		if sp := core.GetFromNamed[*Sprite](owner, clip.Sprite); sp != nil {
			a.sprites[clip.Sprite] = sp
		}
	}

	a.applyFlip()

	if a.Default != "" {
		a.Play(a.Default)
	} else if len(a.Clips) > 0 {
		a.Play(a.Clips[0].Sprite)
	}
}

// Update advances the current clip and writes the frame to its sprite.
func (a *Animator) Update(ctx *core.Context) {
	if !a.playing || a.current == "" {
		return
	}
	clip := a.clips[a.current]
	sprite := a.sprites[a.current]
	if clip.FPS <= 0 || sprite == nil {
		return
	}

	frameCount := sprite.FrameCount()
	if frameCount <= 1 {
		return
	}

	a.timer += ctx.DeltaTime()
	frameDuration := 1.0 / clip.FPS

	finished := false
	for a.timer >= frameDuration {
		a.timer -= frameDuration
		a.frame++

		if a.frame >= frameCount {
			if clip.Loop {
				a.frame = 0
			} else {
				a.frame = frameCount - 1
				a.playing = false
				finished = true
				break
			}
		}
	}

	a.applyFrame()
	if finished {
		a.Emit("animation_finished", a.current)
	}
}

// applyFrame writes the current frame into the current sprite.
func (a *Animator) applyFrame() {
	if sprite := a.sprites[a.current]; sprite != nil {
		sprite.SetFrame(a.frame)
	}
}

// updateVisibility shows exactly the current sprite and hides the rest.
func (a *Animator) updateVisibility() {
	for name, sprite := range a.sprites {
		sprite.SetVisible(name == a.current)
	}
}

// applyFlip mirrors the animator's flip onto every managed sprite.
func (a *Animator) applyFlip() {
	for _, sprite := range a.sprites {
		sprite.SetFlipX(a.FlipX)
		sprite.SetFlipY(a.FlipY)
	}
}

// Play starts the named clip from its first frame.
func (a *Animator) Play(name string) {
	if _, ok := a.clips[name]; !ok {
		return
	}
	a.current = name
	a.frame = 0
	a.timer = 0
	a.playing = true
	a.updateVisibility()
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

// CurrentSprite returns the name of the sprite currently playing ("" when none).
func (a *Animator) CurrentSprite() string { return a.current }

// Frame returns the current frame index.
func (a *Animator) Frame() int { return a.frame }

// SetFlipX sets horizontal flipping and applies it to the managed sprites.
func (a *Animator) SetFlipX(flip bool) {
	a.FlipX = flip
	a.applyFlip()
}

// SetFlipY sets vertical flipping and applies it to the managed sprites.
func (a *Animator) SetFlipY(flip bool) {
	a.FlipY = flip
	a.applyFlip()
}
