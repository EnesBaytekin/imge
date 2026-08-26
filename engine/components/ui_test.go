package components

import (
	"encoding/json"
	"testing"

	"github.com/EnesBaytekin/imge/core"
	"github.com/EnesBaytekin/imge/core/math"
)

// stubInput is a minimal, configurable core.Input for exercising UI components
// without a platform.
type stubInput struct {
	mouse            math.Vector2
	justPressedLeft  bool
	justReleasedLeft bool
	chars            []rune
	enter            bool
	held             map[core.KeyCode]bool // keys currently pressed
}

func (s *stubInput) IsKeyPressed(k core.KeyCode) bool                { return s.held[k] }
func (s *stubInput) IsKeyJustReleased(core.KeyCode) bool             { return false }
func (s *stubInput) IsMouseButtonPressed(core.MouseButton) bool      { return false }
func (s *stubInput) IsMouseButtonJustPressed(core.MouseButton) bool  { return s.justPressedLeft }
func (s *stubInput) IsMouseButtonJustReleased(core.MouseButton) bool { return s.justReleasedLeft }
func (s *stubInput) GetMousePosition() math.Vector2                  { return s.mouse }
func (s *stubInput) GetMouseDelta() math.Vector2                     { return math.Vector2{} }
func (s *stubInput) GetMouseScroll() math.Vector2                    { return math.Vector2{} }
func (s *stubInput) InputChars() []rune                              { return s.chars }
func (s *stubInput) Update()                                         {}

func (s *stubInput) IsKeyJustPressed(k core.KeyCode) bool {
	if k == core.KeyEnter {
		return s.enter
	}
	return false
}

// heldKeys builds a held-key set for stubInput.held.
func heldKeys(keys ...core.KeyCode) map[core.KeyCode]bool {
	m := make(map[core.KeyCode]bool, len(keys))
	for _, k := range keys {
		m[k] = true
	}
	return m
}

// stubTime is a minimal core.Time whose delta can be set per test.
type stubTime struct{ delta float64 }

func (t *stubTime) DeltaTime() float64 { return t.delta }
func (t *stubTime) TotalTime() float64 { return 0 }
func (t *stubTime) FPS() float64       { return 60 }
func (t *stubTime) Tick()              {}
func (t *stubTime) Sleep(float64)      {}

func newButtonInScene() (*core.Scene, *ButtonComponent) {
	scene := core.NewScene("test")
	obj := core.NewObject("btn")
	obj.Transform.Position = math.NewVector2(10, 20)
	btn := &ButtonComponent{}
	btn.Width = 100
	btn.Height = 40
	btn.Event = "clicked"
	btn.SetName("button")
	_ = obj.AddComponent(btn)
	_ = scene.AddObject(obj)
	return scene, btn
}

func newTextInputInScene() (*core.Scene, *TextInputComponent) {
	scene := core.NewScene("test")
	obj := core.NewObject("input")
	obj.Transform.Position = math.NewVector2(10, 20)
	ti := &TextInputComponent{}
	ti.Width = 100
	ti.Height = 24
	ti.SetName("input")
	_ = obj.AddComponent(ti)
	_ = scene.AddObject(obj)
	return scene, ti
}

func TestBaseUIComponentRect(t *testing.T) {
	obj := core.NewObject("w")
	obj.Transform.Position = math.NewVector2(100, 50)
	c := &core.BaseUIComponent{}
	c.SetOwner(obj)
	c.Offset = math.NewVector2(10, 5)
	c.Width = 80
	c.Height = 24

	want := math.NewRect(110, 55, 80, 24)
	if c.Rect() != want {
		t.Fatalf("Rect() = %v, want %v", c.Rect(), want)
	}
	if !c.Contains(math.NewVector2(120, 60)) {
		t.Fatal("expected to contain an inner point")
	}
	if c.Contains(math.NewVector2(0, 0)) {
		t.Fatal("did not expect to contain an outer point")
	}
	if !c.IsVisible() || !c.IsEnabled() {
		t.Fatal("nil Visible/Enabled must default to true")
	}
}

