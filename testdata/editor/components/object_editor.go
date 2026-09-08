package components

import (
	stdmath "math"
	"path/filepath"

	"github.com/EnesBaytekin/imge/core"
	"github.com/EnesBaytekin/imge/core/math"
)

// ObjectEditorComponent is the floating, isolated .obj editor opened from the file
// browser's "Edit" button. It renders a single object under its own origin/axes grid
// with the object's draw components plus their debug overlays, and lets the user
// navigate (wheel zoom, middle-drag or Space+drag pan) — but never edits the object's
// transform, since a .obj has no transform (that is a per-instance scene concern).
//
// "Save" writes the object back to its .obj file (name/tags/components/depth/layer/
// ui/draggable), then reloads the active scene if it references that file so instances
// refresh. It is an editing *focus* (see objectEditorActive), not a modal: it pauses the
// viewport and scene tree but keeps the inspector and component-args windows live, so the
// object's components can be edited there while its world view stays open here. The world
// area also supports component picking and drag-to-move: selecting a component (sprite,
// collider, …) and dragging it shifts that component's offset, undoable and written
// through to the .obj. Dismissed by Close or a click on the viewport surface.
type ObjectEditorComponent struct {
	core.BaseUIComponent

	Background  math.Color `json:"background"`
	GridColor   math.Color `json:"grid_color"`
	AxesColor   math.Color `json:"axes_color"`
	TitleText   math.Color `json:"title_text"`
	Accent      math.Color `json:"accent"`       // title bar + Save button
	BorderColor math.Color `json:"border_color"` // outline + Close button

	FontID    string  `json:"font_id"`
	FontSize  float64 `json:"font_size"`
	RowHeight float64 `json:"row_height"`

	path   string       // absolute .obj path
	rel    string       // project-relative .obj path
	obj    *core.Object // the object being edited
	world  *core.Scene  // throwaway scene holding just this object, for DrawWorld
	cam    editorCamera // navigation camera over the object
	framed bool

	saveBtn  *ButtonComponent
	closeBtn *ButtonComponent

	status string

	dragging bool
	dragGrab math.Vector2

	panning   bool
	lastMouse math.Vector2

	selectedComp core.Component // component under edit (sprite/collider/...), or nil

	dragComp      core.Component // component being drag-moved
	dragStartOff  math.Vector2   // its offset when the drag began
	dragGrabWorld math.Vector2   // world point grabbed at drag start
	dragMoved     bool           // the drag moved past the threshold
	draggingComp  bool

	suppress bool // swallow the press that opened the editor (outside-click guard)

	dismiss  bool
	centered bool
}

func (c *ObjectEditorComponent) titleH() float64 { return c.RowHeight + 8 }

// worldRect is the viewport-like area between the title bar and the bottom button row.
func (c *ObjectEditorComponent) worldRect(rect math.Rect) math.Rect {
	return math.NewRect(rect.X(), rect.Y()+c.titleH(), rect.Width(), rect.Height()-c.titleH()-28)
}

