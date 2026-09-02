package components

import (
	"sort"
	"strconv"
	"strings"

	"github.com/EnesBaytekin/imge/core"
	"github.com/EnesBaytekin/imge/core/math"
)

// InspectorComponent is the editor's right panel: it shows the selected target
// object's general info (name, position, rotation, scale, layer, depth, flags, tags)
// as editable rows, plus a list of its components (name + kind). Clicking a component
// row opens the ComponentArgsComponent window to edit that component's arguments.
// Object-property edits write back through the object's setters and record undo
// entries, so they persist with Save and revert with Ctrl+Z.
//
// Each editable property's value is a real engine widget (@TextInput/@CheckBox) added
// as a component on this panel object, so caret movement, Ctrl+word navigation, and
// the checkbox toggle all work instead of a hand-rolled edit buffer. The panel itself
// draws the chrome (title, labels, read-only values, the COMPONENTS list) and polls
// the widgets for commits.
type InspectorComponent struct {
	core.BaseUIComponent

	Background math.Color `json:"background"`
	TitleText  math.Color `json:"title_text"`
	KeyText    math.Color `json:"key_text"`
	ValueText  math.Color `json:"value_text"`
	Section    math.Color `json:"section"`     // "COMPONENTS" header
	Accent     math.Color `json:"accent"`      // title bar
	ErrorColor math.Color `json:"error_color"` // committed-value-parse failure

	FontID    string  `json:"font_id"`
	FontSize  float64 `json:"font_size"`
	RowHeight float64 `json:"row_height"`

	scroll      float64        // component-list scroll offset (pixels, 0 = top)
	bindings    []fieldBinding // value widgets for editable properties
	boundTarget *core.Object   // object the bindings were built for (rebuild on change)
	hoverComp   int            // component-list row under the cursor (-1 = none)
}

// prop is one editable object property: a label, a getter that renders the current
// value as an editable string, and a setter that parses an edited string and applies it
// (nil setter = read-only). get/set capture the object, so an undo closure built from
// them re-applies to the right object.
type prop struct {
	label   string
	get     func() string
	set     func(string) error // nil = read-only
	kind    fieldKind          // kindText / kindCheck (kindColor unused here)
	getBool func() bool        // kindCheck refresh
}

// lookupViewport resolves the editor's ViewportComponent by object name. It is the
// single source of truth for the current selection.
func lookupViewport(scene *core.Scene) *ViewportComponent {
	if scene == nil {
		return nil
	}
	if obj := scene.GetObjectByName("viewport"); obj != nil {
		return core.GetFrom[*ViewportComponent](obj)
	}
	return nil
}

func (c *InspectorComponent) Initialize() {
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
	if c.Section == (math.Color{}) {
		c.Section = math.NewColor(0x8a, 0x8a, 0x9a, 0xff)
	}
	if c.Accent == (math.Color{}) {
		c.Accent = math.NewColor(0x2f, 0x3b, 0x54, 0xff)
	}
	if c.ErrorColor == (math.Color{}) {
		c.ErrorColor = math.NewColor(0xff, 0x5a, 0x5a, 0xff)
	}
	if c.FontSize <= 0 {
		c.FontSize = 6
	}
	if c.RowHeight <= 0 {
		c.RowHeight = 14
	}
	// The inspector is an opaque surface: it blocks pointer events so the @UIManager
	// occludes whatever is drawn behind it (see pointerOwnedElsewhere).
	if c.Blocking == nil {
		c.SetBlocking(true)
	}
}

// titleH returns the title-bar height.
func (c *InspectorComponent) titleH() float64 { return c.RowHeight + 8 }

// compStart returns the content-space y (relative to the panel's top) where the first
// component row begins, given the number of property rows above it.
func (c *InspectorComponent) compStart(nProps int) float64 {
	return c.titleH() + float64(nProps+1)*c.RowHeight
}

