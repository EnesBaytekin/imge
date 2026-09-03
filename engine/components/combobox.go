package components

import (
	"github.com/EnesBaytekin/imge/core"
	"github.com/EnesBaytekin/imge/core/math"
)

// ComboBox is a dropdown selector: a read-only field that, when opened, shows a
// list of items below (or above, when there is no room) for the user to pick one.
// The items are plain strings — the text shown and the value returned on selection
// are the same string. A handler reads the result via GetValue() (or GetIndex()).
//
// Opening and choosing:
//   - click the field to open/close the list; press an item to highlight it, then
//     release over it to select (the button-like release-to-activate feel),
//   - click anywhere else (or press Escape) to close without selecting,
//   - with keyboard focus, Enter/Space opens, ↑/↓ move the highlight, Enter selects,
//     Escape closes.
//
// Dropdown placement is the native-combobox behavior: it opens below the field by
// default, flips above when it would overflow the bottom of the screen, and when it
// fits neither side it opens on the side with more room. A list taller than the
// available room is capped to fit and scrolls — with the mouse wheel, or by dragging
// the thin scrollbar on its right edge — instead of overflowing the screen.
//
// Appearance: the field and the list are flat colors by default, or nine-sliced
// textures (Texture/Border for the field, DropdownTexture/DropdownBorder for the
// list) so a styled control stretches. HoverColor highlights the item under the
// pointer (or keyboard cursor); SelectedColor marks the current value while the
// list is open. Like @Button and @Panel, defaults can be supplied from styles.imge
// under the "@ComboBox" key (a style body fills any of these json-tagged fields).
//
// Export variables (JSON args): items, value, event, placeholder, font_id, size,
// text_color, color, hover_color, selected_color, dropdown_color, texture, border
// {left, top, right, bottom}, dropdown_texture, dropdown_border, outline_color,
// outline_thickness, item_height, scrollbar_color, scrollbar_width, offset, width,
// height, visible, enabled, blocking, group, draw_layer.
type ComboBoxComponent struct {
	core.BaseUIComponent

	// Items is the list of choices; the shown text and the selected value are the
	// same string.
	Items []string `json:"items"`

	// Value is the currently selected item (one of Items, or "" for none).
	Value string `json:"value"`

	// Event is emitted on selection, with the combobox itself as the data (so a
	// handler can read GetValue()/GetIndex() and GetName()/GetOwner()), matching
	// @Button and @Slider.
	Event string `json:"event"`

	// Placeholder is drawn in the field when Value is empty.
	Placeholder string `json:"placeholder"`

	// FontID and Size set the text font. Size 0 (default) = 12, a crisp multiple of
	// the built-in pixel font's 6-unit design grid (6, 12, 18, … stay sharp; other
	// sizes rasterize antialiased and go soft).
	FontID string  `json:"font_id"`
	Size   float64 `json:"size"`

	// TextColor is the field and item text color.
	TextColor math.Color `json:"text_color"`

	// Color is the field's flat fill when Texture is empty.
	Color math.Color `json:"color"`

	// HoverColor highlights the item under the pointer / keyboard cursor.
	HoverColor math.Color `json:"hover_color"`

	// SelectedColor marks the currently selected item while the list is open.
	SelectedColor math.Color `json:"selected_color"`

	// DropdownColor is the list's flat fill when DropdownTexture is empty.
	DropdownColor math.Color `json:"dropdown_color"`

	// Texture/Border opt the field into nine-slice rendering; DropdownTexture/
	// DropdownBorder do the same for the list. An empty texture means a flat fill.
	Texture         string      `json:"texture"`
	Border          math.Border `json:"border"`
	DropdownTexture string      `json:"dropdown_texture"`
	DropdownBorder  math.Border `json:"dropdown_border"`

	// OutlineColor/OutlineThickness stroke the field and list. A transparent color
	// (the default) means no outline; thickness 0 defaults to 1.
	OutlineColor     math.Color `json:"outline_color"`
	OutlineThickness float64    `json:"outline_thickness"`

	// ItemHeight is the height of one list row; 0 (default) = 22.
	ItemHeight float64 `json:"item_height"`

	// ScrollbarColor is the scrollbar thumb color; ScrollbarWidth is its width
	// (0 default = 6). The scrollbar only appears when the capped list can't show
	// every item at once.
	ScrollbarColor math.Color `json:"scrollbar_color"`
	ScrollbarWidth float64    `json:"scrollbar_width"`

	// open is whether the list is currently shown; highlight is the item index under
	// the pointer or keyboard cursor (-1 = none); viewportH is the logical screen
	// height, captured each frame in Draw and used to decide the open direction.
	open      bool
	highlight int
	viewportH float64

	// Press state for button-like selection (choose on release, not press).
	pressedItem   int  // item index pressed (-1 = none)
	pressedHeader bool // press was on the field itself (close on release)
	openedOnPress bool // this press opened the list; ignore its release

	// Dropdown scrolling (when the item list is taller than the space on screen):
	// scrollOffset is how far the list is scrolled down in pixels (0 = first item at
	// top); draggingBar marks a scrollbar drag in progress.
	scrollOffset float64
	draggingBar  bool
}

