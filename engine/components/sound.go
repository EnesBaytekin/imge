package components

import (
	"github.com/EnesBaytekin/imge/core"
)

// Sound plays a sound effect or background music. Set play_on_start to true to
// begin playback automatically on the first frame; otherwise drive playback from
// another component via Play()/Stop().
//
// Export variables (JSON args): sound, volume (0.0-1.0), loop, play_on_start.
type Sound struct {
	core.BaseComponent

	Sound       string  `json:"sound"`
	Volume      float64 `json:"volume"`
	Loop        bool    `json:"loop"`
	PlayOnStart bool    `json:"play_on_start"`

	started bool
}

// Initialize applies defaults.
func (s *Sound) Initialize() {
	if s.Volume <= 0 {
		s.Volume = 1.0
	}
}

// Update auto-plays the sound once on the first frame when play_on_start is set.
func (s *Sound) Update(ctx *core.Context) {
	if s.PlayOnStart && !s.started {
		s.started = true
		s.Play(ctx)
	}
}

// Play starts playback. If loop is true, plays as background music; otherwise as
// a one-shot sound effect.
func (s *Sound) Play(ctx *core.Context) {
	if s.Sound == "" {
		return
	}
	if s.Loop {
		ctx.Audio.PlayMusic(s.Sound, true)
	} else {
		ctx.Audio.PlaySound(s.Sound, s.Volume, 1.0)
	}
}

// Stop stops the currently playing sound or music.
func (s *Sound) Stop(ctx *core.Context) {
	ctx.Audio.StopMusic()
}

// SetVolume sets the playback volume (0.0 to 1.0). Takes effect on the next Play().
func (s *Sound) SetVolume(volume float64) {
	s.Volume = volume
}

// SetLoop sets whether the sound loops continuously.
func (s *Sound) SetLoop(loop bool) {
	s.Loop = loop
}
