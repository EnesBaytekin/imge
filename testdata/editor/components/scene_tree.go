package components

import (
	"sort"

	"github.com/EnesBaytekin/imge/core"
	"github.com/EnesBaytekin/imge/core/math"
)

// SceneTreeComponent is the editor's left panel: a scrollable list of the target
// scene's objects, with a "+" button to add an empty object and an "x" button on each
// row to remove that object (behind a confirm dialog). Clicking a row selects that
// object and syncs the viewport's selection; the selected object's row is highlighted.
// Components are deliberately not listed here — they live in the inspector, so this
// list stays uncluttered. It reads the target scene from the viewport (the editor
// object named "viewport"), so the two panels stay in lockstep.
//
// Input is read directly from ctx.Input (like the viewport) — a background surface,
// not a @UIManager widget. A thin scrollbar on the right (draggable thumb + mouse
// wheel) appears when the object list overflows the panel, so scrolled-out rows stay
// reachable.
//
// Export variables (JSON args): background, title_text, object_text, tag_text, accent,
// error_color, scroll_track, scroll_thumb, font_id, font_size, row_height.
type SceneTreeComponent struct {
	core.BaseUIComponent

	// Theme. Zero means "use the default".
	Background math.Color `json:"background"`
	TitleText  math.Color `json:"title_text"`  // "OBJECTS" header + "+"
	ObjectText math.Color `json:"object_text"` // object name
	TagText    math.Color `json:"tag_text"`    // dim "ui" tag / "x" button
	Accent     math.Color `json:"accent"`      // title bar + selected-row highlight
	ErrorColor math.Color `json:"error_color"` // "x" hover

	ScrollTrack math.Color `json:"scroll_track"`
	ScrollThumb math.Color `json:"scroll_thumb"`

	FontID    string  `json:"font_id"`   // "" = built-in pixel font
	FontSize  float64 `json:"font_size"` // 0 = default (6; the pixel font is grid-aligned to multiples of 6)
	RowHeight float64 `json:"row_height"`

	viewport *ViewportComponent
	scroll   float64 // content scroll offset in pixels (0 = top)

	hovered        *core.Object // object under the cursor (nil = none), for the hover highlight
	hoverX         *core.Object // object whose "x" remove button is under the cursor (nil = none)
	hoverDup       *core.Object // object whose "=" duplicate button is under the cursor (nil = none)
	hoverPlus      bool         // the "+" add-object button is under the cursor
	scrollDragging bool         // the scrollbar thumb is being dragged
	scrollGrab     float64      // mouse Y offset within the thumb when the drag began
}

// treeRow is one selectable line: a scene object at a content y (unscrolled).
type treeRow struct {
	obj *core.Object
	y   float64
}

// titleH returns the title-bar height.
func (t *SceneTreeComponent) titleH() float64 { return t.RowHeight + 8 }

// bodyTop returns the content-space y where the first row begins (below the title bar).
func (t *SceneTreeComponent) bodyTop() float64 { return t.Rect().Y() + t.titleH() + 2 }

// bodyHeight returns the scrollable list height below the title bar.
func (t *SceneTreeComponent) bodyHeight(rect math.Rect) float64 {
	h := rect.Height() - t.titleH()
	if h < 0 {
		h = 0
	}
	return h
}

// plusRect returns the "+" add-object button rect in the title bar's top-right corner.
func (t *SceneTreeComponent) plusRect(rect math.Rect) math.Rect {
	const s = 14.0
	return math.NewRect(rect.X()+rect.Width()-18, rect.Y()+(t.titleH()-s)/2, s, s)
}

// xRect returns the "x" remove-button strip at the right edge of a row, just left of the
// scrollbar so the two never overlap.
func (t *SceneTreeComponent) xRect(rect math.Rect, rowY float64) math.Rect {
	const xw, sbW = 14.0, 6.0
	return math.NewRect(rect.X()+rect.Width()-sbW-xw-2, rowY, xw, t.RowHeight)
}

// dupRect returns the "=" duplicate-button strip at the right edge of a row, immediately
// left of the "x" remove strip so the two controls sit side by side.
func (t *SceneTreeComponent) dupRect(rect math.Rect, rowY float64) math.Rect {
	const dw = 14.0
	xr := t.xRect(rect, rowY)
	return math.NewRect(xr.X()-dw, rowY, dw, t.RowHeight)
}

// scrollTrack returns the scrollbar track rect along the list's right edge.
func (t *SceneTreeComponent) scrollTrack(rect math.Rect) math.Rect {
	const sbW = 6.0
	return math.NewRect(rect.X()+rect.Width()-sbW-2, rect.Y()+t.titleH(), sbW, t.bodyHeight(rect))
}

