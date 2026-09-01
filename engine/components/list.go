package components

import (
	"github.com/EnesBaytekin/imge/core"
	"github.com/EnesBaytekin/imge/core/math"
)

// List is an always-open, scrollable, selectable list of string items — a listbox.
// Unlike @ComboBox (a dropdown that opens and closes), @List shows its rows inline
// inside its own rectangle and clips them to that rect, so a long list scrolls
// instead of overflowing. The items are plain strings: the text shown and the value
// returned on selection are the same string. A handler reads the result via
// GetValue() (or GetIndex()).
//
// Choosing:
//   - click a row to select it (release-to-activate, like @Button),
//   - with keyboard focus, ↑/↓ move the selection, wrapping around, and scroll the
//     selected row into view,
//   - the mouse wheel scrolls the list (the @UIManager forwards it), and the
//     scrollbar on the right edge shows position and can be dragged.
//
// Appearance: a flat color fill or a nine-slice texture (Texture/Border) for the
// body. HoverColor highlights the row under the pointer; SelectedColor marks the
// current value. OutlineColor/OutlineThickness stroke the whole list. Like @Button
// and @Panel, defaults can be supplied from styles.imge under the "@List" key.
//
// Export variables (JSON args): items, value, event, font_id, size, text_color,
// color, hover_color, selected_color, texture, border {left, top, right, bottom},
// outline_color, outline_thickness, item_height, scrollbar_color, scrollbar_width,
// offset, width, height, visible, enabled, blocking, group, draw_layer.
type ListComponent struct {
	core.BaseUIComponent

	// Items is the list of choices; the shown text and the selected value are the
	// same string.
	Items []string `json:"items"`

	// Value is the currently selected item (one of Items, or "" for none).
	Value string `json:"value"`

	// Event is emitted on selection, with the list itself as the data (so a handler
	// can read GetValue()/GetIndex() and GetName()/GetOwner()), matching @Button,
	// @Slider, and @ComboBox.
	Event string `json:"event"`

	// FontID and Size set the text font. Size 0 (default) = 12, a crisp multiple of
	// the built-in pixel font's 6-unit design grid.
	FontID string  `json:"font_id"`
	Size   float64 `json:"size"`

	// TextColor is the item text color.
	TextColor math.Color `json:"text_color"`

	// Color is the list body's flat fill when Texture is empty.
	Color math.Color `json:"color"`

	// HoverColor highlights the row under the pointer.
	HoverColor math.Color `json:"hover_color"`

	// SelectedColor marks the currently selected row.
	SelectedColor math.Color `json:"selected_color"`

	// Texture/Border opt the list body into nine-slice rendering; an empty texture
	// means a flat fill.
	Texture string      `json:"texture"`
	Border  math.Border `json:"border"`

	// OutlineColor/OutlineThickness stroke the list. A transparent color (the
	// default) means no outline; thickness 0 defaults to 1.
	OutlineColor     math.Color `json:"outline_color"`
	OutlineThickness float64    `json:"outline_thickness"`

	// ItemHeight is the height of one row; 0 (default) = 22.
	ItemHeight float64 `json:"item_height"`

	// ScrollbarColor is the thumb color; ScrollbarWidth is its width (0 default = 6).
	// The scrollbar only appears when the content is taller than the list.
	ScrollbarColor math.Color `json:"scrollbar_color"`
	ScrollbarWidth float64    `json:"scrollbar_width"`

	// scrollOffset is how far the content has been scrolled up, in logical units
	// (0 = items[0] at the top). highlight is the row index under the pointer (-1 =
	// none); pressedItem is the row index pressed (button-like release selection);
	// draggingBar marks a scrollbar drag in progress.
	scrollOffset float64
	highlight    int
	pressedItem  int
	draggingBar  bool
}