func (c *ObjectEditorComponent) Initialize() {
	if c.Background == (math.Color{}) {
		c.Background = viewportBackground
	}
	if c.GridColor == (math.Color{}) {
		c.GridColor = defaultGridColor
	}
	if c.AxesColor == (math.Color{}) {
		c.AxesColor = defaultAxesColor
	}
	if c.TitleText == (math.Color{}) {
		c.TitleText = math.NewColor(0xff, 0xff, 0xff, 0xff)
	}
	if c.Accent == (math.Color{}) {
		c.Accent = math.NewColor(0x2f, 0x3b, 0x54, 0xff)
	}
	if c.BorderColor == (math.Color{}) {
		c.BorderColor = math.NewColor(0x3a, 0x42, 0x57, 0xff)
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

// spawnObjectEditor opens the object editor for the .obj at the project-relative path
// rel. It is a no-op when there is no project, the file can't be loaded, or a modal is
// already open.
func spawnObjectEditor(scene *core.Scene, rel string) {
	if scene == nil || rel == "" || modalOpen() || objectEditorActive() {
		return
	}
	vp := lookupViewport(scene)
	if vp == nil || vp.CurrentProject() == "" {
		return
	}
	abs := filepath.Join(vp.CurrentProject(), rel)

	obj, err := core.LoadObjectFromFile(abs)
	if err != nil {
		console.Print("object editor: " + err.Error())
		return
	}
	// A .obj is a world object: force world-space so DrawWorld draws it under the
	// navigation camera (a template with "ui": true would otherwise be skipped).
	obj.UI = false
	// Record the file provenance so the write-through path (persistObjectFile) updates
	// this .obj on every undoable edit, exactly as it does for file-ref scene objects.
	obj.File = rel

	world := core.NewScene("object_editor")
	world.BackgroundColor = viewportBackground
	if err := world.AddObject(obj); err != nil {
		console.Print("object editor: " + err.Error())
		return
	}
	world.InitializeForRender()

	win := core.NewObject("object_editor")
	win.UI = true
	win.Layer = 3
	win.Transform.Position = math.NewVector2(160, 80) // centered lazily on first Update

	editor := &ObjectEditorComponent{}
	editor.SetName("object_editor")
	editor.Width = 480
	editor.Height = 400
	editor.path = abs
	editor.rel = rel
	editor.obj = obj
	editor.world = world
	editor.cam = newEditorCamera()
	win.AddComponent(editor)

	if err := scene.AddObject(win); err != nil {
		return
	}

	editor.Initialize()
	editor.buildWidgets()
	// The object editor is an editing *focus*, not a blocking modal: it pauses the
	// viewport/scene tree (editorNavBlocked) but leaves the inspector and component-args
	// windows interactive so the object can be edited component-by-component alongside it.
	activeObjectEditor = editor
	editor.suppress = true // swallow the press that opened it (outside-click guard)
	raiseToFront(scene, win)
}

func (c *ObjectEditorComponent) buildWidgets() {
	owner := c.GetOwner()
	c.saveBtn = makePanelButton(owner, "save", "Save", math.NewVector2(8, c.Height-24), 120, 20, c.FontID, c.FontSize, c.Accent)
	c.closeBtn = makePanelButton(owner, "close", "Close", math.NewVector2(132, c.Height-24), 120, 20, c.FontID, c.FontSize, c.BorderColor)
}

// frame centers the camera on the object's origin (0,0) — the object's own position,
// which a .obj has no scene transform to move away from. The object and its components
// (whose offsets are relative to that origin) then sit centered in the viewport, so the
// object is visible immediately on open with no panning. Pan/zoom stay free afterward.
func (c *ObjectEditorComponent) frame(worldRect math.Rect) {
	c.cam.frame(math.Zero(), worldRect.Width(), worldRect.Height())
}

func (c *ObjectEditorComponent) doSave() {
	if err := c.obj.SaveToFile(c.path); err != nil {
		c.status = "save error"
		console.Print("object editor: " + err.Error())
		return
	}
	c.status = "saved"
	console.Print("saved " + c.path)
	// Refresh the active scene if it references this .obj so instances pick up the
	// new definition immediately. ReloadScene (not SetScene) reloads the *current*
	// scene — SetScene is a no-op when the name is unchanged.
	if vp := lookupViewport(c.GetScene()); vp != nil && sceneReferencesFile(vp.TargetScene(), c.rel) {
		vp.ReloadScene()
	}
}

// sceneReferencesFile reports whether any object in scene was instantiated from the
// .obj at rel.
func sceneReferencesFile(scene *core.Scene, rel string) bool {
	if scene == nil {
		return false
	}
	for _, obj := range scene.GetSortedObjects() {
		if obj.File == rel {
			return true
		}
	}
	return false
}

func (c *ObjectEditorComponent) Update(ctx *core.Context) {
	if ctx == nil || ctx.Input == nil {
		return
	}
	// A real modal (a component-args window, a confirm dialog) or an open menu bar is up:
	// the world area yields so it never fights an overlapping widget. The editor itself is
	// NOT a modal (see objectEditorActive), so the inspector stays interactive.
	if modalOpen() || menusOpen() {
		return
	}
	c.centerOnce(ctx)

	mouse := ctx.Input.GetMousePosition()
	rect := c.Rect()
	worldRect := c.worldRect(rect)
	local := mouse.Subtract(worldRect.Position)
	over := worldRect.ContainsPoint(mouse)

	// Cede the mouse to any window drawn above the editor (a component-args window) so a
	// click or wheel over it doesn't also pick/zoom the world beneath it.
	blocked := pointerOwnedElsewhere(c.GetScene(), c.GetOwner(), mouse)

	// Title-bar drag moves the window.
	if c.dragging {
		if ctx.Input.IsMouseButtonPressed(core.MouseButtonLeft) {
			c.GetOwner().SetPosition(mouse.X-c.dragGrab.X, mouse.Y-c.dragGrab.Y)
		} else {
			c.dragging = false
		}
	}

	middle := ctx.Input.IsMouseButtonPressed(core.MouseButtonMiddle)
	space := ctx.Input.IsKeyPressed(core.KeySpace)
	left := ctx.Input.IsMouseButtonPressed(core.MouseButtonLeft)

	// Wheel zoom around the cursor.
	if s := ctx.Input.GetMouseScroll(); over && !blocked && s.Y != 0 {
		c.cam.zoomAt(stdmath.Pow(zoomPerNotch, s.Y), local)
	}

	// Pan: middle-drag, or Space while holding the left button.
	requested := middle || (space && left)
	if !c.panning {
		if over && requested {
			c.panning = true
			c.lastMouse = local
		}
	} else if !requested {
		c.panning = false
	} else {
		c.cam.pan(local.Subtract(c.lastMouse))
		c.lastMouse = local
	}

	if c.saveBtn != nil && c.saveBtn.ConsumeClick() {
		c.doSave()
		return
	}
	if c.closeBtn != nil && c.closeBtn.ConsumeClick() {
		c.dismiss = true
		return
	}

	justPressed := ctx.Input.IsMouseButtonJustPressed(core.MouseButtonLeft)

	// Title-bar press starts a drag (tested before picking/outside-click so moving the
	// window never reads as a component pick or a dismissal).
	if justPressed {
		if math.NewRect(rect.X(), rect.Y(), rect.Width(), c.titleH()).ContainsPoint(mouse) {
			c.dragging = true
			c.dragGrab = mouse.Subtract(rect.Position)
			return
		}
	}

	// Swallow the press that opened the editor: that same just-pressed click must never
	// read as a component pick or a dismissal. Hold until that button is released.
	if c.suppress {
		if left {
			return
		}
		c.suppress = false
	}

	// Component pick + arm a drag: a plain left-press (no Space/middle) in the world
	// area selects the component under the cursor and, when it has an offset field,
	// captures the state to drag that offset.
	if over && !blocked && !space && !middle && justPressed {
		c.selectedComp = c.pickComponent(local)
		c.dragComp = nil
		c.draggingComp = false
		c.dragMoved = false
		if c.selectedComp != nil {
			if off, ok := componentOffset(c.selectedComp); ok {
				c.dragComp = c.selectedComp
				c.dragStartOff = off
				c.dragGrabWorld = c.cam.ScreenToWorld(local)
				c.draggingComp = true
			}
		}
	}

	// Live component drag: translate the selected component's offset by the cursor's
	// world-space delta, and record one undoable offset change on release.
	if c.draggingComp && c.dragComp != nil {
		if !left {
			if c.dragMoved {
				if off, ok := componentOffset(c.dragComp); ok {
					recordComponentOffsetChange(c.dragComp, c.dragStartOff, off)
				}
			}
			c.dragComp = nil
			c.draggingComp = false
			c.dragMoved = false
		} else {
			delta := c.cam.ScreenToWorld(local).Subtract(c.dragGrabWorld)
			if !c.dragMoved && delta.Length() >= dragThreshold {
				c.dragMoved = true
			}
			pos := c.dragStartOff.Add(delta)
			// Snap the component's offset to the grid by default (hold Shift to move
			// unsnapped), matching the viewport's object drag.
			if !ctx.Input.IsKeyPressed(core.KeyShift) {
				pos = math.NewVector2(
					stdmath.Round(pos.X/defaultGridStep)*defaultGridStep,
					stdmath.Round(pos.Y/defaultGridStep)*defaultGridStep,
				)
			}
			setComponentOffset(c.dragComp, pos)
		}
	}

	// Outside-click: a left-press on the viewport surface dismisses the editor, returning
	// to normal scene editing. Clicks on the inspector (which stays interactive) or the
	// scene tree do not dismiss — only the open world does.
	if justPressed && !c.draggingComp {
		if mgr := lookupUIManager(c.GetScene()); mgr != nil {
			if vp := lookupViewport(c.GetScene()); vp != nil {
				if mgr.TopmostObjectAt(mouse) == vp.GetOwner() {
					c.dismiss = true
				}
			}
		}
	}
}

// pickComponent returns the topmost component of the edited object under a
// viewport-local point (via the throwaway world's Pick), or nil when the point misses
// every component (or lands on an invisible one).
func (c *ObjectEditorComponent) pickComponent(local math.Vector2) core.Component {
	if c.world == nil {
		return nil
	}
	return c.world.Pick(c.cam.ScreenToWorld(local))
}

func (c *ObjectEditorComponent) centerOnce(ctx *core.Context) {
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

func (c *ObjectEditorComponent) Draw(r core.Renderer) {
	rect := c.Rect()
	if rect.Width() <= 0 || rect.Height() <= 0 {
		return
	}
	worldRect := c.worldRect(rect)
	if !c.framed {
		c.frame(worldRect)
		c.framed = true
	}

	r.SetClipRect(rect)
	r.DrawRect(rect, c.Background)
	r.DrawRectOutline(rect, c.BorderColor, 1)

	_, th := r.MeasureText("Ag", c.FontID, c.FontSize)

	// Title bar.
	r.DrawRect(math.NewRect(rect.X(), rect.Y(), rect.Width(), c.titleH()), c.Accent)
	title := "OBJECT: " + c.obj.Name
	if c.status != "" {
		title = title + " — " + c.status
	}
	r.DrawText(title, c.FontID, c.FontSize, math.NewVector2(rect.X()+6, rect.Y()+(c.titleH()-th)/2), c.TitleText)

	// World area: grid + axes in screen space, then the object under the camera.
	r.SetClipRect(worldRect)
	c.drawGrid(r, worldRect)
	c.drawAxes(r, worldRect)
	r.SetCamera(c.cam.x-worldRect.X()/c.cam.zoom, c.cam.y-worldRect.Y()/c.cam.zoom, c.cam.zoom)
	if c.world != nil {
		c.world.DrawWorld(r, true)
	}
	r.SetCamera(0, 0, 0)
	// Only the selected component is highlighted — the object itself has no selection
	// outline (a .obj's footprint is its components, and a bounds-less component may
	// legitimately draw nothing).
	c.drawComponentSelection(r, worldRect)

	r.SetClipRect(rect)
	r.ClearClip()

	if c.dismiss {
		c.closeSelf()
	}
}

// drawGrid draws a fixed world-space grid, snapped so the origin carries a line.
func (c *ObjectEditorComponent) drawGrid(r core.Renderer, worldRect math.Rect) {
	stepX, stepY := defaultGridStep, defaultGridStep
	left, top := c.cam.x, c.cam.y
	right := c.cam.x + worldRect.Width()/c.cam.zoom
	bottom := c.cam.y + worldRect.Height()/c.cam.zoom

	startX := stdmath.Floor(left/stepX) * stepX
	startY := stdmath.Floor(top/stepY) * stepY
	for i := 0; ; i++ {
		x := startX + float64(i)*stepX
		if x > right {
			break
		}
		p0 := c.cam.WorldToScreen(math.NewVector2(x, top)).Add(worldRect.Position)
		p1 := c.cam.WorldToScreen(math.NewVector2(x, bottom)).Add(worldRect.Position)
		r.DrawLine(p0, p1, c.GridColor, gridLineThickness)
	}
	for i := 0; ; i++ {
		y := startY + float64(i)*stepY
		if y > bottom {
			break
		}
		p0 := c.cam.WorldToScreen(math.NewVector2(left, y)).Add(worldRect.Position)
		p1 := c.cam.WorldToScreen(math.NewVector2(right, y)).Add(worldRect.Position)
		r.DrawLine(p0, p1, c.GridColor, gridLineThickness)
	}
}

// drawAxes draws the world origin axes (X at y=0, Y at x=0) slightly heavier.
func (c *ObjectEditorComponent) drawAxes(r core.Renderer, worldRect math.Rect) {
	left, top := c.cam.x, c.cam.y
	right := c.cam.x + worldRect.Width()/c.cam.zoom
	bottom := c.cam.y + worldRect.Height()/c.cam.zoom

	if top <= 0 && bottom >= 0 {
		p0 := c.cam.WorldToScreen(math.NewVector2(left, 0)).Add(worldRect.Position)
		p1 := c.cam.WorldToScreen(math.NewVector2(right, 0)).Add(worldRect.Position)
		r.DrawLine(p0, p1, c.AxesColor, axesLineThickness)
	}
	if left <= 0 && right >= 0 {
		p0 := c.cam.WorldToScreen(math.NewVector2(0, top)).Add(worldRect.Position)
		p1 := c.cam.WorldToScreen(math.NewVector2(0, bottom)).Add(worldRect.Position)
		r.DrawLine(p0, p1, c.AxesColor, axesLineThickness)
	}
}

// drawComponentSelection outlines the currently selected component's debug bounds in
// the accent color, so the component being edited (and drag-moved) reads as the target.
func (c *ObjectEditorComponent) drawComponentSelection(r core.Renderer, worldRect math.Rect) {
	if c.selectedComp == nil {
		return
	}
	bp, ok := c.selectedComp.(core.DebugBoundsProvider)
	if !ok {
		return
	}
	bounds := bp.DebugBounds()
	tl := c.cam.WorldToScreen(bounds.Position).Add(worldRect.Position)
	br := c.cam.WorldToScreen(bounds.Position.Add(bounds.Size)).Add(worldRect.Position)
	drawClippedOutline(r, worldRect, tl, br, c.Accent, selOutlineThickness)
}

// drawClippedOutline draws a rectangle outline from tl to br as four line segments,
// each clipped to clip — the chunky renderer rasterizes primitives at logical
// resolution, so an unclipped outline of a zoomed-in object would exceed the atlas.
func drawClippedOutline(r core.Renderer, clip math.Rect, tl, br math.Vector2, color math.Color, thickness float64) {
	x0, x1 := clip.Left(), clip.Right()
	y0, y1 := clip.Top(), clip.Bottom()

	if tl.Y >= y0 && tl.Y <= y1 {
		if lo, hi := stdmath.Max(tl.X, x0), stdmath.Min(br.X, x1); hi > lo {
			r.DrawLine(math.NewVector2(lo, tl.Y), math.NewVector2(hi, tl.Y), color, thickness)
		}
	}
	if br.Y >= y0 && br.Y <= y1 {
		if lo, hi := stdmath.Max(tl.X, x0), stdmath.Min(br.X, x1); hi > lo {
			r.DrawLine(math.NewVector2(lo, br.Y), math.NewVector2(hi, br.Y), color, thickness)
		}
	}
	if tl.X >= x0 && tl.X <= x1 {
		if lo, hi := stdmath.Max(tl.Y, y0), stdmath.Min(br.Y, y1); hi > lo {
			r.DrawLine(math.NewVector2(tl.X, lo), math.NewVector2(tl.X, hi), color, thickness)
		}
	}
	if br.X >= x0 && br.X <= x1 {
		if lo, hi := stdmath.Max(tl.Y, y0), stdmath.Min(br.Y, y1); hi > lo {
			r.DrawLine(math.NewVector2(br.X, lo), math.NewVector2(br.X, hi), color, thickness)
		}
	}
}

// closeSelf clears the object-editor focus and destroys the window's object.
func (c *ObjectEditorComponent) closeSelf() {
	if activeObjectEditor == c {
		activeObjectEditor = nil
	}
	if owner := c.GetOwner(); owner != nil {
		owner.Destroy()
	}
}
