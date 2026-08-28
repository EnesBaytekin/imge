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
	mouseHeld        bool
	chars            []rune
	held             map[core.KeyCode]bool // keys currently pressed
	justPressed      map[core.KeyCode]bool // keys pressed this frame
}

func (s *stubInput) IsKeyPressed(k core.KeyCode) bool           { return s.held[k] }
func (s *stubInput) IsKeyJustPressed(k core.KeyCode) bool       { return s.justPressed[k] }
func (s *stubInput) IsKeyJustReleased(core.KeyCode) bool        { return false }
func (s *stubInput) IsMouseButtonPressed(core.MouseButton) bool { return s.mouseHeld }
func (s *stubInput) IsMouseButtonJustPressed(core.MouseButton) bool {
	return s.justPressedLeft
}
func (s *stubInput) IsMouseButtonJustReleased(core.MouseButton) bool {
	return s.justReleasedLeft
}
func (s *stubInput) GetMousePosition() math.Vector2 { return s.mouse }
func (s *stubInput) GetMouseDelta() math.Vector2    { return math.Vector2{} }
func (s *stubInput) GetMouseScroll() math.Vector2   { return math.Vector2{} }
func (s *stubInput) InputChars() []rune             { return s.chars }
func (s *stubInput) Update()                        {}

// heldKeys builds a held-key set for stubInput.held.
func heldKeys(keys ...core.KeyCode) map[core.KeyCode]bool {
	m := make(map[core.KeyCode]bool, len(keys))
	for _, k := range keys {
		m[k] = true
	}
	return m
}

// justPressedKeys builds a just-pressed set for stubInput.justPressed.
func justPressedKeys(keys ...core.KeyCode) map[core.KeyCode]bool {
	return heldKeys(keys...)
}

// stubTime is a minimal core.Time whose delta can be set per test.
type stubTime struct{ delta float64 }

func (t *stubTime) DeltaTime() float64 { return t.delta }
func (t *stubTime) TotalTime() float64 { return 0 }
func (t *stubTime) FPS() float64       { return 60 }
func (t *stubTime) Tick()              {}
func (t *stubTime) Sleep(float64)      {}

// newManagedScene builds a scene with a single @UIManager on a carrier object.
func newManagedScene() (*core.Scene, *UIManagerComponent) {
	scene := core.NewScene("test")
	root := core.NewObject("ui_root")
	mgr := &UIManagerComponent{}
	mgr.SetName("manager")
	_ = root.AddComponent(mgr)
	_ = scene.AddObject(root)
	return scene, mgr
}

// newManagedButton adds a 100×40 button (event "clicked") on a UI object at pos.
func newManagedButton(scene *core.Scene, pos math.Vector2) *ButtonComponent {
	obj := core.NewObject("btn")
	obj.UI = true
	obj.Transform.Position = pos
	btn := &ButtonComponent{}
	btn.Width = 100
	btn.Height = 40
	btn.Event = "clicked"
	btn.SetName("button")
	_ = obj.AddComponent(btn)
	_ = scene.AddObject(obj)
	return btn
}

// newManagedTextInput adds a 100×24 text input on a UI object at pos.
func newManagedTextInput(scene *core.Scene, pos math.Vector2) *TextInputComponent {
	obj := core.NewObject("input")
	obj.UI = true
	obj.Transform.Position = pos
	ti := &TextInputComponent{}
	ti.Width = 100
	ti.Height = 24
	ti.SetName("input")
	_ = obj.AddComponent(ti)
	_ = scene.AddObject(obj)
	return ti
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
	if c.IsFocusable() {
		t.Fatal("a bare BaseUIComponent must not be focusable")
	}
	if c.BlocksPointer() {
		t.Fatal("a bare BaseUIComponent must not block")
	}
}

func TestButtonClickEmitsEvent(t *testing.T) {
	scene, _ := newManagedScene()
	btn := newManagedButton(scene, math.NewVector2(10, 20))
	var clicked any
	btn.On("clicked", func(data any) { clicked = data })

	// First update initializes components and discovers elements.
	scene.Update(&core.Context{Input: &stubInput{}})

	// Hover + press inside.
	in := &stubInput{mouse: math.NewVector2(50, 40), justPressedLeft: true}
	scene.Update(&core.Context{Input: in})
	if !btn.hovered || !btn.pressed {
		t.Fatalf("after press: hovered=%v pressed=%v, want both true", btn.hovered, btn.pressed)
	}

	// Release inside → click.
	in.justPressedLeft = false
	in.justReleasedLeft = true
	scene.Update(&core.Context{Input: in})
	if clicked != btn {
		t.Fatalf("click event data = %v, want the button", clicked)
	}
	if btn.pressed {
		t.Fatal("expected pressed cleared after release")
	}
}

