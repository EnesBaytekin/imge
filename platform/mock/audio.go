package mock

import (
	"fmt"
)

// MockAudio implements core.Audio interface with debug prints.
type MockAudio struct {
	masterVolume float64
	soundVolume  float64
	musicVolume  float64
}

// PlaySound plays a sound effect once.
func (a *MockAudio) PlaySound(soundID string, volume, pitch float64) {
	fmt.Printf("[MockAudio] PlaySound(id=%s, volume=%f, pitch=%f)\n", soundID, volume, pitch)
}

// PlayMusic starts playing background music.
func (a *MockAudio) PlayMusic(musicID string, loop bool) {
	fmt.Printf("[MockAudio] PlayMusic(id=%s, loop=%v)\n", musicID, loop)
}

// StopMusic stops any currently playing music.
func (a *MockAudio) StopMusic() {
	fmt.Println("[MockAudio] StopMusic()")
}

// PauseMusic pauses the current music (can be resumed with ResumeMusic).
func (a *MockAudio) PauseMusic() {
	fmt.Println("[MockAudio] PauseMusic()")
}

// ResumeMusic resumes paused music.
func (a *MockAudio) ResumeMusic() {
	fmt.Println("[MockAudio] ResumeMusic()")
}

// SetMasterVolume sets the overall volume (0.0 to 1.0).
func (a *MockAudio) SetMasterVolume(volume float64) {
	fmt.Printf("[MockAudio] SetMasterVolume(volume=%f)\n", volume)
	a.masterVolume = volume
}

// SetSoundVolume sets the volume for sound effects.
func (a *MockAudio) SetSoundVolume(volume float64) {
	fmt.Printf("[MockAudio] SetSoundVolume(volume=%f)\n", volume)
	a.soundVolume = volume
}

// SetMusicVolume sets the volume for music.
func (a *MockAudio) SetMusicVolume(volume float64) {
	fmt.Printf("[MockAudio] SetMusicVolume(volume=%f)\n", volume)
	a.musicVolume = volume
}