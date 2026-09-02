package components

import (
	"reflect"

	"github.com/EnesBaytekin/imge/core"
	"github.com/EnesBaytekin/imge/core/math"
)

// argWindows is the set of currently-open component-args windows (one per opened
// component, so several can be open at once and dragged side by side). They are spawned
// and destroyed dynamically; the panels yield to any open window via the @UIManager's
// blocking occlusion (see pointerOwnedElsewhere).
var argWindows []*ComponentArgsComponent

// argsBasePosition is where the first spawned window sits; each additional window
// cascades down-right from it (they are draggable, so the user repositions freely).
var argsBasePosition = math.NewVector2(140, 52)

// ComponentArgsComponent is a spawned editor window that edits a single component's
// arguments in place. The inspector opens it when a component row is clicked; its
// "X" button closes it. Arguments are discovered by reflection over the component's
// exported, json-tagged fields (the same fields populated from a component's `args`
// object on load), shown as name/value rows, and written back through reflection
// when a row's edited value is committed. Edits mutate the live component, so the
// viewport reflects them on the next frame; they are not yet serialized back to the
// project (that lands with Save).
//
// Each editable argument's value is a real engine widget (@TextInput/@CheckBox/
// @ColorPicker) added as a component on this window object, so caret movement,
// Ctrl+word navigation, and the color picker all work instead of a hand-rolled
// edit buffer. The window itself draws the chrome (title, labels, read-only values,
// scrollbar) and polls the widgets for commits.
type ComponentArgsComponent struct {
	core.BaseUIComponent

	Background  math.Color `json:"background"`
	TitleText   math.Color `json:"title_text"`
	KeyText     math.Color `json:"key_text"`
	ValueText   math.Color `json:"value_text"`
	Accent      math.Color `json:"accent"` // title-bar highlight
	BorderColor math.Color `json:"border_color"`
	ErrorColor  math.Color `json:"error_color"` // committed-value-parse failure
	ScrollTrack math.Color `json:"scroll_track"`
	ScrollThumb math.Color `json:"scroll_thumb"`

	FontID    string  `json:"font_id"`
	FontSize  float64 `json:"font_size"`
	RowHeight float64 `json:"row_height"`

	target         core.Component // nil = window hidden
	scroll         float64
	scrollDragging bool
	scrollGrab     float64 // mouse Y offset within the thumb when the drag began

	dragging bool         // the title bar is being dragged to move the window
	dragGrab math.Vector2 // mouse offset from the window's top-left when the drag began

	closeHover bool // the close ("X") button is under the cursor

	bindings []fieldBinding // value widgets for editable args (rebuilt on Open/Close)
}

// titleH returns the title-bar height, shared by Draw and Update so their hit tests
// and layout never drift.
func (c *ComponentArgsComponent) titleH() float64 { return c.RowHeight + 8 }

// IsOpen reports whether the window is showing a component.
func (c *ComponentArgsComponent) IsOpen() bool { return c.target != nil }

// Open shows the window for the given component, resetting scroll and rebuilding its
// argument widgets.
func (c *ComponentArgsComponent) Open(comp core.Component) {
	c.target = comp
	c.scroll = 0
	c.rebuildRows()
}

// spawnArgsWindow opens a new component-args window for the given component. If a window
// for that exact component is already open, it returns that one instead of duplicating
// (a different component always gets its own window, so several can be open at once). The
// window is a fresh scene object, cascaded from the last open one.
func spawnArgsWindow(scene *core.Scene, comp core.Component) *ComponentArgsComponent {
	if comp == nil || scene == nil {
		return nil
	}
	for _, w := range argWindows {
		if w != nil && w.IsOpen() && w.target == comp {
			return w // already open for this component
		}
	}

	obj := core.NewObject("component_args")
	obj.UI = true
	obj.Layer = 3 // same layer as the inspector, so the two z-order against each other
	obj.Transform.Position = nextArgsWindowPos()

	args := &ComponentArgsComponent{}
	args.SetName("args")
	args.Width = 220
	args.Height = 180
	obj.AddComponent(args)

	if err := scene.AddObject(obj); err != nil {
		return nil
	}

	// The new object's Initialize is deferred to its first Scene.Update, but the host's
	// style defaults (font, colors, row height) are needed now to build the value
	// widgets. Initialize is idempotent, so running it early (and again on the deferred
	// pass) is safe.
	args.Initialize()
	args.Open(comp)
	argWindows = append(argWindows, args)
	raiseToFront(scene, obj) // a newly opened window appears on top
	return args
}