func TestButtonClickEmitsEvent(t *testing.T) {
	scene, btn := newButtonInScene()
	var clicked any
	btn.On("clicked", func(data any) { clicked = data })

	// First update initializes components and subscribes handlers.
	scene.Update(&core.Context{Input: &stubInput{}})

	// Hover + press inside.
	scene.Update(&core.Context{Input: &stubInput{mouse: math.NewVector2(50, 40), justPressedLeft: true}})
	if !btn.hovered || !btn.pressed {
		t.Fatalf("after press: hovered=%v pressed=%v, want both true", btn.hovered, btn.pressed)
	}

	// Release inside → click.
	scene.Update(&core.Context{Input: &stubInput{mouse: math.NewVector2(50, 40), justReleasedLeft: true}})
	if clicked != btn {
		t.Fatalf("click event data = %v, want the button", clicked)
	}
	if btn.pressed {
		t.Fatal("expected pressed cleared after release")
	}
}

func TestButtonReleaseOutsideDoesNotClick(t *testing.T) {
	scene, btn := newButtonInScene()
	var clicked any
	btn.On("clicked", func(data any) { clicked = data })
	scene.Update(&core.Context{Input: &stubInput{}})

	// Press inside, release outside.
	scene.Update(&core.Context{Input: &stubInput{mouse: math.NewVector2(50, 40), justPressedLeft: true}})
	scene.Update(&core.Context{Input: &stubInput{mouse: math.NewVector2(500, 500), justReleasedLeft: true}})
	if clicked != nil {
		t.Fatalf("release outside fired a click: %v", clicked)
	}
}

func TestButtonDisabledIgnoresInput(t *testing.T) {
	scene, btn := newButtonInScene()
	btn.SetEnabled(false)
	scene.Update(&core.Context{Input: &stubInput{}})

	scene.Update(&core.Context{Input: &stubInput{mouse: math.NewVector2(50, 40), justPressedLeft: true}})
	if btn.hovered || btn.pressed {
		t.Fatalf("disabled button reacted: hovered=%v pressed=%v", btn.hovered, btn.pressed)
	}
}

func TestTextInputAppendAndBackspace(t *testing.T) {
	ti := &TextInputComponent{}
	ti.appendRune('a')
	ti.appendRune('b')
	if ti.Text != "ab" {
		t.Fatalf("Text = %q, want %q", ti.Text, "ab")
	}
	ti.deleteRuneBeforeCaret()
	if ti.Text != "a" {
		t.Fatalf("Text = %q, want %q", ti.Text, "a")
	}
	ti.deleteRuneBeforeCaret()
	ti.deleteRuneBeforeCaret() // empty: must not panic
	if ti.Text != "" {
		t.Fatalf("Text = %q, want empty", ti.Text)
	}
}

func TestTextInputCaretInsertAndDelete(t *testing.T) {
	ti := &TextInputComponent{}
	ti.appendRune('a')
	ti.appendRune('b')
	ti.appendRune('c') // "abc", caret 3
	ti.caret = 1
	ti.appendRune('X') // "aXbc", caret 2
	if ti.Text != "aXbc" {
		t.Fatalf("Text = %q, want %q", ti.Text, "aXbc")
	}
	ti.deleteRuneBeforeCaret() // deletes 'X' → "abc"
	if ti.Text != "abc" {
		t.Fatalf("Text = %q, want %q", ti.Text, "abc")
	}
	if ti.caret != 1 {
		t.Fatalf("caret = %d, want 1", ti.caret)
	}
}

func TestTextInputArrowKeys(t *testing.T) {
	scene, ti := newTextInputInScene()
	ti.Text = "abc"
	ti.caret = 3
	scene.Update(&core.Context{Input: &stubInput{}})

	// Focus.
	scene.Update(&core.Context{Input: &stubInput{mouse: math.NewVector2(50, 30), justPressedLeft: true}})
	if !ti.focused {
		t.Fatal("expected focused after click inside")
	}

	// Left (held one frame) → caret 2.
	scene.Update(&core.Context{Input: &stubInput{mouse: math.NewVector2(50, 30), held: heldKeys(core.KeyLeft)}})
	if ti.caret != 2 {
		t.Fatalf("caret = %d, want 2", ti.caret)
	}
	// Right (held one frame) → caret 3.
	scene.Update(&core.Context{Input: &stubInput{mouse: math.NewVector2(50, 30), held: heldKeys(core.KeyRight)}})
	if ti.caret != 3 {
		t.Fatalf("caret = %d, want 3", ti.caret)
	}
	// Right at the end stays clamped.
	scene.Update(&core.Context{Input: &stubInput{mouse: math.NewVector2(50, 30), held: heldKeys(core.KeyRight)}})
	if ti.caret != 3 {
		t.Fatalf("caret = %d, want 3", ti.caret)
	}
}

