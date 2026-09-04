package components

import (
	"fmt"

	"github.com/EnesBaytekin/imge/core"
	"github.com/EnesBaytekin/imge/core/math"
)

// EditorSettingsComponent is the modal "Editor Settings" window opened by the menu
// bar's Edit → "Editor Settings...". It edits settings that affect only the editor —
// never the built game — currently the viewport grid's width and height spacing. The
// values live on the viewport component (in memory, applied live) and are undoable
// like game settings, but do not mark the scene document dirty (they are not part of
// the saved project).
type EditorSettingsComponent struct {
	core.BaseUIComponent

	Background  math.Color `json:"background"`
	TitleText   math.Color `json:"title_text"`
	KeyText     math.Color `json:"key_text"`
	ValueText   math.Color `json:"value_text"`
	Accent      math.Color `json:"accent"`       // title bar + Done button
	BorderColor math.Color `json:"border_color"` // panel outline
	ErrorColor  math.Color `json:"error_color"`  // committed-value parse failure

	FontID    string  `json:"font_id"`
	FontSize  float64 `json:"font_size"`
	RowHeight float64 `json:"row_height"`

	vp *ViewportComponent

	bindings []fieldBinding
	labels   []string

	doneBtn *ButtonComponent

	status   string
	dismiss  bool
	centered bool

	dragging bool         // the title bar is being dragged to move the window
	dragGrab math.Vector2 // mouse offset from the window's top-left when the drag began
}

func (c *EditorSettingsComponent) titleH() float64 { return c.RowHeight + 8 }

