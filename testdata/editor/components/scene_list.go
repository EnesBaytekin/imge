package components

import (
	"github.com/EnesBaytekin/imge/core"
	"github.com/EnesBaytekin/imge/core/math"
)

// SceneListComponent is the editor's bottom-left panel: a list of the target
// project's .scene files. Clicking a row switches the viewport to edit that scene
// (the active scene's row is highlighted). Each row also carries a "*" set-as-start
// button (filled when that scene is game.imge's initial_scene) and an "x" delete
// button (behind a confirm dialog); the title bar has a "#" button to edit the active
// scene's settings and a "+" button to create a new scene.
//
// Like the scene tree, it reads input directly from ctx.Input (a background surface,
// not a @UIManager widget) and reads the target from the viewport, so the two panels
// stay in lockstep. It draws nothing when no project is loaded.
//
// Export variables (JSON args): background, title_text, scene_text, tag_text, accent,
// error_color, scroll_track, scroll_thumb, font_id, font_size, row_height.
type SceneListComponent struct {
	core.BaseUIComponent

	// Theme. Zero means "use the default".
	Background math.Color `json:"background"`
	TitleText  math.Color `json:"title_text"`  // "SCENES" header + "+" / "#" buttons
	SceneText  math.Color `json:"scene_text"`  // scene name
	TagText    math.Color `json:"tag_text"`    // dim "*" / "x" button
	Accent     math.Color `json:"accent"`      // title bar + active-row highlight
	ErrorColor math.Color `json:"error_color"` // "x" hover

	ScrollTrack math.Color `json:"scroll_track"`
	ScrollThumb math.Color `json:"scroll_thumb"`

	FontID    string  `json:"font_id"`   // "" = built-in pixel font
	FontSize  float64 `json:"font_size"` // 0 = default (6)
	RowHeight float64 `json:"row_height"`

	viewport *ViewportComponent
	scroll   float64 // content scroll offset in pixels (0 = top)

	entries     []sceneEntry // cached scene list
	lastProject string       // project dir the cache was built for
	initial     string       // current game.imge initial_scene (display name)

	hovered        *sceneEntry // scene under the cursor (nil = none)
	hoverX         *sceneEntry // scene whose "x" delete button is under the cursor
	hoverStar      *sceneEntry // scene whose "*" set-as-start button is under the cursor
	hoverGear      bool        // the "#" settings button is under the cursor
	hoverPlus      bool        // the "+" new-scene button is under the cursor
	scrollDragging bool        // the scrollbar thumb is being dragged
	scrollGrab     float64     // mouse Y offset within the thumb when the drag began
}

// lookupSceneList resolves the editor's SceneListComponent by object name. The
// scene-settings and new-scene modals call it to refresh the list after their action.
func lookupSceneList(scene *core.Scene) *SceneListComponent {
	if scene == nil {
		return nil
	}
	if obj := scene.GetObjectByName("scenes"); obj != nil {
		return core.GetFrom[*SceneListComponent](obj)
	}
	return nil
}

// titleH returns the title-bar height.
func (t *SceneListComponent) titleH() float64 { return t.RowHeight + 8 }

// bodyTop returns the content-space y where the first row begins (below the title bar).
func (t *SceneListComponent) bodyTop() float64 { return t.Rect().Y() + t.titleH() + 2 }

// bodyHeight returns the scrollable list height below the title bar.
func (t *SceneListComponent) bodyHeight(rect math.Rect) float64 {
	h := rect.Height() - t.titleH()
	if h < 0 {
		h = 0
	}
	return h
}

// plusRect returns the "+" new-scene button rect in the title bar's top-right corner.
func (t *SceneListComponent) plusRect(rect math.Rect) math.Rect {
	const s = 14.0
	return math.NewRect(rect.X()+rect.Width()-18, rect.Y()+(t.titleH()-s)/2, s, s)
}

