package components

import (
	"fmt"
	"io/fs"
	"log"
	stdmath "math"
	"os"
	"path/filepath"
	"strings"

	"github.com/EnesBaytekin/imge/core"
	imgejson "github.com/EnesBaytekin/imge/core/json"
	"github.com/EnesBaytekin/imge/core/math"
)

// ViewportComponent is the editor's central view: it renders a target scene's
// world into its rectangle under a free pan/zoom camera, with an origin axes +
// fixed world grid drawn underneath. The navigation camera is internal to this
// component and independent of the target scene's own Camera — nothing here writes
// to the target scene or its data, so the game camera (applied when the game runs)
// stays untouched.
//
// Interaction (read directly from ctx.Input; the viewport is a background surface,
// not a @UIManager widget):
//   - middle-drag, or Space + left-drag, pans the view,
//   - the mouse wheel zooms around the cursor (clamped 0.1x–16x).
//
// The target is a *separate* project: Project is its directory, Scene the scene
// file to open ("" = first .scene found). If Project is empty, the IMGE_PROJECT
// environment variable is used; if that is unset too, the viewport draws the grid
// and axes only. Components in the target whose kind isn't registered in this
// build are skipped, so a scene with custom components still opens and shows every
// built-in it understands.
//
// Export variables (JSON args): project, scene, draw_debug, grid_color, axes_color.
type ViewportComponent struct {
	core.BaseUIComponent

	// Project is the target project directory to display ("" = use IMGE_PROJECT,
	// then grid-only if that is unset too).
	Project string `json:"project"`

	// Scene is the scene file to open within Project: a basename ("main") resolved
	// against <project>/scenes/ or <project>/; "" opens the first .scene found.
	Scene string `json:"scene"`

	// DrawDebug toggles the target scene's debug overlay pass (bounds, colliders).
	DrawDebug bool `json:"draw_debug"`

	// GridColor and AxesColor theme the grid and origin axes. Zero means "use the
	// default".
	GridColor math.Color `json:"grid_color"`
	AxesColor math.Color `json:"axes_color"`

	// GridStepX/GridStepY are the fixed spacing (in world units) between grid lines,
	// horizontally and vertically. The grid is a static world lattice — zooming in
	// magnifies it, it does not add finer lines. Zero or negative means "use the
	// default". They are editor-only settings, edited from Edit → "Editor Settings...".
	GridStepX float64 `json:"grid_step_x"`
	GridStepY float64 `json:"grid_step_y"`

	cam    editorCamera
	scene  *core.Scene
	framed bool

	// sceneFile is the resolved path of the loaded target scene ("" when none loaded).
	// Save writes the serialized scene back to it.
	sceneFile string

	// projectDir is the resolved target project directory used for the current load
	// (the project json arg, or the IMGE_PROJECT env fallback). It is what the toolbar
	// shows and what SetProject overrides.
	projectDir string

	// logicalW/logicalH is the target game's logical screen size (game.imge window
	// width/height), read on load so drawViewBounds can outline where the game window
	// will land in world space. Zero when unknown (no project, or game.imge missing).
	logicalW float64
	logicalH float64

	panning   bool
	lastMouse math.Vector2

	// Drag-to-move state: a plain left-press on an object arms a move. The object only
	// starts translating once the cursor passes dragThreshold (so a click still just
	// selects), and the whole gesture records a single undo entry on release.
	dragging        bool
	dragObj         *core.Object
	dragStart       math.Vector2 // object world position at press
	dragGrab        math.Vector2 // world point under the cursor at press
	dragPressScreen math.Vector2 // viewport-local press point (for the threshold)
	dragActive      bool         // threshold passed; translating
	dragMoved       bool         // object actually moved; worth an undo entry

	// selected is the target object the user last picked in the viewport (nil when
	// empty space was clicked). The target scene's debug selection is synced to one of
	// the object's components so its DrawDebug pass still highlights the pick.
	selected *core.Object
}

