package components

import (
	"github.com/EnesBaytekin/imge/core"
	"github.com/EnesBaytekin/imge/core/math"
)

// SceneSettingsComponent is the modal "Scene Settings" window opened by the scene
// list's "#" button. It edits the active scene's name, background color, and camera
// (x/y/zoom) in place through real engine widgets (@TextInput and @ColorPicker),
// reusing the field-binding machinery the game-settings modal uses. The edits mutate
// the live scene so the viewport previews them immediately; "Save" writes the scene
// back to its .scene file (and syncs game.imge's initial_scene if the name changed),
// while "Close" (or a click outside) reverts the live scene to its captured originals.
//
// As with game settings, these are scene-config edits saved explicitly — they do not
// touch the scene-edit undo history.
type SceneSettingsComponent struct {
	core.BaseUIComponent

	Background  math.Color `json:"background"`
	TitleText   math.Color `json:"title_text"`
	KeyText     math.Color `json:"key_text"`
	ValueText   math.Color `json:"value_text"`
	Accent      math.Color `json:"accent"`       // title bar + Save button
	BorderColor math.Color `json:"border_color"` // Close button + panel outline
	ErrorColor  math.Color `json:"error_color"`  // committed-value parse failure

	FontID    string  `json:"font_id"`
	FontSize  float64 `json:"font_size"`
	RowHeight float64 `json:"row_height"`

	sc *core.Scene
	vp *ViewportComponent

	bindings []fieldBinding
	labels   []string

	// Originals captured at spawn, restored on cancel/outside-click.
	origName string
	origBG   math.Color
	origCam  *core.Camera // nil means the scene had no camera

	saveBtn  *ButtonComponent
	closeBtn *ButtonComponent

	dismiss  bool
	centered bool

	dragging bool         // the title bar is being dragged to move the window
	dragGrab math.Vector2 // mouse offset from the window's top-left when the drag began
}

func (c *SceneSettingsComponent) titleH() float64 { return c.RowHeight + 8 }

