package sdl

import (
	"fmt"
	"log"
	"sync"

	"github.com/veandco/go-sdl2/mix"
)

// SDLAudio implements core.Audio interface using SDL_mixer.
type SDLAudio struct {
	// Sound cache
	sounds map[string]*mix.Chunk
	// Music cache
	music map[string]*mix.Music

	// Volume settings
	masterVolume float64
	soundVolume  float64
	musicVolume  float64

	// Current playing music
	currentMusic *mix.Music

	// Mutex for thread safety
	mu sync.RWMutex
}

// NewSDLAudio creates a new SDL audio subsystem.
// Should be called after SDL_mixer initialization.
func NewSDLAudio() *SDLAudio {
	return &SDLAudio{
		sounds:       make(map[string]*mix.Chunk),
		music:        make(map[string]*mix.Music),
		masterVolume: 1.0,
		soundVolume:  1.0,
		musicVolume:  1.0,
		currentMusic: nil,
	}
}

// LoadSound loads a sound effect from file and caches it.
func (a *SDLAudio) LoadSound(soundID, filePath string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if _, exists := a.sounds[soundID]; exists {
		return nil // Already loaded
	}

	chunk, err := mix.LoadWAV(filePath)
	if err != nil {
		return fmt.Errorf("failed to load sound %s: %v", filePath, err)
	}

	a.sounds[soundID] = chunk
	log.Printf("Sound loaded: %s -> %s", soundID, filePath)
	return nil
}

// LoadMusic loads a music track from file and caches it.
func (a *SDLAudio) LoadMusic(musicID, filePath string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if _, exists := a.music[musicID]; exists {
		return nil // Already loaded
	}

	music, err := mix.LoadMUS(filePath)
	if err != nil {
		return fmt.Errorf("failed to load music %s: %v", filePath, err)
	}

	a.music[musicID] = music
	log.Printf("Music loaded: %s -> %s", musicID, filePath)
	return nil
}

// UnloadSound unloads a sound effect from cache.
func (a *SDLAudio) UnloadSound(soundID string) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if chunk, exists := a.sounds[soundID]; exists {
		chunk.Free()
		delete(a.sounds, soundID)
		log.Printf("Sound unloaded: %s", soundID)
	}
}

// UnloadMusic unloads a music track from cache.
func (a *SDLAudio) UnloadMusic(musicID string) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if music, exists := a.music[musicID]; exists {
		music.Free()
		delete(a.music, musicID)
		log.Printf("Music unloaded: %s", musicID)
	}
}

// PlaySound plays a sound effect once.
// soundID identifies a previously loaded sound.
// volume ranges from 0.0 (silent) to 1.0 (full volume).
// pitch ranges from 0.5 (half speed) to 2.0 (double speed) - not supported in SDL_mixer.
func (a *SDLAudio) PlaySound(soundID string, volume, pitch float64) {
	a.mu.RLock()
	chunk, exists := a.sounds[soundID]
	a.mu.RUnlock()

	if !exists {
		log.Printf("Sound not found: %s", soundID)
		return
	}

	// Calculate final volume (master * sound * volume)
	finalVolume := int(a.masterVolume * a.soundVolume * volume * 128.0) // SDL_mixer uses 0-128
	if finalVolume < 0 {
		finalVolume = 0
	}
	if finalVolume > 128 {
		finalVolume = 128
	}

	// Set volume for this chunk
	chunk.Volume(finalVolume)

	// Play sound once (channel -1 = first free channel)
	channel, err := chunk.Play(-1, 0)
	if err != nil {
		log.Printf("Failed to play sound %s: %v", soundID, err)
		return
	}

	// TODO: Pitch control not directly supported by SDL_mixer
	// Could implement by adjusting playback speed, but skip for now
	log.Printf("Playing sound: %s (channel %d, volume %d)", soundID, channel, finalVolume)
}

// PlayMusic starts playing background music.
// musicID identifies a previously loaded music track.
// loop determines if the music should repeat.
func (a *SDLAudio) PlayMusic(musicID string, loop bool) {
	a.mu.RLock()
	music, exists := a.music[musicID]
	a.mu.RUnlock()

	if !exists {
		log.Printf("Music not found: %s", musicID)
		return
	}

	// Stop current music if playing
	if a.currentMusic != nil && mix.PlayingMusic() {
		mix.HaltMusic()
	}

	// Calculate final volume (master * music)
	finalVolume := int(a.masterVolume * a.musicVolume * 128.0)
	if finalVolume < 0 {
		finalVolume = 0
	}
	if finalVolume > 128 {
		finalVolume = 128
	}
	mix.VolumeMusic(finalVolume)

	// Play music
	loops := 0
	if loop {
		loops = -1 // -1 means infinite loop in SDL_mixer
	}

	if err := music.Play(loops); err != nil {
		log.Printf("Failed to play music %s: %v", musicID, err)
		return
	}

	a.currentMusic = music
	log.Printf("Playing music: %s (loop: %v, volume: %d)", musicID, loop, finalVolume)
}

// StopMusic stops any currently playing music.
func (a *SDLAudio) StopMusic() {
	if mix.PlayingMusic() {
		mix.HaltMusic()
		a.currentMusic = nil
		log.Println("Music stopped")
	}
}

// PauseMusic pauses the current music (can be resumed with ResumeMusic).
func (a *SDLAudio) PauseMusic() {
	if mix.PlayingMusic() {
		mix.PauseMusic()
		log.Println("Music paused")
	}
}

// ResumeMusic resumes paused music.
func (a *SDLAudio) ResumeMusic() {
	if mix.PausedMusic() {
		mix.ResumeMusic()
		log.Println("Music resumed")
	}
}

// SetMasterVolume sets the overall volume (0.0 to 1.0).
func (a *SDLAudio) SetMasterVolume(volume float64) {
	a.mu.Lock()
	a.masterVolume = volume
	a.mu.Unlock()

	// Update currently playing sounds and music
	// Note: SDL_mixer doesn't have global master volume,
	// we apply it per sound/music playback
	log.Printf("Master volume set: %f", volume)
}

// SetSoundVolume sets the volume for sound effects.
func (a *SDLAudio) SetSoundVolume(volume float64) {
	a.mu.Lock()
	a.soundVolume = volume
	a.mu.Unlock()

	// Note: Volume applied per sound when played
	log.Printf("Sound volume set: %f", volume)
}

// SetMusicVolume sets the volume for music.
func (a *SDLAudio) SetMusicVolume(volume float64) {
	a.mu.Lock()
	a.musicVolume = volume
	a.mu.Unlock()

	// Update music volume if currently playing
	if mix.PlayingMusic() || mix.PausedMusic() {
		finalVolume := int(a.masterVolume * a.musicVolume * 128.0)
		if finalVolume < 0 {
			finalVolume = 0
		}
		if finalVolume > 128 {
			finalVolume = 128
		}
		mix.VolumeMusic(finalVolume)
	}

	log.Printf("Music volume set: %f", volume)
}

// Cleanup releases SDL_mixer resources.
func (a *SDLAudio) Cleanup() {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Stop any playing music
	if mix.PlayingMusic() || mix.PausedMusic() {
		mix.HaltMusic()
	}

	// Free all sounds
	for id, chunk := range a.sounds {
		chunk.Free()
		delete(a.sounds, id)
	}

	// Free all music
	for id, music := range a.music {
		music.Free()
		delete(a.music, id)
	}

	a.currentMusic = nil
	log.Println("SDL audio cleaned up")
}