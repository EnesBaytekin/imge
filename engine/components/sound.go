// Package components contains built-in game components.
package components

import (
	"github.com/EnesBaytekin/imge/core"
)

// ============================================================================
// @Sound Component
// ============================================================================

// SoundComponent provides sound effect and music playback.
// Use Play() to start playback and Stop() to halt it.
// Configure loop mode for background music or single-shot for sound effects.
type SoundComponent struct {
	core.BaseComponent
	soundID string
	volume  float64
	loop    bool
}

// Initialize parses component configuration from JSON args.
// Supported args:
//
//	sound: string (sound/music identifier)
//	volume: float64 (0.0 to 1.0, default: 1.0)
//	loop: bool (true for continuous playback, default: false)
func (c *SoundComponent) Initialize(args []interface{}) error {
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
		}
	}

	if c.volume <= 0 {
		c.volume = 1.0
	}

	return nil
}

// Play starts playback of the configured sound or music.
// If loop is true, plays as background music (ctx.Audio.PlayMusic).
// Otherwise plays as a one-shot sound effect (ctx.Audio.PlaySound).
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

// SetVolume sets the playback volume (0.0 to 1.0).
func (c *SoundComponent) SetVolume(volume float64) {
	c.volume = volume
	// Note: volume takes effect on next Play() call
}

// SetLoop sets whether the sound should loop continuously.
func (c *SoundComponent) SetLoop(loop bool) {
	c.loop = loop
}

// ============================================================================
// Registration
// ============================================================================

func init() {
	core.RegisterComponent("@Sound", func(args []interface{}) (core.Component, error) {
		return &SoundComponent{}, nil
	})
}