func TestButtonDisabledTexture(t *testing.T) {
	scene, btn := newButtonInScene()
	btn.DisabledTexture = "assets/btn_disabled.png"
	btn.SetEnabled(false)
	scene.Update(&core.Context{Input: &stubInput{}})

	tex, explicit := btn.currentState()
	if tex != "assets/btn_disabled.png" || !explicit {
		t.Fatalf("currentState() = (%q, %v), want disabled texture, explicit", tex, explicit)
	}
	btn.SetEnabled(true)
	tex, explicit = btn.currentState()
	if tex != "" || explicit {
		t.Fatalf("currentState() = (%q, %v), want empty (no normal texture), non-explicit", tex, explicit)
	}
}

func TestTextInputMaxLength(t *testing.T) {
	ti := &TextInputComponent{MaxLength: 3}
	for _, r := range "hello" {
		ti.appendRune(r)
	}
	if ti.Text != "hel" {
		t.Fatalf("Text = %q, want %q", ti.Text, "hel")
	}
}

func TestTextInputFocusAndTyping(t *testing.T) {
	scene, ti := newTextInputInScene()
	scene.Update(&core.Context{Input: &stubInput{}})

	// Click inside → focus.
	scene.Update(&core.Context{Input: &stubInput{mouse: math.NewVector2(50, 30), justPressedLeft: true}})
	if !ti.focused {
		t.Fatal("expected focused after click inside")
	}

	// Type and backspace.
	scene.Update(&core.Context{Input: &stubInput{mouse: math.NewVector2(50, 30), chars: []rune{'h', 'i'}}})
	if ti.Text != "hi" {
		t.Fatalf("Text = %q, want %q", ti.Text, "hi")
	}
	scene.Update(&core.Context{Input: &stubInput{mouse: math.NewVector2(50, 30), held: heldKeys(core.KeyBackspace)}})
	if ti.Text != "h" {
		t.Fatalf("Text = %q, want %q", ti.Text, "h")
	}

	// Click outside → blur.
	scene.Update(&core.Context{Input: &stubInput{mouse: math.NewVector2(500, 500), justPressedLeft: true}})
	if ti.focused {
		t.Fatal("expected blurred after click outside")
	}
}

// TestUIComponentsDecodeJSONArgs exercises the exact JSON arg shape shown in the
// docs — hex colors, string wrap modes, and snake_case border keys — so a change to
// a color/wrap/border decode can't silently break scene loading.
func TestUIComponentsDecodeJSONArgs(t *testing.T) {
	decode := func(name string, args map[string]any, dst any) {
		t.Helper()
		data, err := json.Marshal(args)
		if err != nil {
			t.Fatalf("%s: marshal: %v", name, err)
		}
		if err := json.Unmarshal(data, dst); err != nil {
			t.Fatalf("%s: decode: %v", name, err)
		}
	}

	var label LabelComponent
	decode("@Label", map[string]any{
		"text": "Inventory", "size": 6.0, "color": "#ffffff",
		"offset":    map[string]any{"x": 20.0, "y": 18.0},
		"max_width": 108.0, "wrap": "word",
	}, &label)
	if label.Wrap != core.WrapWord {
		t.Errorf("@Label wrap = %v, want WrapWord", label.Wrap)
	}
	if label.Color != math.White {
		t.Errorf("@Label color = %v, want White", label.Color)
	}
	if label.Offset != (math.Vector2{X: 20, Y: 18}) {
		t.Errorf("@Label offset = %v", label.Offset)
	}

	var panel PanelComponent
	decode("@Panel", map[string]any{
		"color": "#14141e", "outline_color": "#3b3b4d", "outline_thickness": 1.0,
		"width": 240.0, "height": 232.0,
	}, &panel)
	if panel.Color != math.NewColor(0x14, 0x14, 0x1e, 255) {
		t.Errorf("@Panel color = %v", panel.Color)
	}
	if panel.OutlineThickness != 1 {
		t.Errorf("@Panel outline_thickness = %v", panel.OutlineThickness)
	}

	var btn ButtonComponent
	decode("@Button", map[string]any{
		"text": "OK", "color": "#2e7d32", "event": "login",
		"offset": map[string]any{"x": 20.0, "y": 176.0},
	}, &btn)
	if btn.Color != math.NewColor(0x2e, 0x7d, 0x32, 255) {
		t.Errorf("@Button color = %v", btn.Color)
	}

	var input TextInputComponent
	decode("@TextInput", map[string]any{
		"background_color": "#1e1e28", "placeholder": "name",
		"border": map[string]any{"left": 4.0, "top": 4.0, "right": 4.0, "bottom": 4.0},
	}, &input)
	if input.BackgroundColor != math.NewColor(0x1e, 0x1e, 0x28, 255) {
		t.Errorf("@TextInput background = %v", input.BackgroundColor)
	}
	if input.Border != (math.Border{Left: 4, Top: 4, Right: 4, Bottom: 4}) {
		t.Errorf("@TextInput border = %v", input.Border)
	}
}

