package components

import (
	"strings"
	"unicode/utf8"

	"github.com/EnesBaytekin/imge/core"
	"github.com/EnesBaytekin/imge/core/math"
)

// TextInput is a single-line editable text field. Click to focus (blinking caret,
// drawn as the font's "|" glyph so its height matches the text), type to insert
// characters at the caret, Left/Right move the caret, Backspace/Delete delete the
// rune before/after it, Home/End jump to the start/end, Ctrl+Left/Right jump word
// by word (Ctrl+Left to a word's start, Ctrl+Right to its end), Ctrl+Backspace/
// Ctrl+Delete delete a whole word, and Enter submits. Holding a key auto-repeats
// (one action on press, then a pause, then rapid repeats) for arrows, Backspace,
// Delete, Home/End, and printable characters alike — the classic textbox behavior.
//
// When the text grows wider than the box it scrolls horizontally, but the text only
// moves when the caret leaves the visible window: while the caret is inside the box
// the text stays put, and it scrolls left/right only when the caret crosses an edge.
//
// Background: a flat color, or a nine-sliced texture when texture + border are set.
// Placeholder draws (in placeholder_color) when empty and not focused.
//
// Export variables (JSON args): text, font_id, size, text_color, placeholder_color,
// background_color, texture, border {left, top, right, bottom}, placeholder,
// padding_left, padding_right, max_length, event, offset, width, height, visible,
// enabled, focusable, group, draw_layer.
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

	// PaddingLeft/Right inset the text from the box's left/right edge (logical
	// units). nil means the default (2). Set them to match a nine-slice border so
	// the text starts inside the frame.
	PaddingLeft  *float64 `json:"padding_left"`
	PaddingRight *float64 `json:"padding_right"`

	focused   bool
	showCaret bool
	blink     float64

	// caret is the rune index (0..len) where input is inserted/deleted and the
	// caret is drawn. scroll is the rune index of the first visible rune; it only
	// moves when the caret leaves the visible window, so the text stays put while
	// the caret moves inside the box.
	caret  int
	scroll int

	// Key auto-repeat state for the control keys (arrows/backspace/home/end):
	// holdKey is the key currently held, holdTime how long it has been held, and
	// holdPhase 0 when no key is held, 1 during the initial repeat delay, 2 while
	// repeating rapidly.
	holdKey   core.KeyCode
	holdTime  float64
	holdPhase int

	// Character auto-repeat state: lastChar repeats while its key is held.
	lastChar     rune
	charHoldTime float64
	charPhase    int
}

const defaultInputPadX = 2.0

// Key auto-repeat timing: one action on the initial press, then a pause, then rapid
// repeats while held (the OS textbox rhythm).
const (
	keyRepeatDelay = 0.5
	keyRepeatRate  = 0.05
)

// Initialize marks the input focusable and blocking, and defaults its text/placeholder
// colors.
func (t *TextInputComponent) Initialize() {
	t.Focusable = true
	if t.Blocking == nil {
		b := true
		t.Blocking = &b
	}
	if t.TextColor == (math.Color{}) {
		t.TextColor = math.White
	}
	if t.PlaceholderColor == (math.Color{}) {
		t.PlaceholderColor = math.Gray
	}
}

// SetFocused sets the focus state, driven by a @UIManager. Losing focus clears the
// key/character auto-repeat state.
func (t *TextInputComponent) SetFocused(focused bool) {
	t.focused = focused
	if !focused {
		t.resetRepeat()
	}
}

// HandleInput processes keyboard input while focused. Called each frame by a
// @UIManager when this input holds focus. The text input no longer reads input
// itself; the manager owns focus and routes key/character input here.
func (t *TextInputComponent) HandleInput(ctx *core.Context) {
	if !t.focused || !t.IsVisible() || !t.IsEnabled() {
		t.focused = false
		t.resetRepeat()
		return
	}

	// Blink the caret (~2 Hz).
	t.blink += ctx.DeltaTime()
	t.showCaret = int(t.blink*2)%2 == 0

	t.updateControlKeys(ctx)
	t.updateTyping(ctx)

	if ctx.Input.IsKeyJustPressed(core.KeyEnter) {
		if t.Event != "" {
			t.Emit(t.Event, t.Text)
		}
	}
}

