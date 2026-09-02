package components

import (
	"sort"

	"github.com/EnesBaytekin/imge/core"
	"github.com/EnesBaytekin/imge/core/math"
)

// SceneTreeComponent is the editor's left panel: a scrollable list of the target
// scene's objects. Clicking a row selects that object and syncs the viewport's
// selection; the selected object's row is highlighted. Components are deliberately
// not listed here — they will live in a separate inspector window, so this list stays
// uncluttered. It reads the target scene from the viewport (the editor object named
// "viewport"), so the two panels stay in lockstep.
//
// Input is read directly from ctx.Input (like the viewport) — a background surface,
// not a @UIManager widget.
//
// Export variables (JSON args): background, object_text, tag_text, accent, font_id,
// font_size, row_height.
type SceneTreeComponent struct {
	core.BaseUIComponent

	// Theme. Zero means "use the default".
	Background math.Color `json:"background"`
	ObjectText math.Color `json:"object_text"`
	TagText    math.Color `json:"tag_text"` // dim "ui" tag
	Accent     math.Color `json:"accent"`   // selected-row highlight

	FontID    string  `json:"font_id"`   // "" = built-in pixel font
	FontSize  float64 `json:"font_size"` // 0 = default (6; the pixel font is grid-aligned to multiples of 6)
	RowHeight float64 `json:"row_height"`

	viewport *ViewportComponent
	scroll   float64      // content scroll offset in pixels (0 = top)
	hovered  *core.Object // object under the cursor (nil = none), for the hover highlight
}

// treeRow is one selectable line: a scene object at a content y (unscrolled).
type treeRow struct {
	obj *core.Object
	y   float64
}

func (t *SceneTreeComponent) Initialize() {
	if t.Background == (math.Color{}) {
		t.Background = math.NewColor(0x1d, 0x21, 0x30, 0xff)
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

	y := t.Rect().Y() + 4
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

// maxScroll returns the scroll offset at which the last row is just visible, or 0 when
// the content fits without scrolling.
func (t *SceneTreeComponent) maxScroll(rows []treeRow) float64 {
	if len(rows) == 0 {
		return 0
	}
	content := rows[len(rows)-1].y + t.RowHeight - t.Rect().Y()
	if m := content - t.Rect().Height(); m > 0 {
		return m
	}
	return 0
}

func (t *SceneTreeComponent) Update(ctx *core.Context) {
	if ctx == nil || ctx.Input == nil {
		return
	}
	mouse := ctx.Input.GetMousePosition()
	if !t.Rect().ContainsPoint(mouse) {
		t.hovered = nil
		return
	}

	// Yield to a window drawn above the tree — the @UIManager's blocking occlusion.
	if pointerOwnedElsewhere(t.GetScene(), t.GetOwner(), mouse) {
		t.hovered = nil
		return
	}

	rows := t.rows()
	t.hovered = t.rowAt(rows, mouse.Y)

	// Wheel scrolls the list (wheel up scrolls up).
	if s := ctx.Input.GetMouseScroll(); s.Y != 0 {
		t.scroll -= s.Y * t.RowHeight * 2
		if max := t.maxScroll(rows); t.scroll > max {
			t.scroll = max
		}
		if t.scroll < 0 {
			t.scroll = 0
		}
	}

	if !ctx.Input.IsMouseButtonJustPressed(core.MouseButtonLeft) {
		return
	}
	for _, row := range rows {
		y := row.y - t.scroll
		if mouse.Y >= y && mouse.Y < y+t.RowHeight {
			t.selectRow(row)
			return
		}
	}
}

// rowAt returns the object whose row is under mouseY (screen space, scroll-adjusted),
// or nil. Used for the hover highlight.
func (t *SceneTreeComponent) rowAt(rows []treeRow, mouseY float64) *core.Object {
	for _, row := range rows {
		y := row.y - t.scroll
		if mouseY >= y && mouseY < y+t.RowHeight {
			return row.obj
		}
	}
	return nil
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

	vp := t.viewportComponent()
	var selected *core.Object
	if vp != nil {
		selected = vp.SelectedObject()
	}

	for _, row := range t.rows() {
		y := row.y - t.scroll
		if y+t.RowHeight < rect.Y() || y > rect.Y()+rect.Height() {
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
		// Center on the measured line height (ascent+descent), not the nominal FontSize.
		// The built-in pixel font's line box is taller than its 6-unit design grid, so
		// centering on FontSize pushed the glyphs (and their descenders) below the row.
		// This matches the engine's @List, which centers on MeasureText's height.
		w, th := r.MeasureText(row.obj.Name, t.FontID, t.FontSize)
		ty := y + (t.RowHeight-th)/2
		if ty < y {
			ty = y
		}
		r.DrawText(row.obj.Name, t.FontID, t.FontSize, math.NewVector2(x, ty), t.ObjectText)
		if row.obj.UI {
			r.DrawText("ui", t.FontID, t.FontSize, math.NewVector2(x+w+6, ty), t.TagText)
		}
	}

	r.ClearClip()
}
