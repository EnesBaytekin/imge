package components

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/EnesBaytekin/imge/core"
	"github.com/EnesBaytekin/imge/core/math"
)

// argWindows is the set of currently-open component-args windows (one per opened
// component, so several can be open at once and dragged side by side). They are spawned
// and destroyed dynamically; the panels yield to any open window via the @UIManager's
// blocking occlusion (see pointerOwnedElsewhere).
var argWindows []*ComponentArgsComponent

// focusedArgs is the component-args window the user most recently interacted with. ESC
// closes it (unless a widget inside holds keyboard focus, which takes priority). It is
// nil when no window is focused.
var focusedArgs *ComponentArgsComponent

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

	// spriteTextured records whether the Sprite layout currently shows its texture-only
	// rows (frame size / frame #), so a texture commit that crosses empty↔set rebuilds the
	// rows (those fields are meaningless without a texture).
	spriteTextured bool

	// spriteManaged records whether an Animator currently drives this Sprite (visible/flip/
	// frame are then locked). Detecting an Animator being added/removed — or a clip naming
	// this sprite — mid-session rebuilds the rows so the checkboxes lock/unlock at once.
	spriteManaged bool
}

// titleH returns the title-bar height, shared by Draw and Update so their hit tests
// and layout never drift.
func (c *ComponentArgsComponent) titleH() float64 { return c.RowHeight + 8 }

// requiresH returns the height reserved for the "requires" footer line below the
// argument list — a separator gap plus one text row — or 0 when the target component
// declares no dependencies (via core.Dependable). The footer lives outside the scroll
// area, so the body's usable height is reduced by this amount.
func (c *ComponentArgsComponent) requiresH() float64 {
	if c.target == nil || len(componentRequires(c.target)) == 0 {
		return 0
	}
	return c.RowHeight + 4
}

// isSprite reports whether the window is editing a Sprite, which gets a custom grouped
// layout (common fields + offset + appearance always visible, then a texture section whose
// frame fields appear only once a texture is set).
func (c *ComponentArgsComponent) isSprite() bool {
	_, ok := c.target.(*Sprite)
	return ok
}

// spriteRowBase is the number of always-present rows in the Sprite layout: name,
// draw_layer, group, separator, offset, visible, flip, color, size, separator, texture.
const spriteRowBase = 11

// spriteFrameRows is the number of texture-only rows appended after spriteRowBase
// (frame size and frame #), present only when the sprite has a texture.
const spriteFrameRows = 2

// rowCount returns the number of visible rows (the name row + field rows + separators).
// The Sprite layout appends its texture-only rows only when a texture is set; other
// components use their reflected field count.
func (c *ComponentArgsComponent) rowCount() int {
	if c.isSprite() {
		if spr, _ := c.target.(*Sprite); spr != nil && spr.Texture != "" {
			return spriteRowBase + spriteFrameRows
		}
		return spriteRowBase
	}
	return len(enumerateArgs(c.target)) + 1
}

// IsOpen reports whether the window is showing a component.
func (c *ComponentArgsComponent) IsOpen() bool { return c.target != nil }

// clipsRowRect returns the body rect of the `clips` field row, which for an @Animator
// opens the clips editor on click. The second result is false when the target is not
// an Animator or has no `clips` field, so the callers treat it as a plain row.
func (c *ComponentArgsComponent) clipsRowRect() (math.Rect, bool) {
	anim, ok := c.target.(*Animator)
	if !ok || anim == nil {
		return math.Rect{}, false
	}
	fields := enumerateArgs(c.target)
	for i, f := range fields {
		if f.name != "clips" {
			continue
		}
		rect := c.Rect()
		bodyTop := rect.Y() + c.titleH()
		y := bodyTop + float64(i+1)*c.RowHeight - c.scroll
		return math.NewRect(rect.X(), y, rect.Width(), c.RowHeight), true
	}
	return math.Rect{}, false
}

// Open shows the window for the given component, resetting scroll and rebuilding its
// argument widgets.
func (c *ComponentArgsComponent) Open(comp core.Component) {
	c.target = comp
	c.scroll = 0
	c.rebuildRows()
}