// TestTextInputHeldKeyRepeat verifies the OS-style auto-repeat: one action on the
// initial held frame, then a pause, then rapid repeats while the key stays held.
func TestTextInputHeldKeyRepeat(t *testing.T) {
	scene, ti := newTextInputInScene()
	ti.Text = "abcdef"
	ti.caret = 6
	in := &stubInput{mouse: math.NewVector2(50, 30)}
	tm := &stubTime{}

	scene.Update(&core.Context{Input: in, Time: tm}) // initialize

	// Focus.
	in.justPressedLeft = true
	scene.Update(&core.Context{Input: in, Time: tm})
	in.justPressedLeft = false

	// Hold Left: immediate move on the first held frame.
	in.held = heldKeys(core.KeyLeft)
	tm.delta = 0.1
	scene.Update(&core.Context{Input: in, Time: tm})
	if ti.caret != 5 {
		t.Fatalf("immediate: caret = %d, want 5", ti.caret)
	}

	// Cross the initial repeat delay → one repeat.
	tm.delta = 0.5
	scene.Update(&core.Context{Input: in, Time: tm})
	if ti.caret != 4 {
		t.Fatalf("after delay: caret = %d, want 4", ti.caret)
	}

	// Now rapid: one repeat per rate interval.
	tm.delta = 0.05
	scene.Update(&core.Context{Input: in, Time: tm})
	if ti.caret != 3 {
		t.Fatalf("repeat: caret = %d, want 3", ti.caret)
	}

	// Release stops repeating.
	in.held = nil
	tm.delta = 0.05
	scene.Update(&core.Context{Input: in, Time: tm})
	if ti.caret != 3 {
		t.Fatalf("released: caret = %d, want 3", ti.caret)
	}
}

func TestTextInputHomeEnd(t *testing.T) {
	scene, ti := newTextInputInScene()
	ti.Text = "hello world"
	ti.caret = 5
	scene.Update(&core.Context{Input: &stubInput{}})
	scene.Update(&core.Context{Input: &stubInput{mouse: math.NewVector2(50, 30), justPressedLeft: true}})

	scene.Update(&core.Context{Input: &stubInput{mouse: math.NewVector2(50, 30), held: heldKeys(core.KeyEnd)}})
	if ti.caret != 11 {
		t.Fatalf("End: caret = %d, want 11", ti.caret)
	}
	scene.Update(&core.Context{Input: &stubInput{mouse: math.NewVector2(50, 30), held: heldKeys(core.KeyHome)}})
	if ti.caret != 0 {
		t.Fatalf("Home: caret = %d, want 0", ti.caret)
	}
}

func TestTextInputCtrlWordJump(t *testing.T) {
	scene, ti := newTextInputInScene()
	ti.Text = "one two three"
	scene.Update(&core.Context{Input: &stubInput{}})
	scene.Update(&core.Context{Input: &stubInput{mouse: math.NewVector2(50, 30), justPressedLeft: true}})

	// Ctrl+Right jumps to the start of the next word. Release between presses so
	// each is a fresh key press (holding the same key would auto-repeat instead).
	press := func(keys ...core.KeyCode) {
		scene.Update(&core.Context{Input: &stubInput{mouse: math.NewVector2(50, 30), held: heldKeys(keys...)}})
		scene.Update(&core.Context{Input: &stubInput{mouse: math.NewVector2(50, 30)}}) // release
	}

	press(core.KeyRight, core.KeyControl)
	if ti.caret != 4 {
		t.Fatalf("Ctrl+Right: caret = %d, want 4", ti.caret)
	}
	press(core.KeyRight, core.KeyControl)
	if ti.caret != 8 {
		t.Fatalf("Ctrl+Right: caret = %d, want 8", ti.caret)
	}
	press(core.KeyLeft, core.KeyControl)
	if ti.caret != 4 {
		t.Fatalf("Ctrl+Left: caret = %d, want 4", ti.caret)
	}
}