// Initialize defaults the colors, sizes, and flags, then opts into blocking and
// keyboard focus (a combobox is inherently focusable for ↑/↓/Enter).
func (c *ComboBoxComponent) Initialize() {
	if c.Color == (math.Color{}) {
		c.Color = math.NewColor(42, 42, 56, 255)
	}
	if c.DropdownColor == (math.Color{}) {
		c.DropdownColor = math.NewColor(30, 30, 42, 255)
	}
	if c.HoverColor == (math.Color{}) {
		c.HoverColor = math.NewColor(70, 90, 130, 255)
	}
	if c.SelectedColor == (math.Color{}) {
		c.SelectedColor = math.NewColor(90, 158, 90, 255)
	}
	if c.TextColor == (math.Color{}) {
		c.TextColor = math.White
	}
	if c.OutlineColor == (math.Color{}) {
		c.OutlineColor = math.NewColor(74, 74, 106, 255)
	}
	if c.OutlineThickness <= 0 {
		c.OutlineThickness = 1
	}
	if c.Size <= 0 {
		c.Size = 12
	}
	if c.ItemHeight <= 0 {
		c.ItemHeight = 22
	}
	if c.ScrollbarColor == (math.Color{}) {
		c.ScrollbarColor = math.NewColor(90, 90, 120, 255)
	}
	if c.ScrollbarWidth <= 0 {
		c.ScrollbarWidth = 6
	}
	if c.Blocking == nil {
		b := true
		c.Blocking = &b
	}
	c.Focusable = true
}

// GetValue returns the currently selected item string ("" when none).
func (c *ComboBoxComponent) GetValue() string { return c.Value }

// GetIndex returns the index of the current value in Items, or -1 when the value is
// not present (or the list is empty).
func (c *ComboBoxComponent) GetIndex() int { return c.selectedIndex() }

// SetValue sets the selection silently (no Event) to the given item if it is in
// Items; otherwise it clears the selection.
func (c *ComboBoxComponent) SetValue(v string) {
	c.Value = ""
	for _, item := range c.Items {
		if item == v {
			c.Value = v
			return
		}
	}
}

// Contains reports whether p is over the field or, when open, the list — so the
// manager hit-tests the whole dropdown as one element.
func (c *ComboBoxComponent) Contains(p math.Vector2) bool {
	if c.headerRect().ContainsPoint(p) {
		return true
	}
	if c.open {
		return c.dropdownRect().ContainsPoint(p)
	}
	return false
}

// PointerMove updates the highlighted item from the pointer position. Called by a
// @UIManager every frame the pointer is over the element.
func (c *ComboBoxComponent) PointerMove(pos math.Vector2) {
	if c.open {
		c.highlight = c.itemAt(pos)
	}
}

// PointerLeave resets the highlight to the current selection when the pointer
// leaves the element. Called by a @UIManager.
func (c *ComboBoxComponent) PointerLeave() {
	c.highlight = c.selectedIndex()
}

