package components

import (
	"github.com/EnesBaytekin/imge/core"
)

// SoundComponent plays a sound effect or background music. Set play_on_start to
// true to begin playback automatically on the first frame; otherwise drive
// playback from another component via Play()/Stop().
type SoundComponent struct {
	core.BaseComponent
	soundID     string
	volume      float64
	loop        bool
	playOnStart bool
	started     bool
}

// Initialize parses component configuration from JSON args.
// Supported args: sound, volume (0.0-1.0, default 1.0), loop, play_on_start.
func (c *SoundComponent) Initialize(args []interface{}) error {
	c.volume = 1.0

	if len(args) > 0 {
		if argMap, ok := args[0].(map[string]interface{}); ok {
			if id, ok := argMap["sound"].(string); ok {
				c.soundID = id
			}
			if v, ok := argMap["volume"].(float64); ok {
				c.volume = v
			}
			if l, ok := argMap["loop"].(bool); ok {
				c.loop = l
			}
			if p, ok := argMap["play_on_start"].(bool); ok {
				c.playOnStart = p
			}
		}
	}

	return nil
}

// Update auto-plays the sound once on the first frame when play_on_start is set.
func (c *SoundComponent) Update(ctx *core.ComponentContext) {
	if c.playOnStart && !c.started {
		c.started = true
		c.Play(ctx)
	}
}

// Play starts playback. If loop is true, plays as background music; otherwise as
// a one-shot sound effect.
func (c *SoundComponent) Play(ctx *core.ComponentContext) {
	if c.soundID == "" {
		return
	}

	if c.loop {
		ctx.Audio.PlayMusic(c.soundID, true)
	} else {
		ctx.Audio.PlaySound(c.soundID, c.volume, 1.0)
	}
}

// Stop stops the currently playing sound or music.
func (c *SoundComponent) Stop(ctx *core.ComponentContext) {
	ctx.Audio.StopMusic()
}

// SetVolume sets the playback volume (0.0 to 1.0). Takes effect on the next Play().
func (c *SoundComponent) SetVolume(volume float64) {
	c.volume = volume
}

// SetLoop sets whether the sound loops continuously.
func (c *SoundComponent) SetLoop(loop bool) {
	c.loop = loop
}

func init() {
	core.RegisterComponent("@Sound", func(args []interface{}) (core.Component, error) {
		return &SoundComponent{}, nil
	})
}