func (t *SceneTreeComponent) Initialize() {
	if t.Background == (math.Color{}) {
		t.Background = math.NewColor(0x1d, 0x21, 0x30, 0xff)
	}
	if t.TitleText == (math.Color{}) {
		t.TitleText = math.NewColor(0xff, 0xff, 0xff, 0xff)
	}
	if t.ObjectText == (math.Color{}) {
		t.ObjectText = math.NewColor(0xe6, 0xe6, 0xef, 0xff)
	}
	if t.TagText == (math.Color{}) {
		t.TagText = math.NewColor(0x6b, 0x73, 0x85, 0xff)
	}
	if t.Accent == (math.Color{}) {
		t.Accent = math.NewColor(0x2f, 0x3b, 0x54, 0xff)
	}
	if t.ErrorColor == (math.Color{}) {
		t.ErrorColor = math.NewColor(0xff, 0x5a, 0x5a, 0xff)
	}
	if t.ScrollTrack == (math.Color{}) {
		t.ScrollTrack = math.NewColor(0x2a, 0x30, 0x42, 0xff)
	}
	if t.ScrollThumb == (math.Color{}) {
		t.ScrollThumb = math.NewColor(0x4a, 0x55, 0x70, 0xff)
	}
	if t.FontSize <= 0 {
		t.FontSize = 6
	}
	if t.RowHeight <= 0 {
		t.RowHeight = 14
	}
	// The tree is an opaque surface: it blocks pointer events so the @UIManager
	// occludes whatever is drawn behind it (see pointerOwnedElsewhere).
	if t.Blocking == nil {
		t.SetBlocking(true)
	}
}

// viewportComponent returns the editor's ViewportComponent, looked up lazily by the
// "viewport" object name and cached.
func (t *SceneTreeComponent) viewportComponent() *ViewportComponent {
	if t.viewport == nil {
		if scene := t.GetScene(); scene != nil {
			if obj := scene.GetObjectByName("viewport"); obj != nil {
				t.viewport = core.GetFrom[*ViewportComponent](obj)
			}
		}
	}
	return t.viewport
}

// rows lays out one row per target-scene object, sorted by name. Returns nil when the
// target scene isn't loaded yet.
func (t *SceneTreeComponent) rows() []treeRow {
	vp := t.viewportComponent()
	if vp == nil || vp.TargetScene() == nil {
		return nil
	}
	objs := vp.TargetScene().GetSortedObjects()
	sort.Slice(objs, func(i, j int) bool { return objs[i].Name < objs[j].Name })

	y := t.bodyTop()
	rows := make([]treeRow, 0, len(objs))
	for _, obj := range objs {
		if obj == nil {
			continue
		}
		rows = append(rows, treeRow{obj: obj, y: y})
		y += t.RowHeight
	}
	return rows
}

// contentHeight returns the total height of the row list (from bodyTop), including a
// small bottom pad, so the scrollbar proportion tracks the overflow.
func (t *SceneTreeComponent) contentHeight(rows []treeRow) float64 {
	if len(rows) == 0 {
		return 0
	}
	return rows[len(rows)-1].y + t.RowHeight - t.bodyTop() + 2
}

// maxScroll returns the scroll offset at which the last row is just visible, or 0 when
// the content fits without scrolling.
func (t *SceneTreeComponent) maxScroll(rows []treeRow) float64 {
	if m := t.contentHeight(rows) - t.bodyHeight(t.Rect()); m > 0 {
		return m
	}
	return 0
}

// clampScroll keeps the scroll offset within [0, maxScroll].
func (t *SceneTreeComponent) clampScroll(rows []treeRow) {
	if max := t.maxScroll(rows); t.scroll > max {
		t.scroll = max
	}
	if t.scroll < 0 {
		t.scroll = 0
	}
}