// Press opens the list (when closed) or records what was pressed (when open), so
// the selection can happen on release. Called by a @UIManager on a left-button press.
func (c *ComboBoxComponent) Press(pos math.Vector2) {
	if !c.IsEnabled() {
		return
	}
	if !c.open {
		c.openDropdown()
		c.openedOnPress = true
		return
	}
	c.openedOnPress = false
	c.pressedItem = -1
	c.pressedHeader = false
	// A press on the scrollbar is a scroll gesture, owned by the uiSlider gesture
	// (BeginAdjust, which runs just before this), so it is skipped here.
	if c.draggingBar {
		return
	}
	if idx := c.itemAt(pos); idx >= 0 {
		c.pressedItem = idx
		c.highlight = idx
	} else if c.headerRect().ContainsPoint(pos) {
		c.pressedHeader = true
	}
}

// Release completes the gesture: selecting the pressed item when released over it,
// or closing the list when the field was pressed and released. Called by a
// @UIManager on a left-button release.
func (c *ComboBoxComponent) Release(pos math.Vector2) {
	if !c.open {
		c.pressedItem = -1
		c.pressedHeader = false
		c.openedOnPress = false
		return
	}
	// The release that follows the opening click must not immediately close the list.
	if c.openedOnPress {
		c.openedOnPress = false
		return
	}
	idx := c.itemAt(pos)
	if c.pressedItem >= 0 && idx == c.pressedItem {
		c.selectIndex(idx)
	} else if c.pressedHeader && c.headerRect().ContainsPoint(pos) {
		c.closeDropdown()
	}
	c.pressedItem = -1
	c.pressedHeader = false
}

// BeginAdjust starts a scrollbar drag when the press lands on the scrollbar, so the
// drag survives the pointer leaving the list (the @UIManager owns the gesture and
// keeps calling Adjust). Called by a @UIManager on a left-button press.
func (c *ComboBoxComponent) BeginAdjust(pos math.Vector2) {
	if !c.IsEnabled() {
		return
	}
	if c.open && c.scrollbarVisible() && c.scrollbarRect().ContainsPoint(pos) {
		c.draggingBar = true
		c.scrollToPointer(pos)
	}
}

// Adjust scrolls the list while a scrollbar drag is held. Called by a @UIManager
// every frame the left button stays down.
func (c *ComboBoxComponent) Adjust(pos math.Vector2) {
	if c.draggingBar {
		c.scrollToPointer(pos)
	}
}

// EndAdjust ends a scrollbar drag. Called by a @UIManager when the left button is
// released.
func (c *ComboBoxComponent) EndAdjust() {
	c.draggingBar = false
}

// Scroll adjusts the scroll offset from a mouse-wheel delta (a @UIManager forwards
// the frame's wheel). The y sign follows ebitengine's Wheel(): positive y = wheel up
// = earlier rows, so the offset decreases.
func (c *ComboBoxComponent) Scroll(delta math.Vector2) {
	if !c.open {
		return
	}
	c.scrollOffset -= delta.Y * c.itemHeight()
	c.clampScroll()
}

// SetFocused closes the list when focus leaves the combobox (e.g. a click outside).
// Opening stays on Press/Enter so the first click doesn't both focus and open.
func (c *ComboBoxComponent) SetFocused(focused bool) {
	if !focused {
		c.closeDropdown()
	}
}

// HandleInput handles keyboard: Enter/Space opens, ↑/↓ move the highlight, Enter
// selects, Escape closes. Called by a @UIManager while the combobox has focus.
func (c *ComboBoxComponent) HandleInput(ctx *core.Context) {
	if !c.IsEnabled() {
		return
	}
	in := ctx.Input

	if c.open {
		switch {
		case in.IsKeyJustPressed(core.KeyEscape):
			c.closeDropdown()
		case in.IsKeyJustPressed(core.KeyUp):
			c.moveHighlight(-1)
		case in.IsKeyJustPressed(core.KeyDown):
			c.moveHighlight(1)
		case in.IsKeyJustPressed(core.KeyEnter):
			if c.highlight >= 0 {
				c.selectIndex(c.highlight)
			} else {
				c.closeDropdown()
			}
		}
		return
	}

	if in.IsKeyJustPressed(core.KeyEnter) || in.IsKeyJustPressed(core.KeySpace) {
		c.openDropdown()
	}
}