const (
	viewportMinZoom = 0.1
	viewportMaxZoom = 16.0
	zoomPerNotch    = 1.15

	// defaultGridStep is the fixed spacing (in world units) between grid lines. The
	// grid is a static world lattice: it does not get denser or sparser with zoom.
	defaultGridStep = 32.0

	// dragThreshold is the screen-space distance (viewport pixels) the cursor must move
	// before a left-press becomes a drag-to-move, so a plain click still just selects.
	dragThreshold = 3.0

	// Origin markers for objects with no debug bounds (no sprite/collider/other
	// drawable shape): a small "+" at their origin, with a matching pick radius so they
	// stay clickable in screen space.
	emptyMarkerHalf      = 5.0
	emptyMarkerThickness = 1.5
	emptyPickRadius      = 9.0

	// Line thicknesses in screen pixels. The grid, axes, and selection overlays are
	// drawn in screen space (camera off), so a value here is the exact on-screen
	// width at every zoom level.
	gridLineThickness   = 1.0
	axesLineThickness   = 2.0
	selOutlineThickness = 2.0
	viewRectThickness   = 2.0
)

var (
	viewportBackground = math.NewColor(0x15, 0x17, 0x1e, 0xff)
	defaultGridColor   = math.NewColor(0x33, 0x3a, 0x4e, 0xff)
	defaultAxesColor   = math.NewColor(0x6b, 0x73, 0x85, 0xff)

	selectionColor = math.NewColor(0xff, 0xff, 0xff, 0xff) // white outline

	emptyMarkerColor = math.NewColor(0x9f, 0xa8, 0xbf, 0xff) // origin "+" for bounds-less objects

	viewRectColor = math.NewColor(0x4f, 0xd1, 0xc5, 0xff) // teal outline of the game's logical screen area
)

// editorCamera is the viewport's navigation camera. (x, y) is the world point at
// the viewport's top-left corner — the same convention as core.Camera — and zoom is
// the scale. It maps between world space and viewport-local space; the caller
// handles the viewport's screen offset when setting the renderer camera.
type editorCamera struct {
	x, y float64
	zoom float64
}

func newEditorCamera() editorCamera { return editorCamera{zoom: 1} }

// ScreenToWorld maps a viewport-local point (origin at the viewport top-left) to
// world coordinates.
func (c *editorCamera) ScreenToWorld(p math.Vector2) math.Vector2 {
	return math.NewVector2(c.x+p.X/c.zoom, c.y+p.Y/c.zoom)
}

// WorldToScreen maps a world point to viewport-local space.
func (c *editorCamera) WorldToScreen(w math.Vector2) math.Vector2 {
	return math.NewVector2((w.X-c.x)*c.zoom, (w.Y-c.y)*c.zoom)
}

// pan shifts the view by a screen-space delta (viewport-local pixels): dragging the
// cursor by d keeps the grabbed world point under the cursor.
func (c *editorCamera) pan(d math.Vector2) {
	c.x -= d.X / c.zoom
	c.y -= d.Y / c.zoom
}

// zoomAt scales zoom by factor, keeping the world point under the cursor fixed.
func (c *editorCamera) zoomAt(factor float64, cursor math.Vector2) {
	anchor := c.ScreenToWorld(cursor)
	z := c.zoom * factor
	if z < viewportMinZoom {
		z = viewportMinZoom
	}
	if z > viewportMaxZoom {
		z = viewportMaxZoom
	}
	c.zoom = z
	c.x = anchor.X - cursor.X/z
	c.y = anchor.Y - cursor.Y/z
}

// frame centers the camera on a world point at the current zoom.
func (c *editorCamera) frame(world math.Vector2, vw, vh float64) {
	c.x = world.X - vw/(2*c.zoom)
	c.y = world.Y - vh/(2*c.zoom)
}

// Initialize loads the target scene and sets theming defaults.
func (c *ViewportComponent) Initialize() {
	c.cam = newEditorCamera()
	if c.GridColor == (math.Color{}) {
		c.GridColor = defaultGridColor
	}
	if c.AxesColor == (math.Color{}) {
		c.AxesColor = defaultAxesColor
	}
	if c.GridStepX <= 0 {
		c.GridStepX = defaultGridStep
	}
	if c.GridStepY <= 0 {
		c.GridStepY = defaultGridStep
	}
	// The viewport is an opaque surface: it blocks pointer events so the @UIManager
	// occludes whatever is drawn behind it (see pointerOwnedElsewhere).
	if c.Blocking == nil {
		c.SetBlocking(true)
	}
	c.loadTarget()
}

