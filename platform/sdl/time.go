package sdl

import (
	"time"

	"github.com/veandco/go-sdl2/sdl"
)

// SDLTime implements core.Time interface using SDL2 for timing.
type SDLTime struct {
	start          time.Time
	lastUpdate     time.Time
	deltaTime      float64
	totalTime      float64
	frameCount     int
	lastFPSUpdate  time.Time
	fps            float64
}

// DeltaTime returns the time elapsed since the last frame in seconds.
func (t *SDLTime) DeltaTime() float64 {
	return t.deltaTime
}

// TotalTime returns the total time elapsed since the game started in seconds.
func (t *SDLTime) TotalTime() float64 {
	return t.totalTime
}

// FPS returns the current frames per second.
func (t *SDLTime) FPS() float64 {
	return t.fps
}

// Tick should be called once per frame to update timing.
func (t *SDLTime) Tick() {
	now := time.Now()

	// Calculate delta time
	if t.lastUpdate.IsZero() {
		// First frame, use a small delta
		t.deltaTime = 1.0 / 60.0 // Assume 60 FPS
		t.start = now
		t.lastFPSUpdate = now
	} else {
		t.deltaTime = now.Sub(t.lastUpdate).Seconds()

		// Clamp delta time to reasonable values
		// Prevents large spikes when debugging or pausing
		if t.deltaTime > 0.1 { // Max 100ms (10 FPS)
			t.deltaTime = 0.1
		}
		if t.deltaTime < 0.0001 { // Min 0.1ms (10,000 FPS)
			t.deltaTime = 1.0 / 60.0 // Fallback to 60 FPS
		}
	}

	t.lastUpdate = now
	t.totalTime = now.Sub(t.start).Seconds()
	t.frameCount++

	// Update FPS every second
	if now.Sub(t.lastFPSUpdate) >= time.Second {
		t.fps = float64(t.frameCount) / now.Sub(t.lastFPSUpdate).Seconds()
		t.frameCount = 0
		t.lastFPSUpdate = now
	}
}

// Sleep pauses execution for the specified number of seconds.
// Uses SDL_Delay for more accurate timing with SDL.
func (t *SDLTime) Sleep(seconds float64) {
	// Convert seconds to milliseconds
	ms := uint32(seconds * 1000.0)
	if ms > 0 {
		sdl.Delay(ms)
	}
}

// Cleanup releases any SDL timing resources.
func (t *SDLTime) Cleanup() {
	// No SDL resources to clean up for timing
}