// gearRect returns the "#" settings button rect, just left of the "+" button.
func (t *SceneListComponent) gearRect(rect math.Rect) math.Rect {
	const s, gap = 14.0, 2.0
	pr := t.plusRect(rect)
	return math.NewRect(pr.X()-s-gap, rect.Y()+(t.titleH()-s)/2, s, s)
}

// xRect returns the "x" delete-button strip at the right edge of a row, just left of
// the scrollbar so the two never overlap.
func (t *SceneListComponent) xRect(rect math.Rect, rowY float64) math.Rect {
	const xw, sbW = 14.0, 6.0
	return math.NewRect(rect.X()+rect.Width()-sbW-xw-2, rowY, xw, t.RowHeight)
}

// starRect returns the "*" set-as-start strip immediately left of the "x" strip.
func (t *SceneListComponent) starRect(rect math.Rect, rowY float64) math.Rect {
	const sw = 14.0
	xr := t.xRect(rect, rowY)
	return math.NewRect(xr.X()-sw, rowY, sw, t.RowHeight)
}

// scrollTrack returns the scrollbar track rect along the list's right edge.
func (t *SceneListComponent) scrollTrack(rect math.Rect) math.Rect {
	const sbW = 6.0
	return math.NewRect(rect.X()+rect.Width()-sbW-2, rect.Y()+t.titleH(), sbW, t.bodyHeight(rect))
}