// props builds the editable property list for an object. Each setter applies the parsed
// value through the object's own API, so side effects (scene name/tag/depth indexing)
// stay consistent. Only "ui" is a bool (@CheckBox); the rest are text. "active" has no
// setter and stays read-only (the viewport owns activation).
func (c *InspectorComponent) props(obj *core.Object) []prop {
	if obj == nil {
		return nil
	}
	tags := func() string {
		t := make([]string, 0, len(obj.Tags))
		for tag := range obj.Tags {
			t = append(t, tag)
		}
		sort.Strings(t)
		if len(t) == 0 {
			return "-"
		}
		return strings.Join(t, ", ")
	}
	return []prop{
		{"name", func() string { return obj.Name }, func(s string) error { return obj.SetName(s) }, kindText, nil},
		{"x", func() string { return formatFloat(obj.GetPosition().X) }, func(s string) error {
			f, err := parseFloat(s)
			if err != nil {
				return err
			}
			obj.SetPosition(f, obj.GetPosition().Y)
			return nil
		}, kindText, nil},
		{"y", func() string { return formatFloat(obj.GetPosition().Y) }, func(s string) error {
			f, err := parseFloat(s)
			if err != nil {
				return err
			}
			obj.SetPosition(obj.GetPosition().X, f)
			return nil
		}, kindText, nil},
		{"rotation", func() string { return formatFloat(math.RadiansToDegrees(obj.GetRotation())) }, func(s string) error {
			f, err := parseFloat(s)
			if err != nil {
				return err
			}
			obj.SetRotation(math.DegreesToRadians(f))
			return nil
		}, kindText, nil},
		{"scale", func() string { sc := obj.GetScale(); return formatFloat(sc.X) + ", " + formatFloat(sc.Y) }, func(s string) error {
			x, y, err := parseTwoFloats(s)
			if err != nil {
				return err
			}
			obj.SetScale(x, y)
			return nil
		}, kindText, nil},
		{"layer", func() string { return strconv.Itoa(obj.GetLayer()) }, func(s string) error {
			n, err := strconv.Atoi(strings.TrimSpace(s))
			if err != nil {
				return err
			}
			obj.SetLayer(n)
			return nil
		}, kindText, nil},
		{"depth", func() string { return formatFloat(obj.GetDepth()) }, func(s string) error {
			f, err := parseFloat(s)
			if err != nil {
				return err
			}
			return obj.SetDepth(f)
		}, kindText, nil},
		{"ui", func() string { return strconv.FormatBool(obj.UI) }, func(s string) error {
			b, err := parseBool(s)
			if err != nil {
				return err
			}
			obj.UI = b
			return nil
		}, kindCheck, func() bool { return obj.UI }},
		{"active", func() string { return strconv.FormatBool(obj.Active) }, nil, kindText, nil},
		{"tags", tags, func(s string) error { return setTags(obj, s) }, kindText, nil},
	}
}

// buildBindings converts the object's editable properties into field bindings, one
// widget per property. Read-only rows (nil setter) are skipped — the host draws them.
func (c *InspectorComponent) buildBindings(obj *core.Object) []fieldBinding {
	props := c.props(obj)
	out := make([]fieldBinding, 0, len(props))
	for i := range props {
		p := &props[i]
		if p.set == nil {
			continue
		}
		// "scale" is a Vector2, edited as two side-by-side boxes (x | y) instead of
		// a single "x, y" string.
		if p.label == "scale" {
			sx := fieldBinding{
				key: "scale_x", row: i, col: 0, parts: 2, kind: kindText,
				get: func() string { return formatFloat(obj.GetScale().X) },
				apply: func(s string) error {
					f, err := parseFloat(s)
					if err != nil {
						return err
					}
					sc := obj.GetScale()
					obj.SetScale(f, sc.Y)
					return nil
				},
			}
			sy := fieldBinding{
				key: "scale_y", row: i, col: 1, parts: 2, kind: kindText,
				get: func() string { return formatFloat(obj.GetScale().Y) },
				apply: func(s string) error {
					f, err := parseFloat(s)
					if err != nil {
						return err
					}
					sc := obj.GetScale()
					obj.SetScale(sc.X, f)
					return nil
				},
			}
			sx.old = sx.get()
			sy.old = sy.get()
			out = append(out, sx, sy)
			continue
		}
		b := fieldBinding{
			key:   p.label,
			row:   i,
			parts: 1,
			kind:  p.kind,
			get:   p.get,
			apply: p.set,
		}
		if p.kind == kindCheck {
			b.getBool = p.getBool
		}
		b.old = b.get()
		out = append(out, b)
	}
	return out
}