// openDropdown opens the list, highlighting the current selection (or the first
// item when nothing is selected).
func (c *ComboBoxComponent) openDropdown() {
	c.open = true
	c.highlight = c.selectedIndex()
	if c.highlight < 0 && len(c.Items) > 0 {
		c.highlight = 0
	}
}

// closeDropdown closes the list and clears the highlight and press state.
func (c *ComboBoxComponent) closeDropdown() {
	c.open = false
	c.highlight = -1
	c.pressedItem = -1
	c.pressedHeader = false
	c.openedOnPress = false
}

// selectIndex sets Value to Items[i], closes the list, and emits Event (only when
// the value actually changed).
func (c *ComboBoxComponent) selectIndex(i int) {
	if i < 0 || i >= len(c.Items) {
		return
	}
	c.closeDropdown()
	if c.Items[i] == c.Value {
		return
	}
	c.Value = c.Items[i]
	if c.Event != "" {
		c.Emit(c.Event, c)
	}
}

// moveHighlight moves the keyboard cursor by delta, wrapping around.
func (c *ComboBoxComponent) moveHighlight(delta int) {
	n := len(c.Items)
	if n == 0 {
		return
	}
	if c.highlight < 0 {
		if delta < 0 {
			c.highlight = n - 1
		} else {
			c.highlight = 0
		}
		return
	}
	c.highlight = (c.highlight + delta) % n
	if c.highlight < 0 {
		c.highlight += n
	}
}

// selectedIndex returns the index of Value in Items, or -1 when absent.
func (c *ComboBoxComponent) selectedIndex() int {
	for i, item := range c.Items {
		if item == c.Value {
			return i
		}
	}
	return -1
}

// itemAt returns the index of the item under pos, or -1.
func (c *ComboBoxComponent) itemAt(pos math.Vector2) int {
	dr := c.dropdownRect()
	if !dr.ContainsPoint(pos) {
		return -1
	}
	// The scrollbar strip is not an item.
	if c.scrollbarVisible() && c.scrollbarRect().ContainsPoint(pos) {
		return -1
	}
	idx := int((pos.Y - dr.Top() + c.scrollOffset) / c.itemHeight())
	if idx < 0 || idx >= len(c.Items) {
		return -1
	}
	return idx
}

// headerRect returns the field rectangle (the element's own rect).
func (c *ComboBoxComponent) headerRect() math.Rect { return c.Rect() }

// dropdownRect returns the open list rectangle, below or above the field.
func (c *ComboBoxComponent) dropdownRect() math.Rect {
	hr := c.headerRect()
	h := c.dropdownHeight()
	if c.openUp() {
		return math.NewRect(hr.Left(), hr.Top()-h, hr.Width(), h)
	}
	return math.NewRect(hr.Left(), hr.Bottom(), hr.Width(), h)
}

// contentHeight returns the full height of all items (unscrolled).
func (c *ComboBoxComponent) contentHeight() float64 {
	return float64(len(c.Items)) * c.itemHeight()
}

// dropdownHeight returns the visible list height: the full content height capped to
// the room available on the side the list opens toward, so a long list scrolls
// instead of overflowing the screen. Unknown viewport (viewportH == 0) means no cap.
func (c *ComboBoxComponent) dropdownHeight() float64 {
	ch := c.contentHeight()
	if c.viewportH <= 0 {
		return ch
	}
	room := c.availableRoom()
	if room <= 0 {
		return ch
	}
	if ch > room {
		return room
	}
	return ch
}

// availableRoom returns the vertical room (screen pixels) on the side the list opens:
// below the field when it opens down, above it when it opens up.
func (c *ComboBoxComponent) availableRoom() float64 {
	hr := c.headerRect()
	if c.openUp() {
		return hr.Top()
	}
	return c.viewportH - hr.Bottom()
}

