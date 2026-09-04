package components

import (
	"path/filepath"
	"strconv"
	"strings"

	"github.com/EnesBaytekin/imge/core"
	imgejson "github.com/EnesBaytekin/imge/core/json"
	"github.com/EnesBaytekin/imge/core/math"
)

// GameSettingsComponent is the modal "Game Settings" window opened by the menu bar's
// Game → "Game Settings...". It loads the target project's game.imge (window + game
// settings) and edits it in place through real engine widgets (@TextInput for numbers
// and strings, @CheckBox for booleans), reusing the same field-binding machinery the
// args window and inspector use. "Save" writes the edited config back to game.imge;
// "Close", or a click outside, dismisses without saving.
//
// Edits here mutate the in-memory config only — they do not touch the scene-edit undo
// history (game.imge is a separate document, saved explicitly), so Ctrl+Z never mixes
// game settings with scene edits.
type GameSettingsComponent struct {
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

	cfg  *imgejson.GameConfig
	path string // absolute path to the target's game.imge

	bindings []fieldBinding
	labels   []string

	saveBtn  *ButtonComponent
	closeBtn *ButtonComponent

	status   string // last save result, shown in the title bar
	dismiss  bool
	centered bool

	dragging bool         // the title bar is being dragged to move the window
	dragGrab math.Vector2 // mouse offset from the window's top-left when the drag began
}

func (c *GameSettingsComponent) titleH() float64 { return c.RowHeight + 8 }