// rebuildRows detaches the old property widgets and builds fresh ones for the newly
// selected object. Called only on selection change (a structural change); never
// per-frame, or a focused widget would lose focus every frame.
func (c *InspectorComponent) rebuildRows(obj *core.Object) {
	c.removeWidgets()
	c.bindings = c.buildBindings(obj)
	rect := c.Rect()
	valX := rect.X() + 64
	valueW := rect.Width() - 64 - 4
	for i := range c.bindings {
		b := &c.bindings[i]
		pw := partWidth(valueW, b.parts)
		x := partX(valX, b.col, pw)
		y := rect.Y() + c.titleH() + float64(b.row)*c.RowHeight
		b.widget = makeFieldWidget(b, c.GetOwner(), math.NewVector2(x, y), pw, c.RowHeight, c.FontID, c.FontSize, c.ValueText)
	}
}

// removeWidgets detaches every property widget from the panel object.
func (c *InspectorComponent) removeWidgets() {
	removeWidgets(c.bindings, c.GetOwner())
}

// pollAndRefresh runs the per-frame widget pass (commit polling + live-sync), then
// re-lays out the value widgets.
func (c *InspectorComponent) pollAndRefresh(ctx *core.Context) {
	syncBindings(c.bindings, ctx, c.ValueText, c.ErrorColor)
	c.layoutRows()
}

// layoutRows repositions each property widget to its row (the property list is not
// scrollable, so scroll is 0).
func (c *InspectorComponent) layoutRows() {
	rect := c.Rect()
	valueW := rect.Width() - 64 - 4
	layoutWidgets(c.bindings, c.GetOwner(), rect.Y()+c.titleH(), rect.X()+64, valueW, 0, c.RowHeight, rect.Y()+rect.Height())
}

func (c *InspectorComponent) Update(ctx *core.Context) {
	if ctx == nil || ctx.Input == nil {
		return
	}
	vp := lookupViewport(c.GetScene())
	var obj *core.Object
	if vp != nil {
		obj = vp.SelectedObject()
	}

	// Rebuild the property widgets only when the selected object changes (a structural
	// change). Changing selection discards any in-progress edit, matching the previous
	// hand-rolled behavior.
	if c.boundTarget != obj {
		c.boundTarget = obj
		c.rebuildRows(obj)
	}

	// Commit any widget change and live-sync the model first, before hover-gated logic,
	// so commits fire even after the pointer leaves the panel.
	c.pollAndRefresh(ctx)

	mouse := ctx.Input.GetMousePosition()
	if !c.Rect().ContainsPoint(mouse) {
		c.hoverComp = -1
		return
	}

	// Yield to a window drawn above the inspector (e.g. a component-args window
	// dragged over it) — the @UIManager's blocking occlusion, see pointerOwnedElsewhere.
	if pointerOwnedElsewhere(c.GetScene(), c.GetOwner(), mouse) {
		c.hoverComp = -1
		return
	}

	var comps []core.Component
	if obj != nil {
		comps = obj.ComponentsInDrawOrder()
	}
	props := c.props(obj)
	compY := c.Rect().Y() + c.compStart(len(props))
	available := c.Rect().Height() - c.compStart(len(props))

	// Track the component row under the cursor (for the hover highlight), so the
	// COMPONENTS list reads as clickable.
	c.hoverComp = -1
	for i := range comps {
		y := compY + float64(i)*c.RowHeight - c.scroll
		if mouse.Y >= y && mouse.Y < y+c.RowHeight {
			c.hoverComp = i
			break
		}
	}

	// Wheel scrolls the component list, unless a widget holds focus (so a focused
	// TextInput never scrolls out from under the caret).
	if s := ctx.Input.GetMouseScroll(); s.Y != 0 {
		if mgr := lookupUIManager(c.GetScene()); mgr == nil || !mgr.HasFocus() {
			c.scroll -= s.Y * c.RowHeight * 2
			if max := c.maxScroll(len(comps), available); c.scroll > max {
				c.scroll = max
			}
			if c.scroll < 0 {
				c.scroll = 0
			}
		}
	}

	if !ctx.Input.IsMouseButtonJustPressed(core.MouseButtonLeft) {
		return
	}

	// Component rows: click to open the args window. Property rows are the widgets'
	// job (a click focuses the TextInput or toggles the CheckBox).
	for i, comp := range comps {
		y := compY + float64(i)*c.RowHeight - c.scroll
		if mouse.Y >= y && mouse.Y < y+c.RowHeight {
			spawnArgsWindow(c.GetScene(), comp)
			return
		}
	}
}