// maxScroll returns the largest legal scroll offset (content height minus visible
// height), or 0 when the list fits without scrolling.
func (c *ComboBoxComponent) maxScroll() float64 {
	m := c.contentHeight() - c.dropdownHeight()
	if m < 0 {
		return 0
	}
	return m
}

// clampScroll keeps the scroll offset within [0, maxScroll].
func (c *ComboBoxComponent) clampScroll() {
	c.scrollOffset = clampf(c.scrollOffset, 0, c.maxScroll())
}

// scrollbarWidth returns the scrollbar width.
func (c *ComboBoxComponent) scrollbarWidth() float64 {
	if c.ScrollbarWidth <= 0 {
		return 6
	}
	return c.ScrollbarWidth
}

// scrollbarVisible reports whether the capped list scrolls (so a scrollbar should be
// drawn and dragged).
func (c *ComboBoxComponent) scrollbarVisible() bool {
	return c.maxScroll() > 0
}

// scrollbarRect returns the scrollbar track rectangle on the dropdown's right edge.
func (c *ComboBoxComponent) scrollbarRect() math.Rect {
	dr := c.dropdownRect()
	w := c.scrollbarWidth()
	return math.NewRect(dr.Right()-w, dr.Top(), w, dr.Height())
}

// thumbRect returns the scrollbar thumb rectangle for the current scroll position.
func (c *ComboBoxComponent) thumbRect() math.Rect {
	track := c.scrollbarRect()
	dr := c.dropdownRect()
	contentH := c.contentHeight()
	thumbH := dr.Height() * dr.Height() / contentH
	if thumbH < 16 {
		thumbH = 16
	}
	if thumbH > dr.Height() {
		thumbH = dr.Height()
	}
	frac := 0.0
	if max := c.maxScroll(); max > 0 {
		frac = c.scrollOffset / max
	}
	y := track.Top() + frac*(track.Height()-thumbH)
	return math.NewRect(track.Left(), y, track.Width(), thumbH)
}

// scrollToPointer centers the scrollbar thumb on pos.Y (a drag or a track click).
func (c *ComboBoxComponent) scrollToPointer(pos math.Vector2) {
	track := c.scrollbarRect()
	thumbH := c.thumbRect().Height()
	avail := track.Height() - thumbH
	frac := 0.0
	if avail > 0 {
		frac = (pos.Y - track.Top() - thumbH/2) / avail
	}
	c.scrollOffset = clampf(frac, 0, 1) * c.maxScroll()
}

// itemHeight returns the row height.
func (c *ComboBoxComponent) itemHeight() float64 {
	if c.ItemHeight <= 0 {
		return 22
	}
	return c.ItemHeight
}

// textSize returns the font size.
func (c *ComboBoxComponent) textSize() float64 {
	if c.Size <= 0 {
		return 12
	}
	return c.Size
}

// openUp reports whether the list should open above the field: no (the default)
// when it fits below, yes when it only fits above, and whichever side has more room
// when it fits neither. Unknown viewport (viewportH == 0) defaults to below.
func (c *ComboBoxComponent) openUp() bool {
	if c.viewportH <= 0 {
		return false
	}
	ch := c.contentHeight()
	hr := c.headerRect()
	below := c.viewportH - hr.Bottom()
	above := hr.Top()
	if ch <= below {
		return false
	}
	if ch <= above {
		return true
	}
	return above > below
}

// GetDrawLayer returns a raised draw layer while the dropdown is open, so the list
// draws (and is hit-tested) above the object's other components for that moment.
func (c *ComboBoxComponent) GetDrawLayer() int {
	if c.open {
		return c.DrawLayer + popupLayerOffset
	}
	return c.DrawLayer
}

