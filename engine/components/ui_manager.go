package components

import (
	"sort"

	"github.com/EnesBaytekin/imge/core"
	"github.com/EnesBaytekin/imge/core/math"
)

// UIManager routes pointer and keyboard input across a scene's UI elements. Put it
// on a dedicated "carrier" object (e.g. one named ui_root, holding only this
// component); it is the single reader of ctx.Input, so UI widgets never read the
// mouse or keyboard themselves. Each frame it:
//
//   - discovers the UI elements it manages: every component on every UI=true object
//     matching Tags (no Tags = all UI objects), in draw order,
//   - hit-tests the pointer (topmost element first) and pushes hover/press/click to
//     that element, honoring each element's blocking flag for occlusion,
//   - owns keyboard focus: clicking a focusable focuses it, Tab/Shift+Tab move focus
//     in geometric row-reading order, and the focused element receives key input,
//   - drags any Draggable object by its non-interactive surface.
//
// There is normally one manager per scene. Give two managers different Tags to manage
// disjoint sets of UI objects (e.g. separate windows) without interfering.
//
// Export variables (JSON args): tags.
type UIManagerComponent struct {
	core.BaseComponent

	// Tags restricts the manager to UI objects carrying at least one of these tags.
	// Empty means every UI object in the scene.
	Tags []string `json:"tags"`

	// AutoRaise brings the clicked object to the front of the managed set (normal
	// window behavior). nil defaults to true; set false to keep the manual depth
	// order from the scene.
	AutoRaise *bool `json:"auto_raise"`

	// elements is the discovered UI elements in draw order (back to front), rebuilt
	// once per scene frame. owners is the same scan's distinct object list, used to
	// lift a clicked object above its siblings.
	elements []uiElement
	owners   []*core.Object
	frame    uint64

	hovered    uiPointer
	pressed    uiPointer
	focused    uiFocusable
	adjusting  uiSlider
	pointed    uiPointable
	dragging   *core.Object
	dragOffset math.Vector2
}

// uiElement is the shape every UI component satisfies by embedding
// core.BaseUIComponent. A @UIManager works against this interface (not concrete
// types), so custom UI components participate in routing for free.
type uiElement interface {
	core.Component
	Rect() math.Rect
	Contains(math.Vector2) bool
	IsVisible() bool
	IsEnabled() bool
	BlocksPointer() bool
}

// uiPointer is a UI element that reacts to pointer hover/press/click (a button).
type uiPointer interface {
	uiElement
	SetHovered(bool)
	SetPressed(bool)
	Activate()
}

// uiFocusable is a UI element that can hold keyboard focus (a text input).
type uiFocusable interface {
	uiElement
	IsFocusable() bool
	SetFocused(bool)
	HandleInput(*core.Context)
}

// uiSlider is a UI element the user adjusts by pressing and dragging — a slider or
// any continuous control. The manager owns the gesture: on press it calls
// BeginAdjust, then Adjust every held frame, then EndAdjust on release (or when the
// button is no longer held). The element itself maps pointer positions to its value.
type uiSlider interface {
	uiElement
	BeginAdjust(pos math.Vector2)
	Adjust(pos math.Vector2)
	EndAdjust()
}

// uiPointable is a UI element that acts on the exact pointer position, so it can
// hit-test its own sub-regions — a combobox's dropdown items, a menu's entries, a
// color wheel. Unlike uiPointer (any click = one action), a pointable element
// decides what to do from where the pointer is. The manager calls PointerMove every
// frame the pointer is over the element, PointerLeave once when it leaves, Press on
// each left-button press, and Release on each release — each with the position, so
// the element can select on release the way a button activates on release.
type uiPointable interface {
	uiElement
	PointerMove(pos math.Vector2)
	PointerLeave()
	Press(pos math.Vector2)
	Release(pos math.Vector2)
}

// popupLayerOffset is added to a component's draw layer while it has an open popup
// (a combobox dropdown, a color picker panel), so the popup draws — and is
// hit-tested — above the object's other components for that moment.
const popupLayerOffset = 10000

