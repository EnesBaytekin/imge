package components

import (
	"path/filepath"
	"strconv"

	"github.com/EnesBaytekin/imge/core"
	imgejson "github.com/EnesBaytekin/imge/core/json"
	"github.com/EnesBaytekin/imge/core/math"
)

// FileBrowserComponent is the floating "Browse Files" window (File → "Browse Files…").
// It shows the target project's directory tree on the left (collapsible directories)
// and a per-type preview on the right: images render their texture, .obj files show
// their name + component count with an "Edit" button, .scene files show their display
// name, and anything else shows "no preview". It is modal: while open the other
// hand-rolled panels yield (see modalOpen) and an outside click dismisses it.
//
// Input is read directly from ctx.Input (like the viewport and scene tree), and the
// tree is hand-drawn rather than built from widget children.
type FileBrowserComponent struct {
	core.BaseUIComponent

	Background  math.Color `json:"background"`
	TitleText   math.Color `json:"title_text"`
	DirText     math.Color `json:"dir_text"`  // directory rows
	FileText    math.Color `json:"file_text"` // file rows
	PreviewText math.Color `json:"preview_text"`
	Accent      math.Color `json:"accent"`       // title bar + selected row
	BorderColor math.Color `json:"border_color"` // outline + close "x"
	ScrollTrack math.Color `json:"scroll_track"`
	ScrollThumb math.Color `json:"scroll_thumb"`

	FontID    string  `json:"font_id"`
	FontSize  float64 `json:"font_size"`
	RowHeight float64 `json:"row_height"`

	projectDir string
	entries    []fileEntry
	expanded   map[string]bool
	cursor     string // rel of the keyboard-navigated row (dir or file), "" = none
	selected   string // rel path of the selected file, "" = none
	scroll     float64
	pick       bool              // pick mode: click a file to select it and dismiss
	pickFilter func(string) bool // selectable-file predicate (nil = no filtering in pick mode)
	onPick     func(string)      // invoked with the chosen rel path before dismissing

	dragging bool
	dragGrab math.Vector2

	hoverClose     bool
	hoverEdit      bool
	scrollDragging bool
	scrollGrab     float64
	dismiss        bool
	centered       bool
}

// treeW is the fixed width of the left (tree) pane; the preview fills the remainder.
const treeW = 240.0

func (c *FileBrowserComponent) titleH() float64 { return c.RowHeight + 8 }

func (c *FileBrowserComponent) treePane(rect math.Rect) math.Rect {
	return math.NewRect(rect.X(), rect.Y()+c.titleH(), treeW, rect.Height()-c.titleH())
}

func (c *FileBrowserComponent) previewPane(rect math.Rect) math.Rect {
	return math.NewRect(rect.X()+treeW, rect.Y()+c.titleH(), rect.Width()-treeW, rect.Height()-c.titleH())
}

func (c *FileBrowserComponent) closeRect(rect math.Rect) math.Rect {
	const s = 12.0
	return math.NewRect(rect.X()+rect.Width()-s-4, rect.Y()+(c.titleH()-s)/2, s, s)
}

func (c *FileBrowserComponent) editRect(rect math.Rect) math.Rect {
	pr := c.previewPane(rect)
	return math.NewRect(pr.X()+8, pr.Y()+pr.Height()-28, 64, 20)
}