// Initialize defaults the colors, sizes, and flags, then opts into blocking and
// keyboard focus (a list is focusable for ↑/↓ navigation).
func (l *ListComponent) Initialize() {
	if l.Color == (math.Color{}) {
		l.Color = math.NewColor(30, 30, 42, 255)
	}
	if l.HoverColor == (math.Color{}) {
		l.HoverColor = math.NewColor(70, 90, 130, 255)
	}
	if l.SelectedColor == (math.Color{}) {
		l.SelectedColor = math.NewColor(90, 158, 90, 255)
	}
	if l.TextColor == (math.Color{}) {
		l.TextColor = math.White
	}
	if l.OutlineColor == (math.Color{}) {
		l.OutlineColor = math.NewColor(74, 74, 106, 255)
	}
	if l.ScrollbarColor == (math.Color{}) {
		l.ScrollbarColor = math.NewColor(90, 90, 120, 255)
	}
	if l.OutlineThickness <= 0 {
		l.OutlineThickness = 1
	}
	if l.Size <= 0 {
		l.Size = 12
	}
	if l.ItemHeight <= 0 {
		l.ItemHeight = 22
	}
	if l.ScrollbarWidth <= 0 {
		l.ScrollbarWidth = 6
	}
	if l.Blocking == nil {
		b := true
		l.Blocking = &b
	}
	l.Focusable = true
}

// GetValue returns the currently selected item string ("" when none).
func (l *ListComponent) GetValue() string { return l.Value }

// GetIndex returns the index of the current value in Items, or -1 when the value is
// not present (or the list is empty).
func (l *ListComponent) GetIndex() int { return l.selectedIndex() }

// SetValue sets the selection silently (no Event) to the given item if it is in
// Items; otherwise it clears the selection.
func (l *ListComponent) SetValue(v string) {
	l.Value = ""
	for _, item := range l.Items {
		if item == v {
			l.Value = v
			return
		}
	}
}

// SetItems replaces the item list. The selection is kept when it is still present;
// otherwise it is cleared. No Event is emitted.
func (l *ListComponent) SetItems(items []string) {
	l.Items = items
	if l.selectedIndex() < 0 {
		l.Value = ""
	}
}

// Contains reports whether p is over the list.
func (l *ListComponent) Contains(p math.Vector2) bool {
	return l.Rect().ContainsPoint(p)
}

// PointerMove updates the highlighted row from the pointer. Called by a @UIManager
// each frame the pointer is over the element.
func (l *ListComponent) PointerMove(pos math.Vector2) {
	l.highlight = l.itemAt(pos)
}

// PointerLeave clears the hover highlight. Called by a @UIManager.
func (l *ListComponent) PointerLeave() {
	l.highlight = -1
}

// Press records a row press (to select it on release). The scrollbar drag is owned
// by the uiSlider gesture (BeginAdjust), which runs just before this, so a press on
// the scrollbar is skipped here. Called by a @UIManager on a left-button press.
func (l *ListComponent) Press(pos math.Vector2) {
	if !l.IsEnabled() {
		return
	}
	l.pressedItem = -1
	if l.draggingBar {
		return // scrollbar drag in progress
	}
	if idx := l.itemAt(pos); idx >= 0 {
		l.pressedItem = idx
		l.highlight = idx
	}
}

// Release completes a row click: it selects the pressed row when released over it.
// Called by a @UIManager on a left-button release.
func (l *ListComponent) Release(pos math.Vector2) {
	idx := l.itemAt(pos)
	if l.pressedItem >= 0 && idx == l.pressedItem {
		l.selectIndex(idx)
	}
	l.pressedItem = -1
}

// BeginAdjust starts a scrollbar drag when the press lands on the scrollbar, so the
// drag survives the pointer leaving the list (the @UIManager owns the gesture and
// keeps calling Adjust). Called by a @UIManager on a left-button press.
func (l *ListComponent) BeginAdjust(pos math.Vector2) {
	if !l.IsEnabled() {
		return
	}
	if l.scrollbarVisible() && l.scrollbarRect().ContainsPoint(pos) {
		l.draggingBar = true
		l.scrollToPointer(pos)
	}
}

// Adjust scrolls the list while a scrollbar drag is held. Called by a @UIManager
// every frame the left button stays down.
func (l *ListComponent) Adjust(pos math.Vector2) {
	if l.draggingBar {
		l.scrollToPointer(pos)
	}
}

// EndAdjust ends a scrollbar drag. Called by a @UIManager when the left button is
// released.
func (l *ListComponent) EndAdjust() {
	l.draggingBar = false
}

// SetFocused is a no-op: the list stays open and keeps its selection whether or not
// it has keyboard focus (unlike a dropdown, which closes when focus leaves).
func (l *ListComponent) SetFocused(focused bool) {}