// padLeft/padRight return the horizontal inset for the text, defaulting to
// defaultInputPadX when not configured.
func (t *TextInputComponent) padLeft() float64 {
	if t.PaddingLeft != nil {
		return *t.PaddingLeft
	}
	return defaultInputPadX
}

func (t *TextInputComponent) padRight() float64 {
	if t.PaddingRight != nil {
		return *t.PaddingRight
	}
	return defaultInputPadX
}

// resetRepeat clears the key/character auto-repeat state (on blur or disable).
func (t *TextInputComponent) resetRepeat() {
	t.holdKey = core.KeyUnknown
	t.holdTime = 0
	t.holdPhase = 0
	t.lastChar = 0
	t.charHoldTime = 0
	t.charPhase = 0
}

// updateControlKeys handles the held control keys with OS-style auto-repeat: an
// immediate action on the first held frame, then a pause, then rapid repeats.
func (t *TextInputComponent) updateControlKeys(ctx *core.Context) {
	key := t.heldControlKey(ctx.Input)
	dt := ctx.DeltaTime()
	ctrl := ctx.Input.IsKeyPressed(core.KeyControl)

	if key != t.holdKey {
		t.holdKey = key
		t.holdTime = 0
		t.holdPhase = 0
		if key != core.KeyUnknown {
			t.doControlKey(key, ctrl) // immediate first action
			t.holdPhase = 1
		}
		return
	}
	if key == core.KeyUnknown {
		return
	}

	t.holdTime += dt
	switch t.holdPhase {
	case 1: // waiting out the initial repeat delay
		if t.holdTime >= keyRepeatDelay {
			t.doControlKey(key, ctrl)
			t.holdTime -= keyRepeatDelay
			t.holdPhase = 2
		}
	case 2: // rapid repeat
		for t.holdTime >= keyRepeatRate {
			t.doControlKey(key, ctrl)
			t.holdTime -= keyRepeatRate
		}
	}
}

// heldControlKey returns the control key currently held, or KeyUnknown. Ordering
// matters only if several are held at once; the first match wins.
func (t *TextInputComponent) heldControlKey(in core.Input) core.KeyCode {
	switch {
	case in.IsKeyPressed(core.KeyLeft):
		return core.KeyLeft
	case in.IsKeyPressed(core.KeyRight):
		return core.KeyRight
	case in.IsKeyPressed(core.KeyBackspace):
		return core.KeyBackspace
	case in.IsKeyPressed(core.KeyDelete):
		return core.KeyDelete
	case in.IsKeyPressed(core.KeyHome):
		return core.KeyHome
	case in.IsKeyPressed(core.KeyEnd):
		return core.KeyEnd
	}
	return core.KeyUnknown
}

// doControlKey applies one control-key action. ctrl is whether Control is held,
// which turns Left/Right into word jumps and Backspace/Delete into word deletes.
func (t *TextInputComponent) doControlKey(key core.KeyCode, ctrl bool) {
	switch key {
	case core.KeyLeft:
		if ctrl {
			t.moveCaretWord(-1)
		} else {
			t.moveCaret(-1)
		}
	case core.KeyRight:
		if ctrl {
			t.moveCaretWord(+1)
		} else {
			t.moveCaret(+1)
		}
	case core.KeyBackspace:
		if ctrl {
			t.deleteWordBeforeCaret()
		} else {
			t.deleteRuneBeforeCaret()
		}
	case core.KeyDelete:
		if ctrl {
			t.deleteWordAfterCaret()
		} else {
			t.deleteRuneAfterCaret()
		}
	case core.KeyHome:
		t.caret = 0
	case core.KeyEnd:
		t.caret = utf8.RuneCountInString(t.Text)
	}
}