func (m *UIManagerComponent) Update(ctx *core.Context) {
	scene := m.GetScene()
	if scene == nil {
		return
	}
	if scene.FrameNumber() != m.frame {
		m.frame = scene.FrameNumber()
		m.rebuild(scene)
	}

	pos := ctx.Input.GetMousePosition()
	target := m.targetAt(pos)
	m.updateHover(target)
	m.updatePointed(target, pos)

	if ctx.Input.IsMouseButtonJustPressed(core.MouseButtonLeft) {
		m.onPress(target, pos)
	}
	if ctx.Input.IsMouseButtonJustReleased(core.MouseButtonLeft) {
		m.onRelease(target, pos)
	}

	// Drag a draggable object while the left button stays held.
	if m.dragging != nil {
		if ctx.Input.IsMouseButtonPressed(core.MouseButtonLeft) {
			m.dragging.Transform.Position = pos.Add(m.dragOffset)
		} else {
			m.dragging = nil
		}
	}

	// Keep adjusting a slider (or any uiSlider) while the left button stays held,
	// and end the gesture when it is released. onRelease already cleared it on the
	// release frame, so this only drives the held frames in between.
	if m.adjusting != nil {
		if ctx.Input.IsMouseButtonPressed(core.MouseButtonLeft) {
			m.adjusting.Adjust(pos)
		} else {
			m.adjusting.EndAdjust()
			m.adjusting = nil
		}
	}

	if ctx.Input.IsKeyJustPressed(core.KeyTab) {
		if ctx.Input.IsKeyPressed(core.KeyShift) {
			m.focusPrev()
		} else {
			m.focusNext()
		}
	}

	// Route keyboard input to the focused element; drop focus if it vanished.
	if m.focused != nil {
		if !m.focused.IsVisible() || !m.focused.IsEnabled() {
			m.setFocus(nil)
		} else {
			m.focused.HandleInput(ctx)
		}
	}
}

// rebuild rescans the scene and collects the managed UI elements in draw order.
func (m *UIManagerComponent) rebuild(scene *core.Scene) {
	m.elements = m.elements[:0]
	m.owners = m.owners[:0]
	for _, obj := range scene.GetSortedObjects() {
		if !obj.UI || !obj.Active || obj.IsDestroyed() {
			continue
		}
		if !m.matchesTags(obj) {
			continue
		}
		added := false
		for _, comp := range obj.ComponentsInDrawOrder() {
			if el, ok := comp.(uiElement); ok {
				m.elements = append(m.elements, el)
				added = true
			}
		}
		if added {
			m.owners = append(m.owners, obj)
		}
	}
}

// matchesTags reports whether obj is in scope: any of the manager's tags matches, or
// no tags means every UI object.
func (m *UIManagerComponent) matchesTags(obj *core.Object) bool {
	if len(m.Tags) == 0 {
		return true
	}
	for _, tag := range m.Tags {
		if obj.HasTag(tag) {
			return true
		}
	}
	return false
}

// targetAt returns the topmost enabled+visible element under pos that either blocks
// or is interactive. Non-blocking, non-interactive elements (labels) are transparent
// to the pointer; a blocking element is opaque and stops the search.
func (m *UIManagerComponent) targetAt(pos math.Vector2) uiElement {
	for i := len(m.elements) - 1; i >= 0; i-- {
		el := m.elements[i]
		if !el.IsVisible() || !el.IsEnabled() {
			continue
		}
		if !el.Contains(pos) {
			continue
		}
		if el.BlocksPointer() {
			return el
		}
		if _, ok := el.(uiPointer); ok {
			return el
		}
		if _, ok := el.(uiFocusable); ok {
			return el
		}
	}
	return nil
}