// Update handles pan and zoom directly from ctx.Input. It is render-only with
// respect to the target: the target scene is never updated.
func (c *ViewportComponent) Update(ctx *core.Context) {
	if ctx == nil || ctx.Input == nil {
		return
	}
	// A modal or an open menu bar is up: this panel is inert.
	if modalOpen() || menusOpen() {
		return
	}
	rect := c.Rect()
	mouse := ctx.Input.GetMousePosition()
	local := mouse.Subtract(rect.Position)
	over := rect.ContainsPoint(mouse)

	middle := ctx.Input.IsMouseButtonPressed(core.MouseButtonMiddle)
	space := ctx.Input.IsKeyPressed(core.KeySpace)
	left := ctx.Input.IsMouseButtonPressed(core.MouseButtonLeft)

	// Cede the mouse to any window drawn above the viewport (a floating args window,
	// an open @ColorPicker popup, or the inspector/tree when they overlap) so a click
	// or wheel over it doesn't also zoom/pick the viewport beneath. The @UIManager's
	// blocking occlusion decides this generically.
	blocked := pointerOwnedElsewhere(c.GetScene(), c.GetOwner(), mouse)

	// Wheel zoom around the cursor (scroll up zooms in).
	if s := ctx.Input.GetMouseScroll(); over && !blocked && s.Y != 0 {
		c.cam.zoomAt(stdmath.Pow(zoomPerNotch, s.Y), local)
	}

	// A plain left-press (no Space, no middle-drag, not over an open window) picks the
	// topmost world object and arms a possible drag-to-move. Clicking empty space clears
	// the selection (and starts no drag).
	if over && !blocked && !space && !middle && ctx.Input.IsMouseButtonJustPressed(core.MouseButtonLeft) {
		c.beginDrag(local)
	}

	// Drag-to-move: while the grabbed object's left button stays down (and Space/middle
	// haven't taken over), translate it by the cursor's world-space delta.
	if c.dragging {
		if !left || middle || space {
			c.finishDrag()
		} else {
			c.updateDrag(ctx, local)
		}
	}

	// Pan: hold the middle button, or Space while holding the left button. Once
	// started the drag keeps following the cursor even outside the viewport.
	requested := middle || (space && left)

	if !c.panning {
		if over && requested {
			c.panning = true
			c.lastMouse = local
		}
		return
	}
	if !requested {
		c.panning = false
		return
	}
	c.cam.pan(local.Subtract(c.lastMouse))
	c.lastMouse = local
}

// selectAt picks the topmost world object under a viewport-local point and highlights
// it through the target scene's debug selection. A miss clears the selection.
func (c *ViewportComponent) selectAt(local math.Vector2) {
	if c.scene == nil {
		return
	}
	comp := c.scene.Pick(c.cam.ScreenToWorld(local))
	var obj *core.Object
	if comp != nil {
		obj = comp.GetOwner()
	} else {
		// scene.Pick only sees DebugBoundsProvider bounds; a bounds-less object (no
		// sprite/collider) has none, so fall back to its small origin hitbox.
		obj = c.pickEmptyObject(local)
	}
	c.Select(obj)
	if obj != nil {
		log.Printf("viewport: selected object %q", obj.Name)
	}
}

// pickEmptyObject returns the topmost bounds-less world object whose origin is within
// emptyPickRadius of a viewport-local point, or nil. Empty objects render as a small
// "+" at their origin and have no debug bounds, so this gives them a clickable area.
func (c *ViewportComponent) pickEmptyObject(local math.Vector2) *core.Object {
	objs := c.scene.GetSortedObjects()
	for i := len(objs) - 1; i >= 0; i-- {
		obj := objs[i]
		if obj == nil || !obj.Active || obj.IsDestroyed() || obj.UI {
			continue
		}
		if _, ok := objectBounds(obj); ok {
			continue // has real bounds; scene.Pick already covers it
		}
		screen := c.cam.WorldToScreen(obj.GetPosition())
		if screen.Subtract(local).Length() <= emptyPickRadius {
			return obj
		}
	}
	return nil
}

// beginDrag handles a plain left-press: it selects the object under the cursor (via
// selectAt) and, if one was hit, captures the state needed to drag it. No undo entry is
// recorded here — that happens on release, once, only if the object actually moved.
func (c *ViewportComponent) beginDrag(local math.Vector2) {
	c.selectAt(local)
	c.dragging = false
	c.dragObj = nil
	c.dragActive = false
	c.dragMoved = false
	if c.selected == nil {
		return
	}
	c.dragging = true
	c.dragObj = c.selected
	c.dragStart = c.dragObj.GetPosition()
	c.dragGrab = c.cam.ScreenToWorld(local)
	c.dragPressScreen = local
}