// HandleInput handles keyboard navigation: ↑/↓ move the selection (wrapping around)
// and scroll it into view. Called by a @UIManager while the list has focus.
func (l *ListComponent) HandleInput(ctx *core.Context) {
	if !l.IsEnabled() || len(l.Items) == 0 {
		return
	}
	in := ctx.Input
	switch {
	case in.IsKeyJustPressed(core.KeyUp):
		l.moveSelection(-1)
	case in.IsKeyJustPressed(core.KeyDown):
		l.moveSelection(1)
	case in.IsKeyJustPressed(core.KeyHome):
		l.selectIndex(0)
	case in.IsKeyJustPressed(core.KeyEnd):
		l.selectIndex(len(l.Items) - 1)
	}
}

// Scroll adjusts the scroll offset from a mouse-wheel delta (a @UIManager forwards
// the frame's wheel). The y sign follows ebitengine's Wheel(): positive y = wheel up
// = earlier rows, so the offset decreases.
func (l *ListComponent) Scroll(delta math.Vector2) {
	l.scrollOffset -= delta.Y * l.itemHeight()
	l.clampScroll()
}

// moveSelection moves the selection by delta (wrapping around) and scrolls it into
// view. Selection is immediate for a listbox, so the Event fires on ↑/↓ too.
func (l *ListComponent) moveSelection(delta int) {
	n := len(l.Items)
	if n == 0 {
		return
	}
	idx := l.selectedIndex()
	if idx < 0 {
		if delta < 0 {
			idx = n - 1
		} else {
			idx = 0
		}
	} else {
		idx = (idx + delta) % n
		if idx < 0 {
			idx += n
		}
	}
	l.selectIndex(idx)
}

// selectIndex sets Value to Items[i] and emits Event (only when the value actually
// changed), then scrolls the selected row into view.
func (l *ListComponent) selectIndex(i int) {
	if i < 0 || i >= len(l.Items) {
		return
	}
	if l.Items[i] != l.Value {
		l.Value = l.Items[i]
		if l.Event != "" {
			l.Emit(l.Event, l)
		}
	}
	l.scrollIntoView(i)
}

// selectedIndex returns the index of Value in Items, or -1 when absent.
func (l *ListComponent) selectedIndex() int {
	for i, item := range l.Items {
		if item == l.Value {
			return i
		}
	}
	return -1
}

// itemAt returns the index of the row under pos, or -1. The position is in screen
// space; the scroll offset shifts which row sits where. The scrollbar area is not a
// row.
func (l *ListComponent) itemAt(pos math.Vector2) int {
	rect := l.Rect()
	if !rect.ContainsPoint(pos) {
		return -1
	}
	if l.scrollbarVisible() && l.scrollbarRect().ContainsPoint(pos) {
		return -1
	}
	idx := int((pos.Y - rect.Top() + l.scrollOffset) / l.itemHeight())
	if idx < 0 || idx >= len(l.Items) {
		return -1
	}
	return idx
}

// itemHeight returns the row height.
func (l *ListComponent) itemHeight() float64 {
	if l.ItemHeight <= 0 {
		return 22
	}
	return l.ItemHeight
}

// textSize returns the font size.
func (l *ListComponent) textSize() float64 {
	if l.Size <= 0 {
		return 12
	}
	return l.Size
}

// contentHeight returns the total height of all rows.
func (l *ListComponent) contentHeight() float64 {
	return float64(len(l.Items)) * l.itemHeight()
}

// maxScroll returns the largest legal scroll offset (content height minus viewport
// height), or 0 when the content fits.
func (l *ListComponent) maxScroll() float64 {
	m := l.contentHeight() - l.Rect().Height()
	if m < 0 {
		return 0
	}
	return m
}

// clampScroll keeps the scroll offset within [0, maxScroll].
func (l *ListComponent) clampScroll() {
	l.scrollOffset = clampf(l.scrollOffset, 0, l.maxScroll())
}

// scrollIntoView adjusts the scroll offset so row i is fully visible.
func (l *ListComponent) scrollIntoView(i int) {
	ih := l.itemHeight()
	top := float64(i) * ih
	bottom := top + ih
	if top < l.scrollOffset {
		l.scrollOffset = top
	} else if bottom > l.scrollOffset+l.Rect().Height() {
		l.scrollOffset = bottom - l.Rect().Height()
	}
	l.clampScroll()
}

// scrollbarVisible reports whether the content overflows the list (so the scrollbar
// should be drawn and dragged).
func (l *ListComponent) scrollbarVisible() bool {
	return l.maxScroll() > 0
}