// Draw renders the field, then the open list.
func (c *ComboBoxComponent) Draw(r core.Renderer) {
	if !c.IsVisible() {
		return
	}
	if _, h := r.GetViewportSize(); h > 0 {
		c.viewportH = float64(h)
	}

	hr := c.headerRect()
	if c.Texture != "" {
		core.DrawNineSlice(r, c.Texture, c.Border, hr)
	} else {
		r.DrawRect(hr, c.Color)
	}
	c.drawOutline(r, hr)

	c.drawFieldText(r, hr)
	c.drawArrow(r, hr)

	if c.open {
		c.drawDropdown(r)
	}
}

// drawOutline strokes rect with the outline color.
func (c *ComboBoxComponent) drawOutline(r core.Renderer, rect math.Rect) {
	if c.OutlineThickness > 0 && c.OutlineColor.A > 0 {
		r.DrawRectOutline(rect, c.OutlineColor, c.OutlineThickness)
	}
}

// drawFieldText draws the value (or placeholder) in the field, left-aligned.
func (c *ComboBoxComponent) drawFieldText(r core.Renderer, hr math.Rect) {
	text := c.Value
	if text == "" {
		text = c.Placeholder
	}
	if text == "" {
		return
	}
	size := c.textSize()
	_, th := r.MeasureText(text, c.FontID, size)
	x := hr.Left() + 8
	y := hr.Center().Y - th/2
	r.DrawText(text, c.FontID, size, math.NewVector2(x, y), c.TextColor)
}

// drawArrow draws a small "∨" chevron on the right edge of the field.
func (c *ComboBoxComponent) drawArrow(r core.Renderer, hr math.Rect) {
	cx := hr.Right() - 12
	cy := hr.Center().Y
	w := 4.0
	h := 2.5
	r.DrawLine(math.NewVector2(cx-w, cy-h), math.NewVector2(cx, cy+h), c.TextColor, 2)
	r.DrawLine(math.NewVector2(cx+w, cy-h), math.NewVector2(cx, cy+h), c.TextColor, 2)
}

// drawDropdown draws the list and its items with their highlights.
func (c *ComboBoxComponent) drawDropdown(r core.Renderer) {
	dr := c.dropdownRect()
	c.clampScroll()

	// Clip everything drawn next (body + rows) to the dropdown rect, so rows that
	// scroll past the top/bottom edge are discarded rather than bleeding into the
	// field or the surrounding window.
	r.SetClipRect(dr)

	if c.DropdownTexture != "" {
		core.DrawNineSlice(r, c.DropdownTexture, c.DropdownBorder, dr)
	} else {
		r.DrawRect(dr, c.DropdownColor)
	}

	ih := c.itemHeight()
	size := c.textSize()
	sel := c.selectedIndex()
	// Rows shrink by the scrollbar width when it is visible, so text never runs under it.
	rowW := dr.Width()
	if c.scrollbarVisible() {
		rowW -= c.scrollbarWidth()
	}
	for i, item := range c.Items {
		rowTop := dr.Top() + float64(i)*ih - c.scrollOffset
		if rowTop+ih <= dr.Top() || rowTop >= dr.Bottom() {
			continue // culled; the clip also catches overflow, this skips the draw work
		}
		row := math.NewRect(dr.Left(), rowTop, rowW, ih)
		if i == c.highlight {
			r.DrawRect(row, c.HoverColor)
		} else if i == sel {
			r.DrawRect(row, c.SelectedColor)
		}
		_, th := r.MeasureText(item, c.FontID, size)
		x := row.Left() + 8
		y := row.Center().Y - th/2
		r.DrawText(item, c.FontID, size, math.NewVector2(x, y), c.TextColor)
	}

	// Scrollbar (drawn inside the clip so its thumb stays within the dropdown).
	if c.scrollbarVisible() {
		c.drawScrollbar(r)
	}

	r.ClearClip()

	c.drawOutline(r, dr)
}

// drawScrollbar draws the thumb over a dim track on the dropdown's right edge.
func (c *ComboBoxComponent) drawScrollbar(r core.Renderer) {
	track := c.scrollbarRect()
	r.DrawRect(track, c.DropdownColor.Lerp(math.Black, 0.3))
	r.DrawRect(c.thumbRect(), c.ScrollbarColor)
}