func (t *SceneTreeComponent) Update(ctx *core.Context) {
	if ctx == nil || ctx.Input == nil {
		return
	}
	// A modal or an open menu bar is up: this panel is inert.
	if modalOpen() || menusOpen() {
		return
	}
	mouse := ctx.Input.GetMousePosition()
	rect := t.Rect()

	// A scrollbar drag keeps following the cursor even outside the panel.
	if t.scrollDragging {
		if ctx.Input.IsMouseButtonPressed(core.MouseButtonLeft) {
			rows := t.rows()
			t.scroll = scrollFromThumb(t.scrollTrack(rect), t.contentHeight(rows), t.maxScroll(rows), mouse.Y, t.scrollGrab)
			t.clampScroll(rows)
		} else {
			t.scrollDragging = false
		}
	}

	if !rect.ContainsPoint(mouse) {
		t.hovered = nil
		t.hoverX = nil
		t.hoverDup = nil
		t.hoverPlus = false
		return
	}

	// Yield to a window drawn above the tree — the @UIManager's blocking occlusion.
	if pointerOwnedElsewhere(t.GetScene(), t.GetOwner(), mouse) {
		t.hovered = nil
		t.hoverX = nil
		t.hoverDup = nil
		t.hoverPlus = false
		return
	}

	rows := t.rows()

	// Wheel scrolls the list (wheel up scrolls up), before the hover/click tests so all
	// of them use the same (post-scroll) row layout this frame.
	if s := ctx.Input.GetMouseScroll(); s.Y != 0 {
		t.scroll -= s.Y * t.RowHeight * 2
		t.clampScroll(rows)
	}

	ri := t.rowIndex(rows, mouse.Y)
	t.hovered = nil
	t.hoverX = nil
	t.hoverDup = nil
	if ri >= 0 {
		t.hovered = rows[ri].obj
		if t.xRect(rect, rows[ri].y-t.scroll).ContainsPoint(mouse) {
			t.hoverX = rows[ri].obj
		}
		if t.dupRect(rect, rows[ri].y-t.scroll).ContainsPoint(mouse) {
			t.hoverDup = rows[ri].obj
		}
	}
	t.hoverPlus = t.plusRect(rect).ContainsPoint(mouse)

	if !ctx.Input.IsMouseButtonJustPressed(core.MouseButtonLeft) {
		return
	}

	// "+" button: directly add an empty object and select it, so its name can be edited
	// in the inspector.
	if t.hoverPlus {
		if vp := t.viewportComponent(); vp != nil {
			if scene := vp.TargetScene(); scene != nil {
				if obj := addObjectTo(scene); obj != nil {
					vp.SelectSilent(obj)
				}
			}
		}
		return
	}
	// "x" strip: confirm removal of that object. The object is captured now (before it
	// can be removed), so the confirm callback removes the right one.
	if t.hoverX != nil {
		obj := t.hoverX
		spawnConfirmDialog(t.GetScene(), "Delete \""+obj.Name+"\"?", func() {
			if vp := t.viewportComponent(); vp != nil {
				if scene := vp.TargetScene(); scene != nil {
					removeObjectFrom(scene, obj)
					if vp.SelectedObject() == obj {
						vp.SelectSilent(nil)
					}
				}
			}
		})
		return
	}
	// "=" strip: duplicate that object (a fresh copy with a unique name), then select
	// the copy so its name can be edited right away.
	if t.hoverDup != nil {
		obj := t.hoverDup
		if vp := t.viewportComponent(); vp != nil {
			if scene := vp.TargetScene(); scene != nil {
				if dup := duplicateObject(scene, obj); dup != nil {
					vp.SelectSilent(dup)
				}
			}
		}
		return
	}
	// Scrollbar press: grabbing the thumb starts a drag, clicking the track jumps the
	// thumb to the cursor.
	if t.handleScrollbarPress(mouse, rows, rect) {
		return
	}
	// Row click: select.
	if ri >= 0 {
		t.selectRow(rows[ri])
	}
}

// rowIndex returns the index of the row under mouseY (screen space, scroll-adjusted),
// or -1 when none.
func (t *SceneTreeComponent) rowIndex(rows []treeRow, mouseY float64) int {
	for i, row := range rows {
		y := row.y - t.scroll
		if mouseY >= y && mouseY < y+t.RowHeight {
			return i
		}
	}
	return -1
}

// handleScrollbarPress consumes a click on the scrollbar: grabbing the thumb starts a
// drag, and clicking the track jumps the thumb (centered) to the cursor. Returns true
// when the press landed on the scrollbar.
func (t *SceneTreeComponent) handleScrollbarPress(mouse math.Vector2, rows []treeRow, rect math.Rect) bool {
	track := t.scrollTrack(rect)
	contentH := t.contentHeight(rows)
	max := t.maxScroll(rows)
	thumb, ok := scrollThumb(track, contentH, t.scroll, max)
	if !ok {
		return false
	}
	if thumb.ContainsPoint(mouse) {
		t.scrollDragging = true
		t.scrollGrab = mouse.Y - thumb.Y()
		return true
	}
	if track.ContainsPoint(mouse) {
		t.scroll = scrollFromThumb(track, contentH, max, mouse.Y, thumb.Height()/2)
		t.clampScroll(rows)
		return true
	}
	return false
}