// updateHover moves the hover state to the pointer target, if any.
func (m *UIManagerComponent) updateHover(target uiElement) {
	var hp uiPointer
	if target != nil {
		hp, _ = target.(uiPointer)
	}
	if m.hovered == hp {
		return
	}
	if m.hovered != nil {
		m.hovered.SetHovered(false)
	}
	m.hovered = hp
	if m.hovered != nil {
		m.hovered.SetHovered(true)
	}
}

// updatePointed routes per-frame pointer position to a pointable target (a
// combobox, menu) so it can highlight its own sub-regions, and clears that hover
// when the pointer moves off it.
func (m *UIManagerComponent) updatePointed(target uiElement, pos math.Vector2) {
	pt, ok := target.(uiPointable)
	if !ok {
		if m.pointed != nil {
			m.pointed.PointerLeave()
			m.pointed = nil
		}
		return
	}
	if m.pointed != nil && m.pointed != pt {
		m.pointed.PointerLeave()
	}
	m.pointed = pt
	pt.PointerMove(pos)
}

// onPress handles a left-button press: it focuses a focusable target (or blurs on
// anything else), marks a pointer target pressed, and starts a drag on a draggable
// object's non-interactive surface.
func (m *UIManagerComponent) onPress(target uiElement, pos math.Vector2) {
	if target != nil {
		m.RaiseToFront(target.GetOwner())
	}

	if f, ok := target.(uiFocusable); ok {
		m.setFocus(f)
	} else {
		m.setFocus(nil)
	}

	if p, ok := target.(uiPointer); ok {
		p.SetPressed(true)
		m.pressed = p
	} else {
		m.pressed = nil
	}

	// A slider (or any continuous control) begins its drag gesture on press.
	if a, ok := target.(uiSlider); ok {
		m.adjusting = a
		a.BeginAdjust(pos)
	}

	// A combobox (or any pointable control) acts on the exact press position, so it
	// can hit-test its own sub-regions (dropdown items, menu entries).
	if pt, ok := target.(uiPointable); ok {
		pt.Press(pos)
	}

	if target != nil && target.GetOwner() != nil && target.GetOwner().Draggable && !m.isInteractive(target) {
		m.dragging = target.GetOwner()
		m.dragOffset = m.dragging.Transform.Position.Subtract(pos)
	}
}

// isInteractive reports whether a UI element handles the pointer itself (a button,
// text input, slider, or pointable control). Such elements never start an object
// drag, even when their owner is Draggable.
func (m *UIManagerComponent) isInteractive(el uiElement) bool {
	if _, ok := el.(uiPointer); ok {
		return true
	}
	if _, ok := el.(uiFocusable); ok {
		return true
	}
	if _, ok := el.(uiSlider); ok {
		return true
	}
	if _, ok := el.(uiPointable); ok {
		return true
	}
	return false
}

// onRelease handles a left-button release: it activates the pressed pointer if the
// release lands on it, ends a slider drag, and completes a pointable gesture.
func (m *UIManagerComponent) onRelease(target uiElement, pos math.Vector2) {
	if m.pressed != nil {
		if uiElement(m.pressed) == target {
			m.pressed.Activate()
		}
		m.pressed.SetPressed(false)
		m.pressed = nil
	}
	m.dragging = nil

	// End any slider adjustment on release, regardless of where the cursor is.
	if m.adjusting != nil {
		m.adjusting.EndAdjust()
		m.adjusting = nil
	}

	// A pointable control (combobox) completes its gesture on release, so it can
	// select the item the pointer was released over — the button-like behavior.
	if pt, ok := target.(uiPointable); ok {
		pt.Release(pos)
	}
}

