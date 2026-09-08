package components

import (
	"os"
	"path/filepath"
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
	hoverPlus   bool           // the "+" add-component button is under the cursor
	hoverX      int            // component row whose "x" remove button is under the cursor (-1 = none)
	hoverDup    int            // component row whose "=" duplicate button is under the cursor (-1 = none)
	hoverAction bool           // the make-unique / make-object title-bar button is under the cursor
	hoverEdit   bool           // the "edit" (open object editor) title-bar button is under the cursor

	tagInput    *TextInputComponent // inline "add tag" field (Enter adds a tag)
	tagScroll   float64             // tags-list scroll offset (pixels, 0 = top)
	hoverTagAdd bool                // the "+" add-tag button is under the cursor
	hoverTagX   int                 // tag row whose "x" remove button is under the cursor (-1 = none)
	lastSelComp core.Component      // last selected component (object editor), for args-window sync
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

// tagListVisible is the number of tag rows shown before the tags list scrolls.
const tagListVisible = 5

// tagHeaderY returns the content-space y (relative to the panel's top) of the "TAGS"
// header row, given the number of property rows above it.
func (c *InspectorComponent) tagHeaderY(nProps int) float64 {
	return c.titleH() + float64(nProps)*c.RowHeight
}

// tagListY returns the content-space y of the first tag row (below the header).
func (c *InspectorComponent) tagListY(nProps int) float64 {
	return c.tagHeaderY(nProps) + c.RowHeight
}

// tagListH returns the visible height of the scrollable tag list.
func (c *InspectorComponent) tagListH() float64 {
	return tagListVisible * c.RowHeight
}

// compStart returns the content-space y (relative to the panel's top) where the first
// component row begins, given the number of property rows above it. The COMPONENTS
// header sits one row above this, and the tags section sits between the properties and
// the components.
func (c *InspectorComponent) compStart(nProps int) float64 {
	return c.tagListY(nProps) + c.tagListH() + c.RowHeight
}

// tagAddRect returns the inline add-tag input rect within the TAGS header row.
func (c *InspectorComponent) tagAddRect(rect math.Rect, nProps int) math.Rect {
	y := rect.Y() + c.tagHeaderY(nProps)
	const labelW, plusW = 40.0, 18.0
	return math.NewRect(rect.X()+labelW, y+1, rect.Width()-labelW-plusW-4, c.RowHeight-2)
}

// tagPlusRect returns the "+" add-tag button rect at the header row's right edge.
func (c *InspectorComponent) tagPlusRect(rect math.Rect, nProps int) math.Rect {
	y := rect.Y() + c.tagHeaderY(nProps)
	const s = 14.0
	return math.NewRect(rect.X()+rect.Width()-18, y+(c.RowHeight-s)/2, s, s)
}

// tagXRect returns the "x" remove-button strip at the right edge of a tag row.
func (c *InspectorComponent) tagXRect(rect math.Rect, rowY float64) math.Rect {
	const w = 14.0
	return math.NewRect(rect.X()+rect.Width()-w, rowY, w, c.RowHeight)
}

// tagMaxScroll returns the scroll offset at which the last tag row is just visible.
func (c *InspectorComponent) tagMaxScroll(nTags int) float64 {
	if m := float64(nTags)*c.RowHeight - c.tagListH(); m > 0 {
		return m
	}
	return 0
}

// clampTagScroll keeps the tags-list scroll offset within [0, tagMaxScroll].
func (c *InspectorComponent) clampTagScroll(nTags int) {
	if max := c.tagMaxScroll(nTags); c.tagScroll > max {
		c.tagScroll = max
	}
	if c.tagScroll < 0 {
		c.tagScroll = 0
	}
}

// plusRect returns the "+" add-component button rect in the COMPONENTS header row's
// top-right corner, computed from the header row (one row above the list).
func (c *InspectorComponent) plusRect(rect math.Rect, nProps int) math.Rect {
	y := rect.Y() + c.compStart(nProps) - c.RowHeight // the header row
	const s = 14.0
	return math.NewRect(rect.X()+rect.Width()-18, y+(c.RowHeight-s)/2, s, s)
}

// xRect returns the "x" remove button strip at the right edge of a component row.
func (c *InspectorComponent) xRect(rect math.Rect, rowY float64) math.Rect {
	const w = 14.0
	return math.NewRect(rect.X()+rect.Width()-w, rowY, w, c.RowHeight)
}

// dupRect returns the "=" duplicate button strip at the right edge of a component row,
// immediately left of the "x" remove strip so the two controls sit side by side.
func (c *InspectorComponent) dupRect(rect math.Rect, rowY float64) math.Rect {
	const w = 14.0
	xr := c.xRect(rect, rowY)
	return math.NewRect(xr.X()-w, rowY, w, c.RowHeight)
}

// actionRect returns the title-bar button for the two-way provenance conversion: "make
// unique" (drop the file reference, inline the definition) when the object is
// file-referenced, or "make object" (save the inline object as a .obj) otherwise.
func (c *InspectorComponent) actionRect(rect math.Rect) math.Rect {
	const w = 46.0
	return math.NewRect(rect.X()+rect.Width()-w-6, rect.Y()+(c.titleH()-16)/2, w, 16)
}

// editRect returns the title-bar "edit" button, immediately left of the action button.
// It opens the object editor for a file-referenced object.
func (c *InspectorComponent) editRect(rect math.Rect) math.Rect {
	ar := c.actionRect(rect)
	const w = 34.0
	return math.NewRect(ar.X()-w-4, ar.Y(), w, 16)
}

// actionLabel returns the label for the title-bar provenance button.
func (c *InspectorComponent) actionLabel(obj *core.Object) string {
	if obj != nil && obj.File != "" {
		return "unique"
	}
	return "object"
}

// props builds the editable property list for an object. Each setter applies the parsed
// value through the object's own API, so side effects (scene name/tag/depth indexing)
// stay consistent. Only "ui" is a bool (@CheckBox); the rest are text. "active" has no
// setter and stays read-only (the viewport owns activation).
func (c *InspectorComponent) props(obj *core.Object, inObjEditor bool) []prop {
	if obj == nil {
		return nil
	}
	out := []prop{
		{"name", func() string { return obj.Name }, func(s string) error { return obj.SetName(s) }, kindText, nil},
		{"file", func() string {
			if obj.File != "" {
				return obj.File
			}
			return "inline"
		}, nil, kindText, nil},
	}
	// The transform and active flag are scene-owned (per-instance) — a .obj has none —
	// so the object editor hides them. "ui" is also hidden there: the editor forces the
	// template to world-space to render it, so toggling it would make the object vanish.
	if !inObjEditor {
		out = append(out,
			prop{"x", func() string { return formatFloat(obj.GetPosition().X) }, func(s string) error {
				f, err := parseFloat(s)
				if err != nil {
					return err
				}
				obj.SetPosition(f, obj.GetPosition().Y)
				return nil
			}, kindText, nil},
			prop{"y", func() string { return formatFloat(obj.GetPosition().Y) }, func(s string) error {
				f, err := parseFloat(s)
				if err != nil {
					return err
				}
				obj.SetPosition(obj.GetPosition().X, f)
				return nil
			}, kindText, nil},
			prop{"rotation", func() string { return formatFloat(math.RadiansToDegrees(obj.GetRotation())) }, func(s string) error {
				f, err := parseFloat(s)
				if err != nil {
					return err
				}
				obj.SetRotation(math.DegreesToRadians(f))
				return nil
			}, kindText, nil},
			prop{"scale", func() string { sc := obj.GetScale(); return formatFloat(sc.X) + ", " + formatFloat(sc.Y) }, func(s string) error {
				x, y, err := parseTwoFloats(s)
				if err != nil {
					return err
				}
				obj.SetScale(x, y)
				return nil
			}, kindText, nil},
		)
	}
	out = append(out,
		prop{"layer", func() string { return strconv.Itoa(obj.GetLayer()) }, func(s string) error {
			n, err := strconv.Atoi(strings.TrimSpace(s))
			if err != nil {
				return err
			}
			obj.SetLayer(n)
			return nil
		}, kindText, nil},
		prop{"depth", func() string { return formatFloat(obj.GetDepth()) }, func(s string) error {
			f, err := parseFloat(s)
			if err != nil {
				return err
			}
			return obj.SetDepth(f)
		}, kindText, nil},
	)
	if !inObjEditor {
		out = append(out,
			prop{"ui", func() string { return strconv.FormatBool(obj.UI) }, func(s string) error {
				b, err := parseBool(s)
				if err != nil {
					return err
				}
				obj.UI = b
				return nil
			}, kindCheck, func() bool { return obj.UI }},
			prop{"active", func() string { return strconv.FormatBool(obj.Active) }, nil, kindText, nil},
		)
	}
	return out
}

// buildBindings converts the object's editable properties into field bindings, one
// widget per property. Read-only rows (nil setter) are skipped — the host draws them.
func (c *InspectorComponent) buildBindings(obj *core.Object) []fieldBinding {
	props := c.props(obj, objectEditorActive())
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
		// name is .obj-owned on a file-referenced object, so a commit (and any undo/redo)
		// writes through to the shared template. (tags are edited by their own add/remove
		// controls, which persist directly.)
		if p.label == "name" {
			b.afterApply = func() { persistObjectFile(obj) }
		}
		out = append(out, b)
	}
	return out
}

