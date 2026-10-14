// Package core contains platform-agnostic game engine logic.
// This file defines the Tween primitive: a small time-based value interpolator for
// easing a float (a score counter, a camera zoom, an alpha fade, …) toward a target.
package core

// Easing maps normalized time t (0..1) to an eased progress (0..1). A linear easing
// returns t unchanged; the quad/cubic variants accelerate and/or decelerate the
// transition for a more natural feel. It is shared by Tween and Camera.ZoomTo.
type Easing func(t float64) float64

// EaseLinear is the identity easing: constant speed from start to finish.
func EaseLinear(t float64) float64 { return t }

// EaseInQuad accelerates from rest (slow start, fast end).
func EaseInQuad(t float64) float64 { return t * t }

// EaseOutQuad decelerates to rest (fast start, slow end).
func EaseOutQuad(t float64) float64 { return t * (2 - t) }

// EaseInOutQuad accelerates then decelerates (slow at both ends).
func EaseInOutQuad(t float64) float64 {
	if t < 0.5 {
		return 2 * t * t
	}
	return 1 - 2*(t-1)*(t-1)
}

// EaseOutCubic is a gentler deceleration than EaseOutQuad — a good default for
// camera zooms and score counters.
func EaseOutCubic(t float64) float64 {
	u := 1 - t
	return 1 - u*u*u
}

// EaseInOutCubic accelerates then decelerates with a steeper curve than EaseInOutQuad.
func EaseInOutCubic(t float64) float64 {
	if t < 0.5 {
		return 4 * t * t * t
	}
	u := -2*t + 2
	return 1 - u*u*u/2
}

// Tween interpolates a float64 from From to To over Duration seconds, calling
// OnUpdate each frame with the current value and OnDone once when it completes.
// It is a plain value the owning component drives: hold it on the component and call
// tween.Update(ctx.DeltaTime()) in that component's Update. A nil Easing is linear.
type Tween struct {
	From, To float64
	Duration float64
	Elapsed  float64
	Easing   Easing
	OnUpdate func(value float64)
	OnDone   func()

	done bool
}

// Update advances the tween by dt seconds and fires OnUpdate with the interpolated
// value. When elapsed reaches Duration it clamps to To, fires OnDone, and becomes a
// no-op afterward. A tween with Duration <= 0 completes on its first Update.
func (t *Tween) Update(dt float64) {
	if t.done {
		return
	}
	if t.Duration <= 0 {
		t.done = true
		if t.OnUpdate != nil {
			t.OnUpdate(t.To)
		}
		if t.OnDone != nil {
			t.OnDone()
		}
		return
	}

	t.Elapsed += dt
	p := t.Elapsed / t.Duration
	if p >= 1 {
		p = 1
		t.done = true
	}
	t.apply(p)
	if t.done && t.OnDone != nil {
		t.OnDone()
	}
}

// apply interpolates toward p (0..1) and reports it via OnUpdate.
func (t *Tween) apply(p float64) {
	ease := t.Easing
	if ease == nil {
		ease = EaseLinear
	}
	if t.OnUpdate != nil {
		t.OnUpdate(t.From + (t.To-t.From)*ease(p))
	}
}

// Done reports whether the tween has finished (including the zero-duration case).
func (t *Tween) Done() bool { return t.done }

// Reset restarts the tween from its From value with a fresh clock, for reuse.
func (t *Tween) Reset() {
	t.Elapsed = 0
	t.done = false
}
