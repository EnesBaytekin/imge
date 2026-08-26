package components

import (
	"strings"
	"unicode/utf8"

	"github.com/EnesBaytekin/imge/core"
	"github.com/EnesBaytekin/imge/core/math"
)

// TextInput is a single-line editable text field. Click to focus (blinking caret),
// type to insert characters at the caret, Left/Right move the caret, Backspace
// deletes the rune before it, Enter submits. When the text grows wider than the
// box it scrolls horizontally so the caret stays visible — the classic single-line
// textbox behavior.
//
// Background: a flat color, or a nine-sliced texture when texture + border are set.
// Placeholder draws (in placeholder_color) when empty and not focused.
//
// Export variables (JSON args): text, font_id, size, text_color, placeholder_color,
// background_color, texture, border {left, top, right, bottom}, placeholder,
// max_length, event, offset, width, height, visible, enabled, focusable, group,
// draw_layer.
type TextInputComponent struct {
	core.BaseUIComponent

	Text   string  `json:"text"`
	FontID string  `json:"font_id"`
	Size   float64 `json:"size"`

	TextColor        math.Color `json:"text_color"`
	PlaceholderColor math.Color `json:"placeholder_color"`
	BackgroundColor  math.Color `json:"background_color"`

	Texture string      `json:"texture"`
	Border  math.Border `json:"border"`

	Placeholder string `json:"placeholder"`
	MaxLength   int    `json:"max_length"`
	Event       string `json:"event"`

	focused   bool
	showCaret bool
	blink     float64

	// caret is the rune index (0..len) where input is inserted/deleted and the
	// caret is drawn.
	caret int
}

const inputPadX = 2.0

// Initialize marks the input focusable and defaults its text/placeholder colors.
func (t *TextInputComponent) Initialize() {
	t.Focusable = true
	if t.TextColor == (math.Color{}) {
		t.TextColor = math.White
	}
	if t.PlaceholderColor == (math.Color{}) {
		t.PlaceholderColor = math.Gray
	}
}

func (t *TextInputComponent) Update(ctx *core.Context) {
	if !t.IsVisible() || !t.IsEnabled() {
		t.focused = false
		return
	}

	if ctx.Input.IsMouseButtonJustPressed(core.MouseButtonLeft) {
		t.focused = t.Contains(ctx.Input.GetMousePosition())
	}
	if !t.focused {
		return
	}

	// Blink the caret (~2 Hz).
	t.blink += ctx.DeltaTime()
	t.showCaret = int(t.blink*2)%2 == 0

	if ctx.Input.IsKeyJustPressed(core.KeyLeft) {
		t.moveCaret(-1)
	}
	if ctx.Input.IsKeyJustPressed(core.KeyRight) {
		t.moveCaret(+1)
	}

	for _, r := range ctx.Input.InputChars() {
		t.appendRune(r)
	}
	if ctx.Input.IsKeyJustPressed(core.KeyBackspace) {
		t.deleteRuneBeforeCaret()
	}
	if ctx.Input.IsKeyJustPressed(core.KeyEnter) {
		if t.Event != "" {
			t.Emit(t.Event, t.Text)
		}
	}
}

// moveCaret shifts the caret by delta runes, clamped to [0, len].
func (t *TextInputComponent) moveCaret(delta int) {
	t.caret += delta
	if t.caret < 0 {
		t.caret = 0
	}
	if n := utf8.RuneCountInString(t.Text); t.caret > n {
		t.caret = n
	}
}

// appendRune inserts a rune at the caret, respecting MaxLength (0 = unlimited).
func (t *TextInputComponent) appendRune(r rune) {
	if t.MaxLength > 0 && utf8.RuneCountInString(t.Text) >= t.MaxLength {
		return
	}
	t.moveCaret(0) // clamp caret to a valid range
	b := byteOffset(t.Text, t.caret)
	t.Text = t.Text[:b] + string(r) + t.Text[b:]
	t.caret++
}

// deleteRuneBeforeCaret removes the rune before the caret (Backspace), if any.
func (t *TextInputComponent) deleteRuneBeforeCaret() {
	if t.caret <= 0 {
		return
	}
	t.moveCaret(0) // clamp caret to a valid range
	b := byteOffset(t.Text, t.caret)
	_, size := utf8.DecodeLastRuneInString(t.Text[:b])
	t.Text = t.Text[:b-size] + t.Text[b:]
	t.caret--
}

func (t *TextInputComponent) Draw(r core.Renderer) {
	if !t.IsVisible() {
		return
	}
	rect := t.Rect()
	if t.Texture != "" {
		core.DrawNineSlice(r, t.Texture, t.Border, rect)
	} else if t.BackgroundColor.A > 0 {
		r.DrawRect(rect, t.BackgroundColor)
	}

	_, lineH := r.MeasureText("M", t.FontID, t.Size)
	x := rect.X() + inputPadX
	y := rect.Y() + (rect.Height()-lineH)/2

	if t.Text == "" && !t.focused && t.Placeholder != "" {
		r.DrawText(t.Placeholder, t.FontID, t.Size, math.NewVector2(x, y), t.PlaceholderColor)
		return
	}

	measure := func(s string) float64 {
		w, _ := r.MeasureText(s, t.FontID, t.Size)
		return w
	}
	inner := rect.Width() - 2*inputPadX
	visible := t.Text
	caretX := x + measure(sliceRunes(t.Text, 0, t.caret))
	if inner > 0 {
		v, cx := t.visibleAndCaretX(inner, measure)
		visible = v
		caretX = x + cx
	}

	r.DrawText(visible, t.FontID, t.Size, math.NewVector2(x, y), t.TextColor)

	if t.focused && t.showCaret {
		r.DrawRect(math.NewRect(caretX, rect.Y()+2, 1, rect.Height()-4), t.TextColor)
	}
}

// visibleAndCaretX returns the substring to draw (scrolled so the caret stays
// visible within innerWidth) and the caret's x offset from the text origin.
func (t *TextInputComponent) visibleAndCaretX(innerWidth float64, measure func(string) float64) (string, float64) {
	n := utf8.RuneCountInString(t.Text)
	t.moveCaret(0) // clamp caret to a valid range

	// Advance the scroll start until the caret fits inside innerWidth.
	scroll := 0
	for scroll < t.caret {
		if measure(sliceRunes(t.Text, scroll, t.caret)) <= innerWidth {
			break
		}
		scroll++
	}

	// Visible text: the runes from scroll that fit innerWidth (at least one rune).
	visible := t.Text
	if scroll > 0 || measure(t.Text) > innerWidth {
		var sb strings.Builder
		for _, r := range sliceRunes(t.Text, scroll, n) {
			next := sb.String() + string(r)
			if sb.Len() > 0 && measure(next) > innerWidth {
				break
			}
			sb.WriteRune(r)
		}
		visible = sb.String()
	}

	return visible, measure(sliceRunes(t.Text, scroll, t.caret))
}

// sliceRunes returns the substring from the a-th rune (inclusive) to the b-th rune
// (exclusive). Out-of-range indices are clamped.
func sliceRunes(s string, a, b int) string {
	n := utf8.RuneCountInString(s)
	if a < 0 {
		a = 0
	}
	if b > n {
		b = n
	}
	if a >= b {
		return ""
	}
	return s[byteOffset(s, a):byteOffset(s, b)]
}

// byteOffset returns the byte index of the runeIdx-th rune (0-based).
func byteOffset(s string, runeIdx int) int {
	b := 0
	for i := 0; i < runeIdx; i++ {
		_, size := utf8.DecodeRuneInString(s[b:])
		b += size
	}
	return b
}
