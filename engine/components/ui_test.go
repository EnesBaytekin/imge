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
	backspace        bool
	enter            bool
	left             bool
	right            bool
}

func (s *stubInput) IsKeyPressed(core.KeyCode) bool                 { return false }
func (s *stubInput) IsKeyJustReleased(core.KeyCode) bool            { return false }
func (s *stubInput) IsMouseButtonPressed(core.MouseButton) bool     { return false }
func (s *stubInput) IsMouseButtonJustPressed(core.MouseButton) bool { return s.justPressedLeft }
func (s *stubInput) IsMouseButtonJustReleased(core.MouseButton) bool {
	return s.justReleasedLeft
}
func (s *stubInput) GetMousePosition() math.Vector2 { return s.mouse }
func (s *stubInput) GetMouseDelta() math.Vector2    { return math.Vector2{} }
func (s *stubInput) GetMouseScroll() math.Vector2   { return math.Vector2{} }
func (s *stubInput) InputChars() []rune             { return s.chars }
func (s *stubInput) Update()                        {}

func (s *stubInput) IsKeyJustPressed(k core.KeyCode) bool {
	switch k {
	case core.KeyBackspace:
		return s.backspace
	case core.KeyEnter:
		return s.enter
	case core.KeyLeft:
		return s.left
	case core.KeyRight:
		return s.right
	}
	return false
}

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

	// Left → caret 2.
	scene.Update(&core.Context{Input: &stubInput{mouse: math.NewVector2(50, 30), left: true}})
	if ti.caret != 2 {
		t.Fatalf("caret = %d, want 2", ti.caret)
	}
	// Right → caret 3.
	scene.Update(&core.Context{Input: &stubInput{mouse: math.NewVector2(50, 30), right: true}})
	if ti.caret != 3 {
		t.Fatalf("caret = %d, want 3", ti.caret)
	}
	// Right at the end stays clamped.
	scene.Update(&core.Context{Input: &stubInput{mouse: math.NewVector2(50, 30), right: true}})
	if ti.caret != 3 {
		t.Fatalf("caret = %d, want 3", ti.caret)
	}
}

func TestButtonDisabledTexture(t *testing.T) {
	scene, btn := newButtonInScene()
	btn.DisabledTexture = "assets/btn_disabled.png"
	btn.SetEnabled(false)
	scene.Update(&core.Context{Input: &stubInput{}})

	if got := btn.currentTexture(); got != "assets/btn_disabled.png" {
		t.Fatalf("currentTexture() = %q, want disabled texture", got)
	}
	btn.SetEnabled(true)
	if got := btn.currentTexture(); got != "" {
		t.Fatalf("currentTexture() = %q, want empty (no normal texture)", got)
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
	scene.Update(&core.Context{Input: &stubInput{mouse: math.NewVector2(50, 30), backspace: true}})
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