func (t *SceneListComponent) Initialize() {
	if t.Background == (math.Color{}) {
		t.Background = math.NewColor(0x1d, 0x21, 0x30, 0xff)
	}
	if t.TitleText == (math.Color{}) {
		t.TitleText = math.NewColor(0xff, 0xff, 0xff, 0xff)
	}
	if t.SceneText == (math.Color{}) {
		t.SceneText = math.NewColor(0xe6, 0xe6, 0xef, 0xff)
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
	// The list is an opaque surface: it blocks pointer events so the @UIManager
	// occludes whatever is drawn behind it (see pointerOwnedElsewhere).
	if t.Blocking == nil {
		t.SetBlocking(true)
	}
}

// viewportComponent returns the editor's ViewportComponent, looked up lazily by the
// "viewport" object name and cached.
func (t *SceneListComponent) viewportComponent() *ViewportComponent {
	if t.viewport == nil {
		if scene := t.GetScene(); scene != nil {
			if obj := scene.GetObjectByName("viewport"); obj != nil {
				t.viewport = core.GetFrom[*ViewportComponent](obj)
			}
		}
	}
	return t.viewport
}

// refresh re-scans the target project's scenes and the current initial_scene, then
// resets the scroll. Called when the project changes and after create/delete/settings.
func (t *SceneListComponent) refresh() {
	t.entries = nil
	t.initial = ""
	vp := t.viewportComponent()
	if vp != nil {
		dir := vp.CurrentProject()
		t.lastProject = dir
		t.entries = listScenes(dir)
		t.initial = initialSceneName(dir)
	}
	t.scroll = 0
}

// rowY returns the content-space (unscrolled) y of row i.
func (t *SceneListComponent) rowY(i int) float64 {
	return t.bodyTop() + float64(i)*t.RowHeight
}

// rowIndex returns the index of the row under mouseY (screen space, scroll-adjusted),
// or -1 when none.
func (t *SceneListComponent) rowIndex(mouseY float64) int {
	for i := range t.entries {
		y := t.rowY(i) - t.scroll
		if mouseY >= y && mouseY < y+t.RowHeight {
			return i
		}
	}
	return -1
}

// contentHeight returns the total height of the row list (from bodyTop), including a
// small bottom pad, so the scrollbar proportion tracks the overflow.
func (t *SceneListComponent) contentHeight() float64 {
	if len(t.entries) == 0 {
		return 0
	}
	return float64(len(t.entries))*t.RowHeight + 2
}

// maxScroll returns the scroll offset at which the last row is just visible, or 0 when
// the content fits without scrolling.
func (t *SceneListComponent) maxScroll() float64 {
	if m := t.contentHeight() - t.bodyHeight(t.Rect()); m > 0 {
		return m
	}
	return 0
}

// clampScroll keeps the scroll offset within [0, maxScroll].
func (t *SceneListComponent) clampScroll() {
	if max := t.maxScroll(); t.scroll > max {
		t.scroll = max
	}
	if t.scroll < 0 {
		t.scroll = 0
	}
}

func (t *SceneListComponent) Update(ctx *core.Context) {
	if ctx == nil || ctx.Input == nil {
		return
	}
	// Re-scan when the target project changes — even under a modal (e.g. right after
	// Open Project), so the list reflects the new project the moment the modal closes.
	if vp := t.viewportComponent(); vp != nil && vp.CurrentProject() != t.lastProject {
		t.refresh()
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
			t.scroll = scrollFromThumb(t.scrollTrack(rect), t.contentHeight(), t.maxScroll(), mouse.Y, t.scrollGrab)
			t.clampScroll()
		} else {
			t.scrollDragging = false
		}
	}

	if !rect.ContainsPoint(mouse) {
		t.clearHover()
		return
	}

	// Yield to a window drawn above the list — the @UIManager's blocking occlusion.
	if pointerOwnedElsewhere(t.GetScene(), t.GetOwner(), mouse) {
		t.clearHover()
		return
	}

	// Wheel scrolls the list (wheel up scrolls up), before the hover/click tests so all
	// of them use the same (post-scroll) row layout this frame.
	if s := ctx.Input.GetMouseScroll(); s.Y != 0 {
		t.scroll -= s.Y * t.RowHeight * 2
		t.clampScroll()
	}

	ri := t.rowIndex(mouse.Y)
	t.clearHover()
	if ri >= 0 {
		entry := &t.entries[ri]
		t.hovered = entry
		y := t.rowY(ri) - t.scroll
		if t.xRect(rect, y).ContainsPoint(mouse) {
			t.hoverX = entry
		}
		if t.starRect(rect, y).ContainsPoint(mouse) {
			t.hoverStar = entry
		}
	}
	t.hoverPlus = t.plusRect(rect).ContainsPoint(mouse)
	t.hoverGear = t.gearRect(rect).ContainsPoint(mouse)

	if !ctx.Input.IsMouseButtonJustPressed(core.MouseButtonLeft) {
		return
	}

	if t.hoverPlus {
		spawnNewScene(t.GetScene())
		return
	}
	if t.hoverGear {
		spawnSceneSettings(t.GetScene())
		return
	}
	// "x" strip: confirm removal of that scene. The entry is copied now (before the
	// list can be refreshed), so the confirm callback deletes the right one.
	if t.hoverX != nil {
		entry := *t.hoverX
		spawnConfirmDialog(t.GetScene(), "Delete \""+entry.name+"\"?", func() {
			t.deleteScene(entry)
		})
		return
	}
	// "*" strip: mark that scene as the game's start scene.
	if t.hoverStar != nil {
		entry := *t.hoverStar
		if vp := t.viewportComponent(); vp != nil {
			if err := setInitialScene(vp.CurrentProject(), entry.name); err != nil {
				console.Print("set initial scene: " + err.Error())
			}
			t.initial = entry.name
		}
		return
	}
	// Scrollbar press: grabbing the thumb starts a drag, clicking the track jumps the
	// thumb to the cursor.
	if t.handleScrollbarPress(mouse, rect) {
		return
	}
	// Row click: switch to that scene.
	if ri >= 0 {
		if vp := t.viewportComponent(); vp != nil {
			vp.SetScene(t.entries[ri].file)
		}
	}
}

// clearHover resets the hover state.
func (t *SceneListComponent) clearHover() {
	t.hovered = nil
	t.hoverX = nil
	t.hoverStar = nil
	t.hoverPlus = false
	t.hoverGear = false
}