// destroyArgsWindow closes and removes an open window, freeing its scene object.
func destroyArgsWindow(w *ComponentArgsComponent) {
	if w == nil {
		return
	}
	for i, v := range argWindows {
		if v == w {
			argWindows = append(argWindows[:i], argWindows[i+1:]...)
			break
		}
	}
	w.target = nil
	if owner := w.GetOwner(); owner != nil {
		owner.Destroy()
	}
}

// closeAllArgsWindows destroys every open window. Called when the target project switches,
// since windows reference the previous project's live components.
func closeAllArgsWindows() {
	for _, w := range append([]*ComponentArgsComponent(nil), argWindows...) {
		destroyArgsWindow(w)
	}
	argWindows = nil
}

// nextArgsWindowPos cascades each new window down-right from the base position so newly
// opened windows don't stack exactly on top of the previous one.
func nextArgsWindowPos() math.Vector2 {
	n := len(argWindows)
	return argsBasePosition.Add(math.NewVector2(float64(n)*18, float64(n)*18))
}

func (c *ComponentArgsComponent) Initialize() {
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
	if c.ScrollTrack == (math.Color{}) {
		c.ScrollTrack = math.NewColor(0x2a, 0x30, 0x42, 0xff)
	}
	if c.ScrollThumb == (math.Color{}) {
		c.ScrollThumb = math.NewColor(0x4a, 0x55, 0x70, 0xff)
	}
	if c.FontSize <= 0 {
		c.FontSize = 6
	}
	if c.RowHeight <= 0 {
		c.RowHeight = 14
	}
	// A floating window is an opaque surface: it blocks pointer events so the @UIManager
	// occludes whatever is drawn behind it (see pointerOwnedElsewhere).
	if c.Blocking == nil {
		c.SetBlocking(true)
	}
}

func (c *ComponentArgsComponent) Update(ctx *core.Context) {
	if c.target == nil || ctx == nil || ctx.Input == nil {
		return
	}
	fields := enumerateArgs(c.target)
	mouse := ctx.Input.GetMousePosition()
	rect := c.Rect()

	// Commit any widget change and live-sync the model first, before any hover-gated
	// logic, so commits fire even after the pointer leaves the window.
	c.pollAndRefresh(ctx)

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

	// Recompute after any drag this frame so the hit tests below use the fresh position.
	rect = c.Rect()
	c.closeHover = math.NewRect(rect.X()+rect.Width()-18, rect.Y()+2, 14, 14).ContainsPoint(mouse)

	// A scrollbar drag keeps following the cursor even outside the window.
	if c.scrollDragging {
		if ctx.Input.IsMouseButtonPressed(core.MouseButtonLeft) {
			c.scroll = scrollFromThumb(c.scrollTrack(rect), float64(len(fields))*c.RowHeight, c.maxScroll(fields), mouse.Y, c.scrollGrab)
			c.layoutRows()
		} else {
			c.scrollDragging = false
		}
	}

	if !rect.ContainsPoint(mouse) {
		return
	}

	// Yield to a window drawn above this one (the @UIManager's blocking occlusion). A
	// click over a lower window must not leak through to a higher one beneath the
	// cursor, so a covered window ignores the pointer.
	if pointerOwnedElsewhere(c.GetScene(), c.GetOwner(), mouse) {
		return
	}

	// Wheel scrolls the argument list, unless a widget holds focus (so a focused
	// TextInput never scrolls out from under the caret).
	if s := ctx.Input.GetMouseScroll(); s.Y != 0 {
		if mgr := lookupUIManager(c.GetScene()); mgr == nil || !mgr.HasFocus() {
			c.scroll -= s.Y * c.RowHeight * 2
			c.clampScroll(fields)
			c.layoutRows()
		}
	}

	if ctx.Input.IsMouseButtonJustPressed(core.MouseButtonLeft) {
		// Close button: a small square in the title bar's top-right corner.
		closeRect := math.NewRect(rect.X()+rect.Width()-18, rect.Y()+2, 14, 14)
		if closeRect.ContainsPoint(mouse) {
			destroyArgsWindow(c)
			return
		}
		// Title bar (excluding the close button): start dragging the window.
		if math.NewRect(rect.X(), rect.Y(), rect.Width(), c.titleH()).ContainsPoint(mouse) {
			c.dragging = true
			c.dragGrab = mouse.Subtract(rect.Position)
			return
		}
		c.handleScrollbarPress(mouse, fields, rect)
	}
}

// clampScroll keeps the scroll offset within [0, maxScroll].
func (c *ComponentArgsComponent) clampScroll(fields []argField) {
	if max := c.maxScroll(fields); c.scroll > max {
		c.scroll = max
	}
	if c.scroll < 0 {
		c.scroll = 0
	}
}