// updateDrag translates the grabbed object by the cursor's world-space delta. The move
// is inert until the cursor passes dragThreshold (so a click doesn't nudge the object),
// and snaps to the grid step by default (hold Shift to move unsnapped).
func (c *ViewportComponent) updateDrag(ctx *core.Context, local math.Vector2) {
	if !c.dragActive {
		if local.Subtract(c.dragPressScreen).Length() < dragThreshold {
			return
		}
		c.dragActive = true
	}
	pos := c.dragStart.Add(c.cam.ScreenToWorld(local).Subtract(c.dragGrab))
	if !ctx.Input.IsKeyPressed(core.KeyShift) {
		stepX := c.GridStepX
		if stepX <= 0 {
			stepX = defaultGridStep
		}
		stepY := c.GridStepY
		if stepY <= 0 {
			stepY = defaultGridStep
		}
		pos = math.NewVector2(stdmath.Round(pos.X/stepX)*stepX, stdmath.Round(pos.Y/stepY)*stepY)
	}
	c.dragObj.SetPosition(pos.X, pos.Y)
	c.dragMoved = true
}

// finishDrag ends a drag-to-move, recording a single undo entry if the object moved.
func (c *ViewportComponent) finishDrag() {
	if c.dragObj != nil && c.dragMoved {
		obj := c.dragObj
		oldPos := c.dragStart
		newPos := obj.GetPosition()
		history.record(
			"moved object",
			func() { obj.SetPosition(oldPos.X, oldPos.Y) },
			func() { obj.SetPosition(newPos.X, newPos.Y) },
			true,
		)
	}
	c.dragging = false
	c.dragObj = nil
	c.dragActive = false
	c.dragMoved = false
}

// TargetScene returns the loaded target scene, or nil if none has been loaded yet
// (no project, or the load failed). Other editor panels read the target through this
// so they stay in lockstep with the viewport instead of loading their own copy.
func (c *ViewportComponent) TargetScene() *core.Scene { return c.scene }

// CurrentProject returns the resolved target project directory, or "" when no project
// is configured. The toolbar shows this in its path field.
func (c *ViewportComponent) CurrentProject() string { return c.projectDir }

// SetProject switches the target to the given project directory and reloads its scene.
// It clears the selection and any open component-args window, since both reference the
// previous scene. This is the runtime "pick a project directory" entry point the
// toolbar calls.
func (c *ViewportComponent) SetProject(dir string) {
	// Flush the outgoing project's editor settings before switching away, so a change
	// of project mid-session doesn't lose the previous project's grid/camera/selection.
	c.saveEditorPrefs()
	c.Project = dir
	c.selected = nil
	c.dragging = false
	c.dragObj = nil
	c.dragActive = false
	c.dragMoved = false
	c.scene = nil
	c.sceneFile = ""
	c.framed = false
	history.clear() // undo entries reference the previous project's live components
	closeAllArgsWindows()
	closeActiveModal() // a modal references the previous project's live components
	c.loadTarget()
}

// Save serializes the loaded target scene and writes it back to its .scene file. It
// returns an error when no scene is loaded (no project, or the load failed). On success
// it marks the document clean, so the unsaved-changes prompt stays quiet until the next
// dirty edit.
func (c *ViewportComponent) Save() error {
	if c.scene == nil {
		return fmt.Errorf("no scene loaded")
	}
	if c.sceneFile == "" {
		return fmt.Errorf("no scene file to save to")
	}
	if err := c.scene.SaveToFile(c.sceneFile); err != nil {
		return err
	}
	history.markSaved()
	return nil
}

// RefreshLogicalSize re-reads the target project's game.imge window size so the
// logical-screen outline stays in sync after the game settings modal changes it.
func (c *ViewportComponent) RefreshLogicalSize() {
	if c.projectDir == "" {
		return
	}
	if cfg, err := imgejson.LoadGameConfig(filepath.Join(c.projectDir, "game.imge")); err == nil {
		c.logicalW = float64(cfg.Window.Width)
		c.logicalH = float64(cfg.Window.Height)
	}
}

