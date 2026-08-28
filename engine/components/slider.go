package components

import (
	stdmath "math"

	"github.com/EnesBaytekin/imge/core"
	"github.com/EnesBaytekin/imge/core/math"
)

// Slider is a continuous control the user adjusts by pressing and dragging the
// thumb along a track: a horizontal bar with a grab handle, back by the classic
// "min/max range" idea. It maps the thumb's position to a numeric value between
// Min and Max, so a handler elsewhere can read that value (via the emitted event
// or GetValue()) and use it — e.g. a speed slider driving a character's velocity.
//
// Value is always a float64, quantized by Step (no separate int/float flag):
//   - Step = 1     -> whole numbers only (the common "integer slider"),
//   - Step = 0.5   -> halves,
//   - Step = 0     -> continuous (any float in [Min, Max]).
// Snapping is exact: with Step >= 1 the result is a clean integer; with a fractional
// Step the result is rounded to Step's own decimal places so 0.3 stays 0.3 and not
// 0.30000000000000004.
//
// Appearance: the track is a flat color, or a nine-sliced texture (TrackTexture +
// TrackBorder) so a styled bar stretches; the filled portion from Min up to the
// current value draws in FillColor; the thumb is a circle, or a texture
// (ThumbTexture, scaled to ThumbSize). Thickness sets the track height (0 = the
// full element height), and PadLeft/PadRight inset the track from the element's
// edges so the thumb stays inside (they default to half the thumb size).
//
// Interaction: press anywhere on the slider to grab it, then drag to adjust. Each
// change emits Event (if non-empty) with the slider itself as the data, so a
// handler receives both which slider (GetName()/GetOwner()) and what value
// (GetValue()) — the same convention as @Button. A @UIManager owns the gesture
// (BeginAdjust/Adjust/EndAdjust), like it does for buttons and text inputs.
//
// Export variables (JSON args): min, max, value, step, event, thickness,
// thumb_size, pad_left, pad_right, track_color, fill_color, thumb_color,
// track_texture, track_border {left, top, right, bottom}, thumb_texture, offset,
// width, height, visible, enabled, blocking, group, draw_layer.
type SliderComponent struct {
	core.BaseUIComponent

	// Min/Max bound the value. An unset range defaults to 0..100; a degenerate or
	// inverted range (Max <= Min) is repaired to Min..Min+1 in Initialize.
	Min float64 `json:"min"`
	Max float64 `json:"max"`

	// Value is the current value, always clamped to [Min, Max] and quantized by Step.
	Value float64 `json:"value"`

	// Step is the snap increment. 0 = continuous; otherwise Value lands on whole
	// multiples of Step within [Min, Max].
	Step float64 `json:"step"`

	// Event is emitted on every user-driven change, with the slider as the data.
	Event string `json:"event"`

	// Thickness is the track height. nil (default) = 4; 0 = the full element height.
	Thickness *float64 `json:"thickness"`

	// ThumbSize is the thumb diameter (and the thumb texture's target size).
	// 0 (default) = 14.
	ThumbSize float64 `json:"thumb_size"`

	// PadLeft/PadRight inset the track from the element's left/right edge. nil
	// (default) = half the thumb size, so the thumb stays inside the element at the
	// extremes.
	PadLeft  *float64 `json:"pad_left"`
	PadRight *float64 `json:"pad_right"`

	// TrackColor/FillColor/ThumbColor are the flat fallbacks when no texture is set.
	TrackColor math.Color `json:"track_color"`
	FillColor  math.Color `json:"fill_color"`
	ThumbColor math.Color `json:"thumb_color"`

	// TrackTexture, when set, is nine-sliced with TrackBorder over the track.
	TrackTexture string      `json:"track_texture"`
	TrackBorder  math.Border `json:"track_border"`

	// ThumbTexture, when set, is drawn (scaled to ThumbSize) instead of a circle.
	ThumbTexture string `json:"thumb_texture"`

	dragging bool
}

// Initialize defaults the range, thumb size, colors, and blocking flag, then
// clamps+snaps the configured Value into range.
func (s *SliderComponent) Initialize() {
	if s.Max <= s.Min {
		if s.Min == 0 && s.Max == 0 {
			s.Max = 100
		} else {
			s.Max = s.Min + 1
		}
	}
	if s.ThumbSize <= 0 {
		s.ThumbSize = 14
	}
	if s.TrackColor == (math.Color{}) {
		s.TrackColor = math.DarkGray
	}
	if s.FillColor == (math.Color{}) {
		s.FillColor = math.NewColor(90, 158, 90, 255)
	}
	if s.ThumbColor == (math.Color{}) {
		s.ThumbColor = math.White
	}
	if s.Blocking == nil {
		b := true
		s.Blocking = &b
	}
	s.SetValue(s.Value) // clamp + snap the configured value without emitting
}

// GetValue returns the current (already snapped) value.
func (s *SliderComponent) GetValue() float64 { return s.Value }

// SetValue sets the value, snapping and clamping it to the configured range. It is
// silent — it does not emit Event (that is reserved for user-driven changes).
func (s *SliderComponent) SetValue(v float64) {
	s.Value = s.clamp(s.snap(v))
}

// BeginAdjust starts a drag gesture, snapping the value to the pointer's position.
// Called by a @UIManager on press.
func (s *SliderComponent) BeginAdjust(pos math.Vector2) {
	if !s.IsEnabled() {
		return
	}
	s.dragging = true
	s.Adjust(pos)
}