func (c *FileBrowserComponent) Initialize() {
	if c.Background == (math.Color{}) {
		c.Background = math.NewColor(0x1d, 0x21, 0x30, 0xff)
	}
	if c.TitleText == (math.Color{}) {
		c.TitleText = math.NewColor(0xff, 0xff, 0xff, 0xff)
	}
	if c.DirText == (math.Color{}) {
		c.DirText = math.NewColor(0x9f, 0xc7, 0xff, 0xff)
	}
	if c.FileText == (math.Color{}) {
		c.FileText = math.NewColor(0xe6, 0xe6, 0xef, 0xff)
	}
	if c.PreviewText == (math.Color{}) {
		c.PreviewText = math.NewColor(0x6b, 0x73, 0x85, 0xff)
	}
	if c.Accent == (math.Color{}) {
		c.Accent = math.NewColor(0x2f, 0x3b, 0x54, 0xff)
	}
	if c.BorderColor == (math.Color{}) {
		c.BorderColor = math.NewColor(0x3a, 0x42, 0x57, 0xff)
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
	if c.Blocking == nil {
		c.SetBlocking(true)
	}
}

// spawnFileBrowser opens the file-browser window (File → "Browse Files…") for the current
// target project. It is a no-op when there is no project or a modal is already open.
func spawnFileBrowser(scene *core.Scene) {
	spawnFileBrowserMode(scene, false, nil, nil)
}

// spawnObjPicker opens the file browser in pick mode: filtered to .obj files, clicking
// one loads it into the target scene as a file reference. Used by the scene tree's "+".
func spawnObjPicker(scene *core.Scene) {
	spawnFilePicker(scene, func(rel string) bool { return fileTypeOf(rel) == "obj" }, func(rel string) {
		if vp := lookupViewport(scene); vp != nil {
			if obj := vp.AddObjectFromFile(rel); obj != nil {
				vp.SelectSilent(obj)
			}
		}
	})
}

// spawnFilePicker opens the file browser in pick mode filtered to files that pass
// `filter`, invoking onPick with the chosen project-relative path. It is the generic
// "select a file from the project tree" used by the sprite texture field and the
// object picker.
func spawnFilePicker(scene *core.Scene, filter func(string) bool, onPick func(string)) {
	spawnFileBrowserMode(scene, true, filter, onPick)
}

func spawnFileBrowserMode(scene *core.Scene, pick bool, filter func(string) bool, onPick func(string)) {
	if scene == nil || modalOpen() {
		return
	}
	vp := lookupViewport(scene)
	if vp == nil || vp.CurrentProject() == "" {
		return
	}
	dir := vp.CurrentProject()

	name := "file_browser"
	if pick {
		name = "file_picker"
	}
	obj := core.NewObject(name)
	obj.UI = true
	obj.Layer = 3
	obj.Transform.Position = math.NewVector2(160, 80) // centered lazily on first Update

	browser := &FileBrowserComponent{}
	browser.SetName(name)
	browser.Width = 560
	browser.Height = 360
	browser.projectDir = dir
	browser.pick = pick
	browser.pickFilter = filter
	browser.onPick = onPick
	browser.entries = listProjectFiles(dir)
	browser.expanded = make(map[string]bool)
	for _, e := range browser.entries {
		if e.isDir {
			browser.expanded[e.rel] = true
		}
	}
	obj.AddComponent(browser)

	if err := scene.AddObject(obj); err != nil {
		return
	}

	browser.Initialize()
	setModal(browser)
	raiseToFront(scene, obj)
}

// visibleEntries returns the tree rows that should be shown, hiding a node whose
// ancestor directory is collapsed.
func (c *FileBrowserComponent) visibleEntries() []fileEntry {
	open := map[string]bool{"": true}
	var out []fileEntry
	for _, e := range c.entries {
		if e.isDir {
			if open[e.parent] {
				out = append(out, e)
			}
			open[e.rel] = open[e.parent] && c.expanded[e.rel]
		} else if open[e.parent] {
			if c.pick && c.pickFilter != nil && !c.pickFilter(e.rel) {
				continue // pick mode lists only files the filter accepts
			}
			out = append(out, e)
		}
	}
	return out
}

func (c *FileBrowserComponent) contentHeight(visible []fileEntry) float64 {
	if len(visible) == 0 {
		return 0
	}
	return float64(len(visible)) * c.RowHeight
}

func (c *FileBrowserComponent) maxScroll(visible []fileEntry, rect math.Rect) float64 {
	bodyH := c.treePane(rect).Height() - 2
	if m := c.contentHeight(visible) - bodyH; m > 0 {
		return m
	}
	return 0
}

func (c *FileBrowserComponent) clampScroll(visible []fileEntry, rect math.Rect) {
	if max := c.maxScroll(visible, rect); c.scroll > max {
		c.scroll = max
	}
	if c.scroll < 0 {
		c.scroll = 0
	}
}

// rowAt returns the index of the visible entry under mouseY (screen space), or -1.
func (c *FileBrowserComponent) rowAt(visible []fileEntry, rect math.Rect, mouseY float64) int {
	pane := c.treePane(rect)
	top := pane.Y() + 2
	for i := range visible {
		y := top + float64(i)*c.RowHeight - c.scroll
		if mouseY >= y && mouseY < y+c.RowHeight {
			return i
		}
	}
	return -1
}

// moveCursor shifts the tree cursor by delta rows (through the visible entries), keeping
// it in view and, when it lands on a file, selecting that file so the preview follows.
func (c *FileBrowserComponent) moveCursor(visible []fileEntry, rect math.Rect, delta int) {
	if len(visible) == 0 {
		return
	}
	idx := -1
	for i, e := range visible {
		if e.rel == c.cursor {
			idx = i
			break
		}
	}
	if idx < 0 {
		// No cursor yet: start at the first row (or the last when moving up).
		if delta < 0 {
			idx = 0
		} else {
			idx = -1
		}
	}
	idx += delta
	if idx < 0 {
		idx = 0
	}
	if idx >= len(visible) {
		idx = len(visible) - 1
	}
	c.setCursor(visible[idx], visible, rect)
}

// setCursor moves the cursor to a row and, for files, updates the selection. It also
// scrolls the row into view.
func (c *FileBrowserComponent) setCursor(e fileEntry, visible []fileEntry, rect math.Rect) {
	c.cursor = e.rel
	if !e.isDir {
		c.selected = e.rel
	}
	c.scrollCursorIntoView(visible, rect)
}

// scrollCursorIntoView scrolls the tree so the cursor row is visible within the pane.
func (c *FileBrowserComponent) scrollCursorIntoView(visible []fileEntry, rect math.Rect) {
	idx := -1
	for i, e := range visible {
		if e.rel == c.cursor {
			idx = i
			break
		}
	}
	if idx < 0 {
		return
	}
	pane := c.treePane(rect)
	top := pane.Y() + 2
	y := top + float64(idx)*c.RowHeight - c.scroll
	if y < pane.Y() {
		c.scroll = top + float64(idx)*c.RowHeight - pane.Y()
	} else if y+c.RowHeight > pane.Y()+pane.Height() {
		c.scroll = top + float64(idx)*c.RowHeight + c.RowHeight - (pane.Y() + pane.Height())
	}
	c.clampScroll(visible, rect)
}

// activateCursor "opens" the cursor row: a directory toggles its expand/collapse, a file
// is selected (or, in pick mode, loaded into the scene and the window dismissed).
func (c *FileBrowserComponent) activateCursor(visible []fileEntry) {
	idx := -1
	for i, e := range visible {
		if e.rel == c.cursor {
			idx = i
			break
		}
	}
	if idx < 0 {
		return
	}
	e := visible[idx]
	if e.isDir {
		c.expanded[e.rel] = !c.expanded[e.rel]
		return
	}
	if c.pick {
		c.pickFile(e.rel)
		return
	}
	c.selected = e.rel
}

// pickFile completes a pick-mode selection: it hands the chosen project-relative path to
// the pick callback (if any) and dismisses the window. The keyboard path (activateCursor)
// and the mouse path (tree-row click) both land here.
func (c *FileBrowserComponent) pickFile(rel string) {
	if c.onPick != nil {
		c.onPick(rel)
	}
	c.dismiss = true
}

// cursorIsDir reports whether the cursor row is a directory (it has no preview).
func (c *FileBrowserComponent) cursorIsDir() bool {
	for _, e := range c.entries {
		if e.rel == c.cursor {
			return e.isDir
		}
	}
	return false
}

func (c *FileBrowserComponent) Update(ctx *core.Context) {
	if ctx == nil || ctx.Input == nil {
		return
	}
	c.centerOnce(ctx)

	mouse := ctx.Input.GetMousePosition()
	rect := c.Rect()

	// Drag-to-move via the title bar (tested before everything else so moving the
	// window never reads as a row click or an outside dismiss).
	if c.dragging {
		if ctx.Input.IsMouseButtonPressed(core.MouseButtonLeft) {
			c.GetOwner().SetPosition(mouse.X-c.dragGrab.X, mouse.Y-c.dragGrab.Y)
		} else {
			c.dragging = false
		}
	}

	visible := c.visibleEntries()

	// Scrollbar drag keeps following the cursor even outside the pane.
	if c.scrollDragging {
		if ctx.Input.IsMouseButtonPressed(core.MouseButtonLeft) {
			track := c.scrollTrack(rect)
			c.scroll = scrollFromThumb(track, c.contentHeight(visible), c.maxScroll(visible, rect), mouse.Y, c.scrollGrab)
			c.clampScroll(visible, rect)
		} else {
			c.scrollDragging = false
		}
	}

	// Hover states.
	c.hoverClose = c.closeRect(rect).ContainsPoint(mouse)
	c.hoverEdit = !c.pick && c.selected != "" && fileTypeOf(c.selected) == "obj" && c.editRect(rect).ContainsPoint(mouse)

	// Wheel scrolls the tree.
	if s := ctx.Input.GetMouseScroll(); s.Y != 0 && c.treePane(rect).ContainsPoint(mouse) {
		c.scroll -= s.Y * c.RowHeight * 2
		c.clampScroll(visible, rect)
	}

	// Keyboard: arrows move the cursor, Enter activates the row. Processed every frame
	// (before the mouse-press early-out) so the tree navigates without the mouse.
	if ctx.Input.IsKeyJustPressed(core.KeyUp) {
		c.moveCursor(visible, rect, -1)
	}
	if ctx.Input.IsKeyJustPressed(core.KeyDown) {
		c.moveCursor(visible, rect, 1)
	}
	if ctx.Input.IsKeyJustPressed(core.KeyEnter) {
		c.activateCursor(visible)
	}

	if !ctx.Input.IsMouseButtonJustPressed(core.MouseButtonLeft) {
		return
	}

	// Close button.
	if c.hoverClose {
		c.dismiss = true
		return
	}

	// Edit button → open the object editor for the selected .obj.
	if c.hoverEdit {
		scene := c.GetScene()
		rel := c.selected
		closeActiveModal() // dismiss this window
		spawnObjectEditor(scene, rel)
		return
	}

	// Scrollbar press.
	if c.handleScrollbarPress(mouse, visible, rect) {
		return
	}

	// Tree row click: toggle a directory, select a file (or, in pick mode, load it).
	if ri := c.rowAt(visible, rect, mouse.Y); ri >= 0 {
		e := visible[ri]
		c.cursor = e.rel
		if e.isDir {
			c.expanded[e.rel] = !c.expanded[e.rel]
		} else if c.pick {
			c.pickFile(e.rel)
		} else {
			c.selected = e.rel
		}
		return
	}

	// Title-bar press starts a drag.
	if math.NewRect(rect.X(), rect.Y(), rect.Width(), c.titleH()).ContainsPoint(mouse) {
		c.dragging = true
		c.dragGrab = mouse.Subtract(rect.Position)
		return
	}

	if modalOutsideClick(c.GetScene(), c.GetOwner(), ctx) {
		c.dismiss = true
	}
}

func (c *FileBrowserComponent) scrollTrack(rect math.Rect) math.Rect {
	pane := c.treePane(rect)
	const sbW = 6.0
	return math.NewRect(pane.X()+pane.Width()-sbW-2, pane.Y(), sbW, pane.Height())
}

func (c *FileBrowserComponent) handleScrollbarPress(mouse math.Vector2, visible []fileEntry, rect math.Rect) bool {
	track := c.scrollTrack(rect)
	contentH := c.contentHeight(visible)
	max := c.maxScroll(visible, rect)
	thumb, ok := scrollThumb(track, contentH, c.scroll, max)
	if !ok {
		return false
	}
	if thumb.ContainsPoint(mouse) {
		c.scrollDragging = true
		c.scrollGrab = mouse.Y - thumb.Y()
		return true
	}
	if track.ContainsPoint(mouse) {
		c.scroll = scrollFromThumb(track, contentH, max, mouse.Y, thumb.Height()/2)
		c.clampScroll(visible, rect)
		return true
	}
	return false
}

func (c *FileBrowserComponent) centerOnce(ctx *core.Context) {
	if c.centered || c.GetOwner() == nil {
		return
	}
	c.centered = true
	if ctx.Renderer == nil {
		return
	}
	vw, vh := ctx.Renderer.GetViewportSize()
	if vw <= 0 || vh <= 0 {
		return
	}
	c.GetOwner().SetPosition((float64(vw)-c.Width)/2, (float64(vh)-c.Height)/2)
}

func (c *FileBrowserComponent) Draw(r core.Renderer) {
	rect := c.Rect()
	if rect.Width() <= 0 || rect.Height() <= 0 {
		return
	}
	r.SetClipRect(rect)
	r.DrawRect(rect, c.Background)
	r.DrawRectOutline(rect, c.BorderColor, 1)

	_, th := r.MeasureText("Ag", c.FontID, c.FontSize)

	// Title bar + close "x".
	r.DrawRect(math.NewRect(rect.X(), rect.Y(), rect.Width(), c.titleH()), c.Accent)
	title := "FILES"
	if c.pick {
		title = "SELECT FILE"
	}
	r.DrawText(title, c.FontID, c.FontSize, math.NewVector2(rect.X()+6, rect.Y()+(c.titleH()-th)/2), c.TitleText)
	cr := c.closeRect(rect)
	if c.hoverClose {
		r.DrawRect(cr, c.Background.Lerp(math.White, 0.12))
	}
	xw, xh := r.MeasureText("x", c.FontID, c.FontSize)
	r.DrawText("x", c.FontID, c.FontSize, math.NewVector2(cr.X()+(cr.Width()-xw)/2, cr.Y()+(cr.Height()-xh)/2), c.TitleText)

	// Tree pane.
	treePane := c.treePane(rect)
	r.DrawRect(treePane, c.Background.Lerp(math.White, 0.04))
	visible := c.visibleEntries()
	top := treePane.Y() + 2
	r.SetClipRect(treePane)
	for i, e := range visible {
		y := top + float64(i)*c.RowHeight - c.scroll
		if y+c.RowHeight < treePane.Y() || y > treePane.Y()+treePane.Height() {
			continue
		}
		if e.rel == c.cursor {
			r.DrawRect(math.NewRect(treePane.X(), y, treePane.Width(), c.RowHeight), c.Accent)
		}
		indent := treePane.X() + 4 + float64(e.depth)*12
		ty := y + (c.RowHeight-th)/2
		if ty < y {
			ty = y
		}
		label := e.name
		color := c.FileText
		if e.isDir {
			label = "▸ " + e.name
			if c.expanded[e.rel] {
				label = "▾ " + e.name
			}
			color = c.DirText
		}
		r.DrawText(label, c.FontID, c.FontSize, math.NewVector2(indent, ty), color)
	}
	r.SetClipRect(rect)
	if thumb, ok := scrollThumb(c.scrollTrack(rect), c.contentHeight(visible), c.scroll, c.maxScroll(visible, rect)); ok {
		drawScrollbar(r, c.scrollTrack(rect), thumb, c.ScrollTrack, c.ScrollThumb)
	}

	// Preview pane.
	c.drawPreview(r, rect)

	r.ClearClip()

	if c.dismiss {
		c.closeSelf()
	}
}

func (c *FileBrowserComponent) drawPreview(r core.Renderer, rect math.Rect) {
	pr := c.previewPane(rect)
	// A directory row has no preview: show that instead of lingering on the file that was
	// selected before the cursor moved onto the directory.
	if c.cursor != "" && c.cursorIsDir() {
		r.DrawText("directory — no preview", c.FontID, c.FontSize, math.NewVector2(pr.X()+8, pr.Y()+8), c.PreviewText)
		return
	}
	if c.selected == "" {
		hint := "select a file"
		if c.pick {
			hint = "click a file to select it"
		}
		r.DrawText(hint, c.FontID, c.FontSize, math.NewVector2(pr.X()+8, pr.Y()+8), c.PreviewText)
		return
	}
	abs := filepath.Join(c.projectDir, c.selected)
	switch fileTypeOf(c.selected) {
	case "image":
		sw, sh := r.GetTextureSize(c.selected)
		if sw <= 0 || sh <= 0 {
			r.DrawText("(cannot load image)", c.FontID, c.FontSize, math.NewVector2(pr.X()+8, pr.Y()+8), c.PreviewText)
			return
		}
		r.DrawText("IMAGE", c.FontID, c.FontSize, math.NewVector2(pr.X()+8, pr.Y()+8), c.PreviewText)
		r.DrawText(strconv.Itoa(int(sw))+" x "+strconv.Itoa(int(sh)), c.FontID, c.FontSize, math.NewVector2(pr.X()+8, pr.Y()+24), c.FileText)
		// Center the texture in the area below the two header rows.
		pad := 8.0
		headerH := 2*c.RowHeight + 2*pad
		areaTop := pr.Y() + headerH
		maxW := pr.Width() - 2*pad
		maxH := pr.Height() - headerH - pad
		if maxW <= 0 || maxH <= 0 {
			return
		}
		scale := maxW / sw
		if shScale := maxH / sh; shScale < scale {
			scale = shScale
		}
		if scale > 4 {
			scale = 4 // don't blow a tiny sprite up past 4x
		}
		w, h := sw*scale, sh*scale
		x := pr.X() + (pr.Width()-w)/2
		y := areaTop + (maxH-h)/2
		r.DrawTexture(c.selected, math.Rect{}, math.NewVector2(x, y), math.NewVector2(scale, scale), 0, math.ColorTransform{})
		// Thin outline so the image's bounds read clearly against the dark preview.
		r.DrawRectOutline(math.NewRect(x, y, w, h), c.PreviewText, 1)
	case "obj":
		cfg, err := imgejson.LoadObjectConfig(abs)
		if err != nil {
			r.DrawText("(cannot load .obj)", c.FontID, c.FontSize, math.NewVector2(pr.X()+8, pr.Y()+8), c.PreviewText)
			return
		}
		r.DrawText("OBJECT", c.FontID, c.FontSize, math.NewVector2(pr.X()+8, pr.Y()+8), c.PreviewText)
		r.DrawText(cfg.Name, c.FontID, c.FontSize, math.NewVector2(pr.X()+8, pr.Y()+24), c.FileText)
		n := len(cfg.Components)
		detail := "no components"
		if n == 1 {
			detail = "1 component"
		} else if n > 1 {
			detail = strconv.Itoa(n) + " components"
		}
		r.DrawText(detail, c.FontID, c.FontSize, math.NewVector2(pr.X()+8, pr.Y()+40), c.PreviewText)

		// Edit button.
		er := c.editRect(rect)
		if c.hoverEdit {
			r.DrawRect(er, c.Accent.Lerp(math.White, 0.1))
		} else {
			r.DrawRect(er, c.Accent)
		}
		bw, bh := r.MeasureText("Edit", c.FontID, c.FontSize)
		r.DrawText("Edit", c.FontID, c.FontSize, math.NewVector2(er.X()+(er.Width()-bw)/2, er.Y()+(er.Height()-bh)/2), c.TitleText)
	case "scene":
		cfg, err := imgejson.LoadSceneConfig(abs)
		if err != nil {
			r.DrawText("(cannot load .scene)", c.FontID, c.FontSize, math.NewVector2(pr.X()+8, pr.Y()+8), c.PreviewText)
			return
		}
		r.DrawText("SCENE", c.FontID, c.FontSize, math.NewVector2(pr.X()+8, pr.Y()+8), c.PreviewText)
		r.DrawText(cfg.Name, c.FontID, c.FontSize, math.NewVector2(pr.X()+8, pr.Y()+24), c.FileText)
	default:
		r.DrawText("no preview", c.FontID, c.FontSize, math.NewVector2(pr.X()+8, pr.Y()+8), c.PreviewText)
	}
}

// closeSelf clears the modal state and destroys the window's object.
func (c *FileBrowserComponent) closeSelf() {
	clearModal()
	if owner := c.GetOwner(); owner != nil {
		owner.Destroy()
	}
}