// saveEditorPrefs writes the editor-only viewport settings — grid spacing/colors, the
// navigation camera, and the last selection — to the target project's .imge.editor
// cache. It is a no-op when no project is loaded. Called on project switch and window
// close so these settings survive an editor restart.
func (c *ViewportComponent) saveEditorPrefs() {
	if c.projectDir == "" {
		return
	}
	s := editorSettings{
		FormatVersion: 1,
		GridStepX:     c.GridStepX,
		GridStepY:     c.GridStepY,
		GridColor:     c.GridColor.HexString(),
		AxesColor:     c.AxesColor.HexString(),
		Camera: &editorCameraSettings{
			X:    c.cam.x,
			Y:    c.cam.y,
			Zoom: c.cam.zoom,
		},
	}
	if c.selected != nil {
		s.SelectedObject = c.selected.Name
	}
	if err := writeEditorSettings(c.projectDir, s); err != nil {
		log.Printf("viewport: failed to write %s: %v", editorSettingsPath(c.projectDir), err)
	}
}

// loadEditorPrefs restores the target project's saved editor settings (grid, camera,
// last selection) from its .imge.editor cache. A missing or malformed cache is
// ignored, leaving the defaults (and the first-frame camera framing) in place.
func (c *ViewportComponent) loadEditorPrefs() {
	s, err := readEditorSettings(c.projectDir)
	if err != nil {
		return
	}
	if s.GridStepX > 0 {
		c.GridStepX = s.GridStepX
	}
	if s.GridStepY > 0 {
		c.GridStepY = s.GridStepY
	}
	if s.GridColor != "" {
		if col, err := math.ParseHex(s.GridColor); err == nil {
			c.GridColor = col
		}
	}
	if s.AxesColor != "" {
		if col, err := math.ParseHex(s.AxesColor); err == nil {
			c.AxesColor = col
		}
	}
	if s.Camera != nil {
		zoom := s.Camera.Zoom
		if zoom <= 0 {
			zoom = 1
		}
		c.cam = editorCamera{x: s.Camera.X, y: s.Camera.Y, zoom: zoom}
		c.framed = true // a saved camera wins over the first-frame auto-framing
	}
	if s.SelectedObject != "" && c.scene != nil {
		if obj := c.scene.GetObjectByName(s.SelectedObject); obj != nil {
			c.applySelection(obj)
		}
	}
}

// SelectedObject returns the target object the user last picked, or nil.
func (c *ViewportComponent) SelectedObject() *core.Object { return c.selected }

// Select sets the current selection to the given target object and keeps the target
// scene's debug selection in sync (pointed at one of the object's bound-reporting
// components) so its DrawDebug pass still highlights the pick. Passing nil clears the
// selection. Other panels (e.g. the object tree) call this to drive selection, so the
// viewport stays the single source of truth.
func (c *ViewportComponent) Select(obj *core.Object) {
	old := c.selected
	if old == obj {
		return
	}
	// Selection is itself undoable: record the transition so Ctrl+Z restores the prior
	// selection first, then the next undo reverts the edit underneath it visibly. It is
	// not dirty — selection is navigation, not a project-data change.
	label := "deselected"
	if old != nil {
		label = "deselected " + old.Name
	}
	if obj != nil {
		label = "selected " + obj.Name
	}
	history.record(
		label,
		func() { c.applySelection(old) },
		func() { c.applySelection(obj) },
		false,
	)
	c.applySelection(obj)
}

// SelectSilent changes the selection without recording an undo step. It is for
// programmatic side effects (selecting a freshly added/duplicated object, or clearing
// the selection after a removal) where the selection change is part of another
// undoable action, not a user's deliberate click.
func (c *ViewportComponent) SelectSilent(obj *core.Object) {
	c.applySelection(obj)
}

// applySelection sets the current selection and keeps the target scene's debug
// selection in sync. It never records history; Select and SelectSilent both delegate
// here.
func (c *ViewportComponent) applySelection(obj *core.Object) {
	c.selected = obj
	if c.scene != nil {
		c.scene.SetDebugSelection(debugPick(obj))
	}
}

// debugPick returns the object's first component that reports debug bounds (so the
// scene's debug overlay can highlight something for a selected object), or nil.
func debugPick(obj *core.Object) core.Component {
	if obj == nil {
		return nil
	}
	for _, comp := range obj.ComponentsInDrawOrder() {
		if _, ok := comp.(core.DebugBoundsProvider); ok {
			return comp
		}
	}
	return nil
}