// rebuildRows detaches the old property widgets and builds fresh ones for the newly
// selected object. Called only on selection change (a structural change); never
// per-frame, or a focused widget would lose focus every frame.
func (c *InspectorComponent) rebuildRows(obj *core.Object) {
	c.removeWidgets()
	c.rebuildTagInput(obj)
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

// rebuildTagInput (re)builds the inline "add tag" @TextInput as a child of the panel
// object. It is detached and rebuilt on selection change (matching rebuildRows), and
// repositioned per-frame by layoutRows.
func (c *InspectorComponent) rebuildTagInput(obj *core.Object) {
	if c.tagInput != nil {
		c.GetOwner().RemoveComponent(c.tagInput.GetName())
		c.tagInput = nil
	}
	if obj == nil {
		return
	}
	ti := &TextInputComponent{}
	ti.FontID = c.FontID
	ti.Size = c.FontSize
	ti.TextColor = c.ValueText
	ti.PlaceholderColor = c.KeyText
	ti.BackgroundColor = fieldBackground
	ti.OutlineColor = fieldOutline
	ti.OutlineThickness = 1
	ti.Placeholder = "add tag..."
	ti.Height = c.RowHeight - 2
	ti.DrawLayer = 1
	ti.SetName("tags_add")
	if err := c.GetOwner().AddComponent(ti); err != nil {
		return
	}
	ti.Initialize()
	c.tagInput = ti
}

// commitTagInput adds the current text of the add-tag input as a new tag and clears the
// input, so Enter (or the "+" button) both add-and-reset in one gesture.
func (c *InspectorComponent) commitTagInput(obj *core.Object) {
	if c.tagInput == nil {
		return
	}
	tag := strings.TrimSpace(c.tagInput.Text)
	c.tagInput.Text = ""
	if tag == "" {
		return
	}
	addTag(obj, tag)
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
	if c.tagInput != nil {
		obj := inspectorTarget(c.GetScene())
		ar := c.tagAddRect(rect, len(c.props(obj, objectEditorActive())))
		if ar.Width() > 0 {
			c.tagInput.Width = ar.Width()
			c.tagInput.Height = ar.Height()
			c.tagInput.SetOffset(ar.Position.Subtract(c.GetOwner().Transform.Position))
		}
	}
}

func (c *InspectorComponent) Update(ctx *core.Context) {
	if ctx == nil || ctx.Input == nil {
		return
	}
	// A modal or an open menu bar is up: this panel is inert.
	if modalOpen() || menusOpen() {
		return
	}
	// The inspector shows the viewport's selection normally, but while the object editor
	// is open it redirects to that editor's object, so a .obj can be edited through the
	// same inspector rather than a separate one.
	obj := inspectorTarget(c.GetScene())

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

	// Add-tag input: Enter commits a new tag (the input may be focused even when the
	// pointer has left the panel).
	if c.tagInput != nil && c.tagInput.IsFocused() && ctx.Input.IsKeyJustPressed(core.KeyEnter) {
		c.commitTagInput(obj)
	}

	// Selected component (object editor): open its args window when the selection
	// changes, so the component being edited always has its window up.
	if sel := selectedComponent(obj); sel != c.lastSelComp {
		c.lastSelComp = sel
		if sel != nil {
			spawnArgsWindow(c.GetScene(), sel)
		}
	}

	rect := c.Rect()
	mouse := ctx.Input.GetMousePosition()
	if !rect.ContainsPoint(mouse) {
		c.hoverComp = -1
		c.hoverPlus = false
		c.hoverX = -1
		c.hoverDup = -1
		c.hoverAction = false
		c.hoverEdit = false
		c.hoverTagAdd = false
		c.hoverTagX = -1
		return
	}

	// Yield to a window drawn above the inspector (e.g. a component-args window
	// dragged over it) — the @UIManager's blocking occlusion, see pointerOwnedElsewhere.
	if pointerOwnedElsewhere(c.GetScene(), c.GetOwner(), mouse) {
		c.hoverComp = -1
		c.hoverPlus = false
		c.hoverX = -1
		c.hoverDup = -1
		c.hoverAction = false
		c.hoverEdit = false
		c.hoverTagAdd = false
		c.hoverTagX = -1
		return
	}

	var comps []core.Component
	if obj != nil {
		comps = obj.ComponentsInDrawOrder()
	}
	inObjEditor := objectEditorActive()
	props := c.props(obj, inObjEditor)
	compY := rect.Y() + c.compStart(len(props))
	available := rect.Height() - c.compStart(len(props))

	// Track the component row (and its "x" strip) plus the "+" button under the
	// cursor, so the COMPONENTS list reads as clickable.
	c.hoverComp = -1
	c.hoverX = -1
	c.hoverDup = -1
	c.hoverPlus = c.plusRect(rect, len(props)).ContainsPoint(mouse)
	c.hoverAction = !inObjEditor && obj != nil && c.actionRect(rect).ContainsPoint(mouse)
	c.hoverEdit = !inObjEditor && obj != nil && obj.File != "" && c.editRect(rect).ContainsPoint(mouse)
	for i := range comps {
		y := compY + float64(i)*c.RowHeight - c.scroll
		if mouse.Y >= y && mouse.Y < y+c.RowHeight {
			c.hoverComp = i
			if c.xRect(rect, y).ContainsPoint(mouse) {
				c.hoverX = i
			}
			if c.dupRect(rect, y).ContainsPoint(mouse) {
				c.hoverDup = i
			}
			break
		}
	}

	// Tags section: the "+" add button and each tag row's "x" remove button under the
	// cursor, so the tags list reads as clickable.
	tags := sortedTags(obj)
	c.hoverTagAdd = false
	c.hoverTagX = -1
	if obj != nil {
		c.hoverTagAdd = c.tagPlusRect(rect, len(props)).ContainsPoint(mouse)
		listY := rect.Y() + c.tagListY(len(props))
		for i := range tags {
			y := listY + float64(i)*c.RowHeight - c.tagScroll
			if y+c.RowHeight < listY || y > listY+c.tagListH() {
				continue
			}
			if mouse.Y >= y && mouse.Y < y+c.RowHeight {
				if c.tagXRect(rect, y).ContainsPoint(mouse) {
					c.hoverTagX = i
				}
				break
			}
		}
	}
	c.clampTagScroll(len(tags))

	// Wheel scrolls the tags list when the cursor is over it, otherwise the component
	// list — unless a widget holds focus (so a focused TextInput never scrolls out from
	// under the caret).
	if s := ctx.Input.GetMouseScroll(); s.Y != 0 {
		if mgr := lookupUIManager(c.GetScene()); mgr == nil || !mgr.HasFocus() {
			if obj != nil {
				listY := rect.Y() + c.tagListY(len(props))
				if mouse.Y >= listY && mouse.Y < listY+c.tagListH() {
					c.tagScroll -= s.Y * c.RowHeight * 2
					c.clampTagScroll(len(tags))
				} else {
					c.scroll -= s.Y * c.RowHeight * 2
					if max := c.maxScroll(len(comps), available); c.scroll > max {
						c.scroll = max
					}
					if c.scroll < 0 {
						c.scroll = 0
					}
				}
			} else {
				c.scroll -= s.Y * c.RowHeight * 2
				if max := c.maxScroll(len(comps), available); c.scroll > max {
					c.scroll = max
				}
				if c.scroll < 0 {
					c.scroll = 0
				}
			}
		}
	}

	if !ctx.Input.IsMouseButtonJustPressed(core.MouseButtonLeft) {
		return
	}

	// Title-bar "edit" button: open the object editor for this file-referenced object.
	if c.hoverEdit {
		if obj != nil && obj.File != "" {
			spawnObjectEditor(c.GetScene(), obj.File)
		}
		return
	}

	// Title-bar provenance button: convert file↔inline (make unique / make object).
	if c.hoverAction {
		if obj != nil && obj.File != "" {
			makeUnique(obj)
		} else {
			makeObject(c.GetScene(), obj)
		}
		return
	}

	// "+" button: open the add-component modal for the selected object.
	if c.hoverPlus {
		spawnAddComponentPanel(c.GetScene(), obj)
		return
	}
	// "x" strip: open the remove-confirm modal for that component. The component is
	// captured now (before it can be removed), so the confirm callback removes the
	// right one.
	if c.hoverX >= 0 && c.hoverX < len(comps) {
		comp := comps[c.hoverX]
		spawnConfirmDialog(c.GetScene(), "Delete \""+comp.GetName()+"\"?", func() {
			removeComponent(comp)
		})
		return
	}

	// "=" strip: duplicate that component (a fresh copy with a unique name).
	if c.hoverDup >= 0 && c.hoverDup < len(comps) {
		duplicateComponent(obj, comps[c.hoverDup])
		return
	}

	// Tags section: "+" commits the text in the inline add-tag field, "x" removes that
	// tag. (The field itself is a real widget, so clicking it focuses it, not routed here.)
	if c.hoverTagAdd {
		c.commitTagInput(obj)
		return
	}
	if c.hoverTagX >= 0 && c.hoverTagX < len(tags) {
		removeTag(obj, tags[c.hoverTagX])
		return
	}

	// Component rows: click to open the args window, and in the object editor also move
	// the editor selection to that component (so clicking a row selects it in the editor
	// view). Property rows are the widgets' job (a click focuses the TextInput or toggles
	// the CheckBox).
	for i, comp := range comps {
		y := compY + float64(i)*c.RowHeight - c.scroll
		if mouse.Y >= y && mouse.Y < y+c.RowHeight {
			if inObjEditor {
				activeObjectEditor.selectedComp = comp
			}
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

	obj := inspectorTarget(c.GetScene())
	if obj == nil {
		r.DrawText("no selection", c.FontID, c.FontSize, math.NewVector2(rect.X()+6, rect.Y()+c.titleH()+2), c.KeyText)
		r.ClearClip()
		return
	}

	// Title-bar buttons. In the object editor (editing a .obj), the provenance action
	// (make unique / make object) is hidden — that object is not a scene object there.
	// The "edit" button opens the object editor for a file-referenced scene object.
	inObjEditor := objectEditorActive()
	if !inObjEditor && obj.File != "" {
		er := c.editRect(rect)
		if c.hoverEdit {
			r.DrawRect(er, c.Accent.Lerp(math.White, 0.14))
		} else {
			r.DrawRect(er, c.Background.Lerp(math.White, 0.08))
		}
		ew, eh := r.MeasureText("edit", c.FontID, c.FontSize)
		r.DrawText("edit", c.FontID, c.FontSize, math.NewVector2(er.X()+(er.Width()-ew)/2, er.Y()+(er.Height()-eh)/2), c.TitleText)
	}
	if !inObjEditor {
		ar := c.actionRect(rect)
		if c.hoverAction {
			r.DrawRect(ar, c.Accent.Lerp(math.White, 0.14))
		} else {
			r.DrawRect(ar, c.Background.Lerp(math.White, 0.08))
		}
		label := c.actionLabel(obj)
		aw, ah := r.MeasureText(label, c.FontID, c.FontSize)
		r.DrawText(label, c.FontID, c.FontSize, math.NewVector2(ar.X()+(ar.Width()-aw)/2, ar.Y()+(ar.Height()-ah)/2), c.TitleText)
	}

	// Property rows: the host draws the name label and any read-only value; editable
	// values are drawn by their widgets (layer 1, above this chrome).
	props := c.props(obj, inObjEditor)
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

	// Tags section: a "TAGS" header with an inline add-tag field (the widget draws itself
	// on layer 1 above this chrome) and a "+" button, then a scrollable list of tag rows
	// each with an "x" remove button.
	tagHeaderY := rect.Y() + c.tagHeaderY(len(props))
	tsY := tagHeaderY + (c.RowHeight-th)/2
	r.DrawText("TAGS", c.FontID, c.FontSize, math.NewVector2(rect.X()+6, tsY), c.Section)
	tplus := c.tagPlusRect(rect, len(props))
	if c.hoverTagAdd {
		r.DrawRect(tplus, c.Background.Lerp(math.White, 0.12))
	}
	tpw, tph := r.MeasureText("+", c.FontID, c.FontSize)
	r.DrawText("+", c.FontID, c.FontSize, math.NewVector2(tplus.X()+(tplus.Width()-tpw)/2, tplus.Y()+(tplus.Height()-tph)/2), c.Section)

	tags := sortedTags(obj)
	tagListTop := rect.Y() + c.tagListY(len(props))
	r.SetClipRect(math.NewRect(rect.X(), tagListTop, rect.Width(), c.tagListH()))
	for i, tag := range tags {
		y := tagListTop + float64(i)*c.RowHeight - c.tagScroll
		if y+c.RowHeight < tagListTop || y > tagListTop+c.tagListH() {
			continue
		}
		ty := y + (c.RowHeight-th)/2
		if ty < y {
			ty = y
		}
		r.DrawText(tag, c.FontID, c.FontSize, math.NewVector2(rect.X()+6, ty), c.ValueText)
		xr := c.tagXRect(rect, y)
		xColor := c.KeyText
		if i == c.hoverTagX {
			r.DrawRect(xr, c.ErrorColor)
			xColor = c.TitleText
		}
		txw, txh := r.MeasureText("x", c.FontID, c.FontSize)
		r.DrawText("x", c.FontID, c.FontSize, math.NewVector2(xr.X()+(xr.Width()-txw)/2, y+(c.RowHeight-txh)/2), xColor)
	}
	r.SetClipRect(rect)

	// Components section header — one row above the scrollable list, with a "+"
	// add-component button in its top-right corner.
	compY := rect.Y() + c.compStart(len(props))
	sty := (compY - c.RowHeight) + (c.RowHeight-th)/2
	r.DrawText("COMPONENTS", c.FontID, c.FontSize, math.NewVector2(rect.X()+6, sty), c.Section)
	plus := c.plusRect(rect, len(props))
	if c.hoverPlus {
		r.DrawRect(plus, c.Background.Lerp(math.White, 0.12))
	}
	pw, ph := r.MeasureText("+", c.FontID, c.FontSize)
	r.DrawText("+", c.FontID, c.FontSize, math.NewVector2(plus.X()+(plus.Width()-pw)/2, plus.Y()+(plus.Height()-ph)/2), c.Section)

	// Component rows (scrollable): name + kind + an "x" remove button, clipped to the
	// list region below the header so scrolled-out rows don't bleed over it.
	comps := obj.ComponentsInDrawOrder()
	sel := selectedComponent(obj)
	r.SetClipRect(math.NewRect(rect.X(), compY, rect.Width(), rect.Height()-(compY-rect.Y())))
	for i, comp := range comps {
		y := compY + float64(i)*c.RowHeight - c.scroll
		if y+c.RowHeight < compY || y > rect.Y()+rect.Height() {
			continue
		}
		if sel != nil && sel == comp {
			r.DrawRect(math.NewRect(rect.X(), y, rect.Width(), c.RowHeight), c.Accent.Lerp(math.White, 0.10))
		} else if i == c.hoverComp {
			r.DrawRect(math.NewRect(rect.X(), y, rect.Width(), c.RowHeight), c.Background.Lerp(math.White, 0.07))
		}
		ty := y + (c.RowHeight-th)/2
		if ty < y {
			ty = y
		}
		r.DrawText(comp.GetName(), c.FontID, c.FontSize, math.NewVector2(rect.X()+6, ty), c.ValueText)
		r.DrawText(comp.GetKind(), c.FontID, c.FontSize, math.NewVector2(valX, ty), c.KeyText)

		// "!" badge when the component declares required kinds (core.Dependable) that
		// are missing from this object — a compact cue that a dependency is unmet. The
		// args window spells out the full "requires" list when the row is opened.
		if deps := componentRequires(comp); len(missingDependencies(obj, deps)) > 0 {
			wr := math.NewRect(c.dupRect(rect, y).X()-14, y, 14, c.RowHeight)
			ww, wh := r.MeasureText("!", c.FontID, c.FontSize)
			r.DrawText("!", c.FontID, c.FontSize, math.NewVector2(wr.X()+(wr.Width()-ww)/2, y+(c.RowHeight-wh)/2), c.ErrorColor)
		}

		// "=" duplicate button in the strip left of the "x" remove button.
		dr := c.dupRect(rect, y)
		dupColor := c.KeyText
		if i == c.hoverDup {
			r.DrawRect(dr, c.Accent)
			dupColor = c.TitleText
		}
		dw, dh := r.MeasureText("=", c.FontID, c.FontSize)
		r.DrawText("=", c.FontID, c.FontSize, math.NewVector2(dr.X()+(dr.Width()-dw)/2, y+(c.RowHeight-dh)/2), dupColor)

		// "x" remove button in the right-edge strip (red on hover).
		xr := c.xRect(rect, y)
		xColor := c.KeyText
		if i == c.hoverX {
			r.DrawRect(xr, c.ErrorColor)
			xColor = c.TitleText
		}
		xw, xh := r.MeasureText("x", c.FontID, c.FontSize)
		r.DrawText("x", c.FontID, c.FontSize, math.NewVector2(xr.X()+(xr.Width()-xw)/2, y+(c.RowHeight-xh)/2), xColor)
	}

	r.ClearClip()
}

// sortedTags returns the object's tags sorted by name, for stable list rendering.
func sortedTags(obj *core.Object) []string {
	if obj == nil {
		return nil
	}
	tags := make([]string, 0, len(obj.Tags))
	for tag := range obj.Tags {
		tags = append(tags, tag)
	}
	sort.Strings(tags)
	return tags
}

// addTag adds a tag to obj and records an undoable entry that removes it. It is a no-op
// when the tag is empty or already present. The write-through (persistObjectFile) keeps
// a file-referenced object's .obj in sync.
func addTag(obj *core.Object, tag string) {
	if obj == nil || tag == "" || obj.HasTag(tag) {
		return
	}
	obj.AddTag(tag)
	history.record(
		"added tag "+tag,
		func() { obj.RemoveTag(tag); persistObjectFile(obj) },
		func() { obj.AddTag(tag); persistObjectFile(obj) },
		true,
	)
	persistObjectFile(obj)
}

// removeTag removes a tag from obj and records an undoable entry that restores it.
func removeTag(obj *core.Object, tag string) {
	if obj == nil || tag == "" || !obj.HasTag(tag) {
		return
	}
	obj.RemoveTag(tag)
	history.record(
		"removed tag "+tag,
		func() { obj.AddTag(tag); persistObjectFile(obj) },
		func() { obj.RemoveTag(tag); persistObjectFile(obj) },
		true,
	)
	persistObjectFile(obj)
}

// makeUnique drops a file-referenced object's File reference, inlining its current
// definition (name/tags/components) so the scene serializes it inline from now on. The
// in-memory object is unchanged apart from provenance, so the conversion is lossless.
// Records an undo entry that restores the reference.
func makeUnique(obj *core.Object) {
	if obj == nil || obj.File == "" {
		return
	}
	rel := obj.File
	obj.File = ""
	history.record(
		"made object unique",
		func() { obj.File = rel },
		func() { obj.File = "" },
		true,
	)
}

// makeObject prompts for a filename and saves an inline object's current definition as a
// .obj under the project's objects/ directory, recording its file reference so it becomes
// a file-referenced object. The transform stays scene-owned, so placement is unchanged.
// The prompt auto-appends ".obj" when missing and refuses to overwrite an existing file.
func makeObject(scene *core.Scene, obj *core.Object) {
	if obj == nil || obj.File != "" {
		return
	}
	vp := lookupViewport(scene)
	if vp == nil || vp.CurrentProject() == "" {
		return
	}
	spawnTextPrompt(scene, "SAVE AS OBJECT", obj.Name, "Save", func(name string) string {
		name = strings.TrimSpace(name)
		if name == "" {
			return "name is required"
		}
		if !strings.HasSuffix(name, ".obj") {
			name += ".obj"
		}
		rel := filepath.Join("objects", filepath.Base(name))
		abs := filepath.Join(vp.CurrentProject(), rel)
		if _, err := os.Stat(abs); err == nil {
			return "already exists: " + rel
		}
		if err := obj.SaveToFile(abs); err != nil {
			return err.Error()
		}
		obj.File = rel
		history.record(
			"made object from file",
			func() { obj.File = "" },
			func() { obj.File = rel },
			true,
		)
		return ""
	})
}
