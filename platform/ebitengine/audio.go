package ebitengine

import (
	"bytes"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/audio/mp3"
	"github.com/hajimehoshi/ebiten/v2/audio/vorbis"
	"github.com/hajimehoshi/ebiten/v2/audio/wav"
)

const defaultSampleRate = 44100

// sound holds fully-decoded PCM data in the audio context's format.
type sound struct {
	pcm []byte
}

// Audio implements core.Audio using Ebitengine's audio package.
type Audio struct {
	context *audio.Context

	sounds map[string]*sound
	music  map[string]*sound

	// players keeps references to active one-shot players alive. Ebitengine stops
	// a player when it is garbage collected, so we retain them until they finish.
	players []*audio.Player

	musicPlayer *audio.Player

	masterVolume float64
	soundVolume  float64
	musicVolume  float64
}

func newAudio() *Audio {
	return &Audio{
		context:      audio.NewContext(defaultSampleRate),
		sounds:       make(map[string]*sound),
		music:        make(map[string]*sound),
		masterVolume: 1.0,
		soundVolume:  1.0,
		musicVolume:  1.0,
	}
}

// load decodes an audio file and caches its PCM data. soundID is used as the
// cache key; path is resolved from the working directory or assets/.
func (a *Audio) load(cache map[string]*sound, soundID, path string) error {
	if _, ok := cache[soundID]; ok {
		return nil
	}

	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	var src io.Reader
	switch strings.ToLower(filepath.Ext(path)) {
	case ".mp3":
		s, err := mp3.Decode(a.context, f)
		if err != nil {
			return err
		}
		src = s
	case ".ogg":
		s, err := vorbis.Decode(a.context, f)
		if err != nil {
			return err
		}
		src = s
	default: // treat as WAV
		s, err := wav.Decode(a.context, f)
		if err != nil {
			return err
		}
		src = s
	}

	pcm, err := io.ReadAll(src)
	if err != nil {
		return err
	}

	cache[soundID] = &sound{pcm: pcm}
	return nil
}

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

// PlaySound plays a sound effect once.
func (a *Audio) PlaySound(soundID string, volume, pitch float64) {
	if err := a.load(a.sounds, soundID, resolveAssetPath(soundID)); err != nil {
		log.Printf("ebitengine: sound not found: %q (%v)", soundID, err)
		return
	}

	s := a.sounds[soundID]
	p, err := audio.NewPlayer(a.context, bytes.NewReader(s.pcm))
	if err != nil {
		log.Printf("ebitengine: failed to play sound %q: %v", soundID, err)
		return
	}

	p.SetVolume(clamp01(volume * a.masterVolume * a.soundVolume))
	p.Play()
	a.players = append(a.players, p)
	a.prunePlayers()
}

// PlayMusic starts background music playback, optionally looping.
func (a *Audio) PlayMusic(musicID string, loop bool) {
	if err := a.load(a.music, musicID, resolveAssetPath(musicID)); err != nil {
		log.Printf("ebitengine: music not found: %q (%v)", musicID, err)
		return
	}

	a.StopMusic()

	s := a.music[musicID]
	var src io.Reader
	if loop {
		src = audio.NewInfiniteLoop(bytes.NewReader(s.pcm), int64(len(s.pcm)))
	} else {
		src = bytes.NewReader(s.pcm)
	}

	p, err := audio.NewPlayer(a.context, src)
	if err != nil {
		log.Printf("ebitengine: failed to play music %q: %v", musicID, err)
		return
	}

	p.SetVolume(clamp01(a.masterVolume * a.musicVolume))
	p.Play()
	a.musicPlayer = p
}

// StopMusic stops any currently playing music.
func (a *Audio) StopMusic() {
	if a.musicPlayer != nil {
		a.musicPlayer.Close()
		a.musicPlayer = nil
	}
}

// PauseMusic pauses the current music.
func (a *Audio) PauseMusic() {
	if a.musicPlayer != nil {
		a.musicPlayer.Pause()
	}
}

// ResumeMusic resumes paused music.
func (a *Audio) ResumeMusic() {
	if a.musicPlayer != nil {
		a.musicPlayer.Play()
	}
}

// SetMasterVolume sets the overall volume (0.0 to 1.0).
func (a *Audio) SetMasterVolume(volume float64) {
	a.masterVolume = clamp01(volume)
	if a.musicPlayer != nil {
		a.musicPlayer.SetVolume(clamp01(a.masterVolume * a.musicVolume))
	}
}

// SetSoundVolume sets the volume for sound effects.
func (a *Audio) SetSoundVolume(volume float64) {
	a.soundVolume = clamp01(volume)
}

// SetMusicVolume sets the volume for music.
func (a *Audio) SetMusicVolume(volume float64) {
	a.musicVolume = clamp01(volume)
	if a.musicPlayer != nil {
		a.musicPlayer.SetVolume(clamp01(a.masterVolume * a.musicVolume))
	}
}

// prunePlayers drops finished one-shot players so the slice doesn't grow without bound.
func (a *Audio) prunePlayers() {
	kept := a.players[:0]
	for _, p := range a.players {
		if p.IsPlaying() {
			kept = append(kept, p)
		}
	}
	a.players = kept
}

// close stops all playback and releases resources.
func (a *Audio) close() {
	a.StopMusic()
	for _, p := range a.players {
		p.Close()
	}
	a.players = nil
}
