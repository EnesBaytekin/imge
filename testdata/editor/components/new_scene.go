package components

import (
	"os"
	"strings"

	"github.com/EnesBaytekin/imge/core"
	"github.com/EnesBaytekin/imge/core/math"
)

// NewSceneComponent is the modal "New Scene" dialog opened by the scene list's "+"
// button. It shows a @TextInput for the scene name and Create/Cancel buttons. Create
// writes a fresh, empty .scene file with that name and switches the viewport to it;
// Cancel, or a click outside, closes. An invalid or duplicate name shows an error line
// and keeps the dialog open.
type NewSceneComponent struct {
	core.BaseUIComponent

	Background  math.Color `json:"background"`
	TitleText   math.Color `json:"title_text"`
	Accent      math.Color `json:"accent"`       // Create button
	BorderColor math.Color `json:"border_color"` // Cancel button + panel outline
	ErrorColor  math.Color `json:"error_color"`  // validation error line
	FontID      string     `json:"font_id"`
	FontSize    float64    `json:"font_size"`

	nameInput *TextInputComponent
	ok        *ButtonComponent
	cancel    *ButtonComponent
	errText   string
	dismiss   bool
	centered  bool
}

func (c *NewSceneComponent) titleH() float64 { return 18 }

func (c *NewSceneComponent) Initialize() {
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

// spawnNewScene opens the new-scene modal. It is a no-op when a modal is already open
// or there is no viewport.
func spawnNewScene(scene *core.Scene) {
	if scene == nil || modalOpen() {
		return
	}
	if lookupViewport(scene) == nil {
		return
	}

	obj := core.NewObject("new_scene")
	obj.UI = true
	obj.Layer = 3
	obj.Transform.Position = math.NewVector2(160, 80) // centered lazily on first Update

	panel := &NewSceneComponent{}
	panel.SetName("new_scene")
	panel.Width = 300
	panel.Height = 96
	obj.AddComponent(panel)

	if err := scene.AddObject(obj); err != nil {
		return
	}

	panel.Initialize()
	panel.buildWidgets()
	setModal(panel)
	raiseToFront(scene, obj)
}

// buildWidgets creates the name field and Create/Cancel buttons as children of the same
// object. Each is initialized manually (the object's own Initialize already ran).
func (c *NewSceneComponent) buildWidgets() {
	owner := c.GetOwner()

	ti := &TextInputComponent{}
	ti.FontID = c.FontID
	ti.Size = c.FontSize
	ti.TextColor = c.TitleText
	ti.PlaceholderColor = c.BorderColor
	ti.BackgroundColor = fieldBackground
	ti.OutlineColor = fieldOutline
	ti.OutlineThickness = 1
	ti.Placeholder = "scene name..."
	ti.Width = 284
	ti.Height = 20
	ti.DrawLayer = 1
	ti.SetName("name")
	ti.SetOffset(math.NewVector2(8, 26))
	owner.AddComponent(ti)
	ti.Initialize()
	c.nameInput = ti

	c.ok = makePanelButton(owner, "ok", "Create", math.NewVector2(8, 56), 138, 20, c.FontID, c.FontSize, c.Accent)
	c.cancel = makePanelButton(owner, "cancel", "Cancel", math.NewVector2(152, 56), 140, 20, c.FontID, c.FontSize, c.BorderColor)
}

func (c *NewSceneComponent) Update(ctx *core.Context) {
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
	// Enter in the name field commits too, so the flow matches a typed-in name.
	if c.nameInput != nil && c.nameInput.IsFocused() && ctx.Input.IsKeyJustPressed(core.KeyEnter) {
		c.commit()
		if c.dismiss {
			return
		}
	}

	if modalOutsideClick(c.GetScene(), c.GetOwner(), ctx) {
		c.dismiss = true
	}
}

// commit validates the name, creates the scene, and switches the viewport to it. On
// failure it sets errText and leaves the dialog open.
func (c *NewSceneComponent) commit() {
	if c.nameInput == nil {
		return
	}
	name := strings.TrimSpace(c.nameInput.Text)
	switch {
	case name == "":
		c.errText = "name is required"
		return
	case strings.ContainsAny(name, `/\`):
		c.errText = "name can't contain / or \\"
		return
	}

	vp := lookupViewport(c.GetScene())
	if vp == nil || vp.CurrentProject() == "" {
		c.errText = "no project loaded"
		return
	}
	dir := vp.CurrentProject()

	if _, err := os.Stat(scenePath(dir, name)); err == nil {
		c.errText = "a scene with that name already exists"
		return
	}
	if err := createSceneFile(dir, name); err != nil {
		c.errText = err.Error()
		return
	}

	// Refresh the list, then switch to the new scene to start editing it.
	if sl := lookupSceneList(c.GetScene()); sl != nil {
		sl.refresh()
	}
	vp.SetScene(name)
	c.dismiss = true
}

// centerOnce repositions the panel to the screen center on its first frame.
func (c *NewSceneComponent) centerOnce(ctx *core.Context) {
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

func (c *NewSceneComponent) Draw(r core.Renderer) {
	rect := c.Rect()
	if rect.Width() <= 0 || rect.Height() <= 0 {
		return
	}
	r.SetClipRect(rect)
	r.DrawRect(rect, c.Background)
	r.DrawRectOutline(rect, c.BorderColor, 1)

	_, th := r.MeasureText("Ag", c.FontID, c.FontSize)
	r.DrawRect(math.NewRect(rect.X(), rect.Y(), rect.Width(), c.titleH()), c.Accent)
	r.DrawText("NEW SCENE", c.FontID, c.FontSize, math.NewVector2(rect.X()+6, rect.Y()+(c.titleH()-th)/2), c.TitleText)

	if c.errText != "" {
		r.DrawText(c.errText, c.FontID, c.FontSize, math.NewVector2(rect.X()+8, rect.Y()+80), c.ErrorColor)
	}

	r.ClearClip()

	// Finalize dismissal after drawing (and after all Updates).
	if c.dismiss {
		c.close()
	}
}

// close clears the modal state and destroys the panel's object.
func (c *NewSceneComponent) close() {
	clearModal()
	if owner := c.GetOwner(); owner != nil {
		owner.Destroy()
	}
}