// scrollTrack returns the scrollbar track rect along the window body's right edge.
func (c *ComponentArgsComponent) scrollTrack(rect math.Rect) math.Rect {
	const w = 6.0
	return math.NewRect(rect.X()+rect.Width()-w-2, rect.Y()+c.titleH(), w, rect.Height()-c.titleH())
}

// handleScrollbarPress consumes a click on the scrollbar: grabbing the thumb starts a
// drag, and clicking the track jumps the thumb (centered) to the cursor.
func (c *ComponentArgsComponent) handleScrollbarPress(mouse math.Vector2, fields []argField, rect math.Rect) {
	track := c.scrollTrack(rect)
	contentH := float64(len(fields)) * c.RowHeight
	max := c.maxScroll(fields)
	thumb, ok := scrollThumb(track, contentH, c.scroll, max)
	if !ok {
		return
	}
	if thumb.ContainsPoint(mouse) {
		c.scrollDragging = true
		c.scrollGrab = mouse.Y - thumb.Y()
		return
	}
	if track.ContainsPoint(mouse) {
		c.scroll = scrollFromThumb(track, contentH, max, mouse.Y, thumb.Height()/2)
		c.layoutRows()
	}
}

// maxScroll returns the scroll offset at which the last argument row is just visible.
func (c *ComponentArgsComponent) maxScroll(fields []argField) float64 {
	if len(fields) == 0 {
		return 0
	}
	body := c.Rect().Height() - c.titleH()
	if m := float64(len(fields))*c.RowHeight - body; m > 0 {
		return m
	}
	return 0
}

func (c *ComponentArgsComponent) Draw(r core.Renderer) {
	if c.target == nil {
		return
	}
	rect := c.Rect()
	if rect.Width() <= 0 || rect.Height() <= 0 {
		return
	}

	r.SetClipRect(rect)
	r.DrawRect(rect, c.Background)
	r.DrawRectOutline(rect, c.BorderColor, 1)

	// Line height is constant for a font+size, so measure once and reuse it for
	// every row's vertical centering.
	_, th := r.MeasureText("Ag", c.FontID, c.FontSize)

	// Title bar: "<name>  <kind>" on the left, "X" on the right.
	titleY := rect.Y() + (c.titleH()-th)/2
	r.DrawRect(math.NewRect(rect.X(), rect.Y(), rect.Width(), c.titleH()), c.Accent)
	r.DrawText(c.target.GetName()+"  "+c.target.GetKind(), c.FontID, c.FontSize, math.NewVector2(rect.X()+6, titleY), c.TitleText)
	xColor := c.TitleText
	if c.closeHover {
		xColor = c.ErrorColor // red on hover: a clear "close" affordance
	}
	r.DrawText("X", c.FontID, c.FontSize, math.NewVector2(rect.X()+rect.Width()-16, titleY), xColor)

	// Argument rows, clipped to the body below the title bar so partially-scrolled rows
	// are cut off rather than blinking out whole — a realistic scroll feel. Editable
	// rows are drawn by their widgets (layer 1, above this chrome); the host draws the
	// name label and any read-only value.
	fields := enumerateArgs(c.target)
	bodyTop := rect.Y() + c.titleH()
	body := math.NewRect(rect.X(), bodyTop, rect.Width(), rect.Height()-c.titleH())
	r.SetClipRect(body)
	valX := rect.X() + rect.Width()/2
	for i, f := range fields {
		y := bodyTop + float64(i)*c.RowHeight - c.scroll
		// Skip only rows fully scrolled out of the body; a partly-visible row is drawn
		// and clipped to `body`.
		if y+c.RowHeight <= bodyTop || y >= rect.Y()+rect.Height() {
			continue
		}
		ty := y + (c.RowHeight-th)/2
		if ty < y {
			ty = y
		}

		r.DrawText(f.name, c.FontID, c.FontSize, math.NewVector2(rect.X()+6, ty), c.KeyText)

		if !f.editable {
			r.DrawText(formatArg(f.value), c.FontID, c.FontSize, math.NewVector2(valX, ty), c.KeyText)
		}
	}

	// Scrollbar, drawn on top when the argument list overflows the body.
	r.SetClipRect(rect)
	track := c.scrollTrack(rect)
	if thumb, ok := scrollThumb(track, float64(len(fields))*c.RowHeight, c.scroll, c.maxScroll(fields)); ok {
		drawScrollbar(r, track, thumb, c.ScrollTrack, c.ScrollThumb)
	}

	r.ClearClip()
}

// ============================================================================
// Widget-host: build/rebuild the value widgets, poll commits, live-sync, layout.
// ============================================================================