// updateTyping inserts freshly typed characters and auto-repeats the last one while
// its key stays held.
func (t *TextInputComponent) updateTyping(ctx *core.Context) {
	dt := ctx.DeltaTime()
	chars := ctx.Input.InputChars()
	if len(chars) > 0 {
		for _, r := range chars {
			t.appendRune(r)
		}
		t.lastChar = chars[len(chars)-1]
		t.charHoldTime = 0
		t.charPhase = 1
		return
	}

	if t.lastChar == 0 {
		return
	}
	key, ok := charKeyCode(t.lastChar)
	if !ok || !ctx.Input.IsKeyPressed(key) {
		t.lastChar = 0
		t.charPhase = 0
		return
	}

	t.charHoldTime += dt
	switch t.charPhase {
	case 1:
		if t.charHoldTime >= keyRepeatDelay {
			t.appendRune(t.lastChar)
			t.charHoldTime -= keyRepeatDelay
			t.charPhase = 2
		}
	case 2:
		for t.charHoldTime >= keyRepeatRate {
			t.appendRune(t.lastChar)
			t.charHoldTime -= keyRepeatRate
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

// moveCaretWord moves the caret word by word. delta > 0 jumps to the end of the word
// to the right (so a following space stays right of the caret); delta < 0 jumps to the
// start of the word to the left (so a preceding space stays left of the caret).
func (t *TextInputComponent) moveCaretWord(delta int) {
	if delta > 0 {
		t.caret = t.wordEnd(t.caret)
	} else {
		t.caret = t.wordStart(t.caret)
	}
}

// wordStart returns the index of the start of the word to the left of i (skipping any
// spaces between), the target of Ctrl+Left and Ctrl+Backspace. Spaces before that word
// are preserved.
func (t *TextInputComponent) wordStart(i int) int {
	runes := []rune(t.Text)
	for i > 0 && isSpaceRune(runes[i-1]) {
		i--
	}
	for i > 0 && !isSpaceRune(runes[i-1]) {
		i--
	}
	return i
}

// wordEnd returns the index just past the end of the word to the right of i (skipping
// any spaces between), the target of Ctrl+Right and Ctrl+Delete. Spaces after that word
// are preserved.
func (t *TextInputComponent) wordEnd(i int) int {
	runes := []rune(t.Text)
	n := len(runes)
	for i < n && isSpaceRune(runes[i]) {
		i++
	}
	for i < n && !isSpaceRune(runes[i]) {
		i++
	}
	return i
}

func isSpaceRune(r rune) bool {
	return r == ' ' || r == '\t' || r == '\n'
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

// deleteRuneAfterCaret removes the rune after the caret (Delete), if any. The caret
// stays put.
func (t *TextInputComponent) deleteRuneAfterCaret() {
	t.moveCaret(0) // clamp caret to a valid range
	b := byteOffset(t.Text, t.caret)
	if b >= len(t.Text) {
		return
	}
	_, size := utf8.DecodeRuneInString(t.Text[b:])
	t.Text = t.Text[:b] + t.Text[b+size:]
}

// deleteRange removes the runes in [a, b) and moves the caret to a.
func (t *TextInputComponent) deleteRange(a, b int) {
	t.moveCaret(0) // clamp caret to a valid range
	n := utf8.RuneCountInString(t.Text)
	if a < 0 {
		a = 0
	}
	if b > n {
		b = n
	}
	if a >= b {
		return
	}
	ab := byteOffset(t.Text, a)
	bb := byteOffset(t.Text, b)
	t.Text = t.Text[:ab] + t.Text[bb:]
	t.caret = a
}

// deleteWordBeforeCaret removes the word ending at the caret (Ctrl+Backspace), leaving
// any spaces between the caret and the word. The caret moves to the word's start.
func (t *TextInputComponent) deleteWordBeforeCaret() {
	t.deleteRange(t.wordStart(t.caret), t.caret)
}

// deleteWordAfterCaret removes the word starting at the caret (Ctrl+Delete), leaving any
// spaces after it. The caret stays put.
func (t *TextInputComponent) deleteWordAfterCaret() {
	t.deleteRange(t.caret, t.wordEnd(t.caret))
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
	x := rect.X() + t.padLeft()
	y := rect.Y() + (rect.Height()-lineH)/2

	if t.Text == "" && !t.focused && t.Placeholder != "" {
		r.DrawText(t.Placeholder, t.FontID, t.Size, math.NewVector2(x, y), t.PlaceholderColor)
		return
	}

	measure := func(s string) float64 {
		w, _ := r.MeasureText(s, t.FontID, t.Size)
		return w
	}
	inner := rect.Width() - t.padLeft() - t.padRight()

	t.syncScroll(inner, measure)

	visible := t.visibleFromScroll(inner, measure)
	caretX := x + measure(sliceRunes(t.Text, t.scroll, t.caret))

	r.DrawText(visible, t.FontID, t.Size, math.NewVector2(x, y), t.TextColor)

	if t.focused && t.showCaret {
		t.drawCaret(r, caretX, y, rect)
	}
}

// drawCaret draws the blinking caret as the font's "|" glyph (centered on x) so its
// height matches the text instead of a full-height line. Falls back to a thin rectangle
// if the font has no "|".
func (t *TextInputComponent) drawCaret(r core.Renderer, x, y float64, rect math.Rect) {
	if cw, _ := r.MeasureText("|", t.FontID, t.Size); cw > 0 {
		r.DrawText("|", t.FontID, t.Size, math.NewVector2(x-cw/2, y), t.TextColor)
		return
	}
	r.DrawRect(math.NewRect(x, rect.Y()+2, 1, rect.Height()-4), t.TextColor)
}

// syncScroll keeps scroll such that the caret stays inside the visible window of
// innerWidth logical units. It changes scroll only when the caret crosses an edge
// (or the whole text shrinks to fit) — never while the caret moves within the box.
func (t *TextInputComponent) syncScroll(inner float64, measure func(string) float64) {
	t.moveCaret(0) // clamp caret to a valid range
	if inner <= 0 {
		return
	}
	// If the whole text fits there is nothing to scroll.
	if measure(t.Text) <= inner {
		t.scroll = 0
	}
	if t.caret < t.scroll {
		t.scroll = t.caret
	}
	for t.scroll < t.caret && measure(sliceRunes(t.Text, t.scroll, t.caret)) > inner {
		t.scroll++
	}
}

// visibleFromScroll returns the runes starting at scroll that fit within innerWidth
// (always at least one rune when scroll is valid).
func (t *TextInputComponent) visibleFromScroll(inner float64, measure func(string) float64) string {
	if inner <= 0 {
		return t.Text
	}
	n := utf8.RuneCountInString(t.Text)
	if t.scroll >= n {
		return ""
	}
	var sb strings.Builder
	for _, r := range sliceRunes(t.Text, t.scroll, n) {
		next := sb.String() + string(r)
		if sb.Len() > 0 && measure(next) > inner {
			break
		}
		sb.WriteRune(r)
	}
	return sb.String()
}

// charKeyCode maps an ASCII rune back to its key so a held character key can be
// detected for auto-repeat. Returns (KeyUnknown, false) for runes without a
// dedicated key (symbols, IME input), which simply do not auto-repeat.
func charKeyCode(r rune) (core.KeyCode, bool) {
	switch {
	case r >= 'a' && r <= 'z':
		return core.KeyA + core.KeyCode(r-'a'), true
	case r >= 'A' && r <= 'Z':
		return core.KeyA + core.KeyCode(r-'A'), true
	case r >= '0' && r <= '9':
		return core.Key0 + core.KeyCode(r-'0'), true
	case r == ' ':
		return core.KeySpace, true
	}
	return core.KeyUnknown, false
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