// selectRow selects the clicked row's object in the viewport.
func (t *SceneTreeComponent) selectRow(row treeRow) {
	if vp := t.viewportComponent(); vp != nil {
		vp.Select(row.obj)
	}
}

func (t *SceneTreeComponent) Draw(r core.Renderer) {
	rect := t.Rect()
	if rect.Width() <= 0 || rect.Height() <= 0 {
		return
	}

	r.SetClipRect(rect)
	r.DrawRect(rect, t.Background)

	// Line height is constant for a font+size.
	_, th := r.MeasureText("Ag", t.FontID, t.FontSize)

	// Title bar: "OBJECTS" on the left, "+" add button on the right.
	r.DrawRect(math.NewRect(rect.X(), rect.Y(), rect.Width(), t.titleH()), t.Accent)
	titleY := rect.Y() + (t.titleH()-th)/2
	r.DrawText("OBJECTS", t.FontID, t.FontSize, math.NewVector2(rect.X()+6, titleY), t.TitleText)
	plus := t.plusRect(rect)
	if t.hoverPlus {
		r.DrawRect(plus, t.Background.Lerp(math.White, 0.12))
	}
	pw, ph := r.MeasureText("+", t.FontID, t.FontSize)
	r.DrawText("+", t.FontID, t.FontSize, math.NewVector2(plus.X()+(plus.Width()-pw)/2, plus.Y()+(plus.Height()-ph)/2), t.TitleText)

	vp := t.viewportComponent()
	var selected *core.Object
	if vp != nil {
		selected = vp.SelectedObject()
	}

	// Object rows, clipped to the body below the title bar so scrolled-out rows don't
	// bleed over the header.
	rows := t.rows()
	bodyTop := t.bodyTop()
	r.SetClipRect(math.NewRect(rect.X(), bodyTop, rect.Width(), t.bodyHeight(rect)))
	for _, row := range rows {
		y := row.y - t.scroll
		if y+t.RowHeight < bodyTop || y > rect.Y()+rect.Height() {
			continue
		}
		isSelected := selected != nil && selected == row.obj
		if t.hovered != nil && t.hovered == row.obj && !isSelected {
			// A faint highlight under the cursor, distinct from (and below) the
			// stronger selection highlight, so hovered rows read as clickable.
			r.DrawRect(math.NewRect(rect.X(), y, rect.Width(), t.RowHeight), t.Background.Lerp(math.White, 0.07))
		}
		if isSelected {
			r.DrawRect(math.NewRect(rect.X(), y, rect.Width(), t.RowHeight), t.Accent)
		}

		x := rect.X() + 6
		// Center on the measured line height (ascent+descent), not the nominal FontSize,
		// matching the engine's @List and the inspector.
		w, th2 := r.MeasureText(row.obj.Name, t.FontID, t.FontSize)
		ty := y + (t.RowHeight-th2)/2
		if ty < y {
			ty = y
		}
		r.DrawText(row.obj.Name, t.FontID, t.FontSize, math.NewVector2(x, ty), t.ObjectText)
		if row.obj.UI {
			r.DrawText("ui", t.FontID, t.FontSize, math.NewVector2(x+w+6, ty), t.TagText)
		}

		// "=" duplicate button in the strip left of the "x" remove button.
		dr := t.dupRect(rect, y)
		dupColor := t.TagText
		if t.hoverDup != nil && t.hoverDup == row.obj {
			r.DrawRect(dr, t.Accent)
			dupColor = t.TitleText
		}
		dw, dh := r.MeasureText("=", t.FontID, t.FontSize)
		r.DrawText("=", t.FontID, t.FontSize, math.NewVector2(dr.X()+(dr.Width()-dw)/2, y+(t.RowHeight-dh)/2), dupColor)

		// "x" remove button in the right-edge strip (red on hover).
		xr := t.xRect(rect, y)
		xColor := t.TagText
		if t.hoverX != nil && t.hoverX == row.obj {
			r.DrawRect(xr, t.ErrorColor)
			xColor = t.TitleText
		}
		xw, xh := r.MeasureText("x", t.FontID, t.FontSize)
		r.DrawText("x", t.FontID, t.FontSize, math.NewVector2(xr.X()+(xr.Width()-xw)/2, y+(t.RowHeight-xh)/2), xColor)
	}

	// Scrollbar, drawn on top when the object list overflows the body.
	r.SetClipRect(rect)
	track := t.scrollTrack(rect)
	if thumb, ok := scrollThumb(track, t.contentHeight(rows), t.scroll, t.maxScroll(rows)); ok {
		drawScrollbar(r, track, thumb, t.ScrollTrack, t.ScrollThumb)
	}

	r.ClearClip()
}
