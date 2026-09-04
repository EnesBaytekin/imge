package components

import (
	"strings"

	"github.com/EnesBaytekin/imge/core"
	"github.com/EnesBaytekin/imge/core/math"
)

// OpenProjectPanelComponent is the modal "Open Project" dialog opened by the menu
// bar's File → "Open Project...". It shows a @TextInput seeded with the current
// project directory, and Open/Cancel buttons. Open switches the viewport to the typed
// directory (via ViewportComponent.SetProject) and closes; Cancel, Enter in the field,
// or a click outside closes (Enter also opens). A full file-selector can replace the
// plain path field later — for now a path is enough.
//
// It is modal (like AddComponentPanelComponent): while open, every other hand-rolled
// panel yields (see modalOpen), and an outside click dismisses it, finalized in Draw.
type OpenProjectPanelComponent struct {
	core.BaseUIComponent

	Background  math.Color `json:"background"`
	TitleText   math.Color `json:"title_text"`
	Accent      math.Color `json:"accent"`       // Open button
	BorderColor math.Color `json:"border_color"` // Cancel button + panel outline
	FontID      string     `json:"font_id"`
	FontSize    float64    `json:"font_size"`

	pathInput *TextInputComponent
	ok        *ButtonComponent
	cancel    *ButtonComponent
	dismiss   bool
	centered  bool
}

func (c *OpenProjectPanelComponent) titleH() float64 { return 18 }

func (c *OpenProjectPanelComponent) Initialize() {
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
	if c.FontSize <= 0 {
		c.FontSize = 6
	}
	if c.Blocking == nil {
		c.SetBlocking(true)
	}
}

// spawnOpenProject opens the open-project modal. It is a no-op when a modal is already
// open or there is no viewport.
func spawnOpenProject(scene *core.Scene) {
	if scene == nil || modalOpen() {
		return
	}
	if lookupViewport(scene) == nil {
		return
	}

	obj := core.NewObject("open_project_panel")
	obj.UI = true
	obj.Layer = 3
	obj.Transform.Position = math.NewVector2(160, 80) // centered lazily on first Update

	panel := &OpenProjectPanelComponent{}
	panel.SetName("open_project_panel")
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

// buildWidgets creates the path field and Open/Cancel buttons as children of the same
// object. Each is initialized manually (the object's own Initialize already ran).
func (c *OpenProjectPanelComponent) buildWidgets() {
	owner := c.GetOwner()

	ti := &TextInputComponent{}
	ti.FontID = c.FontID
	ti.Size = c.FontSize
	ti.TextColor = c.TitleText
	ti.PlaceholderColor = c.BorderColor
	ti.BackgroundColor = fieldBackground
	ti.OutlineColor = fieldOutline
	ti.OutlineThickness = 1
	ti.Placeholder = "project directory..."
	ti.Width = 284
	ti.Height = 20
	ti.DrawLayer = 1
	ti.SetName("path")
	ti.SetOffset(math.NewVector2(8, 26))
	owner.AddComponent(ti)
	ti.Initialize()
	// Seed the current project so the user edits it rather than retyping from scratch.
	if vp := lookupViewport(c.GetScene()); vp != nil {
		ti.Text = vp.CurrentProject()
	}
	c.pathInput = ti

	c.ok = makePanelButton(owner, "ok", "Open", math.NewVector2(8, 56), 138, 20, c.FontID, c.FontSize, c.Accent)
	c.cancel = makePanelButton(owner, "cancel", "Cancel", math.NewVector2(152, 56), 140, 20, c.FontID, c.FontSize, c.BorderColor)
}

func (c *OpenProjectPanelComponent) Update(ctx *core.Context) {
	if ctx == nil || ctx.Input == nil {
		return
	}
	c.centerOnce(ctx)

	// Open/Cancel are polled (events never reach dynamically-added widgets).
	if c.cancel != nil && c.cancel.ConsumeClick() {
		c.dismiss = true
		return
	}
	if c.ok != nil && c.ok.ConsumeClick() {
		c.commit()
		c.dismiss = true
		return
	}
	// Enter in the path field commits too, so the flow matches a typed-in directory.
	if c.pathInput != nil && c.pathInput.IsFocused() && ctx.Input.IsKeyJustPressed(core.KeyEnter) {
		c.commit()
		c.dismiss = true
		return
	}

	if modalOutsideClick(c.GetScene(), c.GetOwner(), ctx) {
		c.dismiss = true
	}
}

// commit switches the viewport to the typed directory (ignoring an empty field).
func (c *OpenProjectPanelComponent) commit() {
	if c.pathInput == nil {
		return
	}
	path := strings.TrimSpace(c.pathInput.Text)
	if path == "" {
		return
	}
	if vp := lookupViewport(c.GetScene()); vp != nil {
		vp.SetProject(path)
	}
}

// centerOnce repositions the panel to the screen center on its first frame.
func (c *OpenProjectPanelComponent) centerOnce(ctx *core.Context) {
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

func (c *OpenProjectPanelComponent) Draw(r core.Renderer) {
	rect := c.Rect()
	if rect.Width() <= 0 || rect.Height() <= 0 {
		return
	}
	r.SetClipRect(rect)
	r.DrawRect(rect, c.Background)
	r.DrawRectOutline(rect, c.BorderColor, 1)

	_, th := r.MeasureText("Ag", c.FontID, c.FontSize)
	r.DrawRect(math.NewRect(rect.X(), rect.Y(), rect.Width(), c.titleH()), c.Accent)
	r.DrawText("OPEN PROJECT", c.FontID, c.FontSize, math.NewVector2(rect.X()+6, rect.Y()+(c.titleH()-th)/2), c.TitleText)

	r.ClearClip()

	// Finalize dismissal after drawing (and after all Updates).
	if c.dismiss {
		c.close()
	}
}

// close clears the modal state and destroys the panel's object.
func (c *OpenProjectPanelComponent) close() {
	clearModal()
	if owner := c.GetOwner(); owner != nil {
		owner.Destroy()
	}
}