// rebuildRows detaches the old argument widgets and builds fresh ones from the target
// component's current reflection schema. Called only on Open (a structural change);
// never per-frame, or a focused widget would lose focus every frame.
func (c *ComponentArgsComponent) rebuildRows() {
	c.removeWidgets()
	c.bindings = nil
	if c.target == nil {
		return
	}
	fields := enumerateArgs(c.target)
	rect := c.Rect()
	valX := rect.X() + rect.Width()/2
	valueW := rect.Width()/2 - 8 // leave room for the scrollbar
	for i := range fields {
		f := &fields[i]
		if !f.editable {
			continue
		}
		for _, b := range c.bindingsFor(*f) {
			b.row = i
			pw := partWidth(valueW, b.parts)
			x := partX(valX, b.col, pw)
			y := rect.Y() + c.titleH() + float64(i)*c.RowHeight - c.scroll
			b.widget = makeFieldWidget(&b, c.GetOwner(), math.NewVector2(x, y), pw, c.RowHeight, c.FontID, c.FontSize, c.ValueText)
			c.bindings = append(c.bindings, b)
		}
	}
}

// bindingsFor builds the field binding(s) for one reflection-discovered argument: a
// single widget for scalar/bool/color fields, or one widget per component for
// math.Vector2 (x, y) and math.Border (left, top, right, bottom), laid out side by side.
// The closures capture the live reflect.Value (which points into the target component),
// so get/apply always read/write the current value.
func (c *ComponentArgsComponent) bindingsFor(f argField) []fieldBinding {
	fv := f.value
	t := fv.Type()

	switch {
	case t == reflect.TypeOf(math.Color{}):
		b := fieldBinding{
			key:   f.name,
			parts: 1,
			kind:  kindColor,
			get:   func() string { return formatArg(fv) },
			apply: func(s string) error { return setArg(fv, s) },
		}
		b.getColor = func() math.Color { return fv.Interface().(math.Color) }
		b.old = b.get()
		return []fieldBinding{b}
	case fv.Kind() == reflect.Bool:
		b := fieldBinding{
			key:   f.name,
			parts: 1,
			kind:  kindCheck,
			get:   func() string { return formatArg(fv) },
			apply: func(s string) error { return setArg(fv, s) },
		}
		b.getBool = func() bool { return fv.Bool() }
		b.old = b.get()
		return []fieldBinding{b}
	case t == reflect.TypeOf(math.Vector2{}):
		return []fieldBinding{
			c.vectorPart(f, 0, "x", 2),
			c.vectorPart(f, 1, "y", 2),
		}
	case t == reflect.TypeOf(math.Border{}):
		return []fieldBinding{
			c.vectorPart(f, 0, "left", 4),
			c.vectorPart(f, 1, "top", 4),
			c.vectorPart(f, 2, "right", 4),
			c.vectorPart(f, 3, "bottom", 4),
		}
	default:
		b := fieldBinding{
			key:   f.name,
			parts: 1,
			kind:  kindText,
			get:   func() string { return formatArg(fv) },
			apply: func(s string) error { return setArg(fv, s) },
		}
		b.old = b.get()
		return []fieldBinding{b}
	}
}

// vectorPart builds one scalar text binding for a component of a math.Vector2 or
// math.Border field. The closure captures that component's reflect.Value (a float64
// pointing into the target component), so each part reads/writes just its own value.
func (c *ComponentArgsComponent) vectorPart(f argField, idx int, suffix string, parts int) fieldBinding {
	fv := f.value.Field(idx)
	b := fieldBinding{
		key:   f.name + "_" + suffix,
		col:   idx,
		parts: parts,
		kind:  kindText,
		get:   func() string { return formatFloat(fv.Float()) },
		apply: func(s string) error {
			v, err := parseFloat(s)
			if err != nil {
				return err
			}
			fv.SetFloat(v)
			return nil
		},
	}
	b.old = b.get()
	return b
}

// removeWidgets detaches every argument widget from the window object. RemoveComponent
// is synchronous and unsubscribes events, so this is safe to call mid-frame.
func (c *ComponentArgsComponent) removeWidgets() {
	removeWidgets(c.bindings, c.GetOwner())
}

// pollAndRefresh runs the per-frame widget pass (commit polling + live-sync), then
// re-lays out the value widgets (reposition for scroll).
func (c *ComponentArgsComponent) pollAndRefresh(ctx *core.Context) {
	syncBindings(c.bindings, ctx, c.ValueText, c.ErrorColor)
	c.layoutRows()
}

// layoutRows repositions each argument widget to its row and hides scrolled-out rows.
func (c *ComponentArgsComponent) layoutRows() {
	rect := c.Rect()
	valueW := rect.Width()/2 - 8 // leave room for the scrollbar
	layoutWidgets(c.bindings, c.GetOwner(), rect.Y()+c.titleH(), rect.X()+rect.Width()/2, valueW, c.scroll, c.RowHeight, rect.Y()+rect.Height())
}