func (c *GameSettingsComponent) Initialize() {
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

// spawnGameSettings opens the game-settings modal for the current target project. It
// is a no-op when there is no project, its game.imge can't be read, or a modal is
// already open.
func spawnGameSettings(scene *core.Scene) {
	if scene == nil || modalOpen() {
		return
	}
	vp := lookupViewport(scene)
	if vp == nil || vp.CurrentProject() == "" {
		return
	}
	path := filepath.Join(vp.CurrentProject(), "game.imge")
	cfg, err := imgejson.LoadGameConfig(path)
	if err != nil {
		console.Print("game settings: " + err.Error())
		return
	}

	obj := core.NewObject("game_settings")
	obj.UI = true
	obj.Layer = 3
	obj.Transform.Position = math.NewVector2(140, 60) // centered lazily on first Update

	win := &GameSettingsComponent{}
	win.SetName("game_settings")
	win.Width = 300
	win.Height = 232
	win.cfg = cfg
	win.path = path
	obj.AddComponent(win)

	if err := scene.AddObject(obj); err != nil {
		return
	}

	win.Initialize()
	win.buildWidgets()
	setModal(win)
	raiseToFront(scene, obj)
}

// addField registers one editable config field: key (widget name), display label, and
// the get/apply closures that read/write the config struct.
func (c *GameSettingsComponent) addField(key, label string, kind fieldKind, get func() string, apply func(string) error, getBool func() bool) {
	b := fieldBinding{
		key:     key,
		row:     len(c.bindings),
		parts:   1,
		kind:    kind,
		get:     get,
		apply:   apply,
		getBool: getBool,
	}
	b.old = get()
	c.bindings = append(c.bindings, b)
	c.labels = append(c.labels, label)
}

// buildWidgets registers every game.imge field and creates its widget, then the
// Save/Close buttons. All widgets are children of this window object, initialized
// manually (the object's own Initialize already ran).
func (c *GameSettingsComponent) buildWidgets() {
	owner := c.GetOwner()
	cfg := c.cfg

	c.addField("name", "Name", kindText,
		func() string { return cfg.Name },
		func(s string) error { cfg.Name = s; return nil }, nil)

	c.addField("window_title", "Title", kindText,
		func() string { return cfg.Window.Title },
		func(s string) error { cfg.Window.Title = s; return nil }, nil)

	c.addField("width", "Width", kindText,
		func() string { return strconv.Itoa(cfg.Window.Width) },
		intApply(&cfg.Window.Width), nil)

	c.addField("height", "Height", kindText,
		func() string { return strconv.Itoa(cfg.Window.Height) },
		intApply(&cfg.Window.Height), nil)

	c.addField("fullscreen", "Fullscreen", kindCheck,
		func() string { return strconv.FormatBool(cfg.Window.Fullscreen) },
		boolApply(&cfg.Window.Fullscreen),
		func() bool { return cfg.Window.Fullscreen })

	c.addField("resizable", "Resizable", kindCheck,
		func() string { return strconv.FormatBool(cfg.Window.Resizable) },
		boolApply(&cfg.Window.Resizable),
		func() bool { return cfg.Window.Resizable })

	c.addField("pixel_per_unit", "Pixel / Unit", kindText,
		func() string { return strconv.Itoa(cfg.Window.PixelPerUnit) },
		intApply(&cfg.Window.PixelPerUnit), nil)

	c.addField("scale", "Scale", kindText,
		func() string { return strconv.Itoa(cfg.Window.Scale) },
		intApply(&cfg.Window.Scale), nil)

	c.addField("smooth_shapes", "Smooth Shapes", kindCheck,
		func() string { return strconv.FormatBool(cfg.Window.SmoothShapes) },
		boolApply(&cfg.Window.SmoothShapes),
		func() bool { return cfg.Window.SmoothShapes })

	c.addField("target_fps", "Target FPS", kindText,
		func() string { return strconv.Itoa(cfg.Game.TargetFPS) },
		intApply(&cfg.Game.TargetFPS), nil)

	c.addField("initial_scene", "Initial Scene", kindText,
		func() string { return cfg.Game.InitialScene },
		func(s string) error { cfg.Game.InitialScene = s; return nil }, nil)

	// Build the value widgets (single-part, no scroll: the window is sized to fit all
	// eleven fields).
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

// intApply returns a string→int setter that parses and stores into *p.
func intApply(p *int) func(string) error {
	return func(s string) error {
		n, err := strconv.Atoi(strings.TrimSpace(s))
		if err != nil {
			return err
		}
		*p = n
		return nil
	}
}

// boolApply returns a string→bool setter that parses and stores into *p.
func boolApply(p *bool) func(string) error {
	return func(s string) error {
		v, err := parseBool(s)
		if err != nil {
			return err
		}
		*p = v
		return nil
	}
}

// commitField applies a committed value through the binding without recording undo
// (game.imge is saved explicitly, not part of the scene-edit history).
func commitField(b *fieldBinding, s string) error {
	if s == b.old {
		return nil
	}
	if err := b.apply(s); err != nil {
		return err
	}
	b.old = s
	return nil
}

// pollCommits detects committed widget changes each frame — a CheckBox toggle, a
// TextInput Enter or blur — and applies them. TextInput parse failures tint the box
// error-red and keep focus (Enter) or revert the text (blur). Mirrors the args window's
// pollCommits but without undo recording.
func (c *GameSettingsComponent) pollCommits(ctx *core.Context) {
	for i := range c.bindings {
		b := &c.bindings[i]
		switch b.kind {
		case kindCheck:
			cb := b.widget.(*CheckBoxComponent)
			_ = commitStringDirty(b, strconv.FormatBool(cb.GetChecked()), false)
		default:
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
}

// commitAll force-applies every widget's current value before Save, so a TextInput
// still being edited (no Enter/blur yet) is captured.
func (c *GameSettingsComponent) commitAll() {
	for i := range c.bindings {
		b := &c.bindings[i]
		switch b.kind {
		case kindCheck:
			_ = commitField(b, strconv.FormatBool(b.widget.(*CheckBoxComponent).GetChecked()))
		default:
			_ = commitField(b, b.widget.(*TextInputComponent).Text)
		}
	}
}

func (c *GameSettingsComponent) Update(ctx *core.Context) {
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
		c.dismiss = true
		return
	}
	if c.saveBtn != nil && c.saveBtn.ConsumeClick() {
		c.commitAll()
		if err := imgejson.SaveGameConfig(c.cfg, c.path); err != nil {
			c.status = "save error"
			console.Print("game settings: " + err.Error())
		} else {
			c.status = "saved"
			console.Print("saved " + c.path)
			// The logical-size outline in the viewport reflects game.imge; re-read it so
			// a changed window width/height updates the overlay immediately.
			if vp := lookupViewport(c.GetScene()); vp != nil {
				vp.RefreshLogicalSize()
			}
		}
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
func (c *GameSettingsComponent) centerOnce(ctx *core.Context) {
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

func (c *GameSettingsComponent) Draw(r core.Renderer) {
	rect := c.Rect()
	if rect.Width() <= 0 || rect.Height() <= 0 {
		return
	}
	r.SetClipRect(rect)
	r.DrawRect(rect, c.Background)
	r.DrawRectOutline(rect, c.BorderColor, 1)

	_, th := r.MeasureText("Ag", c.FontID, c.FontSize)

	// Title bar with a save-status suffix when present.
	r.DrawRect(math.NewRect(rect.X(), rect.Y(), rect.Width(), c.titleH()), c.Accent)
	title := "GAME SETTINGS"
	if c.status != "" {
		title = "GAME SETTINGS — " + c.status
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

	// Finalize dismissal after drawing (and after all Updates).
	if c.dismiss {
		c.closeSelf()
	}
}

// closeSelf clears the modal state and destroys the window's object.
func (c *GameSettingsComponent) closeSelf() {
	clearModal()
	if owner := c.GetOwner(); owner != nil {
		owner.Destroy()
	}
}