func (c *EditorSettingsComponent) Initialize() {
	if c.Background == (math.Color{}) {
		c.Background = math.NewColor(0x1d, 0x21, 0x30, 0xff)
	}
	if c.TitleText == (math.Color{}) {
		c.TitleText = math.NewColor(0xff, 0xff, 0xff, 0xff)
	}
	if c.KeyText == (math.Color{}) {
		c.KeyText = math.NewColor(0x6b, 0x73, 0x85, 0xff)
	}
	if c.ValueText == (math.Color{}) {
		c.ValueText = math.NewColor(0xe6, 0xe6, 0xef, 0xff)
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
	if c.RowHeight <= 0 {
		c.RowHeight = 16
	}
	if c.Blocking == nil {
		c.SetBlocking(true)
	}
}

// spawnEditorSettings opens the editor-settings modal. It is a no-op when there is no
// viewport or a modal is already open.
func spawnEditorSettings(scene *core.Scene) {
	if scene == nil || modalOpen() {
		return
	}
	vp := lookupViewport(scene)
	if vp == nil {
		return
	}

	obj := core.NewObject("editor_settings")
	obj.UI = true
	obj.Layer = 3
	obj.Transform.Position = math.NewVector2(140, 60) // centered lazily on first Update

	win := &EditorSettingsComponent{}
	win.SetName("editor_settings")
	win.Width = 300
	win.Height = 88
	win.vp = vp
	obj.AddComponent(win)

	if err := scene.AddObject(obj); err != nil {
		return
	}

	win.Initialize()
	win.buildWidgets()
	setModal(win)
	raiseToFront(scene, obj)
}

// addField registers one editable setting: key (widget name), display label, and the
// get/apply closures that read/write the viewport's setting.
func (c *EditorSettingsComponent) addField(key, label string, get func() string, apply func(string) error) {
	b := fieldBinding{
		key:   key,
		row:   len(c.bindings),
		parts: 1,
		kind:  kindText,
		get:   get,
		apply: apply,
	}
	b.old = get()
	c.bindings = append(c.bindings, b)
	c.labels = append(c.labels, label)
}

// floatApply returns a string→float setter that parses, requires a positive value,
// and stores into *p.
func floatApply(p *float64) func(string) error {
	return func(s string) error {
		f, err := parseFloat(s)
		if err != nil {
			return err
		}
		if f <= 0 {
			return fmt.Errorf("must be positive")
		}
		*p = f
		return nil
	}
}

// buildWidgets registers the grid width/height fields and the Done button.
func (c *EditorSettingsComponent) buildWidgets() {
	owner := c.GetOwner()
	vp := c.vp

	c.addField("grid_width", "Grid Width",
		func() string { return formatFloat(vp.GridStepX) },
		floatApply(&vp.GridStepX))
	c.addField("grid_height", "Grid Height",
		func() string { return formatFloat(vp.GridStepY) },
		floatApply(&vp.GridStepY))

	rect := c.Rect()
	valX := rect.X() + 130
	valW := rect.Width() - 130 - 12
	for i := range c.bindings {
		b := &c.bindings[i]
		y := rect.Y() + c.titleH() + float64(i)*c.RowHeight
		b.widget = makeFieldWidget(b, owner, math.NewVector2(valX, y), valW, c.RowHeight, c.FontID, c.FontSize, c.ValueText)
	}

	buttonY := c.titleH() + float64(len(c.bindings))*c.RowHeight + 4
	c.doneBtn = makePanelButton(owner, "done", "Done", math.NewVector2(8, buttonY), 284, 20, c.FontID, c.FontSize, c.Accent)
}

// pollCommits applies committed widget changes each frame (Enter or blur), recording
// undo without marking the scene document dirty.
func (c *EditorSettingsComponent) pollCommits(ctx *core.Context) {
	for i := range c.bindings {
		b := &c.bindings[i]
		ti := b.widget.(*TextInputComponent)
		focused := ti.IsFocused()
		if focused && ctx.Input.IsKeyJustPressed(core.KeyEnter) {
			if err := commitStringDirty(b, ti.Text, false); err != nil {
				ti.TextColor = c.ErrorColor
			} else {
				ti.TextColor = c.ValueText
			}
		}
		if b.wasFocused && !focused {
			if err := commitStringDirty(b, ti.Text, false); err != nil {
				ti.Text = b.get() // revert on blur-error
			}
			ti.TextColor = c.ValueText
		}
		b.wasFocused = focused
	}
}

func (c *EditorSettingsComponent) Update(ctx *core.Context) {
	if ctx == nil || ctx.Input == nil {
		return
	}
	c.centerOnce(ctx)

	mouse := ctx.Input.GetMousePosition()

	// Drag-to-move: while the title bar is held, follow the cursor (even outside the
	// window). Moving the owner carries the value widgets with it, since their offsets
	// are relative to the owner's transform.
	if c.dragging {
		if ctx.Input.IsMouseButtonPressed(core.MouseButtonLeft) {
			c.GetOwner().SetPosition(mouse.X-c.dragGrab.X, mouse.Y-c.dragGrab.Y)
		} else {
			c.dragging = false
		}
	}

	c.pollCommits(ctx)

	if c.doneBtn != nil && c.doneBtn.ConsumeClick() {
		c.dismiss = true
		return
	}

	// Title-bar press starts a drag (tested before the outside-click check, so moving
	// the window never reads as a dismissal).
	if ctx.Input.IsMouseButtonJustPressed(core.MouseButtonLeft) {
		rect := c.Rect()
		if math.NewRect(rect.X(), rect.Y(), rect.Width(), c.titleH()).ContainsPoint(mouse) {
			c.dragging = true
			c.dragGrab = mouse.Subtract(rect.Position)
			return
		}
	}

	if modalOutsideClick(c.GetScene(), c.GetOwner(), ctx) {
		c.dismiss = true
	}
}

// centerOnce repositions the window to the screen center on its first frame.
func (c *EditorSettingsComponent) centerOnce(ctx *core.Context) {
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

func (c *EditorSettingsComponent) Draw(r core.Renderer) {
	rect := c.Rect()
	if rect.Width() <= 0 || rect.Height() <= 0 {
		return
	}
	r.SetClipRect(rect)
	r.DrawRect(rect, c.Background)
	r.DrawRectOutline(rect, c.BorderColor, 1)

	_, th := r.MeasureText("Ag", c.FontID, c.FontSize)

	// Title bar.
	r.DrawRect(math.NewRect(rect.X(), rect.Y(), rect.Width(), c.titleH()), c.Accent)
	title := "EDITOR SETTINGS"
	if c.status != "" {
		title = "EDITOR SETTINGS — " + c.status
	}
	r.DrawText(title, c.FontID, c.FontSize, math.NewVector2(rect.X()+6, rect.Y()+(c.titleH()-th)/2), c.TitleText)

	// Field name labels (the value widgets draw themselves as layer-1 children).
	for i := range c.bindings {
		y := rect.Y() + c.titleH() + float64(i)*c.RowHeight
		ty := y + (c.RowHeight-th)/2
		if ty < y {
			ty = y
		}
		r.DrawText(c.labels[i], c.FontID, c.FontSize, math.NewVector2(rect.X()+8, ty), c.KeyText)
	}

	r.ClearClip()

	if c.dismiss {
		c.closeSelf()
	}
}

// closeSelf clears the modal state and destroys the window's object.
func (c *EditorSettingsComponent) closeSelf() {
	clearModal()
	if owner := c.GetOwner(); owner != nil {
		owner.Destroy()
	}
}