// scrollbarRect returns the scrollbar track rectangle on the list's right edge.
func (l *ListComponent) scrollbarRect() math.Rect {
	rect := l.Rect()
	w := l.scrollbarWidth()
	return math.NewRect(rect.Right()-w, rect.Top(), w, rect.Height())
}

// thumbRect returns the scrollbar thumb rectangle for the current scroll position.
func (l *ListComponent) thumbRect() math.Rect {
	track := l.scrollbarRect()
	rect := l.Rect()
	contentH := l.contentHeight()
	thumbH := rect.Height() * rect.Height() / contentH
	if thumbH < 16 {
		thumbH = 16
	}
	if thumbH > rect.Height() {
		thumbH = rect.Height()
	}
	maxScroll := l.maxScroll()
	frac := 0.0
	if maxScroll > 0 {
		frac = l.scrollOffset / maxScroll
	}
	y := track.Top() + frac*(track.Height()-thumbH)
	return math.NewRect(track.Left(), y, track.Width(), thumbH)
}

// scrollToPointer centers the scrollbar thumb on pos.Y (a drag or a track click).
func (l *ListComponent) scrollToPointer(pos math.Vector2) {
	track := l.scrollbarRect()
	thumbH := l.thumbRect().Height()
	avail := track.Height() - thumbH
	frac := 0.0
	if avail > 0 {
		frac = (pos.Y - track.Top() - thumbH/2) / avail
	}
	l.scrollOffset = clampf(frac, 0, 1) * l.maxScroll()
}

// scrollbarWidth returns the scrollbar width.
func (l *ListComponent) scrollbarWidth() float64 {
	if l.ScrollbarWidth <= 0 {
		return 6
	}
	return l.ScrollbarWidth
}

// Draw renders the list body, then the visible rows, clipped to the list rectangle
// so a long list scrolls instead of overflowing, then the scrollbar and outline.
func (l *ListComponent) Draw(r core.Renderer) {
	if !l.IsVisible() {
		return
	}
	rect := l.Rect()
	if rect.Width() <= 0 || rect.Height() <= 0 {
		return
	}
	l.clampScroll()

	// Clip everything drawn next (body + rows) to the list rect, so rows that scroll
	// past the top/bottom edge are discarded rather than bleeding into neighbors.
	r.SetClipRect(rect)

	if l.Texture != "" {
		core.DrawNineSlice(r, l.Texture, l.Border, rect)
	} else {
		r.DrawRect(rect, l.Color)
	}
	l.drawRows(r, rect)

	// The scrollbar is drawn inside the clip so its thumb stays within the body.
	if l.scrollbarVisible() {
		l.drawScrollbar(r)
	}

	r.ClearClip()

	l.drawOutline(r, rect)
}

// drawRows draws the visible rows (with their highlights) within the clip. The row
// width shrinks by the scrollbar width when the scrollbar is visible, so text never
// runs under it.
func (l *ListComponent) drawRows(r core.Renderer, rect math.Rect) {
	ih := l.itemHeight()
	size := l.textSize()
	sel := l.selectedIndex()
	rowW := rect.Width()
	if l.scrollbarVisible() {
		rowW -= l.scrollbarWidth()
	}

	for i, item := range l.Items {
		rowTop := rect.Top() + float64(i)*ih - l.scrollOffset
		rowBottom := rowTop + ih
		if rowBottom < rect.Top() || rowTop > rect.Bottom() {
			continue // culled; the clip also catches overflow, this skips the draw work
		}
		row := math.NewRect(rect.Left(), rowTop, rowW, ih)
		if i == l.highlight {
			r.DrawRect(row, l.HoverColor)
		} else if i == sel {
			r.DrawRect(row, l.SelectedColor)
		}
		_, th := r.MeasureText(item, l.FontID, size)
		x := row.Left() + 8
		y := row.Center().Y - th/2
		r.DrawText(item, l.FontID, size, math.NewVector2(x, y), l.TextColor)
	}
}

// drawScrollbar draws the thumb over a dim track on the right edge.
func (l *ListComponent) drawScrollbar(r core.Renderer) {
	track := l.scrollbarRect()
	r.DrawRect(track, l.Color)
	r.DrawRect(l.thumbRect(), l.ScrollbarColor)
}

// drawOutline strokes rect with the outline color.
func (l *ListComponent) drawOutline(r core.Renderer, rect math.Rect) {
	if l.OutlineThickness > 0 && l.OutlineColor.A > 0 {
		r.DrawRectOutline(rect, l.OutlineColor, l.OutlineThickness)
	}
}