// TestTextInputScrollStaysPut verifies the scroll anchor: moving the caret within
// the visible window must not scroll the text; only crossing an edge scrolls it.
func TestTextInputScrollStaysPut(t *testing.T) {
	ti := &TextInputComponent{}
	ti.Text = "abcdefgh"
	measure := func(s string) float64 { return float64(len([]rune(s))) } // 1 unit/run

	// Box shows 3 runes; caret at the end scrolls to rune 5 ("fgh").
	ti.caret = 8
	ti.syncScroll(3, measure)
	if ti.scroll != 5 {
		t.Fatalf("scroll = %d, want 5", ti.scroll)
	}

	// Caret moves left within the window → text must stay put.
	for _, want := range []struct{ caret, scroll int }{{7, 5}, {6, 5}, {5, 5}} {
		ti.caret = want.caret
		ti.syncScroll(3, measure)
		if ti.scroll != want.scroll {
			t.Fatalf("caret=%d: scroll = %d, want %d (text stays put)", want.caret, ti.scroll, want.scroll)
		}
	}

	// Caret crosses the left edge → scroll left by one.
	ti.caret = 4
	ti.syncScroll(3, measure)
	if ti.scroll != 4 {
		t.Fatalf("caret=4 (left of window): scroll = %d, want 4", ti.scroll)
	}
}

func TestButtonStateTint(t *testing.T) {
	btn := &ButtonComponent{}
	btn.Color = math.NewColor(100, 100, 100, 255)

	if got := btn.stateColor(); got != btn.Color {
		t.Fatalf("normal stateColor = %v, want base %v", got, btn.Color)
	}

	btn.hovered = true
	if hover := btn.stateColor(); !(hover.R > btn.Color.R && hover.G > btn.Color.G && hover.B > btn.Color.B) {
		t.Fatalf("hover should brighten: %v vs base %v", hover, btn.Color)
	}

	btn.hovered = false
	btn.pressed = true
	if pressed := btn.stateColor(); !(pressed.R < btn.Color.R) {
		t.Fatalf("pressed should darken: %v vs base %v", pressed, btn.Color)
	}
	btn.pressed = false

	btn.SetEnabled(false)
	if dis := btn.stateColor(); dis.R != dis.G || dis.G != dis.B {
		t.Fatalf("disabled should desaturate to gray: %v", dis)
	} else if dis.R >= btn.Color.R {
		t.Fatalf("disabled should dim: %v vs base %v", dis, btn.Color)
	}
}

func TestButtonStateTransform(t *testing.T) {
	btn := &ButtonComponent{}

	if btn.stateTransform().Brightness != 0 {
		t.Fatal("normal transform should be identity")
	}
	btn.hovered = true
	if btn.stateTransform().Brightness <= 0 {
		t.Fatal("hover transform should brighten")
	}
	btn.hovered = false
	btn.pressed = true
	if btn.stateTransform().Brightness >= 0 {
		t.Fatal("pressed transform should darken")
	}
	btn.pressed = false
	btn.SetEnabled(false)
	if !btn.stateTransform().Grayscale {
		t.Fatal("disabled transform should desaturate")
	}
}

func TestButtonStateTextureFallbackTint(t *testing.T) {
	btn := &ButtonComponent{NormalTexture: "assets/btn.png"}
	btn.hovered = true // no hover texture → fall back to normal, non-explicit

	tex, explicit := btn.currentState()
	if tex != "assets/btn.png" || explicit {
		t.Fatalf("hover fallback: currentState = (%q, %v), want (normal, false)", tex, explicit)
	}

	btn.HoverTexture = "assets/btn_hover.png"
	tex, explicit = btn.currentState()
	if tex != "assets/btn_hover.png" || !explicit {
		t.Fatalf("explicit hover: currentState = (%q, %v), want (hover, true)", tex, explicit)
	}
}
