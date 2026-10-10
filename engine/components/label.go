package components

import (
	"github.com/EnesBaytekin/imge/core"
	"github.com/EnesBaytekin/imge/core/math"
)

// Label draws text at the owner's position plus its offset. It is the simplest UI
// element: no interaction, no background. Text is left-aligned (alignment is
// deferred to a later phase).
//
// max_width > 0 wraps the text to that width using the given wrap mode; otherwise
// the text draws as a single line.
//
// Export variables (JSON args): text, font_id, size, color, max_width, wrap,
// offset, width, height, visible, group, draw_layer.
type LabelComponent struct {
	core.BaseUIComponent

	Text   string  `json:"text"`
	FontID string  `json:"font_id"` // "" = built-in pixel font
	Size   float64 `json:"size"`    // 0 = font default

	Color math.Color `json:"color"`

	// MaxWidth > 0 wraps text to this width (in logical units); 0 = single line.
	MaxWidth float64       `json:"max_width"`
	Wrap     core.WrapMode `json:"wrap"`
}

// Initialize defaults the text color to white when not specified.
func (l *LabelComponent) Initialize() {
	if l.Color == (math.Color{}) {
		l.Color = math.White
	}
}

func (l *LabelComponent) Draw(r core.Renderer) {
	if !l.IsVisible() {
		return
	}
	pos := l.Position()
	if l.MaxWidth > 0 {
		r.DrawTextWrapped(l.Text, l.FontID, l.Size, l.MaxWidth, l.Wrap, false, pos, l.Color)
		return
	}
	r.DrawText(l.Text, l.FontID, l.Size, pos, l.Color)
}

// MeasuredBounds reports the label's on-screen rectangle for editor hit-testing. An
// explicit Width×Height (the BaseUIComponent size) is used when set; otherwise the
// bounds is the rendered text extent measured via the renderer, so a label that only
// declares an offset and text is still selectable by the area its text actually draws.
func (l *LabelComponent) MeasuredBounds(r core.Renderer) math.Rect {
	if l.Width > 0 && l.Height > 0 {
		return l.Rect()
	}
	if r == nil {
		return l.Rect()
	}
	pos := l.Position()
	if l.MaxWidth > 0 {
		w, h := r.MeasureTextWrapped(l.Text, l.FontID, l.Size, l.MaxWidth, l.Wrap, false)
		return math.NewRect(pos.X, pos.Y, w, h)
	}
	w, h := r.MeasureText(l.Text, l.FontID, l.Size)
	return math.NewRect(pos.X, pos.Y, w, h)
}