// handleScrollbarPress consumes a click on the scrollbar: grabbing the thumb starts a
// drag, and clicking the track jumps the thumb (centered) to the cursor. Returns true
// when the press landed on the scrollbar.
func (t *SceneListComponent) handleScrollbarPress(mouse math.Vector2, rect math.Rect) bool {
	track := t.scrollTrack(rect)
	contentH := t.contentHeight()
	max := t.maxScroll()
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
		t.clampScroll()
		return true
	}
	return false
}

// deleteScene removes a scene file and re-resolves the surrounding state: if it was
// the active scene, the viewport moves to the first remaining scene (or clears); if it
// was the initial scene, initial_scene moves to the new active/first scene.
func (t *SceneListComponent) deleteScene(entry sceneEntry) {
	vp := t.viewportComponent()
	if vp == nil {
		return
	}
	dir := vp.CurrentProject()
	if dir == "" {
		return
	}
	wasActive := vp.CurrentSceneName() == entry.file
	wasInitial := t.initial == entry.name

	// Capture the file's bytes before removing it so the deletion is undoable.
	if !captureDeletedScene(entry, dir, wasActive, wasInitial) {
		console.Print("delete scene: cannot read scene for undo")
	}

	if err := deleteSceneFile(entry.path); err != nil {
		console.Print("delete scene: " + err.Error())
		return
	}
	// A scene deletion is a top-level undo boundary: the next Ctrl+Z restores the
	// deleted scene (restoreLastDeletedScene) instead of undoing an in-scene edit.
	// Clearing history also drops entries that reference live objects the scene
	// switch below invalidates anyway.
	history.clear()
	// Drop the deleted scene from the viewport first, so the switch below doesn't
	// auto-save it back over the just-removed file.
	if wasActive {
		vp.ClearScene()
	}
	if wasInitial {
		remaining := listScenes(dir)
		next := ""
		if len(remaining) > 0 {
			next = remaining[0].name
		}
		if err := setInitialScene(dir, next); err != nil {
			console.Print("set initial scene: " + err.Error())
		}
	}
	t.refresh()
	if wasActive && len(t.entries) > 0 {
		vp.SetScene(t.entries[0].file)
	}
}

// restoreLastDeletedScene undoes the most recent scene deletion: it rewrites the
// deleted .scene file, restores initial_scene when the deleted scene was the start
// scene, refreshes the list, and reopens the scene in the viewport when it was the
// active one. Returns true when there was a deletion to undo.
func (t *SceneListComponent) restoreLastDeletedScene() bool {
	if lastDeletedScene == nil {
		return false
	}
	d := *lastDeletedScene

	if err := restoreDeletedSceneFile(d); err != nil {
		console.Print("restore scene: " + err.Error())
		return false
	}
	// Only consume the slot once the file is safely back on disk.
	lastDeletedScene = nil

	if d.wasInitial {
		if err := setInitialScene(d.projectDir, d.name); err != nil {
			console.Print("restore initial scene: " + err.Error())
		}
	}
	t.refresh()
	if d.wasActive {
		if vp := t.viewportComponent(); vp != nil {
			vp.SetScene(d.file)
		}
	}
	return true
}