// objectBounds returns the union of the object's debug bounds (from every component
// that reports them), plus whether any were found. Objects whose components report no
// bounds — or only a degenerate (zero-area) box, e.g. a sprite with an empty texture
// path — have no bounds; the selection outline falls back to marking their origin.
func objectBounds(obj *core.Object) (math.Rect, bool) {
	var b math.Rect
	found := false
	for _, comp := range obj.ComponentsInDrawOrder() {
		if bp, ok := comp.(core.DebugBoundsProvider); ok {
			r := bp.DebugBounds()
			if !found {
				b = r
				found = true
			} else {
				b = b.Union(r)
			}
		}
	}
	if found && (b.Width() <= 0 || b.Height() <= 0) {
		return b, false // degenerate: fall back to the origin "+" marker
	}
	return b, found
}

// Draw renders the viewport: clipped to its rect, it fills the target background,
// draws the grid and axes, then the target scene's world under the editor camera.
func (c *ViewportComponent) Draw(r core.Renderer) {
	rect := c.Rect()
	if rect.Width() <= 0 || rect.Height() <= 0 {
		return
	}
	if !c.framed {
		c.cam.frame(math.Zero(), rect.Width(), rect.Height())
		c.framed = true
	}

	r.SetClipRect(rect)

	// Background: the target scene's own clear color, or the editor's dark paper.
	bg := viewportBackground
	if c.scene != nil {
		bg = c.scene.BackgroundColor
	}
	r.DrawRect(math.NewRect(rect.X(), rect.Y(), rect.Width(), rect.Height()), bg)

	// Grid and axes are drawn in screen space (camera off) so their line thickness
	// is a true constant number of screen pixels at every zoom. In world space the
	// renderer rasterizes lines at logical resolution, so a fixed world thickness
	// fades to nothing as zoom grows.
	c.drawGrid(r, rect)
	c.drawAxes(r, rect)

	// World space: position the camera so world (cam.x, cam.y) lands at the
	// viewport's top-left (rect.Position) on screen.
	r.SetCamera(c.cam.x-rect.X()/c.cam.zoom, c.cam.y-rect.Y()/c.cam.zoom, c.cam.zoom)
	if c.scene != nil {
		c.scene.DrawWorld(r, c.DrawDebug)
	}
	r.SetCamera(0, 0, 0)

	// Selection: a screen-space outline drawn on top of the world.
	c.drawEmptyMarkers(r, rect)
	c.drawViewBounds(r, rect)
	c.drawSelection(r, rect)

	r.ClearClip()
}

// drawGrid draws a fixed world-space grid: vertical lines every GridStepX world
// units and horizontal lines every GridStepY world units, snapped so x=0 (and y=0)
// always carries a line. The steps are fixed in world space, so zooming magnifies the
// lattice instead of adding lines. Each line is drawn in screen space at a constant
// gridLineThickness pixels.
func (c *ViewportComponent) drawGrid(r core.Renderer, rect math.Rect) {
	stepX := c.GridStepX
	if stepX <= 0 {
		stepX = defaultGridStep
	}
	stepY := c.GridStepY
	if stepY <= 0 {
		stepY = defaultGridStep
	}
	left, top := c.cam.x, c.cam.y
	right := c.cam.x + rect.Width()/c.cam.zoom
	bottom := c.cam.y + rect.Height()/c.cam.zoom

	startX := stdmath.Floor(left/stepX) * stepX
	startY := stdmath.Floor(top/stepY) * stepY

	for i := 0; ; i++ {
		x := startX + float64(i)*stepX
		if x > right {
			break
		}
		p0 := c.cam.WorldToScreen(math.NewVector2(x, top)).Add(rect.Position)
		p1 := c.cam.WorldToScreen(math.NewVector2(x, bottom)).Add(rect.Position)
		r.DrawLine(p0, p1, c.GridColor, gridLineThickness)
	}
	for i := 0; ; i++ {
		y := startY + float64(i)*stepY
		if y > bottom {
			break
		}
		p0 := c.cam.WorldToScreen(math.NewVector2(left, y)).Add(rect.Position)
		p1 := c.cam.WorldToScreen(math.NewVector2(right, y)).Add(rect.Position)
		r.DrawLine(p0, p1, c.GridColor, gridLineThickness)
	}
}