// RaiseToFront lifts obj above every other managed object in its own layer so it
// draws on top. Clicking a window — its background or any of its controls — brings
// it to the front, the normal window behavior.
//
// The reorder stays inside obj's layer: objects in a higher layer (e.g. an
// always-on-top header) are untouched, and obj never crosses into one. Depths are
// re-slotted to a compact 0..n-1 range within the layer, so repeated raises can't
// push depths to grow without bound (and never need a magic "very high" number).
//
// It is also useful programmatically: a component that reopens a hidden window
// (e.g. an "Open" button) can call RaiseToFront after SetActive(true) so the window
// reappears on top instead of buried under whatever was in front of it.
func (m *UIManagerComponent) RaiseToFront(obj *core.Object) {
	if obj == nil {
		return
	}
	if m.AutoRaise != nil && !*m.AutoRaise {
		return
	}

	// Collect every managed object in obj's layer, in draw order, with obj itself
	// moved to the front. obj may not be in m.owners yet (e.g. just reactivated).
	var order []*core.Object
	for _, o := range m.owners {
		if o == obj || o.Layer != obj.Layer {
			continue
		}
		order = append(order, o)
	}
	order = append(order, obj)

	// Re-slot compact depths within the layer: 0..n-1, front = highest.
	for i, o := range order {
		if o.Depth != float64(i) {
			o.SetDepth(float64(i))
		}
	}
}

// setFocus moves focus to f, updating the old and new focused elements.
func (m *UIManagerComponent) setFocus(f uiFocusable) {
	if m.focused == f {
		return
	}
	if m.focused != nil {
		m.focused.SetFocused(false)
	}
	m.focused = f
	if m.focused != nil {
		m.focused.SetFocused(true)
	}
}

// focusNext moves focus to the next focusable in geometric row-reading order,
// wrapping around.
func (m *UIManagerComponent) focusNext() {
	order := m.focusablesInTabOrder()
	if len(order) == 0 {
		return
	}
	idx := -1
	if m.focused != nil {
		for i, f := range order {
			if f == m.focused {
				idx = i
				break
			}
		}
	}
	m.setFocus(order[(idx+1)%len(order)])
}

// focusPrev moves focus to the previous focusable in geometric row-reading order,
// wrapping around.
func (m *UIManagerComponent) focusPrev() {
	order := m.focusablesInTabOrder()
	if len(order) == 0 {
		return
	}
	idx := len(order) - 1
	if m.focused != nil {
		for i, f := range order {
			if f == m.focused {
				idx = i
				break
			}
		}
	}
	m.setFocus(order[(idx-1+len(order))%len(order)])
}

// focusablesInTabOrder returns the enabled+visible focusables in geometric
// row-reading order: elements whose vertical extents overlap are grouped into a row
// (top to bottom), and each row is read left to right.
func (m *UIManagerComponent) focusablesInTabOrder() []uiFocusable {
	var fs []uiFocusable
	for _, el := range m.elements {
		f, ok := el.(uiFocusable)
		if !ok || !f.IsFocusable() || !f.IsVisible() || !f.IsEnabled() {
			continue
		}
		fs = append(fs, f)
	}
	if len(fs) == 0 {
		return nil
	}

	// Sort by top edge (then left), so rows are built top to bottom.
	sort.SliceStable(fs, func(i, j int) bool {
		ri, rj := fs[i].Rect(), fs[j].Rect()
		if ri.Y() != rj.Y() {
			return ri.Y() < rj.Y()
		}
		return ri.X() < rj.X()
	})

	type row struct {
		minY, maxY float64
		items      []uiFocusable
	}
	var rows []row
	for _, f := range fs {
		r := f.Rect()
		top, bottom := r.Y(), r.Y()+r.Height()
		if len(rows) > 0 {
			last := len(rows) - 1
			if top <= rows[last].maxY && bottom >= rows[last].minY {
				rows[last].items = append(rows[last].items, f)
				if top < rows[last].minY {
					rows[last].minY = top
				}
				if bottom > rows[last].maxY {
					rows[last].maxY = bottom
				}
				continue
			}
		}
		rows = append(rows, row{minY: top, maxY: bottom, items: []uiFocusable{f}})
	}

	var order []uiFocusable
	for i := range rows {
		sort.SliceStable(rows[i].items, func(a, b int) bool {
			return rows[i].items[a].Rect().X() < rows[i].items[b].Rect().X()
		})
		order = append(order, rows[i].items...)
	}
	return order
}
