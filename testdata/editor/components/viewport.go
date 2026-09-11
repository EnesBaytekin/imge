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

	cam   editorCamera
	scene *core.Scene

	// objectCams is the object editor's pan/zoom per project-relative .obj path, loaded
	// from and saved to .imge.editor. The object editor reads/updates it so each .obj
	// reopens at the view it was left at.
	objectCams map[string]editorCameraSettings

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

	// pixelStep is the target game's world-units-per-pixel resolution (1 / PPU), used by
	// drag snapping so Alt-drag lands on a representable pixel. Defaults to 1 (PPU = 1).
	pixelStep float64

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
	gridLineThickness         = 1.0
	axesLineThickness         = 2.0
	selOutlineThickness       = 2.0
	viewRectThickness         = 2.0
	unzoomedViewRectThickness = 1.0
)

var (
	viewportBackground = math.NewColor(0x15, 0x17, 0x1e, 0xff)
	defaultGridColor   = math.NewColor(0x33, 0x3a, 0x4e, 0xff)
	defaultAxesColor   = math.NewColor(0x6b, 0x73, 0x85, 0xff)

	selectionColor = math.NewColor(0xff, 0xff, 0xff, 0xff) // white outline

	emptyMarkerColor = math.NewColor(0x9f, 0xa8, 0xbf, 0xff) // origin "+" for bounds-less objects

	viewRectColor = math.NewColor(0x4f, 0xd1, 0xc5, 0xff) // teal outline of the game's logical screen area

	viewRectUnzoomedColor = math.NewColor(0xff, 0xff, 0xff, 0xff) // white outline of the unzoomed (zoom=1) screen area
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
	// A modal, the object editor's focus, or an open menu bar is up: this panel is inert.
	if editorNavBlocked() {
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
	world := c.cam.ScreenToWorld(local)
	comp := c.scene.Pick(world)
	var obj *core.Object
	if comp != nil {
		obj = comp.GetOwner()
	} else if ui := c.pickUIObject(world); ui != nil {
		// scene.Pick skips UI objects (they are screen-space in the game); the editor
		// draws them in world space, so they get their own hit-test.
		obj = ui
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

// pickUIObject returns the topmost UI object whose debug bounds contain the world point,
// or nil. Scene.Pick skips UI objects (they are screen-space in the game), but the editor
// draws them in world space, so viewport click-selection needs this separate hit-test.
func (c *ViewportComponent) pickUIObject(world math.Vector2) *core.Object {
	objs := c.scene.GetSortedObjects()
	for i := len(objs) - 1; i >= 0; i-- {
		obj := objs[i]
		if obj == nil || !obj.Active || obj.IsDestroyed() || !obj.UI {
			continue
		}
		for _, comp := range obj.ComponentsInDrawOrder() {
			if vp, ok := comp.(core.VisibilityProvider); ok && !vp.IsVisible() {
				continue
			}
			if bp, ok := comp.(core.DebugBoundsProvider); ok && bp.DebugBounds().ContainsPoint(world) {
				return obj
			}
		}
	}
	return nil
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
// and snaps by modifier: no key = whole units, Shift = grid step, Alt = pixel (1/PPU).
func (c *ViewportComponent) updateDrag(ctx *core.Context, local math.Vector2) {
	if !c.dragActive {
		if local.Subtract(c.dragPressScreen).Length() < dragThreshold {
			return
		}
		c.dragActive = true
	}
	pos := c.dragStart.Add(c.cam.ScreenToWorld(local).Subtract(c.dragGrab))
	stepX := c.GridStepX
	if stepX <= 0 {
		stepX = defaultGridStep
	}
	stepY := c.GridStepY
	if stepY <= 0 {
		stepY = defaultGridStep
	}
	pos = snapDragPosition(
		pos, stepX, stepY, c.PixelStep(),
		ctx.Input.IsKeyPressed(core.KeyShift),
		ctx.Input.IsKeyPressed(core.KeyAlt),
	)
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
	c.cam = newEditorCamera()
	history.clear() // undo entries reference the previous project's live components
	closeAllArgsWindows()
	closeActiveModal() // a modal references the previous project's live components
	closeActiveObjectEditor()
	c.loadTarget()
}

// CurrentSceneName returns the loaded scene's filename basename (the stable identity
// the scene list keys on), or "" when no scene is loaded. It is the file name, not the
// scene's display name (the JSON "name" field), because selection/delete key on files.
func (c *ViewportComponent) CurrentSceneName() string {
	if c.sceneFile == "" {
		return ""
	}
	return strings.TrimSuffix(filepath.Base(c.sceneFile), ".scene")
}

// SceneFile returns the loaded scene's resolved .scene path, or "" when none.
func (c *ViewportComponent) SceneFile() string { return c.sceneFile }

// SetScene switches the viewport to another scene within the current project. It
// flushes the outgoing scene's edits (so switching never loses work), resets the
// transient edit state, and loads the new scene without restoring editor prefs (so
// camera/selection reset per scene). It is a no-op when name is empty, no project is
// loaded, or it is already the current scene.
func (c *ViewportComponent) SetScene(name string) {
	if name == "" || c.projectDir == "" || name == c.CurrentSceneName() {
		return
	}
	if c.scene != nil && c.sceneFile != "" {
		if err := c.Save(); err != nil {
			console.Print("scene switch: " + err.Error())
		}
	}
	c.selected = nil
	c.dragging = false
	c.dragObj = nil
	c.dragActive = false
	c.dragMoved = false
	c.scene = nil
	c.sceneFile = ""
	c.cam = newEditorCamera()
	history.clear() // undo entries reference the outgoing scene's live components
	closeAllArgsWindows()
	closeActiveModal()
	closeActiveObjectEditor()
	c.Scene = name
	c.loadProjectScene()
}

// ClearScene unloads the current target scene (used after the last scene is deleted),
// resetting transient state so the viewport draws only the grid/axes. It does not close
// the active modal — the delete-confirm dialog closes itself.
func (c *ViewportComponent) ClearScene() {
	if c.scene == nil && c.sceneFile == "" {
		return
	}
	c.selected = nil
	c.dragging = false
	c.dragObj = nil
	c.dragActive = false
	c.dragMoved = false
	c.scene = nil
	c.sceneFile = ""
	c.cam = newEditorCamera()
	history.clear()
	closeAllArgsWindows()
	closeActiveObjectEditor()
}

// ReloadScene re-reads the current target scene from disk, discarding in-memory edit
// state (selection, drag, undo history, open component-args windows). It is used after an
// external change to the scene's data — specifically the object editor saving a .obj that
// the scene references — so file-referenced instances refresh. The current scene is saved
// first (flushing pending edits) so nothing is lost, and the object editor is intentionally
// left open: it is a focus, not a modal, and must survive the reload its Save triggers.
func (c *ViewportComponent) ReloadScene() {
	if c.scene == nil || c.sceneFile == "" {
		return
	}
	if err := c.Save(); err != nil {
		console.Print("scene reload: " + err.Error())
	}
	c.selected = nil
	c.dragging = false
	c.dragObj = nil
	c.dragActive = false
	c.dragMoved = false
	c.scene = nil
	c.sceneFile = ""
	// The camera is intentionally preserved: ReloadScene refreshes scene data (e.g. a
	// .obj save propagating to file-referenced instances), not navigation, so the user's
	// view stays where it was.
	history.clear() // undo entries reference the outgoing scene's live components
	closeAllArgsWindows()
	c.loadProjectScene()
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

// RefreshLogicalSize re-reads the target project's game.imge window size (and pixel
// resolution) so the logical-screen outline and drag snapping stay in sync after the game
// settings modal changes them.
func (c *ViewportComponent) RefreshLogicalSize() {
	if c.projectDir == "" {
		return
	}
	if cfg, err := imgejson.LoadGameConfig(filepath.Join(c.projectDir, "game.imge")); err == nil {
		c.logicalW = float64(cfg.Window.Width)
		c.logicalH = float64(cfg.Window.Height)
		c.pixelStep = pixelStepFromPPU(cfg.Window.PixelPerUnit)
	}
}

// PixelStep returns the target game's world-units-per-pixel resolution (1 / PPU), or 1
// when it hasn't been resolved yet. Alt-drag snapping uses this so it never returns zero.
func (c *ViewportComponent) PixelStep() float64 {
	if c.pixelStep > 0 {
		return c.pixelStep
	}
	return 1
}

// saveEditorPrefs writes the editor-only viewport settings — grid spacing/colors, the
// navigation camera, and the last selection — to the target project's .imge.editor
// cache. It is a no-op when no project is loaded. Called on project switch and window
// close so these settings survive an editor restart.
func (c *ViewportComponent) saveEditorPrefs() {
	if c.projectDir == "" {
		return
	}
	// Capture the open object editor's current view first, so a window-close (or project
	// switch) while the object editor is still open persists its pan/zoom too.
	if objectEditorActive() {
		activeObjectEditor.captureCam()
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
	if len(c.objectCams) > 0 {
		s.ObjectCams = make(map[string]*editorCameraSettings, len(c.objectCams))
		for rel, cam := range c.objectCams {
			s.ObjectCams[rel] = &editorCameraSettings{X: cam.X, Y: cam.Y, Zoom: cam.Zoom}
		}
	}
	if err := writeEditorSettings(c.projectDir, s); err != nil {
		log.Printf("viewport: failed to write %s: %v", editorSettingsPath(c.projectDir), err)
	}
}

// loadEditorPrefs restores the target project's saved editor settings (grid, camera,
// last selection) from its .imge.editor cache. A missing or malformed cache is
// ignored, leaving the defaults (a top-left-anchored camera at the origin) in place.
func (c *ViewportComponent) loadEditorPrefs() {
	// Reset the per-file object-camera store: it is rebuilt from this project's cache so
	// switching projects never leaks the previous project's object-editor views.
	c.objectCams = nil
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
	}
	if len(s.ObjectCams) > 0 {
		c.objectCams = make(map[string]editorCameraSettings, len(s.ObjectCams))
		for rel, cam := range s.ObjectCams {
			if cam == nil || cam.Zoom <= 0 {
				continue
			}
			c.objectCams[rel] = editorCameraSettings{X: cam.X, Y: cam.Y, Zoom: cam.Zoom}
		}
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

// ============================================================================
// Object add/remove/duplicate (undoable), owned by the viewport because removal
// must tear down editor state that references the object — its open component-args
// windows and a selection pointing at it. The scene tree's + / x / = controls call
// these.
// ============================================================================

// removeObject detaches obj from the target scene, closing any open component-args
// window for its components and clearing a selection that points at it. It does not
// record history — it is the shared body of RemoveObject and the undo/redo closures of
// AddObject/DuplicateObject, so every path that detaches an object tears down its
// editor state consistently.
func (c *ViewportComponent) removeObject(obj *core.Object) {
	if obj == nil || c.scene == nil {
		return
	}
	closeArgsWindowsForObject(obj)
	if c.selected == obj {
		c.SelectSilent(nil)
	}
	c.scene.RemoveObject(obj.GetID())
}

// AddObject appends a fresh empty object to the target scene and records an undo entry
// that removes it (redo re-adds the same object). The scene's AddObject assigns a unique
// default name ("Object", "Object2", ...) and ID; the caller selects the new object so
// its name can be edited in the inspector. Returns the new object, or nil.
func (c *ViewportComponent) AddObject() *core.Object {
	if c.scene == nil {
		return nil
	}
	obj := core.NewObject("")
	if err := c.scene.AddObject(obj); err != nil {
		return nil
	}
	history.record(
		"added object",
		func() { c.removeObject(obj) },
		func() { c.scene.AddObject(obj) },
		true,
	)
	return obj
}

// AddObjectFromFile loads the .obj at the project-relative path rel as a file-referenced
// object (File = rel) and appends it to the target scene, recording an undo entry that
// removes it. The scene's AddObject defers component initialization, but the editor
// renders without running Scene.Update, so the template's components are initialized
// manually (mirrors DuplicateObject). Returns the new object, or nil on failure.
func (c *ViewportComponent) AddObjectFromFile(rel string) *core.Object {
	if c.scene == nil || rel == "" {
		return nil
	}
	obj, err := core.LoadObjectFromFile(filepath.Join(c.projectDir, rel))
	if err != nil {
		console.Print("load .obj: " + err.Error())
		return nil
	}
	obj.File = rel
	for _, comp := range obj.ComponentsInDrawOrder() {
		comp.Initialize()
	}
	if err := c.scene.AddObject(obj); err != nil {
		return nil
	}
	history.record(
		"loaded object from file",
		func() { c.removeObject(obj) },
		func() { c.scene.AddObject(obj) },
		true,
	)
	return obj
}

// RemoveObject detaches obj from the target scene and records an undo entry that re-adds
// the same object pointer. RemoveObject only detaches the object (unsubscribing events
// and clearing its scene ref); it leaves the object's components and name intact, so an
// undone remove brings the object back exactly as it was. Removing also closes any open
// args window for the object's components and clears a selection pointing at it.
func (c *ViewportComponent) RemoveObject(obj *core.Object) {
	if obj == nil || c.scene == nil {
		return
	}
	c.removeObject(obj)
	history.record(
		"removed object "+obj.Name,
		func() { c.scene.AddObject(obj) },
		func() { c.removeObject(obj) },
		true,
	)
}

// DuplicateObject clones obj into the target scene: it copies the object's JSON data
// (its config via ToJSONConfig, plus the live transform and active state) into a new
// object with the same components, and records an undo entry that removes the copy (redo
// re-adds the same object pointer). The scene's AddObject assigns a unique name. A
// file-referenced source keeps its File reference on the copy, so the duplicate stays
// derived from the same .obj template. The copy's components are initialized on its first
// Scene.Update (AddObject defers it), so the injected args plus Initialize defaults land
// exactly as a fresh load would. Returns the copy, or nil.
func (c *ViewportComponent) DuplicateObject(obj *core.Object) *core.Object {
	if obj == nil || c.scene == nil {
		return nil
	}
	cfg := obj.ToJSONConfig()
	// Strip a trailing duplicate counter from the source name so re-duplicating
	// continues the sequence (Object -> Object2 -> Object3) instead of nesting it
	// (Object2 -> Object22 -> Object222). The scene's AddObject re-uniquifies the base.
	dup := core.NewObject(stripNumericSuffix(cfg.Name))
	dup.Transform = obj.Transform
	dup.Active = obj.Active
	dup.Depth = cfg.Depth
	dup.Layer = cfg.Layer
	dup.UI = cfg.UI
	dup.Draggable = cfg.Draggable
	// A duplicate of a file-referenced object stays file-referenced: the copy points at
	// the same .obj template and inherits its components/tags (the tags/components copied
	// below are the template's current definition, needed so the copy renders immediately;
	// on save the scene emits the File reference rather than re-inlining them).
	dup.File = obj.File
	for _, tag := range cfg.Tags {
		dup.AddTag(tag)
	}
	for _, comp := range cfg.Components {
		cmp := buildComponent(comp.Kind, comp.Name, comp.Args)
		if cmp == nil {
			continue
		}
		if err := dup.AddComponent(cmp); err != nil {
			continue
		}
		// Initialize manually: the editor renders the target scene without running
		// Scene.Update, so a component added at runtime never reaches the deferred
		// initializeComponents pass. This mirrors addComponentTo/restoreComponent.
		cmp.Initialize()
	}
	if err := c.scene.AddObject(dup); err != nil {
		return nil
	}
	history.record(
		"duplicated object",
		func() { c.removeObject(dup) },
		func() { c.scene.AddObject(dup) },
		true,
	)
	return dup
}

// objectBounds returns the debug bounds the selection outline is drawn around. Sprites
// are the object's visible footprint, so their bounds win when present: a sprite's
// offset then shifts the outline together with the drawn texture instead of growing the
// box (which is what happens if we union a shifted sprite with an unshifted collider).
// Colliders/triggers are physics and are already outlined separately by their own
// DrawDebug, so they only contribute when the object has no sprite at all.
//
// Objects whose components report no bounds — or only a degenerate (zero-area) box,
// e.g. a sprite with an empty texture path — have no bounds; the selection outline
// falls back to marking their origin.
func objectBounds(obj *core.Object) (math.Rect, bool) {
	var b math.Rect
	found := false

	// Pass 1: visible sprites (the object's visible footprint). Hidden sprites are
	// skipped so an @Animator that keeps only the current frame's sprite visible
	// doesn't inflate the outline with every frame it has hidden.
	for _, comp := range obj.ComponentsInDrawOrder() {
		spr, isSprite := comp.(*Sprite)
		if !isSprite || !spr.IsVisible() {
			continue
		}
		r := spr.DebugBounds()
		if !found {
			b = r
			found = true
		} else {
			b = b.Union(r)
		}
	}

	// Pass 2: no visible sprite — union the non-sprite bounds providers (collider,
	// trigger, ...) so a physics-only object still outlines its hitbox. Invisible
	// sprites are skipped here too (they were already excluded in pass 1); an object
	// whose only shape is a hidden sprite falls through to the origin marker.
	if !found {
		for _, comp := range obj.ComponentsInDrawOrder() {
			if _, isSprite := comp.(*Sprite); isSprite {
				continue
			}
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
	}

	if found && (b.Width() <= 0 || b.Height() <= 0) {
		return b, false // degenerate: fall back to the origin "+" marker
	}
	return b, found
}

// objectLocalBounds mirrors objectBounds (same pass-1 sprites / pass-2 providers rule)
// but returns the union in the object's LOCAL (pre-transform) space instead of world
// space. The selection outline transforms its four corners through the object transform
// to draw a rotated+scaled quad that hugs the object, so it needs the local rect — the
// world AABB alone cannot be de-rotated back to it under non-uniform scale.
func objectLocalBounds(obj *core.Object) (math.Rect, bool) {
	var b math.Rect
	found := false

	// Pass 1: visible sprites (the object's visible footprint), exactly as objectBounds.
	for _, comp := range obj.ComponentsInDrawOrder() {
		spr, isSprite := comp.(*Sprite)
		if !isSprite || !spr.IsVisible() {
			continue
		}
		r := spr.LocalBounds()
		if !found {
			b = r
			found = true
		} else {
			b = b.Union(r)
		}
	}

	// Pass 2: no visible sprite — union the non-sprite bounds providers that expose a
	// local bounds (panel, collider). Same set that objectBounds uses via DebugBounds.
	if !found {
		for _, comp := range obj.ComponentsInDrawOrder() {
			if _, isSprite := comp.(*Sprite); isSprite {
				continue
			}
			if lp, ok := comp.(interface{ LocalBounds() math.Rect }); ok {
				r := lp.LocalBounds()
				if !found {
					b = r
					found = true
				} else {
					b = b.Union(r)
				}
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
	// viewport's top-left (rect.Position) on screen. Both the scene's world objects
	// (DrawWorld) and its UI objects are drawn under this camera, so in the editor a
	// UI object moves with pan/zoom like any other scene object — the viewport acts as
	// the game's "screen", and a UI object's screen coordinates (relative to the game
	// window's top-left, i.e. the world origin) land at the matching world spot.
	r.SetCamera(c.cam.x-rect.X()/c.cam.zoom, c.cam.y-rect.Y()/c.cam.zoom, c.cam.zoom)
	if c.scene != nil {
		c.scene.DrawWorld(r, c.DrawDebug)
		c.drawUIObjects(r)
	}
	r.SetCamera(0, 0, 0)

	// Selection and markers: drawn in screen space on top of the world so they stay
	// crisp at a constant width at any zoom.
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

// drawUIObjects draws the target scene's UI objects under the editor's world camera, so
// a UI object positioned in screen coordinates (relative to the game window's top-left,
// i.e. the world origin) moves with pan/zoom exactly like a world object. In the editor
// the viewport acts as the game's "screen": a UI object at (0,0) sits at the world
// origin, which maps to the viewport's top-left. Only the editor draws UI this way — a
// built game draws UI screen-fixed with no camera (see Scene.Draw). The caller must have
// the world camera active and clears it afterward.
func (c *ViewportComponent) drawUIObjects(r core.Renderer) {
	if c.scene == nil {
		return
	}
	for _, obj := range c.scene.GetSortedObjects() {
		if obj == nil || !obj.Active || obj.IsDestroyed() || !obj.UI {
			continue
		}
		obj.Draw(r)
	}
}

// drawSelection outlines the picked object in white, drawn in screen space on top of the
// world so the outline stays crisp at a constant width at any zoom (it never fills the
// interior, so the object stays fully visible). For a world object the outline is the
// object's actual rotated+scaled quad — its local bounds mapped through the object
// transform — so it hugs the object instead of its axis-aligned enclosing box. A UI
// object has no transform to de-rotate, so it keeps its plain axis-aligned bounds. An
// object whose components report no bounds is marked with a small box at its origin.
//
// The four edges are clipped to the viewport: a large world object zoomed in maps to
// a screen-space box far wider than the viewport, and the chunky renderer rasterizes
// each primitive at logical resolution — an unclipped outline would exceed
// Ebitengine's atlas size limit and panic.
func (c *ViewportComponent) drawSelection(r core.Renderer, rect math.Rect) {
	if c.selected == nil {
		return
	}

	// World objects: map the object's local bounds through its transform (scale → rotate
	// about the origin → translate), then into screen space, and draw that quad. This is
	// the object's true footprint, rotated and scaled just like the object itself.
	if !c.selected.UI {
		local, ok := objectLocalBounds(c.selected)
		if !ok {
			c.drawSelectionOrigin(r, rect)
			return
		}
		x0, y0 := local.X(), local.Y()
		x1, y1 := local.X()+local.Width(), local.Y()+local.Height()
		corners := [4]math.Vector2{
			{X: x0, Y: y0},
			{X: x1, Y: y0},
			{X: x1, Y: y1},
			{X: x0, Y: y1},
		}
		transform := c.selected.Transform
		var pts [4]math.Vector2
		for i := range corners {
			pts[i] = c.cam.WorldToScreen(transform.LocalToWorld(corners[i])).Add(rect.Position)
		}
		for i := 0; i < 4; i++ {
			c.drawClippedLine(r, rect, pts[i], pts[(i+1)%4], selectionColor, selOutlineThickness)
		}
		return
	}

	// UI object: no object transform to de-rotate, so draw its axis-aligned world bounds
	// straight from their corners.
	bounds, ok := objectBounds(c.selected)
	if !ok {
		c.drawSelectionOrigin(r, rect)
		return
	}
	x0, y0 := bounds.X(), bounds.Y()
	x1, y1 := bounds.X()+bounds.Width(), bounds.Y()+bounds.Height()
	corners := [4]math.Vector2{
		{X: x0, Y: y0},
		{X: x1, Y: y0},
		{X: x1, Y: y1},
		{X: x0, Y: y1},
	}
	var pts [4]math.Vector2
	for i := range corners {
		pts[i] = c.cam.WorldToScreen(corners[i]).Add(rect.Position)
	}
	for i := 0; i < 4; i++ {
		c.drawClippedLine(r, rect, pts[i], pts[(i+1)%4], selectionColor, selOutlineThickness)
	}
}

// drawSelectionOrigin marks an object whose components report no bounds with a small box
// at its origin. UI and world objects are both drawn in world space in the editor, so the
// origin maps the same way.
func (c *ViewportComponent) drawSelectionOrigin(r core.Renderer, rect math.Rect) {
	pos := c.cam.WorldToScreen(c.selected.Transform.Position).Add(rect.Position)
	const half = 5.0
	c.drawOutlineEdges(r, rect, math.NewVector2(pos.X-half, pos.Y-half), math.NewVector2(pos.X+half, pos.Y+half), selectionColor, selOutlineThickness)
}

// drawClippedLine draws a line segment clipped to the clip rect, using Liang-Barsky
// clipping so a rotated (non-axis-aligned) selection edge never rasterizes wider than
// the viewport — the chunky renderer would otherwise build an atlas image sized to the
// full unclipped segment and panic for a large zoomed-in object.
func (c *ViewportComponent) drawClippedLine(r core.Renderer, clip math.Rect, a, b math.Vector2, color math.Color, thickness float64) {
	dx := b.X - a.X
	dy := b.Y - a.Y
	x0, y0 := clip.Left(), clip.Top()
	x1, y1 := clip.Right(), clip.Bottom()
	p := [4]float64{-dx, dx, -dy, dy}
	q := [4]float64{a.X - x0, x1 - a.X, a.Y - y0, y1 - a.Y}
	u1, u2 := 0.0, 1.0
	for i := 0; i < 4; i++ {
		if p[i] == 0 {
			if q[i] < 0 {
				return // parallel to this boundary and outside
			}
			continue
		}
		t := q[i] / p[i]
		if p[i] < 0 {
			if t > u2 {
				return
			}
			if t > u1 {
				u1 = t
			}
		} else {
			if t < u1 {
				return
			}
			if t < u2 {
				u2 = t
			}
		}
	}
	r.DrawLine(
		math.NewVector2(a.X+u1*dx, a.Y+u1*dy),
		math.NewVector2(a.X+u2*dx, a.Y+u2*dy),
		color, thickness,
	)
}

// drawViewBounds outlines the target game's visible screen area — the world rectangle
// the game's window will show once the scene camera is applied. The scene camera
// (core.Camera) places the window's top-left at (X, Y) in world space and its zoom
// divides the logical window size (game.imge width/height) to yield the visible world
// extent. With no scene camera the game draws world = screen, i.e. an identity camera
// at the origin. Drawn in screen space (constant line width) so that, while panning
// and zooming the scene, the user can see exactly where the game's window will land.
// When the logical size is unknown (no project, or game.imge can't be read), nothing
// is drawn.
//
// When the camera is zoomed (zoom != 1), a second rectangle is drawn at the same camera
// position showing the window at zoom 1 (a thin white outline). Seeing both at once lets
// the user compare the magnified view against the true 1:1 footprint of the game screen.
func (c *ViewportComponent) drawViewBounds(r core.Renderer, rect math.Rect) {
	if c.logicalW <= 0 || c.logicalH <= 0 {
		return
	}
	camX, camY, camZoom := 0.0, 0.0, 1.0
	if c.scene != nil && c.scene.Camera != nil {
		camX, camY, camZoom = c.scene.Camera.X, c.scene.Camera.Y, c.scene.Camera.Zoom
		if camZoom <= 0 {
			camZoom = 1
		}
	}

	// Zoomed view: the world area the game window shows at the scene camera's zoom.
	tl := c.cam.WorldToScreen(math.NewVector2(camX, camY)).Add(rect.Position)
	br := c.cam.WorldToScreen(math.NewVector2(camX+c.logicalW/camZoom, camY+c.logicalH/camZoom)).Add(rect.Position)
	c.drawOutlineEdges(r, rect, tl, br, viewRectColor, viewRectThickness)

	// Unzoomed reference: when the camera is zoomed, also show the same window at zoom 1.
	// It shares the top-left corner with the zoomed rect; only its width/height differ
	// (logical / 1). Drawn thin and white to read as a reference, distinct from the teal
	// zoomed rect.
	if camZoom != 1 {
		brBase := c.cam.WorldToScreen(math.NewVector2(camX+c.logicalW, camY+c.logicalH)).Add(rect.Position)
		c.drawOutlineEdges(r, rect, tl, brBase, viewRectUnzoomedColor, unzoomedViewRectThickness)
	}
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

// loadTarget resolves and loads the target scene for display, then restores the
// project's saved editor settings (grid, camera, selection). It is the initial-load
// path (Initialize and SetProject); SetScene calls loadProjectScene directly so a
// scene switch resets camera/selection instead of restoring them.
func (c *ViewportComponent) loadTarget() {
	if c.loadProjectScene() {
		c.loadEditorPrefs()
	}
}

// loadProjectScene resolves the configured Project (json arg, or set via SetProject;
// the IMGE_PROJECT env var is a fallback when Project is empty) and the scene within
// it, and loads that scene into c.scene for display. It records the resolved project
// directory and scene file path for Save. It does not touch editor prefs. Returns
// false when there is nothing to load (no project, no scene file, or load error).
func (c *ViewportComponent) loadProjectScene() bool {
	c.logicalW, c.logicalH = 0, 0
	project := c.Project
	if project == "" {
		project = os.Getenv("IMGE_PROJECT")
	}
	if project == "" {
		return false
	}
	// Resolve to an absolute path so every later step — scene resolution, Save, and the
	// working-directory switch below — agrees on one project root regardless of how the
	// path was entered.
	if abs, err := filepath.Abs(project); err == nil {
		project = abs
	}

	// Make the target project the process working directory BEFORE loading the scene, so
	// a file-referenced object's relative .obj path (and any custom sprite/font/audio
	// path) resolves against the project root exactly as the built game does (it
	// os.Chdir's into its extracted project data). This must precede LoadForDisplay: that
	// load resolves each {file: ...} object's template relative to the current directory,
	// and resolving them against the launch CWD is why a freshly-launched editor could
	// come up with its file-referenced objects missing until the project was re-opened.
	// The editor's own scene is already loaded and draws only vectors with the embedded
	// pixel font, so this switch does not affect the editor UI. RUN sets its own cmd.Dir.
	if err := os.Chdir(project); err != nil {
		log.Printf("viewport: failed to chdir to project %q: %v", project, err)
	}

	sceneFile := resolveSceneFile(project, c.Scene)
	if sceneFile == "" {
		log.Printf("viewport: no scene file found in %q (scene=%q)", project, c.Scene)
		return false
	}
	scene := core.NewScene(filepath.Base(sceneFile))
	if err := scene.LoadForDisplay(sceneFile); err != nil {
		log.Printf("viewport: failed to load %s: %v", sceneFile, err)
		return false
	}
	c.Project = project // pin to the resolved absolute dir so a later SetScene re-resolves
	c.projectDir = project
	c.sceneFile = sceneFile
	c.scene = scene

	// Read the target's logical screen size so drawViewBounds can outline the game
	// window's world area. A missing/unreadable game.imge only means no outline — the
	// scene still loads and edits fine.
	if cfg, err := imgejson.LoadGameConfig(filepath.Join(project, "game.imge")); err == nil {
		c.logicalW = float64(cfg.Window.Width)
		c.logicalH = float64(cfg.Window.Height)
		c.pixelStep = pixelStepFromPPU(cfg.Window.PixelPerUnit)
	} else {
		log.Printf("viewport: no logical size (game.imge): %v", err)
	}

	log.Printf("viewport: loaded %s (%d objects)", sceneFile, len(scene.Objects))
	return true
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
