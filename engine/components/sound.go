package components

import (
	"github.com/EnesBaytekin/imge/core"
)

// SoundComponent plays a sound effect or background music. Set play_on_start to
// true to begin playback automatically on the first frame; otherwise drive
// playback from another component via Play()/Stop().
//
// Export variables (JSON args): sound, volume (0.0-1.0), loop, play_on_start.
type SoundComponent struct {
	core.BaseComponent

	Sound       string  `json:"sound"`
	Volume      float64 `json:"volume"`
	Loop        bool    `json:"loop"`
	PlayOnStart bool    `json:"play_on_start"`

	// local state
	started bool
}

// Initialize applies defaults.
func (c *SoundComponent) Initialize() {
	if c.Volume <= 0 {
		c.Volume = 1.0
	}
}

// Update auto-plays the sound once on the first frame when play_on_start is set.
func (c *SoundComponent) Update(ctx *core.Context) {
	if c.PlayOnStart && !c.started {
		c.started = true
		c.Play(ctx)
	}
}

// Play starts playback. If loop is true, plays as background music; otherwise as
// a one-shot sound effect.
func (c *SoundComponent) Play(ctx *core.Context) {
	if c.Sound == "" {
		return
	}

	if c.Loop {
		ctx.Audio.PlayMusic(c.Sound, true)
	} else {
		ctx.Audio.PlaySound(c.Sound, c.Volume, 1.0)
	}
}

// Stop stops the currently playing sound or music.
func (c *SoundComponent) Stop(ctx *core.Context) {
	ctx.Audio.StopMusic()
}

// SetVolume sets the playback volume (0.0 to 1.0). Takes effect on the next Play().
func (c *SoundComponent) SetVolume(volume float64) {
	c.Volume = volume
}

// SetLoop sets whether the sound loops continuously.
func (c *SoundComponent) SetLoop(loop bool) {
	c.Loop = loop
}
