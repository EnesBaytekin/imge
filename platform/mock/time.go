package mock

import (
	"fmt"
	"time"
)

// MockTime implements core.Time interface with debug prints.
type MockTime struct {
	start       time.Time
	lastUpdate  time.Time
	deltaTime   float64
	totalTime   float64
	frameCount  int
	lastFPSUpdate time.Time
	fps         float64
}

// DeltaTime returns the time elapsed since the last frame in seconds.
func (t *MockTime) DeltaTime() float64 {
	fmt.Printf("[MockTime] DeltaTime() = %f\n", t.deltaTime)
	return t.deltaTime
}

// TotalTime returns the total time elapsed since the game started in seconds.
func (t *MockTime) TotalTime() float64 {
	fmt.Printf("[MockTime] TotalTime() = %f\n", t.totalTime)
	return t.totalTime
}

// FPS returns the current frames per second.
func (t *MockTime) FPS() float64 {
	fmt.Printf("[MockTime] FPS() = %f\n", t.fps)
	return t.fps
}

// Tick should be called once per frame to update timing.
func (t *MockTime) Tick() {
	fmt.Println("[MockTime] Tick() called")

	now := time.Now()

	// Calculate delta time (simulate 60 FPS for mock)
	if t.lastUpdate.IsZero() {
		t.deltaTime = 1.0 / 60.0
	} else {
		t.deltaTime = now.Sub(t.lastUpdate).Seconds()
		// Clamp to reasonable values
		if t.deltaTime > 0.1 {
			t.deltaTime = 0.1
		}
		if t.deltaTime < 0.0001 {
			t.deltaTime = 1.0 / 60.0
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
func (t *MockTime) Sleep(seconds float64) {
	fmt.Printf("[MockTime] Sleep(seconds=%f)\n", seconds)
	time.Sleep(time.Duration(seconds * float64(time.Second)))
}