// maxScroll returns the scroll offset at which the last component row is just visible.
func (c *InspectorComponent) maxScroll(nComps int, available float64) float64 {
	if nComps == 0 || available <= 0 {
		return 0
	}
	if m := float64(nComps)*c.RowHeight - available; m > 0 {
		return m
	}
	return 0
}

func (c *InspectorComponent) Draw(r core.Renderer) {
	rect := c.Rect()
	if rect.Width() <= 0 || rect.Height() <= 0 {
		return
	}

	r.SetClipRect(rect)
	r.DrawRect(rect, c.Background)

	// Line height is constant for a font+size.
	_, th := r.MeasureText("Ag", c.FontID, c.FontSize)

	// Title bar.
	r.DrawRect(math.NewRect(rect.X(), rect.Y(), rect.Width(), c.titleH()), c.Accent)
	titleY := rect.Y() + (c.titleH()-th)/2
	r.DrawText("INSPECTOR", c.FontID, c.FontSize, math.NewVector2(rect.X()+6, titleY), c.TitleText)

	vp := lookupViewport(c.GetScene())
	var obj *core.Object
	if vp != nil {
		obj = vp.SelectedObject()
	}
	if obj == nil {
		r.DrawText("no selection", c.FontID, c.FontSize, math.NewVector2(rect.X()+6, rect.Y()+c.titleH()+2), c.KeyText)
		r.ClearClip()
		return
	}

	// Property rows: the host draws the name label and any read-only value; editable
	// values are drawn by their widgets (layer 1, above this chrome).
	props := c.props(obj)
	bodyTop := rect.Y() + c.titleH()
	valX := rect.X() + 64
	for i, p := range props {
		y := bodyTop + float64(i)*c.RowHeight
		ty := y + (c.RowHeight-th)/2
		if ty < y {
			ty = y
		}
		r.DrawText(p.label, c.FontID, c.FontSize, math.NewVector2(rect.X()+6, ty), c.KeyText)
		if p.set == nil {
			r.DrawText(p.get(), c.FontID, c.FontSize, math.NewVector2(valX, ty), c.KeyText) // read-only, dimmed
		}
	}

	// Components section header — one row above the scrollable list.
	compY := rect.Y() + c.compStart(len(props))
	sty := (compY - c.RowHeight) + (c.RowHeight-th)/2
	r.DrawText("COMPONENTS", c.FontID, c.FontSize, math.NewVector2(rect.X()+6, sty), c.Section)

	// Component rows (scrollable): name + kind, clipped to the list region below the
	// header so scrolled-out rows don't bleed over it.
	comps := obj.ComponentsInDrawOrder()
	r.SetClipRect(math.NewRect(rect.X(), compY, rect.Width(), rect.Height()-(compY-rect.Y())))
	for i, comp := range comps {
		y := compY + float64(i)*c.RowHeight - c.scroll
		if y+c.RowHeight < compY || y > rect.Y()+rect.Height() {
			continue
		}
		if i == c.hoverComp {
			r.DrawRect(math.NewRect(rect.X(), y, rect.Width(), c.RowHeight), c.Background.Lerp(math.White, 0.07))
		}
		ty := y + (c.RowHeight-th)/2
		if ty < y {
			ty = y
		}
		r.DrawText(comp.GetName(), c.FontID, c.FontSize, math.NewVector2(rect.X()+6, ty), c.ValueText)
		r.DrawText(comp.GetKind(), c.FontID, c.FontSize, math.NewVector2(valX, ty), c.KeyText)
	}

	r.ClearClip()
}

// setTags replaces the object's tag set from a comma-separated string, adding new tags
// and removing dropped ones through the object's tag API so the scene's tag index stays
// in sync.
func setTags(obj *core.Object, s string) error {
	want := make(map[string]bool)
	for _, t := range strings.Split(s, ",") {
		if t = strings.TrimSpace(t); t != "" {
			want[t] = true
		}
	}
	for tag := range want {
		if !obj.HasTag(tag) {
			obj.AddTag(tag)
		}
	}
	var remove []string
	for tag := range obj.Tags {
		if !want[tag] {
			remove = append(remove, tag)
		}
	}
	for _, tag := range remove {
		obj.RemoveTag(tag)
	}
	return nil
}