func (t *SceneListComponent) Draw(r core.Renderer) {
	rect := t.Rect()
	if rect.Width() <= 0 || rect.Height() <= 0 {
		return
	}

	r.SetClipRect(rect)
	r.DrawRect(rect, t.Background)

	_, th := r.MeasureText("Ag", t.FontID, t.FontSize)

	// Title bar: "SCENES" on the left, "#" (settings) and "+" (new) on the right.
	r.DrawRect(math.NewRect(rect.X(), rect.Y(), rect.Width(), t.titleH()), t.Accent)
	titleY := rect.Y() + (t.titleH()-th)/2
	r.DrawText("SCENES", t.FontID, t.FontSize, math.NewVector2(rect.X()+6, titleY), t.TitleText)

	gear := t.gearRect(rect)
	if t.hoverGear {
		r.DrawRect(gear, t.Background.Lerp(math.White, 0.12))
	}
	gw, gh := r.MeasureText("#", t.FontID, t.FontSize)
	r.DrawText("#", t.FontID, t.FontSize, math.NewVector2(gear.X()+(gear.Width()-gw)/2, gear.Y()+(gear.Height()-gh)/2), t.TitleText)

	plus := t.plusRect(rect)
	if t.hoverPlus {
		r.DrawRect(plus, t.Background.Lerp(math.White, 0.12))
	}
	pw, ph := r.MeasureText("+", t.FontID, t.FontSize)
	r.DrawText("+", t.FontID, t.FontSize, math.NewVector2(plus.X()+(plus.Width()-pw)/2, plus.Y()+(plus.Height()-ph)/2), t.TitleText)

	vp := t.viewportComponent()
	active := ""
	if vp != nil {
		active = vp.CurrentSceneName()
	}

	// Scene rows, clipped to the body below the title bar so scrolled-out rows don't
	// bleed over the header.
	bodyTop := t.bodyTop()
	r.SetClipRect(math.NewRect(rect.X(), bodyTop, rect.Width(), t.bodyHeight(rect)))
	for i := range t.entries {
		entry := &t.entries[i]
		y := t.rowY(i) - t.scroll
		if y+t.RowHeight < bodyTop || y > rect.Y()+rect.Height() {
			continue
		}
		isActive := entry.file == active
		if t.hovered == entry && !isActive {
			// A faint highlight under the cursor, distinct from (and below) the
			// stronger active highlight, so hovered rows read as clickable.
			r.DrawRect(math.NewRect(rect.X(), y, rect.Width(), t.RowHeight), t.Background.Lerp(math.White, 0.07))
		}
		if isActive {
			r.DrawRect(math.NewRect(rect.X(), y, rect.Width(), t.RowHeight), t.Accent)
		}

		x := rect.X() + 6
		_, th2 := r.MeasureText(entry.name, t.FontID, t.FontSize)
		ty := y + (t.RowHeight-th2)/2
		if ty < y {
			ty = y
		}
		r.DrawText(entry.name, t.FontID, t.FontSize, math.NewVector2(x, ty), t.SceneText)

		// "*" set-as-start button: filled (bright) when this scene is the initial scene.
		sr := t.starRect(rect, y)
		starColor := t.TagText
		if entry.name == t.initial {
			starColor = t.TitleText
		}
		if t.hoverStar == entry {
			r.DrawRect(sr, t.Accent)
			starColor = t.TitleText
		}
		sw, sh := r.MeasureText("*", t.FontID, t.FontSize)
		r.DrawText("*", t.FontID, t.FontSize, math.NewVector2(sr.X()+(sr.Width()-sw)/2, y+(t.RowHeight-sh)/2), starColor)

		// "x" delete button in the right-edge strip (red on hover).
		xr := t.xRect(rect, y)
		xColor := t.TagText
		if t.hoverX == entry {
			r.DrawRect(xr, t.ErrorColor)
			xColor = t.TitleText
		}
		xw, xh := r.MeasureText("x", t.FontID, t.FontSize)
		r.DrawText("x", t.FontID, t.FontSize, math.NewVector2(xr.X()+(xr.Width()-xw)/2, y+(t.RowHeight-xh)/2), xColor)
	}

	// Scrollbar, drawn on top when the list overflows the body.
	r.SetClipRect(rect)
	track := t.scrollTrack(rect)
	if thumb, ok := scrollThumb(track, t.contentHeight(), t.scroll, t.maxScroll()); ok {
		drawScrollbar(r, track, thumb, t.ScrollTrack, t.ScrollThumb)
	}

	r.ClearClip()
}