// drawAxes draws the world origin axes (X at y=0, Y at x=0) slightly heavier than
// the grid so the origin reads at a glance, at a constant axesLineThickness pixels.
func (c *ViewportComponent) drawAxes(r core.Renderer, rect math.Rect) {
	left, top := c.cam.x, c.cam.y
	right := c.cam.x + rect.Width()/c.cam.zoom
	bottom := c.cam.y + rect.Height()/c.cam.zoom

	if top <= 0 && bottom >= 0 {
		p0 := c.cam.WorldToScreen(math.NewVector2(left, 0)).Add(rect.Position)
		p1 := c.cam.WorldToScreen(math.NewVector2(right, 0)).Add(rect.Position)
		r.DrawLine(p0, p1, c.AxesColor, axesLineThickness)
	}
	if left <= 0 && right >= 0 {
		p0 := c.cam.WorldToScreen(math.NewVector2(0, top)).Add(rect.Position)
		p1 := c.cam.WorldToScreen(math.NewVector2(0, bottom)).Add(rect.Position)
		r.DrawLine(p0, p1, c.AxesColor, axesLineThickness)
	}
}

// drawEmptyMarkers draws a small "+" at the origin of every world object that has no
// debug bounds (no sprite/collider/other drawable shape), so such objects stay visible
// and clickable in the viewport even though they render nothing themselves. Drawn in
// screen space so the marker keeps a constant size at any zoom.
func (c *ViewportComponent) drawEmptyMarkers(r core.Renderer, rect math.Rect) {
	if c.scene == nil {
		return
	}
	for _, obj := range c.scene.GetSortedObjects() {
		if obj == nil || !obj.Active || obj.IsDestroyed() || obj.UI {
			continue
		}
		if _, ok := objectBounds(obj); ok {
			continue
		}
		p := c.cam.WorldToScreen(obj.GetPosition()).Add(rect.Position)
		h := emptyMarkerHalf
		r.DrawLine(math.NewVector2(p.X-h, p.Y), math.NewVector2(p.X+h, p.Y), emptyMarkerColor, emptyMarkerThickness)
		r.DrawLine(math.NewVector2(p.X, p.Y-h), math.NewVector2(p.X, p.Y+h), emptyMarkerColor, emptyMarkerThickness)
	}
}

// drawSelection outlines the picked object's bounds in white, drawn in screen space
// on top of the world so the outline stays crisp at a constant width at any zoom (it
// never fills the interior, so the object stays fully visible). An object whose
// components report no debug bounds is marked with a small box at its origin instead.
//
// The four edges are clipped to the viewport: a large world object zoomed in maps to
// a screen-space box far wider than the viewport, and the chunky renderer rasterizes
// each primitive at logical resolution — an unclipped outline would exceed
// Ebitengine's atlas size limit and panic.
func (c *ViewportComponent) drawSelection(r core.Renderer, rect math.Rect) {
	if c.selected == nil {
		return
	}
	bounds, ok := objectBounds(c.selected)
	var tl, br math.Vector2
	if !ok {
		pos := c.cam.WorldToScreen(c.selected.Transform.Position).Add(rect.Position)
		const half = 5.0
		tl = math.NewVector2(pos.X-half, pos.Y-half)
		br = math.NewVector2(pos.X+half, pos.Y+half)
	} else {
		tl = c.cam.WorldToScreen(bounds.Position).Add(rect.Position)
		br = c.cam.WorldToScreen(bounds.Position.Add(bounds.Size)).Add(rect.Position)
	}
	c.drawOutlineEdges(r, rect, tl, br, selectionColor, selOutlineThickness)
}

// drawViewBounds outlines the target game's logical screen area — the rectangle from
// the world origin to (logicalWidth, logicalHeight), read from the target project's
// game.imge. Drawn in screen space (constant line width) so that, while panning and
// zooming the scene, the user can see exactly where the game's window will land. When
// the logical size is unknown (no project, or game.imge can't be read), nothing is
// drawn.
func (c *ViewportComponent) drawViewBounds(r core.Renderer, rect math.Rect) {
	if c.logicalW <= 0 || c.logicalH <= 0 {
		return
	}
	tl := c.cam.WorldToScreen(math.Zero()).Add(rect.Position)
	br := c.cam.WorldToScreen(math.NewVector2(c.logicalW, c.logicalH)).Add(rect.Position)
	c.drawOutlineEdges(r, rect, tl, br, viewRectColor, viewRectThickness)
}