// spawnArgsWindow opens a new component-args window for the given component. If a window
// for that logical component (same owner + kind + name) is already open, it returns that
// one instead of duplicating — re-targeting it when the component instance has changed
// (e.g. the object editor re-loaded the same .obj and built a fresh component) and raising
// it to the front. A different component always gets its own window, so several can be
// open at once. The window is a fresh scene object, cascaded from the last open one.
func spawnArgsWindow(scene *core.Scene, comp core.Component) *ComponentArgsComponent {
	if comp == nil || scene == nil {
		return nil
	}
	// Focusing a component-args window moves ESC focus off the object editor, so ESC
	// closes this window first (the editor regains focus when the window closes).
	blurObjectEditor()
	id := argsWindowIdentity(comp)
	for _, w := range argWindows {
		if w == nil || !w.IsOpen() || argsWindowIdentity(w.target) != id {
			continue
		}
		if w.target != comp {
			w.Open(comp) // fresh instance for the same logical component: re-target it
		}
		focusedArgs = w
		raiseToFront(scene, w.GetOwner())
		return w
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
	focusedArgs = args
	return args
}

// argsWindowIdentity returns a stable key identifying "the same component" for window
// dedupe: the owner (its .obj file when file-referenced, else its pointer), plus the
// component kind and name. Reopening an object editor for the same .obj rebuilds the
// component instance, so a pure pointer comparison would open a duplicate; this key
// collapses those to one window.
func argsWindowIdentity(comp core.Component) string {
	if comp == nil {
		return ""
	}
	ownerID := "<no-owner>"
	if owner := comp.GetOwner(); owner != nil {
		if owner.File != "" {
			ownerID = "file:" + owner.File
		} else {
			ownerID = fmt.Sprintf("ptr:%p", owner)
		}
	}
	return ownerID + "|" + comp.GetKind() + "|" + comp.GetName()
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
	if focusedArgs == w {
		focusedArgs = nil
	}
	scene := w.GetScene()
	if owner := w.GetOwner(); owner != nil {
		owner.Destroy()
	}
	// Closing a component window returns ESC focus to the object editor (if it is open),
	// so the next ESC dismisses the editor rather than nothing. focusObjectEditor records
	// the frame so this same ESC press can't also close the editor.
	focusObjectEditor(scene)
}

// closeAllArgsWindows destroys every open window. Called when the target project switches,
// since windows reference the previous project's live components.
func closeAllArgsWindows() {
	for _, w := range append([]*ComponentArgsComponent(nil), argWindows...) {
		destroyArgsWindow(w)
	}
	argWindows = nil
}

// closeArgsWindowFor closes the open component-args window editing the given component,
// if one is open. Called when a component is removed so its window doesn't linger over
// a detached component. Iterates over a copy because destroyArgsWindow mutates argWindows.
func closeArgsWindowFor(comp core.Component) {
	for _, w := range append([]*ComponentArgsComponent(nil), argWindows...) {
		if w != nil && w.target == comp {
			destroyArgsWindow(w)
		}
	}
}

// closeArgsWindowsForObject closes every open component-args window whose target
// component belongs to obj. Called when an object is removed, so a window editing one
// of its components doesn't linger over a detached component (the object-level analog
// of closeArgsWindowFor).
func closeArgsWindowsForObject(obj *core.Object) {
	if obj == nil {
		return
	}
	for _, comp := range obj.ComponentsInDrawOrder() {
		closeArgsWindowFor(comp)
	}
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
	// A modal or an open menu bar is up: this window is inert.
	if modalOpen() || menusOpen() {
		return
	}

	// ESC closes this window when it is the focused one and no widget holds keyboard
	// focus (a focused ColorPicker/ComboBox consumes ESC itself to cancel its panel).
	if ctx.Input.IsKeyJustPressed(core.KeyEscape) && focusedArgs == c {
		if mgr := lookupUIManager(c.GetScene()); mgr == nil || !mgr.HasFocus() {
			destroyArgsWindow(c)
			return
		}
	}

	mouse := ctx.Input.GetMousePosition()
	rect := c.Rect()

	// Commit any widget change and live-sync the model first, before any hover-gated
	// logic, so commits fire even after the pointer leaves the window.
	c.pollAndRefresh(ctx)

	// A texture commit that crossed empty↔set changes the Sprite layout's row set: the
	// frame-size / frame-# fields only exist once a texture is set, so rebuild the rows.
	if c.isSprite() {
		if spr, _ := c.target.(*Sprite); spr != nil && (spr.Texture != "") != c.spriteTextured {
			c.rebuildRows() // rebuildRows also refreshes c.spriteTextured
		} else if spr, _ := c.target.(*Sprite); spr != nil && spriteManagedByAnimator(spr) != c.spriteManaged {
			// An Animator gained/lost this sprite (added as a clip, or a clip removed):
			// rebuild so the visible/flip/frame fields lock or unlock immediately.
			c.rebuildRows()
		}
	}

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
			c.scroll = scrollFromThumb(c.scrollTrack(rect), float64(c.rowCount())*c.RowHeight, c.maxScroll(), mouse.Y, c.scrollGrab)
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
			c.clampScroll()
			c.layoutRows()
		}
	}

	if ctx.Input.IsMouseButtonJustPressed(core.MouseButtonLeft) {
		focusedArgs = c    // any click in the window makes it the focused one
		blurObjectEditor() // so ESC now closes this window, not the object editor
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
		// The Animator's read-only `clips` row is a link into the clips editor.
		if rr, ok := c.clipsRowRect(); ok && rr.ContainsPoint(mouse) {
			spawnAnimatorClips(c.GetScene(), c.target.(*Animator))
			return
		}
		c.handleScrollbarPress(mouse, rect)
	}
}

