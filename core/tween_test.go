package core

import (
	"math"
	"testing"
)

func TestTweenLinearInterpolation(t *testing.T) {
	var got []float64
	tw := &Tween{From: 0, To: 10, Duration: 1, OnUpdate: func(v float64) { got = append(got, v) }}

	tw.Update(0.25)
	tw.Update(0.25)
	tw.Update(0.25)
	tw.Update(0.25) // completes at elapsed == duration

	if len(got) != 4 {
		t.Fatalf("expected 4 updates, got %d", len(got))
	}
	want := []float64{2.5, 5, 7.5, 10}
	for i, w := range want {
		if math.Abs(got[i]-w) > 1e-9 {
			t.Fatalf("update %d: expected %v, got %v", i, w, got[i])
		}
	}
	if !tw.Done() {
		t.Fatalf("expected tween to be done")
	}

	// After completion, further updates are no-ops.
	tw.Update(1)
	if len(got) != 4 {
		t.Fatalf("expected no updates after completion, got %d total", len(got))
	}
}

func TestTweenClampsAndFiresOnDone(t *testing.T) {
	var last float64
	done := false
	tw := &Tween{
		From: 5, To: 15, Duration: 1,
		OnUpdate: func(v float64) { last = v },
		OnDone:   func() { done = true },
	}

	tw.Update(10) // overshoots duration
	if last != 15 {
		t.Fatalf("expected clamped value 15, got %v", last)
	}
	if !done {
		t.Fatalf("expected OnDone to fire")
	}
	if !tw.Done() {
		t.Fatalf("expected Done to report true")
	}
}

func TestTweenZeroDurationSnaps(t *testing.T) {
	var last float64
	done := false
	tw := &Tween{
		From: 1, To: 9, Duration: 0,
		OnUpdate: func(v float64) { last = v },
		OnDone:   func() { done = true },
	}

	tw.Update(0.016)
	if last != 9 {
		t.Fatalf("expected immediate snap to To, got %v", last)
	}
	if !done || !tw.Done() {
		t.Fatalf("expected zero-duration tween to complete on first update")
	}
}

func TestTweenEasing(t *testing.T) {
	values := []float64{}
	tw2 := &Tween{From: 0, To: 100, Duration: 1, Easing: EaseOutQuad, OnUpdate: func(v float64) { values = append(values, v) }}
	tw2.Update(0.5)
	tw2.Update(0.5)
	if len(values) != 2 {
		t.Fatalf("expected 2 updates, got %d", len(values))
	}
	// EaseOutQuad(0.5) = 0.75.
	if math.Abs(values[0]-75) > 1e-9 {
		t.Fatalf("EaseOutQuad midpoint: expected 75, got %v", values[0])
	}
	if values[1] != 100 {
		t.Fatalf("expected final value 100, got %v", values[1])
	}
}

func TestTweenReset(t *testing.T) {
	var got []float64
	tw := &Tween{From: 0, To: 4, Duration: 1, OnUpdate: func(v float64) { got = append(got, v) }}

	tw.Update(1) // complete
	tw.Reset()
	tw.Update(0.5)

	if tw.Done() {
		t.Fatalf("expected reset tween to be in-flight")
	}
	if len(got) != 2 {
		t.Fatalf("expected one value per update (2 total), got %d", len(got))
	}
	// After reset, the clock restarts at From=0.
	if math.Abs(got[1]-2) > 1e-9 {
		t.Fatalf("expected 2 after half of a reset tween, got %v", got[1])
	}
}

func TestEasingEndpoints(t *testing.T) {
	easings := []Easing{EaseLinear, EaseInQuad, EaseOutQuad, EaseInOutQuad, EaseOutCubic, EaseInOutCubic}
	for _, e := range easings {
		if e(0) != 0 {
			t.Fatalf("easing should map 0 -> 0")
		}
		if e(1) != 1 {
			t.Fatalf("easing should map 1 -> 1")
		}
	}
}

func TestCameraZoomTo(t *testing.T) {
	cam := NewCamera()
	cam.ZoomTo(4, 2, EaseLinear)

	// Two one-second steps reach the target exactly.
	cam.updateZoom(1)
	if cam.Zoom == 1 {
		t.Fatalf("expected zoom to have moved after one second")
	}
	cam.updateZoom(1)
	if cam.Zoom != 4 {
		t.Fatalf("expected zoom 4 after 2s, got %v", cam.Zoom)
	}
	if cam.zoomTween != nil {
		t.Fatalf("expected zoom tween to be cleared on completion")
	}
}

func TestCameraZoomToSnaps(t *testing.T) {
	cam := NewCamera()
	cam.ZoomTo(3, 0)
	if cam.Zoom != 3 {
		t.Fatalf("expected immediate zoom 3, got %v", cam.Zoom)
	}
	if cam.zoomTween != nil {
		t.Fatalf("expected no tween for zero duration")
	}
}
