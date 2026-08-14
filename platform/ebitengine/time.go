package ebitengine

import (
	"time"

	"github.com/hajimehoshi/ebiten/v2"
)

// Time implements core.Time using wall-clock timing.
type Time struct {
	start      time.Time
	lastUpdate time.Time
	deltaTime  float64
	totalTime  float64
}

func newTime() *Time {
	return &Time{}
}

// DeltaTime returns the time elapsed since the last frame in seconds.
func (t *Time) DeltaTime() float64 {
	return t.deltaTime
}

// TotalTime returns the total time elapsed since the game started in seconds.
func (t *Time) TotalTime() float64 {
	return t.totalTime
}

// FPS returns the current frames per second as measured by Ebitengine.
func (t *Time) FPS() float64 {
	return ebiten.ActualFPS()
}

// Tick advances the timing state. Called once per frame by the runner.
func (t *Time) Tick() {
	now := time.Now()

	if t.lastUpdate.IsZero() {
		t.deltaTime = 1.0 / 60.0
		t.start = now
	} else {
		t.deltaTime = now.Sub(t.lastUpdate).Seconds()
		// Clamp to avoid huge spikes when the window is dragged or paused.
		if t.deltaTime > 0.1 {
			t.deltaTime = 0.1
		}
		if t.deltaTime <= 0 {
			t.deltaTime = 1.0 / 60.0
		}
	}

	t.lastUpdate = now
	t.totalTime = now.Sub(t.start).Seconds()
}

// Sleep pauses execution for the given number of seconds.
func (t *Time) Sleep(seconds float64) {
	time.Sleep(time.Duration(seconds * float64(time.Second)))
}