// drawOutlineEdges draws a rectangle outline from tl (top-left) to br (bottom-right)
// in screen space as four line segments, each clipped to the clip rect. Clipping keeps
// every primitive no larger than the viewport, so the chunky renderer never builds an
// oversized atlas image for a zoomed-in large object.
func (c *ViewportComponent) drawOutlineEdges(r core.Renderer, clip math.Rect, tl, br math.Vector2, color math.Color, thickness float64) {
	x0, x1 := clip.Left(), clip.Right()
	y0, y1 := clip.Top(), clip.Bottom()

	// Horizontal edges (top at tl.Y, bottom at br.Y), clipped in X.
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
	// Vertical edges (left at tl.X, right at br.X), clipped in Y.
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

// loadTarget resolves and loads the target scene for display. The configured Project
// (json arg, or set via SetProject) wins; the IMGE_PROJECT environment variable is
// only a fallback when Project is empty. On success it records the resolved project
// directory and scene file path for Save.
func (c *ViewportComponent) loadTarget() {
	c.logicalW, c.logicalH = 0, 0
	project := c.Project
	if project == "" {
		project = os.Getenv("IMGE_PROJECT")
	}
	if project == "" {
		return
	}
	// Resolve to an absolute path so every later step — scene resolution, Save, and the
	// working-directory switch below — agrees on one project root regardless of how the
	// path was entered.
	if abs, err := filepath.Abs(project); err == nil {
		project = abs
	}
	sceneFile := resolveSceneFile(project, c.Scene)
	if sceneFile == "" {
		log.Printf("viewport: no scene file found in %q (scene=%q)", project, c.Scene)
		return
	}
	scene := core.NewScene(filepath.Base(sceneFile))
	if err := scene.LoadForDisplay(sceneFile); err != nil {
		log.Printf("viewport: failed to load %s: %v", sceneFile, err)
		return
	}
	c.projectDir = project
	c.sceneFile = sceneFile
	c.scene = scene
	// Make the target project the process working directory so the renderer resolves a
	// sprite's "texture" path (and any custom font/audio path) against the project root,
	// exactly as the built game does (it os.Chdir's into its extracted project data).
	// The editor's own scene is already loaded and draws only vectors with the embedded
	// pixel font, so this switch does not affect the editor UI. RUN already sets its own
	// cmd.Dir, so it is unaffected too.
	if err := os.Chdir(project); err != nil {
		log.Printf("viewport: failed to chdir to project %q: %v", project, err)
	}

	// Read the target's logical screen size so drawViewBounds can outline the game
	// window's world area. A missing/unreadable game.imge only means no outline — the
	// scene still loads and edits fine.
	if cfg, err := imgejson.LoadGameConfig(filepath.Join(project, "game.imge")); err == nil {
		c.logicalW = float64(cfg.Window.Width)
		c.logicalH = float64(cfg.Window.Height)
	} else {
		log.Printf("viewport: no logical size (game.imge): %v", err)
	}

	// Restore the project's saved editor settings (grid, camera, last selection). This
	// runs after the scene is set so the saved selection can be re-applied to it.
	c.loadEditorPrefs()

	log.Printf("viewport: loaded %s (%d objects)", sceneFile, len(scene.Objects))
}

// resolveSceneFile finds a scene file in a project directory. A non-empty scene
// selects a specific file (basename, ".scene" suffix optional, tried under the
// project root and its scenes/ subdir); an empty scene returns the first .scene
// found anywhere under the root.
func resolveSceneFile(projectDir, scene string) string {
	if scene != "" {
		name := scene
		if !strings.HasSuffix(name, ".scene") {
			name += ".scene"
		}
		for _, p := range []string{
			filepath.Join(projectDir, name),
			filepath.Join(projectDir, "scenes", name),
		} {
			if info, err := os.Stat(p); err == nil && !info.IsDir() {
				return p
			}
		}
		return ""
	}

	var found string
	_ = filepath.WalkDir(projectDir, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(d.Name(), ".scene") {
			return nil
		}
		found = p
		return fs.SkipAll
	})
	return found
}