func (c *SceneSettingsComponent) Initialize() {
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

// spawnSceneSettings opens the scene-settings modal for the active target scene. It is
// a no-op when there is no loaded scene or a modal is already open.
func spawnSceneSettings(scene *core.Scene) {
	if scene == nil || modalOpen() {
		return
	}
	vp := lookupViewport(scene)
	if vp == nil || vp.TargetScene() == nil {
		return
	}
	sc := vp.TargetScene()

	obj := core.NewObject("scene_settings")
	obj.UI = true
	obj.Layer = 3
	obj.Transform.Position = math.NewVector2(140, 60) // centered lazily on first Update

	win := &SceneSettingsComponent{}
	win.SetName("scene_settings")
	win.Width = 300
	win.Height = 140
	win.sc = sc
	win.vp = vp
	win.origName = sc.Name
	win.origBG = sc.BackgroundColor
	win.origCam = sc.Camera
	obj.AddComponent(win)

	if err := scene.AddObject(obj); err != nil {
		return
	}

	win.Initialize()
	win.buildWidgets()
	setModal(win)
	raiseToFront(scene, obj)
}

// ensureCam returns the scene's camera, creating a default one if it has none.
func (c *SceneSettingsComponent) ensureCam() *core.Camera {
	if c.sc.Camera == nil {
		c.sc.Camera = core.NewCamera()
	}
	return c.sc.Camera
}

// addField registers one editable scene field: key (widget name), display label, and
// the get/apply closures that read/write the live scene.
func (c *SceneSettingsComponent) addField(key, label string, kind fieldKind, get func() string, apply func(string) error, getColor func() math.Color) {
	b := fieldBinding{
		key:      key,
		row:      len(c.bindings),
		parts:    1,
		kind:     kind,
		get:      get,
		apply:    apply,
		getColor: getColor,
	}
	b.old = get()
	c.bindings = append(c.bindings, b)
	c.labels = append(c.labels, label)
}

// buildWidgets registers every scene field and creates its widget, then the Save/Close
// buttons. All widgets are children of this window object, initialized manually (the
// object's own Initialize already ran).
func (c *SceneSettingsComponent) buildWidgets() {
	owner := c.GetOwner()
	sc := c.sc

	c.addField("name", "Name", kindText,
		func() string { return sc.Name },
		func(s string) error { sc.Name = s; return nil }, nil)

	c.addField("background", "Background", kindColor,
		func() string { return sc.BackgroundColor.HexString() },
		func(s string) error {
			col, err := math.ParseHex(s)
			if err != nil {
				return err
			}
			sc.BackgroundColor = col
			return nil
		},
		func() math.Color { return sc.BackgroundColor })

	c.addField("camera_x", "Camera X", kindText,
		func() string {
			if sc.Camera == nil {
				return "0"
			}
			return formatFloat(sc.Camera.X)
		},
		func(s string) error {
			f, err := parseFloat(s)
			if err != nil {
				return err
			}
			c.ensureCam().X = f
			return nil
		}, nil)

	c.addField("camera_y", "Camera Y", kindText,
		func() string {
			if sc.Camera == nil {
				return "0"
			}
			return formatFloat(sc.Camera.Y)
		},
		func(s string) error {
			f, err := parseFloat(s)
			if err != nil {
				return err
			}
			c.ensureCam().Y = f
			return nil
		}, nil)

	c.addField("camera_zoom", "Camera Zoom", kindText,
		func() string {
			if sc.Camera == nil {
				return "1"
			}
			return formatFloat(sc.Camera.Zoom)
		},
		func(s string) error {
			f, err := parseFloat(s)
			if err != nil {
				return err
			}
			c.ensureCam().Zoom = f
			return nil
		}, nil)

	rect := c.Rect()
	valX := rect.X() + 130
	valW := rect.Width() - 130 - 12
	for i := range c.bindings {
		b := &c.bindings[i]
		y := rect.Y() + c.titleH() + float64(i)*c.RowHeight
		b.widget = makeFieldWidget(b, owner, math.NewVector2(valX, y), valW, c.RowHeight, c.FontID, c.FontSize, c.ValueText)
	}

	buttonY := c.titleH() + float64(len(c.bindings))*c.RowHeight + 4
	c.saveBtn = makePanelButton(owner, "save", "Save", math.NewVector2(8, buttonY), 138, 20, c.FontID, c.FontSize, c.Accent)
	c.closeBtn = makePanelButton(owner, "close", "Close", math.NewVector2(152, buttonY), 140, 20, c.FontID, c.FontSize, c.BorderColor)
}

// commitField applies a committed value through the binding without recording undo
// (the scene is saved explicitly, not part of the scene-edit history).
func (c *SceneSettingsComponent) commitField(b *fieldBinding, s string) error {
	if s == b.old {
		return nil
	}
	if err := b.apply(s); err != nil {
		return err
	}
	b.old = s
	return nil
}

// pollCommits detects committed widget changes each frame — a ColorPicker commit, a
// TextInput Enter or blur — and applies them. Mirrors the game-settings modal's
// pollCommits (no undo recording).
func (c *SceneSettingsComponent) pollCommits(ctx *core.Context) {
	for i := range c.bindings {
		b := &c.bindings[i]
		switch b.kind {
		case kindColor:
			cp := b.widget.(*ColorPickerComponent)
			_ = c.commitField(b, formatColorHex(cp.GetColor()))
		default:
			ti := b.widget.(*TextInputComponent)
			focused := ti.IsFocused()
			if focused && ctx.Input.IsKeyJustPressed(core.KeyEnter) {
				if err := c.commitField(b, ti.Text); err != nil {
					ti.TextColor = c.ErrorColor
				} else {
					ti.TextColor = c.ValueText
				}
			}
			if b.wasFocused && !focused {
				if err := c.commitField(b, ti.Text); err != nil {
					ti.Text = b.get() // revert on blur-error
				}
				ti.TextColor = c.ValueText
			}
			b.wasFocused = focused
		}
	}
}

// commitAll force-applies every widget's current value before Save, so a TextInput
// still being edited (no Enter/blur yet) is captured.
func (c *SceneSettingsComponent) commitAll() {
	for i := range c.bindings {
		b := &c.bindings[i]
		switch b.kind {
		case kindColor:
			_ = c.commitField(b, formatColorHex(b.widget.(*ColorPickerComponent).GetColor()))
		default:
			_ = c.commitField(b, b.widget.(*TextInputComponent).Text)
		}
	}
}

// revert restores the live scene to the originals captured at spawn.
func (c *SceneSettingsComponent) revert() {
	if c.sc == nil {
		return
	}
	c.sc.Name = c.origName
	c.sc.BackgroundColor = c.origBG
	c.sc.Camera = c.origCam
}

// save writes the edited scene back to disk and syncs the scene list (and
// game.imge's initial_scene when the name changed).
func (c *SceneSettingsComponent) save() {
	c.commitAll()

	vp := c.vp
	if vp == nil {
		vp = lookupViewport(c.GetScene())
	}
	if vp == nil {
		c.dismiss = true
		return
	}

	// A rename moves the scene's display name; if it was the initial scene, follow it.
	if c.sc.Name != c.origName && vp.CurrentProject() != "" {
		if initialSceneName(vp.CurrentProject()) == c.origName {
			if err := setInitialScene(vp.CurrentProject(), c.sc.Name); err != nil {
				console.Print("set initial scene: " + err.Error())
			}
		}
	}

	if err := vp.Save(); err != nil {
		console.Print("scene settings: " + err.Error())
	} else {
		console.Print("saved scene settings")
	}

	if sl := lookupSceneList(c.GetScene()); sl != nil {
		sl.refresh()
	}
	c.dismiss = true
}

func (c *SceneSettingsComponent) Update(ctx *core.Context) {
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

	if c.closeBtn != nil && c.closeBtn.ConsumeClick() {
		c.revert()
		c.dismiss = true
		return
	}
	if c.saveBtn != nil && c.saveBtn.ConsumeClick() {
		c.save()
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
		c.revert()
		c.dismiss = true
	}
}

// centerOnce repositions the window to the screen center on its first frame.
func (c *SceneSettingsComponent) centerOnce(ctx *core.Context) {
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

func (c *SceneSettingsComponent) Draw(r core.Renderer) {
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
	r.DrawText("SCENE SETTINGS", c.FontID, c.FontSize, math.NewVector2(rect.X()+6, rect.Y()+(c.titleH()-th)/2), c.TitleText)

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

	// Finalize dismissal after drawing (and after all Updates).
	if c.dismiss {
		c.closeSelf()
	}
}

// closeSelf clears the modal state and destroys the window's object.
func (c *SceneSettingsComponent) closeSelf() {
	clearModal()
	if owner := c.GetOwner(); owner != nil {
		owner.Destroy()
	}
}