// clampScroll keeps the scroll offset within [0, maxScroll].
func (c *ComponentArgsComponent) clampScroll() {
	if max := c.maxScroll(); c.scroll > max {
		c.scroll = max
	}
	if c.scroll < 0 {
		c.scroll = 0
	}
}

// scrollTrack returns the scrollbar track rect along the window body's right edge.
func (c *ComponentArgsComponent) scrollTrack(rect math.Rect) math.Rect {
	const w = 6.0
	return math.NewRect(rect.X()+rect.Width()-w-2, rect.Y()+c.titleH(), w, rect.Height()-c.titleH()-c.requiresH())
}

// handleScrollbarPress consumes a click on the scrollbar: grabbing the thumb starts a
// drag, and clicking the track jumps the thumb (centered) to the cursor.
func (c *ComponentArgsComponent) handleScrollbarPress(mouse math.Vector2, rect math.Rect) {
	track := c.scrollTrack(rect)
	contentH := float64(c.rowCount()) * c.RowHeight
	max := c.maxScroll()
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
// The row count comes from rowCount(), which accounts for the Sprite's fixed layout.
func (c *ComponentArgsComponent) maxScroll() float64 {
	body := c.Rect().Height() - c.titleH() - c.requiresH()
	if m := float64(c.rowCount())*c.RowHeight - body; m > 0 {
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
	bodyTop := rect.Y() + c.titleH()
	body := math.NewRect(rect.X(), bodyTop, rect.Width(), rect.Height()-c.titleH()-c.requiresH())
	r.SetClipRect(body)
	valX := rect.X() + rect.Width()/2

	if c.isSprite() {
		c.drawSpriteRows(r, rect, bodyTop, valX, th)
	} else {
		c.drawGenericRows(r, rect, bodyTop, valX, th)
	}

	// Scrollbar, drawn on top when the argument list overflows the body.
	r.SetClipRect(rect)
	track := c.scrollTrack(rect)
	if thumb, ok := scrollThumb(track, float64(c.rowCount())*c.RowHeight, c.scroll, c.maxScroll()); ok {
		drawScrollbar(r, track, thumb, c.ScrollTrack, c.ScrollThumb)
	}

	// "requires" footer: the component's declared dependencies, below the scroll area.
	c.drawRequiresFooter(r, rect, th)

	r.ClearClip()
}

// drawGenericRows draws the non-Sprite argument rows: the name label plus one label per
// reflected field, with read-only values rendered by the host (editable values are drawn
// by their widgets on layer 1).
func (c *ComponentArgsComponent) drawGenericRows(r core.Renderer, rect math.Rect, bodyTop, valX, th float64) {
	fields := enumerateArgs(c.target)

	// Name row (row 0).
	if y := bodyTop - c.scroll; y+c.RowHeight > bodyTop && y < rect.Y()+rect.Height() {
		ty := y + (c.RowHeight-th)/2
		if ty < y {
			ty = y
		}
		r.DrawText("name", c.FontID, c.FontSize, math.NewVector2(rect.X()+6, ty), c.KeyText)
	}

	for i, f := range fields {
		y := bodyTop + float64(i+1)*c.RowHeight - c.scroll
		if y+c.RowHeight <= bodyTop || y >= rect.Y()+rect.Height() {
			continue
		}
		ty := y + (c.RowHeight-th)/2
		if ty < y {
			ty = y
		}
		r.DrawText(f.name, c.FontID, c.FontSize, math.NewVector2(rect.X()+6, ty), c.KeyText)

		if !f.editable {
			val := formatArg(f.value)
			color := c.KeyText
			// The Animator's `clips` row reads as a link into the clips editor rather
			// than a raw JSON blob, and is highlighted to signal it is clickable.
			if anim, ok := c.target.(*Animator); ok && f.name == "clips" {
				val = fmt.Sprintf("Edit... (%d)", len(anim.Clips))
				color = c.TitleText
			}
			r.DrawText(val, c.FontID, c.FontSize, math.NewVector2(valX, ty), color)
		}
	}
}

// drawSpriteRows draws the Sprite's grouped layout chrome: field labels, separator lines
// between related groups, and dimmed read-only values for the animator-managed fields
// (visible/flip/frame) when a sprite is driven by an Animator. Editable values are drawn
// by their widgets on layer 1.
func (c *ComponentArgsComponent) drawSpriteRows(r core.Renderer, rect math.Rect, bodyTop, valX, th float64) {
	spr, _ := c.target.(*Sprite)
	managed := spriteManagedByAnimator(spr)

	drawLabel := func(row int, label string) {
		y := bodyTop + float64(row)*c.RowHeight - c.scroll
		if y+c.RowHeight <= bodyTop || y >= rect.Y()+rect.Height() {
			return
		}
		ty := y + (c.RowHeight-th)/2
		if ty < y {
			ty = y
		}
		r.DrawText(label, c.FontID, c.FontSize, math.NewVector2(rect.X()+6, ty), c.KeyText)
	}

	// drawValue renders a locked (animator-managed) value, dimmed so it reads as inert.
	drawValue := func(row int, value string) {
		y := bodyTop + float64(row)*c.RowHeight - c.scroll
		if y+c.RowHeight <= bodyTop || y >= rect.Y()+rect.Height() {
			return
		}
		ty := y + (c.RowHeight-th)/2
		if ty < y {
			ty = y
		}
		r.DrawText(value, c.FontID, c.FontSize, math.NewVector2(valX, ty), c.ValueText.Lerp(c.Background, 0.55))
	}

	// drawCheck renders a locked (animator-managed) boolean as a dimmed read-only
	// checkbox: just the box plus a check mark when set, no label. It mirrors the
	// editable checkbox's value-column placement (partWidth/partX) so locked and
	// editable rows line up.
	drawCheck := func(row, col, parts int, checked bool) {
		y := bodyTop + float64(row)*c.RowHeight - c.scroll
		if y+c.RowHeight <= bodyTop || y >= rect.Y()+rect.Height() {
			return
		}
		valueW := rect.Width()/2 - 8
		pw := partWidth(valueW, parts)
		x := partX(valX, col, pw)
		box := math.NewRect(x, y, c.RowHeight, c.RowHeight)
		dim := c.ValueText.Lerp(c.Background, 0.55)
		r.DrawRect(box, c.Background)
		r.DrawRectOutline(box, dim, 1)
		if checked {
			t := c.RowHeight * 0.12
			if t < 1 {
				t = 1
			}
			p1 := math.NewVector2(box.Left()+box.Width()*0.22, box.Top()+box.Height()*0.55)
			p2 := math.NewVector2(box.Left()+box.Width()*0.42, box.Top()+box.Height()*0.78)
			p3 := math.NewVector2(box.Left()+box.Width()*0.80, box.Top()+box.Height()*0.26)
			r.DrawLine(p1, p2, dim, t)
			r.DrawLine(p2, p3, dim, t)
		}
	}

	drawSeparator := func(row int) {
		sepY := bodyTop + float64(row)*c.RowHeight - c.scroll + c.RowHeight/2
		r.DrawLine(math.NewVector2(rect.X()+4, sepY), math.NewVector2(rect.X()+rect.Width()-10, sepY), c.BorderColor, 1)
	}

	drawLabel(0, "name")
	drawLabel(1, "draw_layer")
	drawLabel(2, "group")
	drawSeparator(3)
	drawLabel(4, "offset")

	drawLabel(5, "visible")
	drawLabel(6, "flip")
	if managed {
		// Locked (animator-managed) booleans render as dimmed read-only checkboxes —
		// just the box and check mark, no label — so they read as inert but still show
		// the live value.
		drawCheck(5, 0, 1, spr.IsVisible())
		drawCheck(6, 0, 2, spr.FlipX)
		drawCheck(6, 1, 2, spr.FlipY)
	}

	drawLabel(7, "color")
	drawLabel(8, "size")
	drawSeparator(9)
	drawLabel(10, "texture")

	if spr != nil && spr.Texture != "" {
		drawLabel(11, "frame size")
		drawLabel(12, "frame #")
		if managed {
			drawValue(12, formatFloat(float64(spr.Frame)))
		}
	}
}

// drawRequiresFooter draws the "requires" footer at the bottom of the window: the
// component kinds the target depends on (via core.Dependable), flagged in red when any
// of them are missing from the owner object. A component with no dependencies draws no
// footer and reserves no space (see requiresH).
func (c *ComponentArgsComponent) drawRequiresFooter(r core.Renderer, rect math.Rect, th float64) {
	deps := componentRequires(c.target)
	if len(deps) == 0 {
		return
	}
	footerTop := rect.Y() + rect.Height() - c.requiresH()
	r.DrawLine(math.NewVector2(rect.X(), footerTop), math.NewVector2(rect.X()+rect.Width(), footerTop), c.BorderColor, 1)

	label := "requires: " + strings.Join(deps, ", ")
	color := c.KeyText
	if missing := missingDependencies(c.target.GetOwner(), deps); len(missing) > 0 {
		label += "  (missing: " + strings.Join(missing, ", ") + ")"
		color = c.ErrorColor
	}
	ty := footerTop + 4 + (c.RowHeight-th)/2
	r.DrawText(label, c.FontID, c.FontSize, math.NewVector2(rect.X()+6, ty), color)
}

// ============================================================================
// Widget-host: build/rebuild the value widgets, poll commits, live-sync, layout.
// ============================================================================

// rebuildRows detaches the old argument widgets and builds fresh ones from the target
// component's current reflection schema. Called only on Open or a Sprite tab switch (a
// structural change); never per-frame, or a focused widget would lose focus every frame.
func (c *ComponentArgsComponent) rebuildRows() {
	c.removeWidgets()
	c.bindings = nil
	if c.target == nil {
		return
	}
	if c.isSprite() {
		c.buildSpriteBindings()
	} else {
		c.buildGenericBindings()
	}

	// Every commit here (the component's name or any of its args) mutates the .obj-owned
	// definition, so a file-referenced object writes through to its template on commit and
	// on undo/redo (commitString invokes afterApply for both).
	owner := c.target.GetOwner()
	for i := range c.bindings {
		c.bindings[i].afterApply = func() { persistObjectFile(owner) }
	}

	// Record the Sprite's textured state so a later texture commit can detect empty↔set
	// and rebuild the rows (frame-size / frame-# only exist when textured).
	if c.isSprite() {
		c.spriteTextured = c.target.(*Sprite).Texture != ""
		// Record whether the sprite is animator-managed so a change in management (an
		// Animator added, or a clip naming this sprite) rebuilds the rows and locks the
		// visible/flip/frame fields at once.
		c.spriteManaged = spriteManagedByAnimator(c.target.(*Sprite))
	}
}

// placeBinding positions b's widget at its row/col slot and appends it to c.bindings.
func (c *ComponentArgsComponent) placeBinding(b fieldBinding) {
	rect := c.Rect()
	valX := rect.X() + rect.Width()/2
	valueW := rect.Width()/2 - 8 // leave room for the scrollbar
	pw := partWidth(valueW, b.parts)
	x := partX(valX, b.col, pw)
	y := rect.Y() + c.titleH() + float64(b.row)*c.RowHeight - c.scroll
	b.widget = makeFieldWidget(&b, c.GetOwner(), math.NewVector2(x, y), pw, c.RowHeight, c.FontID, c.FontSize, c.ValueText)
	c.bindings = append(c.bindings, b)
}

// buildGenericBindings builds the name binding plus one binding per reflected editable
// field, in declaration order (the non-Sprite path).
func (c *ComponentArgsComponent) buildGenericBindings() {
	// The name row (row 0) is not a reflected JSON arg — it edits the component's own
	// name, so it is built as a dedicated binding ahead of the reflected fields.
	c.placeBinding(c.buildNameBinding())

	fields := enumerateArgs(c.target)
	for i := range fields {
		f := &fields[i]
		if !f.editable {
			continue
		}
		for _, b := range c.bindingsFor(*f) {
			b.row = i + 1 // row 0 is the name row
			c.placeBinding(b)
		}
	}
}

// spriteField returns the reflected arg field of the Sprite target with the given json
// name (draw_layer/group/texture/offset/...).
func (c *ComponentArgsComponent) spriteField(name string) (argField, bool) {
	for _, f := range enumerateArgs(c.target) {
		if f.name == name {
			return f, true
		}
	}
	return argField{}, false
}

// placePair builds the field `name` as a two-part side-by-side binding at `row` with the
// given part column (0 = left, 1 = right).
func (c *ComponentArgsComponent) placePair(name string, col, row int) {
	f, ok := c.spriteField(name)
	if !ok {
		return
	}
	for _, b := range c.bindingsFor(f) {
		b.row = row
		b.col = col
		b.parts = 2
		c.placeBinding(b)
	}
}

// buildSpriteBindings builds the Sprite's custom grouped layout:
//
//	0 name, 1 draw_layer, 2 group, 3 separator, 4 offset, 5 visible, 6 flip, 7 color,
//	8 size, 9 separator, 10 texture, then (texture only) 11 frame size, 12 frame #.
//
// visible/flip/frame are omitted (locked) when an Animator drives this sprite, so their
// rows show as dimmed read-only values instead of widgets.
func (c *ComponentArgsComponent) buildSpriteBindings() {
	spr, _ := c.target.(*Sprite)
	managed := spriteManagedByAnimator(spr)

	c.placeBinding(c.buildNameBinding()) // row 0

	if f, ok := c.spriteField("draw_layer"); ok {
		for _, b := range c.bindingsFor(f) {
			b.row = 1
			c.placeBinding(b)
		}
	}
	if f, ok := c.spriteField("group"); ok {
		for _, b := range c.bindingsFor(f) {
			b.row = 2
			c.placeBinding(b)
		}
	}
	// offset (Vector2, two side-by-side boxes) is always editable.
	if f, ok := c.spriteField("offset"); ok {
		for _, b := range c.bindingsFor(f) {
			b.row = 4
			c.placeBinding(b)
		}
	}

	// visible (checkbox) and flip are editable unless the animator owns them.
	if !managed {
		c.placeBinding(c.visibleBinding()) // row 5
		c.placePair("flip_x", 0, 6)
		c.placePair("flip_y", 1, 6)
	}

	// color (tint) and size are never animator-owned.
	c.placeBinding(c.tintBinding()) // row 7
	c.placePair("width", 0, 8)
	c.placePair("height", 1, 8)

	// texture (row 10), then the texture-only frame rows.
	if f, ok := c.spriteField("texture"); ok {
		for _, b := range c.bindingsFor(f) {
			b.row = 10
			c.placeBinding(b)
		}
	}
	if spr != nil && spr.Texture != "" {
		c.placePair("frame_width", 0, 11)
		c.placePair("frame_height", 1, 11)
		if !managed {
			c.placeBinding(c.frameSliderBinding()) // row 12
		}
	}
}

// frameSliderBinding builds the Sprite's `frame` argument as an integer slider bounded
// 0..FrameCount()-1 (the max follows the texture size once it loads).
func (c *ComponentArgsComponent) frameSliderBinding() fieldBinding {
	spr := c.target.(*Sprite)
	b := fieldBinding{
		key:   "frame",
		row:   12,
		parts: 1,
		kind:  kindSlider,
		get:   func() string { return formatFloat(float64(spr.Frame)) },
		apply: func(s string) error {
			v, err := parseFloat(s)
			if err != nil {
				return err
			}
			spr.Frame = int(v)
			return nil
		},
		getFloat:   func() float64 { return float64(spr.Frame) },
		sliderMin:  func() float64 { return 0 },
		sliderMax:  func() float64 { return float64(spr.FrameCount() - 1) },
		sliderStep: 1,
	}
	b.old = b.get()
	return b
}

// tintBinding builds the Sprite's `color` argument as a color picker bound to the color
// transform's Tint (the multiply color). An unset Tint reads as white (identity).
func (c *ComponentArgsComponent) tintBinding() fieldBinding {
	spr := c.target.(*Sprite)
	tint := func() math.Color {
		if spr.Color.Tint == (math.Color{}) {
			return math.White
		}
		return spr.Color.Tint
	}
	b := fieldBinding{
		key:   "color",
		row:   7,
		parts: 1,
		kind:  kindColor,
		get:   func() string { return formatColorHex(tint()) },
		apply: func(s string) error {
			col, err := math.ParseHex(s)
			if err != nil {
				return err
			}
			spr.Color.Tint = col
			return nil
		},
		getColor: tint,
	}
	b.old = b.get()
	return b
}

// visibleBinding builds the Sprite's `visible` argument as a checkbox. `visible` is a
// *bool (nil = default true), so it is not reflection-editable; it is bound here through
// IsVisible/SetVisible instead. Only built when the sprite is not animator-managed.
func (c *ComponentArgsComponent) visibleBinding() fieldBinding {
	spr := c.target.(*Sprite)
	b := fieldBinding{
		key:   "visible",
		row:   5,
		parts: 1,
		kind:  kindCheck,
		get:   func() string { return strconv.FormatBool(spr.IsVisible()) },
		apply: func(s string) error {
			v, err := parseBool(s)
			if err != nil {
				return err
			}
			spr.SetVisible(v)
			return nil
		},
		getBool: func() bool { return spr.IsVisible() },
	}
	b.old = b.get()
	return b
}

// browseTexture opens the project file browser to pick the sprite's texture image. On
// selection the chosen project-relative path commits through the texture binding (so it
// records undo and resets the sprite's cached texture size via the binding's apply). It
// is a no-op when no scene is available or a modal is already up.
func (c *ComponentArgsComponent) browseTexture() {
	scene := c.GetScene()
	if scene == nil || modalOpen() {
		return
	}
	idx := -1
	for i := range c.bindings {
		if c.bindings[i].key == "texture" {
			idx = i
			break
		}
	}
	if idx < 0 {
		return
	}
	spawnFilePicker(scene, func(rel string) bool { return fileTypeOf(rel) == "image" },
		func(rel string) {
			if idx < len(c.bindings) {
				_ = commitString(&c.bindings[idx], rel)
			}
		})
}

// buildNameBinding returns the field binding for the component's name. get/apply read
// and write the name on the component itself; apply enforces non-empty and unique
// within the owner object, and renames in place so the owner's name map and insertion
// order stay consistent. Undo is recorded by commitString, not here.
func (c *ComponentArgsComponent) buildNameBinding() fieldBinding {
	b := fieldBinding{
		key:   "_name",
		row:   0,
		parts: 1,
		kind:  kindText,
		get:   func() string { return c.target.GetName() },
		apply: func(s string) error {
			name := strings.TrimSpace(s)
			if name == "" {
				return fmt.Errorf("name cannot be empty")
			}
			if name == c.target.GetName() {
				return nil
			}
			owner := c.target.GetOwner()
			if owner == nil {
				return fmt.Errorf("component has no owner")
			}
			return owner.RenameComponent(c.target.GetName(), name)
		},
	}
	b.old = b.get()
	return b
}

// spriteManagedByAnimator reports whether the sprite is driven by an Animator on the
// same owner (its component name appears in any animator clip). Such a sprite's visible,
// frame, and flip fields are owned by the animator and are locked (dimmed, non-editable)
// in the args window.
func spriteManagedByAnimator(spr *Sprite) bool {
	if spr == nil {
		return false
	}
	owner := spr.GetOwner()
	if owner == nil {
		return false
	}
	name := spr.GetName()
	for _, comp := range owner.ComponentsInDrawOrder() {
		anim, ok := comp.(*Animator)
		if !ok {
			continue
		}
		for _, clip := range anim.Clips {
			if clip.Sprite == name {
				return true
			}
		}
	}
	return false
}

// clipSpriteNames returns the distinct sprite names named by the animator's clips, in
// clip order — the choices for the animator's `default` combobox.
func clipSpriteNames(anim *Animator) []string {
	if anim == nil {
		return nil
	}
	seen := make(map[string]bool, len(anim.Clips))
	names := make([]string, 0, len(anim.Clips))
	for _, clip := range anim.Clips {
		if clip.Sprite != "" && !seen[clip.Sprite] {
			seen[clip.Sprite] = true
			names = append(names, clip.Sprite)
		}
	}
	return names
}

// bindingsFor builds the field binding(s) for one reflection-discovered argument: a
// single widget for scalar/bool/color fields, or one widget per component for
// math.Vector2 (x, y) and math.Border (left, top, right, bottom), laid out side by side.
// The closures capture the live reflect.Value (which points into the target component),
// so get/apply always read/write the current value.
func (c *ComponentArgsComponent) bindingsFor(f argField) []fieldBinding {
	fv := f.value
	t := fv.Type()

	// The Animator's `default` field names the clip played on startup. A free-text box
	// is error-prone (it must name a clip), so it is a combobox of the clip sprite
	// names; selecting one also re-plays it so the editor's visibility updates at once.
	if f.name == "default" {
		if anim, ok := c.target.(*Animator); ok {
			b := fieldBinding{
				key:   f.name,
				parts: 1,
				kind:  kindCombobox,
				get:   func() string { return anim.Default },
				apply: func(s string) error {
					anim.Default = s
					anim.Play(s) // no-op when s has no clip; else shows it and hides the rest
					return nil
				},
				getOptions: func() []string { return clipSpriteNames(anim) },
			}
			b.old = b.get()
			return []fieldBinding{b}
		}
	}

	// The Animator's `flip_x`/`flip_y` booleans must go through SetFlipX/SetFlipY (not a
	// raw reflect write) so the flip is applied to every managed sprite immediately.
	if (f.name == "flip_x" || f.name == "flip_y") && fv.Kind() == reflect.Bool {
		if anim, ok := c.target.(*Animator); ok {
			apply := func(s string) error {
				v, err := parseBool(s)
				if err != nil {
					return err
				}
				if f.name == "flip_x" {
					anim.SetFlipX(v)
				} else {
					anim.SetFlipY(v)
				}
				return nil
			}
			b := fieldBinding{
				key:     f.name,
				parts:   1,
				kind:    kindCheck,
				get:     func() string { return formatArg(fv) },
				apply:   apply,
				getBool: func() bool { return fv.Bool() },
			}
			b.old = b.get()
			return []fieldBinding{b}
		}
	}

	// A Sprite's `texture` is a project-relative file path: it is a file-selector
	// button (kindFile) that opens the project browser, not a free-text box. Committing
	// a new path also resets the sprite's cached texture size so the new image loads
	// immediately (undo/redo re-apply the same setter).
	if f.name == "texture" {
		if spr, ok := c.target.(*Sprite); ok {
			b := fieldBinding{
				key:   f.name,
				parts: 1,
				kind:  kindFile,
				get:   func() string { return spr.Texture },
				apply: func(s string) error {
					spr.SetTexture(s)
					spr.ResetTexture()
					return nil
				},
				fileFilter: func(rel string) bool { return fileTypeOf(rel) == "image" },
			}
			b.onBrowse = func() { c.browseTexture() }
			b.old = b.get()
			return []fieldBinding{b}
		}
	}

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
	layoutWidgets(c.bindings, c.GetOwner(), rect.Y()+c.titleH(), rect.X()+rect.Width()/2, valueW, c.scroll, c.RowHeight, rect.Y()+rect.Height()-c.requiresH())
}