func TestButtonReleaseOutsideDoesNotClick(t *testing.T) {
	scene, _ := newManagedScene()
	btn := newManagedButton(scene, math.NewVector2(10, 20))
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
	scene, _ := newManagedScene()
	btn := newManagedButton(scene, math.NewVector2(10, 20))
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
	scene, _ := newManagedScene()
	ti := newManagedTextInput(scene, math.NewVector2(10, 20))
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
	btn := &ButtonComponent{}
	btn.DisabledTexture = "assets/btn_disabled.png"
	btn.SetEnabled(false)

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
	scene, _ := newManagedScene()
	ti := newManagedTextInput(scene, math.NewVector2(10, 20))
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
// docs — hex colors, string wrap modes, snake_case border keys, and blocking — so a
// change to a color/wrap/border/blocking decode can't silently break scene loading.
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

	var panelNoBlock PanelComponent
	decode("@Panel", map[string]any{"blocking": false}, &panelNoBlock)
	if panelNoBlock.Blocking == nil || *panelNoBlock.Blocking {
		t.Errorf("@Panel blocking:false should decode to a non-nil false")
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
	scene, _ := newManagedScene()
	ti := newManagedTextInput(scene, math.NewVector2(10, 20))
	ti.Text = "abcdef"
	ti.caret = 6
	in := &stubInput{mouse: math.NewVector2(50, 30)}
	tm := &stubTime{}

	scene.Update(&core.Context{Input: in, Time: tm}) // initialize + discover

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
	scene, _ := newManagedScene()
	ti := newManagedTextInput(scene, math.NewVector2(10, 20))
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
	scene, _ := newManagedScene()
	ti := newManagedTextInput(scene, math.NewVector2(10, 20))
	ti.Text = "one two three"
	scene.Update(&core.Context{Input: &stubInput{}})
	scene.Update(&core.Context{Input: &stubInput{mouse: math.NewVector2(50, 30), justPressedLeft: true}})

	// Ctrl+Right jumps to the END of the word to the right (the space stays right of
	// the caret); Ctrl+Left to the START of the word to the left. Release between
	// presses so each is a fresh key press (holding the same key would auto-repeat).
	press := func(keys ...core.KeyCode) {
		scene.Update(&core.Context{Input: &stubInput{mouse: math.NewVector2(50, 30), held: heldKeys(keys...)}})
		scene.Update(&core.Context{Input: &stubInput{mouse: math.NewVector2(50, 30)}}) // release
	}

	press(core.KeyRight, core.KeyControl) // end of "one"
	if ti.caret != 3 {
		t.Fatalf("Ctrl+Right: caret = %d, want 3", ti.caret)
	}
	press(core.KeyRight, core.KeyControl) // end of "two"
	if ti.caret != 7 {
		t.Fatalf("Ctrl+Right: caret = %d, want 7", ti.caret)
	}
	press(core.KeyLeft, core.KeyControl) // start of "two"
	if ti.caret != 4 {
		t.Fatalf("Ctrl+Left: caret = %d, want 4", ti.caret)
	}
}

// TestTextInputDeleteAndWordDelete verifies Delete (runes to the right) plus
// Ctrl+Backspace/Ctrl+Delete (whole words, preserving adjacent spaces).
func TestTextInputDeleteAndWordDelete(t *testing.T) {
	ti := &TextInputComponent{}
	ti.Text = "one two three"
	ti.caret = 3 // between "one" and " two"

	ti.deleteRuneAfterCaret() // delete the space → "onetwo three", caret 3
	if ti.Text != "onetwo three" {
		t.Fatalf("Delete: Text = %q, want %q", ti.Text, "onetwo three")
	}
	if ti.caret != 3 {
		t.Fatalf("Delete: caret = %d, want 3 (stays put)", ti.caret)
	}

	// Ctrl+Delete from caret 3 deletes forward to the end of the next word, keeping
	// the space after that word (" three") intact.
	ti.Text = "one two three"
	ti.caret = 3
	ti.deleteWordAfterCaret() // deletes " two" → "one three", caret 3
	if ti.Text != "one three" {
		t.Fatalf("Ctrl+Delete: Text = %q, want %q", ti.Text, "one three")
	}
	if ti.caret != 3 {
		t.Fatalf("Ctrl+Delete: caret = %d, want 3", ti.caret)
	}

	// Ctrl+Backspace from caret 3 deletes "one" (the space after it stays).
	ti.Text = "one two three"
	ti.caret = 3
	ti.deleteWordBeforeCaret() // deletes "one" → " two three", caret 0
	if ti.Text != " two three" {
		t.Fatalf("Ctrl+Backspace: Text = %q, want %q", ti.Text, " two three")
	}
	if ti.caret != 0 {
		t.Fatalf("Ctrl+Backspace: caret = %d, want 0", ti.caret)
	}
}

// TestTextInputDeleteKey verifies the Delete control key removes the rune to the
// right of the caret through the manager routing path.
func TestTextInputDeleteKey(t *testing.T) {
	scene, _ := newManagedScene()
	ti := newManagedTextInput(scene, math.NewVector2(10, 20))
	ti.Text = "abc"
	ti.caret = 1
	scene.Update(&core.Context{Input: &stubInput{}})
	scene.Update(&core.Context{Input: &stubInput{mouse: math.NewVector2(50, 30), justPressedLeft: true}})

	scene.Update(&core.Context{Input: &stubInput{mouse: math.NewVector2(50, 30), held: heldKeys(core.KeyDelete)}})
	if ti.Text != "ac" {
		t.Fatalf("Delete key: Text = %q, want %q", ti.Text, "ac")
	}
	if ti.caret != 1 {
		t.Fatalf("Delete key: caret = %d, want 1", ti.caret)
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

// TestUIManagerBlockingOcclusion verifies that a blocking panel drawn over a button
// swallows the click, so the button behind it does not fire.
func TestUIManagerBlockingOcclusion(t *testing.T) {
	scene, _ := newManagedScene()

	// A button (bottom, depth 0) covering (0,0)-(100,40).
	behind := core.NewObject("behind")
	behind.UI = true
	behind.Depth = 0
	behindBtn := &ButtonComponent{Event: "behind"}
	behindBtn.Width = 100
	behindBtn.Height = 40
	behindBtn.SetName("button")
	_ = behind.AddComponent(behindBtn)
	_ = scene.AddObject(behind)

	// A blocking panel (top, depth 1) covering the same area.
	front := core.NewObject("front")
	front.UI = true
	front.Depth = 1
	panel := &PanelComponent{}
	panel.Width = 100
	panel.Height = 40
	panel.SetName("panel")
	_ = front.AddComponent(panel)
	_ = scene.AddObject(front)

	var behindFired bool
	behindBtn.On("behind", func(any) { behindFired = true })

	scene.Update(&core.Context{Input: &stubInput{}})

	// Click where both overlap: the panel blocks the button.
	in := &stubInput{mouse: math.NewVector2(50, 20), justPressedLeft: true}
	scene.Update(&core.Context{Input: in})
	if behindBtn.hovered {
		t.Fatal("button behind a blocking panel must not hover")
	}
	in.justPressedLeft = false
	in.justReleasedLeft = true
	scene.Update(&core.Context{Input: in})
	if behindFired {
		t.Fatal("button behind a blocking panel must not click")
	}
}

// TestUIManagerFocusAndTab verifies geometric row-reading tab order and wrapping.
func TestUIManagerFocusAndTab(t *testing.T) {
	scene, _ := newManagedScene()
	win := core.NewObject("win")
	win.UI = true
	_ = scene.AddObject(win)

	mk := func(name string, off math.Vector2) *TextInputComponent {
		ti := &TextInputComponent{}
		ti.Width = 50
		ti.Height = 20
		ti.Offset = off
		ti.SetName(name)
		_ = win.AddComponent(ti)
		return ti
	}
	a := mk("a", math.NewVector2(0, 0))
	b := mk("b", math.NewVector2(100, 0))
	c := mk("c", math.NewVector2(0, 100))

	scene.Update(&core.Context{Input: &stubInput{}}) // initialize + discover

	// First Tab focuses the top-left element.
	scene.Update(&core.Context{Input: &stubInput{justPressed: justPressedKeys(core.KeyTab)}})
	if !a.focused || b.focused || c.focused {
		t.Fatalf("first Tab: a=%v b=%v c=%v, want only a", a.focused, b.focused, c.focused)
	}

	// Same row (left to right): a → b.
	scene.Update(&core.Context{Input: &stubInput{justPressed: justPressedKeys(core.KeyTab)}})
	if !b.focused || a.focused {
		t.Fatalf("second Tab: a=%v b=%v, want only b", a.focused, b.focused)
	}

	// Next row: b → c.
	scene.Update(&core.Context{Input: &stubInput{justPressed: justPressedKeys(core.KeyTab)}})
	if !c.focused || b.focused {
		t.Fatalf("third Tab: b=%v c=%v, want only c", b.focused, c.focused)
	}

	// Wraps around: c → a.
	scene.Update(&core.Context{Input: &stubInput{justPressed: justPressedKeys(core.KeyTab)}})
	if !a.focused || c.focused {
		t.Fatalf("wrap Tab: a=%v c=%v, want only a", a.focused, c.focused)
	}

	// Shift+Tab goes backwards and wraps: a → c.
	scene.Update(&core.Context{Input: &stubInput{justPressed: justPressedKeys(core.KeyTab), held: heldKeys(core.KeyShift)}})
	if !c.focused || a.focused {
		t.Fatalf("Shift+Tab: a=%v c=%v, want only c", a.focused, c.focused)
	}
}

// TestUIManagerDragWindow verifies a Draggable object follows the pointer while its
// non-interactive surface is held, and stops on release.
func TestUIManagerDragWindow(t *testing.T) {
	scene, _ := newManagedScene()
	win := core.NewObject("window")
	win.UI = true
	win.Draggable = true
	win.Transform.Position = math.NewVector2(10, 20)
	_ = scene.AddObject(win)

	panel := &PanelComponent{}
	panel.Width = 200
	panel.Height = 150
	panel.SetName("bg")
	_ = win.AddComponent(panel)

	scene.Update(&core.Context{Input: &stubInput{}})

	// Press on the panel (non-interactive) begins a drag.
	in := &stubInput{mouse: math.NewVector2(50, 40), justPressedLeft: true, mouseHeld: true}
	scene.Update(&core.Context{Input: in})

	// Move while held: the object follows the pointer.
	in.justPressedLeft = false
	in.mouse = math.NewVector2(80, 65)
	scene.Update(&core.Context{Input: in})
	if win.Transform.Position != (math.Vector2{X: 40, Y: 45}) {
		t.Fatalf("after drag: position = %v, want (40, 45)", win.Transform.Position)
	}

	// Release, then move again: the window must not move.
	in.mouseHeld = false
	in.justReleasedLeft = true
	scene.Update(&core.Context{Input: in})
	in.justReleasedLeft = false
	in.mouse = math.NewVector2(200, 200)
	scene.Update(&core.Context{Input: in})
	if win.Transform.Position != (math.Vector2{X: 40, Y: 45}) {
		t.Fatalf("after release: position = %v, want unchanged (40, 45)", win.Transform.Position)
	}
}

// TestUIManagerTagsScope verifies a manager restricted by tags only routes to UI
// objects carrying one of those tags.
func TestUIManagerTagsScope(t *testing.T) {
	scene, mgr := newManagedScene()
	mgr.Tags = []string{"modal"}

	managed := core.NewObject("managed")
	managed.UI = true
	managed.AddTag("modal")
	managed.Transform.Position = math.NewVector2(10, 20)
	mBtn := &ButtonComponent{Event: "managed"}
	mBtn.Width = 100
	mBtn.Height = 40
	mBtn.SetName("button")
	_ = managed.AddComponent(mBtn)
	_ = scene.AddObject(managed)

	unmanaged := core.NewObject("unmanaged")
	unmanaged.UI = true
	unmanaged.Transform.Position = math.NewVector2(300, 20)
	uBtn := &ButtonComponent{Event: "unmanaged"}
	uBtn.Width = 100
	uBtn.Height = 40
	uBtn.SetName("button")
	_ = unmanaged.AddComponent(uBtn)
	_ = scene.AddObject(unmanaged)

	var managedFired, unmanagedFired bool
	mBtn.On("managed", func(any) { managedFired = true })
	uBtn.On("unmanaged", func(any) { unmanagedFired = true })

	scene.Update(&core.Context{Input: &stubInput{}})

	// Click the tagged button → fires.
	in := &stubInput{mouse: math.NewVector2(50, 40), justPressedLeft: true}
	scene.Update(&core.Context{Input: in})
	in.justPressedLeft = false
	in.justReleasedLeft = true
	scene.Update(&core.Context{Input: in})
	if !managedFired {
		t.Fatal("expected the tagged button to fire")
	}

	// Click the untagged button → out of scope, ignored.
	in = &stubInput{mouse: math.NewVector2(340, 40), justPressedLeft: true}
	scene.Update(&core.Context{Input: in})
	in.justPressedLeft = false
	in.justReleasedLeft = true
	scene.Update(&core.Context{Input: in})
	if unmanagedFired {
		t.Fatal("untagged button should be outside the manager's scope")
	}
}

// TestUIManagerRaiseToFront verifies click-to-front window behavior: clicking an
// object lifts it above its siblings, and toggling back and forth keeps the last
// clicked object on top. auto_raise:false disables it.
func TestUIManagerRaiseToFront(t *testing.T) {
	scene, mgr := newManagedScene()

	// Two partially-overlapping windows. back covers x∈[0,100]; front covers
	// x∈[50,150], so x=25 hits only back and x=120 hits only front.
	back := newManagedButton(scene, math.NewVector2(0, 0))
	front := newManagedButton(scene, math.NewVector2(50, 0))

	scene.Update(&core.Context{Input: &stubInput{}})

	// Click the exposed part of the back window → it rises above the front.
	scene.Update(&core.Context{Input: &stubInput{mouse: math.NewVector2(25, 20), justPressedLeft: true}})
	if back.GetOwner().Depth <= front.GetOwner().Depth {
		t.Fatalf("after clicking back: back depth %v <= front depth %v, want back on top",
			back.GetOwner().Depth, front.GetOwner().Depth)
	}

	// Click the exposed part of the front window → now it rises above the back.
	scene.Update(&core.Context{Input: &stubInput{mouse: math.NewVector2(120, 20), justPressedLeft: true}})
	if front.GetOwner().Depth <= back.GetOwner().Depth {
		t.Fatalf("after clicking front: front depth %v <= back depth %v, want front on top",
			front.GetOwner().Depth, back.GetOwner().Depth)
	}

	// Clicking the already-top window must not lower it.
	topDepth := front.GetOwner().Depth
	scene.Update(&core.Context{Input: &stubInput{mouse: math.NewVector2(120, 20), justPressedLeft: true}})
	if front.GetOwner().Depth != topDepth {
		t.Fatalf("re-clicking the top window changed its depth %v → %v", topDepth, front.GetOwner().Depth)
	}

	// With auto_raise disabled, clicking the back window leaves depth alone.
	disabled := false
	mgr.AutoRaise = &disabled
	backDepth := back.GetOwner().Depth
	scene.Update(&core.Context{Input: &stubInput{mouse: math.NewVector2(25, 20), justPressedLeft: true}})
	if back.GetOwner().Depth != backDepth {
		t.Fatalf("auto_raise:false still raised the window: %v → %v", backDepth, back.GetOwner().Depth)
	}
}

// TestUIManagerRaiseToFrontRespectsLayer verifies that click-to-front stays inside
// the clicked object's layer: reordering layer-0 windows must never cross into (or
// disturb) a fixed chrome object in a higher layer, and repeated clicks must keep
// depths bounded rather than growing without limit.
func TestUIManagerRaiseToFrontRespectsLayer(t *testing.T) {
	scene, _ := newManagedScene()

	back := newManagedButton(scene, math.NewVector2(0, 0))
	front := newManagedButton(scene, math.NewVector2(50, 0))

	// A fixed chrome bar in a higher layer: always on top, never reordered by
	// click-to-front among the layer-0 windows.
	header := core.NewObject("header")
	header.UI = true
	header.Layer = 1
	_ = scene.AddObject(header)

	scene.Update(&core.Context{Input: &stubInput{}})

	// Click back: it rises to the top of layer 0, compacting depths there.
	scene.Update(&core.Context{Input: &stubInput{mouse: math.NewVector2(25, 20), justPressedLeft: true}})
	if back.GetOwner().Depth <= front.GetOwner().Depth {
		t.Fatalf("after clicking back: back depth %v <= front depth %v", back.GetOwner().Depth, front.GetOwner().Depth)
	}
	if back.GetOwner().Layer != 0 || header.Layer != 1 {
		t.Fatalf("layers moved: back layer %d, header layer %d", back.GetOwner().Layer, header.Layer)
	}

	// Repeated clicks must keep the layer-0 depths bounded (0..n-1), never growing.
	for i := 0; i < 5; i++ {
		scene.Update(&core.Context{Input: &stubInput{mouse: math.NewVector2(25, 20), justPressedLeft: true}})
	}
	if d := back.GetOwner().Depth; d > 1 {
		t.Fatalf("depth grew past the layer's bound: %v", d)
	}
	if back.GetOwner().Depth <= front.GetOwner().Depth {
		t.Fatalf("back should stay on top after repeated clicks")
	}

	// The sorted order must still put the header (layer 1) last, on top.
	sorted := scene.GetSortedObjects()
	if len(sorted) == 0 || sorted[len(sorted)-1] != header {
		t.Fatalf("header must be the topmost object")
	}
}