// Adjust maps a pointer position to the value and, if it changed, emits Event.
// Called by a @UIManager on every held frame.
func (s *SliderComponent) Adjust(pos math.Vector2) {
	if !s.IsEnabled() {
		return
	}
	v := s.valueFromX(pos.X)
	if v == s.Value {
		return
	}
	s.Value = v
	if s.Event != "" {
		s.Emit(s.Event, s)
	}
}

// EndAdjust ends the drag gesture. Called by a @UIManager on release.
func (s *SliderComponent) EndAdjust() {
	s.dragging = false
}

// snap quantizes v to the nearest Step multiple and, for a fractional Step, rounds
// to Step's own decimal places so repeated values stay clean.
func (s *SliderComponent) snap(v float64) float64 {
	if s.Step <= 0 {
		return v
	}
	v = stdmath.Round(v/s.Step) * s.Step
	if dec := stepDecimals(s.Step); dec > 0 {
		p := stdmath.Pow(10, float64(dec))
		v = stdmath.Round(v*p) / p
	}
	return v
}

// clamp bounds v to [Min, Max].
func (s *SliderComponent) clamp(v float64) float64 {
	if v < s.Min {
		return s.Min
	}
	if v > s.Max {
		return s.Max
	}
	return v
}

// fraction returns Value's position in [0, 1] across [Min, Max].
func (s *SliderComponent) fraction() float64 {
	span := s.Max - s.Min
	if span <= 0 {
		return 0
	}
	f := (s.Value - s.Min) / span
	if f < 0 {
		f = 0
	}
	if f > 1 {
		f = 1
	}
	return f
}

// valueFromX maps a pointer X to the snapped, clamped value for that position.
func (s *SliderComponent) valueFromX(x float64) float64 {
	tr := s.track()
	f := 0.0
	if tr.Width() > 0 {
		f = (x - tr.Left()) / tr.Width()
		if f < 0 {
			f = 0
		}
		if f > 1 {
			f = 1
		}
	}
	return s.snap(s.Min + f*(s.Max-s.Min))
}

// padLeft/padRight return the track's horizontal inset, defaulting to half the
// thumb size so the thumb stays inside the element at the extremes.
func (s *SliderComponent) padLeft() float64 {
	if s.PadLeft != nil {
		return *s.PadLeft
	}
	return s.ThumbSize / 2
}

func (s *SliderComponent) padRight() float64 {
	if s.PadRight != nil {
		return *s.PadRight
	}
	return s.ThumbSize / 2
}

// track returns the track rectangle: the element rect inset by the pads, vertically
// centered at the given thickness.
func (s *SliderComponent) track() math.Rect {
	rect := s.Rect()
	left := rect.Left() + s.padLeft()
	right := rect.Right() - s.padRight()
	if right < left {
		right = left
	}
	cy := rect.Center().Y
	th := s.trackThickness(rect)
	return math.NewRect(left, cy-th/2, right-left, th)
}

// trackThickness returns the track height for the given element rect.
func (s *SliderComponent) trackThickness(rect math.Rect) float64 {
	if s.Thickness == nil {
		return 4
	}
	if *s.Thickness <= 0 {
		return rect.Height()
	}
	return *s.Thickness
}

// thumbScale is a modest size bump while dragging, for tactile feedback.
func (s *SliderComponent) thumbScale() float64 {
	if s.dragging {
		return 1.2
	}
	return 1.0
}

func (s *SliderComponent) Draw(r core.Renderer) {
	if !s.IsVisible() {
		return
	}
	tr := s.track()

	if s.TrackTexture != "" {
		core.DrawNineSlice(r, s.TrackTexture, s.TrackBorder, tr)
	} else {
		r.DrawRect(tr, s.TrackColor)
	}

	// Filled portion from Min up to the current value.
	fillW := tr.Width() * s.fraction()
	if fillW > 0 {
		r.DrawRect(math.NewRect(tr.Left(), tr.Top(), fillW, tr.Height()), s.FillColor)
	}

	center := math.NewVector2(tr.Left()+fillW, tr.Center().Y)
	size := s.ThumbSize * s.thumbScale()
	if s.ThumbTexture != "" {
		s.drawThumbTexture(r, center, size)
	} else {
		r.DrawCircle(center, size/2, s.ThumbColor)
	}
}

// drawThumbTexture draws the thumb texture scaled to `size` (its diameter), centered
// on center.
func (s *SliderComponent) drawThumbTexture(r core.Renderer, center math.Vector2, size float64) {
	tw, th := r.GetTextureSize(s.ThumbTexture)
	if tw <= 0 || th <= 0 {
		return
	}
	pos := math.NewVector2(center.X-size/2, center.Y-size/2)
	r.DrawTexture(s.ThumbTexture, math.Rect{}, pos, math.NewVector2(size/tw, size/th), 0, math.ColorTransform{})
}

// stepDecimals returns how many decimal places Step has (so its multiples round
// cleanly), or 0 for an integer Step. Fractional steps beyond 6 decimals are capped
// at 6.
func stepDecimals(step float64) int {
	if step == stdmath.Trunc(step) {
		return 0
	}
	for i := 1; i <= 6; i++ {
		p := stdmath.Pow(10, float64(i))
		if stdmath.Abs(step*p-stdmath.Round(step*p)) < 1e-9 {
			return i
		}
	}
	return 6
}
