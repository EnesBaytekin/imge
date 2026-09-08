package components

import (
	"github.com/EnesBaytekin/imge/core"
	"github.com/EnesBaytekin/imge/core/math"
)

// TextPromptComponent is a generic modal single-line text prompt: a title, a
// @TextInput seeded with an initial value, and OK/Cancel buttons. OK (or Enter in
// the field) runs onSubmit with the current text; onSubmit returns "" to accept and
// close, or an error message to show and keep the dialog open. It backs the "save as
// object" filename prompt.
type TextPromptComponent struct {
	core.BaseUIComponent

	Background  math.Color `json:"background"`
	TitleText   math.Color `json:"title_text"`
	Accent      math.Color `json:"accent"`       // OK button
	BorderColor math.Color `json:"border_color"` // Cancel button + panel outline
	ErrorColor  math.Color `json:"error_color"`  // validation error line
	FontID      string     `json:"font_id"`
	FontSize    float64    `json:"font_size"`

	title    string
	initial  string
	okLabel  string
	onSubmit func(string) string

	input    *TextInputComponent
	ok       *ButtonComponent
	cancel   *ButtonComponent
	errText  string
	dismiss  bool
	centered bool
}

func (c *TextPromptComponent) titleH() float64 { return 18 }

func (c *TextPromptComponent) Initialize() {
	if c.Background == (math.Color{}) {
		c.Background = math.NewColor(0x1d, 0x21, 0x30, 0xff)
	}
	if c.TitleText == (math.Color{}) {
		c.TitleText = math.NewColor(0xe6, 0xe6, 0xef, 0xff)
	}
	if c.Accent == (math.Color{}) {
		c.Accent = math.NewColor(0x2f, 0x3b, 0x54, 0xff)
	}
	if c.BorderColor == (math.Color{}) {
		c.BorderColor = math.NewColor(0x3a, 0x42, 0x57, 0xff)
	}
	if c.ErrorColor == (math.Color{}) {
		c.ErrorColor = math.NewColor(0xff, 0x5a, 0x5a, 0xff)
	}
	if c.FontSize <= 0 {
		c.FontSize = 6
	}
	if c.Blocking == nil {
		c.SetBlocking(true)
	}
}

// spawnTextPrompt opens a text-prompt modal. onSubmit returns "" on success (closes)
// or an error message to display. It is a no-op when a modal is already open.
func spawnTextPrompt(scene *core.Scene, title, initial, okLabel string, onSubmit func(string) string) {
	if scene == nil || modalOpen() {
		return
	}

	obj := core.NewObject("text_prompt")
	obj.UI = true
	obj.Layer = 3
	obj.Transform.Position = math.NewVector2(160, 80) // centered lazily on first Update

	panel := &TextPromptComponent{}
	panel.SetName("text_prompt")
	panel.Width = 300
	panel.Height = 96
	panel.title = title
	panel.initial = initial
	panel.okLabel = okLabel
	panel.onSubmit = onSubmit
	obj.AddComponent(panel)

	if err := scene.AddObject(obj); err != nil {
		return
	}

	panel.Initialize()
	panel.buildWidgets()
	setModal(panel)
	raiseToFront(scene, obj)
}

// buildWidgets creates the text field and OK/Cancel buttons as children of the same
// object. Each is initialized manually (the object's own Initialize already ran).
func (c *TextPromptComponent) buildWidgets() {
	owner := c.GetOwner()

	ti := &TextInputComponent{}
	ti.FontID = c.FontID
	ti.Size = c.FontSize
	ti.TextColor = c.TitleText
	ti.PlaceholderColor = c.BorderColor
	ti.BackgroundColor = fieldBackground
	ti.OutlineColor = fieldOutline
	ti.OutlineThickness = 1
	ti.Text = c.initial
	ti.Width = 284
	ti.Height = 20
	ti.DrawLayer = 1
	ti.SetName("input")
	ti.SetOffset(math.NewVector2(8, 26))
	owner.AddComponent(ti)
	ti.Initialize()
	c.input = ti

	c.ok = makePanelButton(owner, "ok", c.okLabel, math.NewVector2(8, 56), 138, 20, c.FontID, c.FontSize, c.Accent)
	c.cancel = makePanelButton(owner, "cancel", "Cancel", math.NewVector2(152, 56), 140, 20, c.FontID, c.FontSize, c.BorderColor)
}

func (c *TextPromptComponent) Update(ctx *core.Context) {
	if ctx == nil || ctx.Input == nil {
		return
	}
	c.centerOnce(ctx)

	if c.cancel != nil && c.cancel.ConsumeClick() {
		c.dismiss = true
		return
	}
	if c.ok != nil && c.ok.ConsumeClick() {
		c.commit()
		if c.dismiss {
			return
		}
	}
	// Enter in the field commits too, so the flow matches a typed-in value.
	if c.input != nil && c.input.IsFocused() && ctx.Input.IsKeyJustPressed(core.KeyEnter) {
		c.commit()
		if c.dismiss {
			return
		}
	}

	if modalOutsideClick(c.GetScene(), c.GetOwner(), ctx) {
		c.dismiss = true
	}
}

// commit runs onSubmit with the current text. A non-empty result is an error to show;
// an empty result accepts and dismisses.
func (c *TextPromptComponent) commit() {
	if c.input == nil || c.onSubmit == nil {
		return
	}
	if msg := c.onSubmit(c.input.Text); msg != "" {
		c.errText = msg
		return
	}
	c.dismiss = true
}

// centerOnce repositions the panel to the screen center on its first frame.
func (c *TextPromptComponent) centerOnce(ctx *core.Context) {
	if c.centered || c.GetOwner() == nil {
		return
	}
	c.centered = true
	if ctx.Renderer == nil {
		return
	}
	vw, vh := ctx.Renderer.GetViewportSize()
	if vw <= 0 || vh <= 0 {
		return
	}
	c.GetOwner().SetPosition((float64(vw)-c.Width)/2, (float64(vh)-c.Height)/2)
}

func (c *TextPromptComponent) Draw(r core.Renderer) {
	rect := c.Rect()
	if rect.Width() <= 0 || rect.Height() <= 0 {
		return
	}
	r.SetClipRect(rect)
	r.DrawRect(rect, c.Background)
	r.DrawRectOutline(rect, c.BorderColor, 1)

	_, th := r.MeasureText("Ag", c.FontID, c.FontSize)
	r.DrawRect(math.NewRect(rect.X(), rect.Y(), rect.Width(), c.titleH()), c.Accent)
	r.DrawText(c.title, c.FontID, c.FontSize, math.NewVector2(rect.X()+6, rect.Y()+(c.titleH()-th)/2), c.TitleText)

	if c.errText != "" {
		r.DrawText(c.errText, c.FontID, c.FontSize, math.NewVector2(rect.X()+8, rect.Y()+80), c.ErrorColor)
	}

	r.ClearClip()

	if c.dismiss {
		c.close()
	}
}

// close clears the modal state and destroys the panel's object.
func (c *TextPromptComponent) close() {
	clearModal()
	if owner := c.GetOwner(); owner != nil {
		owner.Destroy()
	}
}
