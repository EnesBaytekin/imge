// Package core contains platform-agnostic game engine logic.
// This file defines the Scene system - a container for game objects.
package core

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"sort"

	corejson "github.com/EnesBaytekin/imge/core/json"
	"github.com/EnesBaytekin/imge/core/math"
)

// ============================================================================
// Scene Definition
// ============================================================================

// Scene represents a collection of game objects that can be updated and drawn together.
type Scene struct {
	// Objects maps object ID to object pointer for O(1) lookup
	Objects map[uint64]*Object

	// nameToID maps object name to object ID for O(1) name lookup
	nameToID map[string]uint64

	// Tags maps tag name to set of object IDs that have that tag
	Tags map[string]map[uint64]bool

	// SortedObjects contains object IDs sorted by depth (ascending)
	// Lower depth drawn first, higher depth drawn last (on top)
	SortedObjects []uint64

	// depthChangedIDs tracks objects whose depth has changed since last sort
	depthChangedIDs map[uint64]bool

	// nextObjectID is the next available unique object ID
	nextObjectID uint64

	// frame counts update cycles, incremented once per Scene.Update.
	frame uint64

	// Name is the scene's identifier
	Name string

	// BackgroundColor is the clear color used each frame before objects draw.
	BackgroundColor math.Color

	// Camera transforms world coordinates to screen coordinates when drawing. Nil
	// means no transform (world = screen).
	Camera *Camera

	// Active controls whether the scene is updated and drawn
	Active bool

	// DebugDraw enables the debug overlay pass: after all normal draws, every
	// component implementing DebugDrawer draws its debug visuals on top. Off by
	// default; the editor and `imge build --debug` turn it on.
	DebugDraw bool

	// debugSelected is the component the editor has selected, so the overlay can
	// pass DebugInfo.Selected to it. Set via SetDebugSelection.
	debugSelected Component

	// EventManager handles the event queue and subscriptions for this scene.
	// Processed after all component Update() calls each frame.
	EventManager *EventManager
}

// ============================================================================
// Scene Creation
// ============================================================================

// NewScene creates a new empty scene.
func NewScene(name string) *Scene {
	return &Scene{
		Objects:         make(map[uint64]*Object),
		nameToID:        make(map[string]uint64),
		Tags:            make(map[string]map[uint64]bool),
		SortedObjects:   make([]uint64, 0),
		depthChangedIDs: make(map[uint64]bool),
		nextObjectID:    1, // Start from 1, 0 is invalid
		Name:            name,
		BackgroundColor: math.Black,
		Active:          true,
		EventManager:    NewEventManager(),
	}
}

// ============================================================================
// Object Management
// ============================================================================

// AddObject adds an object to the scene.
// Assigns a unique ID and updates all internal mappings.
// Returns an error if the object's name conflicts with an existing object.
func (s *Scene) AddObject(obj *Object) error {
	// Generate unique name if needed
	name := obj.Name
	if name == "" {
		name = "Object"
	}
	name = s.generateUniqueName(name)

	// Assign unique ID
	id := s.generateUniqueID()
	obj.SetID(id)
	obj.Name = name
	obj.Scene = s

	// Store object
	s.Objects[id] = obj
	s.nameToID[name] = id

	// Add to sorted list (insert at correct position based on depth)
	s.insertIntoSortedList(id)

	// Add tags to tag mapping
	for tag := range obj.Tags {
		s.addObjectToTag(id, tag)
	}

	// Component Initialize/event-subscription is deferred to the first Update so
	// the object sees a fully-assembled scene (see Scene.Update / initializeComponents).

	return nil
}

// RemoveObject removes an object from the scene by ID.
// Unsubscribes all components from events before removal.
func (s *Scene) RemoveObject(id uint64) {
	obj, exists := s.Objects[id]
	if !exists {
		return
	}

	// Unsubscribe all components from events
	for _, comp := range obj.Components {
		s.EventManager.UnsubscribeAll(comp)
	}

	// Remove from name mapping
	delete(s.nameToID, obj.Name)

	// Remove from tag mappings
	for tag := range obj.Tags {
		s.removeObjectFromTag(id, tag)
	}

	// Remove from sorted list
	s.removeFromSortedList(id)

	// Remove from depth changed tracking
	delete(s.depthChangedIDs, id)

	// Remove from objects map
	delete(s.Objects, id)

	// Clear object's scene reference
	obj.Scene = nil
}

// GetObjectByID retrieves an object by its ID (O(1) lookup).
// Returns nil if the object doesn't exist.
func (s *Scene) GetObjectByID(id uint64) *Object {
	return s.Objects[id]
}

// GetObjectByName retrieves an object by its name (O(1) lookup via nameToID).
// Returns nil if the object doesn't exist.
func (s *Scene) GetObjectByName(name string) *Object {
	id, exists := s.nameToID[name]
	if !exists {
		return nil
	}
	return s.Objects[id]
}

// renameObject changes an object's name, ensuring uniqueness.
// Called by Object.SetName() when the object is in a scene.
func (s *Scene) renameObject(id uint64, newName string) error {
	obj, exists := s.Objects[id]
	if !exists {
		return fmt.Errorf("object with ID %d not found", id)
	}

	// Check if new name is already taken (and not by this object)
	if existingID, taken := s.nameToID[newName]; taken && existingID != id {
		return fmt.Errorf("name '%s' is already taken by another object", newName)
	}

	// Update name mapping
	delete(s.nameToID, obj.Name)
	s.nameToID[newName] = id

	// Update object's name
	obj.Name = newName

	return nil
}

// ============================================================================
// Tag Management
// ============================================================================

// FindObjectsWithTag returns all objects with the given tag (O(1) lookup).
func (s *Scene) FindObjectsWithTag(tag string) []*Object {
	idSet, exists := s.Tags[tag]
	if !exists {
		return []*Object{}
	}

	result := make([]*Object, 0, len(idSet))
	for id := range idSet {
		if obj, exists := s.Objects[id]; exists {
			result = append(result, obj)
		}
	}

	return result
}

// addObjectToTag adds an object ID to a tag's set.
// Called by Object.AddTag().
func (s *Scene) addObjectToTag(id uint64, tag string) {
	if s.Tags[tag] == nil {
		s.Tags[tag] = make(map[uint64]bool)
	}
	s.Tags[tag][id] = true
}

// removeObjectFromTag removes an object ID from a tag's set.
// Called by Object.RemoveTag().
func (s *Scene) removeObjectFromTag(id uint64, tag string) {
	if tagSet, exists := s.Tags[tag]; exists {
		delete(tagSet, id)
		if len(tagSet) == 0 {
			delete(s.Tags, tag)
		}
	}
}

// ============================================================================
// Depth Sorting
// ============================================================================

// markDepthChanged marks an object's depth as changed.
// Called by Object.SetDepth().
func (s *Scene) markDepthChanged(id uint64) {
	s.depthChangedIDs[id] = true
}

// updateSortedObjects updates the sorted list using insertion sort.
// Only processes objects marked as having changed depth.
func (s *Scene) updateSortedObjects() {
	if len(s.depthChangedIDs) == 0 {
		return
	}

	// For each changed object, remove and reinsert at correct position
	for id := range s.depthChangedIDs {
		s.removeFromSortedList(id)
		s.insertIntoSortedList(id)
	}

	// Clear changed IDs
	s.depthChangedIDs = make(map[uint64]bool)
}

// insertIntoSortedList inserts an object ID into the sorted list at the correct position.
// Uses binary search to find insertion point.
func (s *Scene) insertIntoSortedList(id uint64) {
	obj := s.Objects[id]
	if obj == nil {
		return
	}

	layer := obj.Layer
	depth := obj.Depth

	// Find insertion index using binary search. Objects sort by layer first (lower
	// layer first), then by depth within the layer, so a higher layer always draws
	// on top regardless of depth.
	insertIndex := sort.Search(len(s.SortedObjects), func(i int) bool {
		otherID := s.SortedObjects[i]
		otherObj := s.Objects[otherID]
		if otherObj == nil {
			return true // Shouldn't happen, but treat missing objects as infinite depth
		}
		if otherObj.Layer != layer {
			return otherObj.Layer > layer
		}
		return otherObj.Depth >= depth
	})

	// Insert at the found position
	s.SortedObjects = append(s.SortedObjects, 0) // Add zero value at end
	copy(s.SortedObjects[insertIndex+1:], s.SortedObjects[insertIndex:])
	s.SortedObjects[insertIndex] = id
}

// removeFromSortedList removes an object ID from the sorted list.
func (s *Scene) removeFromSortedList(id uint64) {
	for i, existingID := range s.SortedObjects {
		if existingID == id {
			// Remove by slicing
			s.SortedObjects = append(s.SortedObjects[:i], s.SortedObjects[i+1:]...)
			return
		}
	}
}

// GetSortedObjects returns objects in depth order (ascending).
// Calls updateSortedObjects first to ensure the list is up-to-date.
func (s *Scene) GetSortedObjects() []*Object {
	s.updateSortedObjects()

	result := make([]*Object, 0, len(s.SortedObjects))
	for _, id := range s.SortedObjects {
		if obj, exists := s.Objects[id]; exists {
			result = append(result, obj)
		}
	}

	return result
}

// FrameNumber returns the number of update cycles that have begun (1-based). It
// increments once per Scene.Update, so components can use it to detect "this
// frame" conditions (e.g. a @StateMachine's JustEntered).
func (s *Scene) FrameNumber() uint64 {
	return s.frame
}

// ============================================================================
// Lifecycle Methods
// ============================================================================

// Update calls Update on all active objects in the scene.
// Before the first update, it runs each object's component Initialize() (once,
// after the scene is fully assembled). After all component updates, it processes
// the event queue. Depth order doesn't matter for updates.
func (s *Scene) Update(ctx *Context) {
	if !s.Active {
		return
	}

	s.frame++
	ctx.Scene = s

	for _, obj := range s.Objects {
		if obj.IsDestroyed() {
			continue
		}
		obj.initializeComponents()
		if obj.Active {
			obj.Update(ctx)
		}
	}

	// Deliver queued events to subscribed components.
	s.EventManager.Process()

	// Remove destroyed objects
	s.removeDestroyedObjects()
}

// Draw calls Draw on all active objects in the scene, sorted by depth. World
// objects (UI=false) draw under the camera transform; UI objects draw afterward
// in screen space (no camera), so they always sit on top of the world.
func (s *Scene) Draw(renderer Renderer) {
	if !s.Active {
		return
	}

	// Ensure sorted list is up-to-date
	s.updateSortedObjects()

	if s.Camera != nil {
		vw, vh := renderer.GetViewportSize()
		s.Camera.setViewport(float64(vw), float64(vh))
		// Advance the camera toward its follow target after objects move, now that
		// the viewport is known (follow centers the target, which needs the size).
		s.Camera.Tick()
		renderer.SetCamera(s.Camera.X, s.Camera.Y, s.Camera.Zoom)
	} else {
		renderer.SetCamera(0, 0, 0)
	}

	for _, id := range s.SortedObjects {
		obj := s.Objects[id]
		if obj != nil && obj.Active && !obj.IsDestroyed() && !obj.UI {
			obj.Draw(renderer)
		}
	}

	// UI objects draw in raw screen space, on top of the world.
	renderer.SetCamera(0, 0, 0)
	for _, id := range s.SortedObjects {
		obj := s.Objects[id]
		if obj != nil && obj.Active && !obj.IsDestroyed() && obj.UI {
			obj.Draw(renderer)
		}
	}

	// Debug overlay draws last, always on top of the finished frame.
	if s.DebugDraw {
		s.drawDebugOverlay(renderer)
	}
}

// SetDebugDraw enables or disables the debug overlay pass.
func (s *Scene) SetDebugDraw(v bool) { s.DebugDraw = v }

// SetDebugSelection marks a component as the current selection so the debug overlay
// passes it DebugInfo.Selected == true. Pass nil to clear the selection.
func (s *Scene) SetDebugSelection(c Component) { s.debugSelected = c }

// Pick returns the topmost world component under a world-space point whose bounds
// (reported via DebugBoundsProvider) contain it, or nil if nothing is hit. Objects
// are tested back-to-front (reverse draw order) so a click on overlapping objects
// selects the one the user sees on top; within an object, components are likewise
// tested back-to-front. Only active, non-destroyed, non-UI world objects are
// considered — UI objects are screen-space and picked separately. The returned
// component is the one to pass to SetDebugSelection to highlight it.
func (s *Scene) Pick(point math.Vector2) Component {
	s.updateSortedObjects()
	for i := len(s.SortedObjects) - 1; i >= 0; i-- {
		obj := s.Objects[s.SortedObjects[i]]
		if obj == nil || !obj.Active || obj.IsDestroyed() || obj.UI {
			continue
		}
		if comp := pickComponent(obj, point); comp != nil {
			return comp
		}
	}
	return nil
}

// pickComponent returns the topmost DebugBoundsProvider component on obj whose bounds
// contain point, or nil.
func pickComponent(obj *Object, point math.Vector2) Component {
	comps := obj.ComponentsInDrawOrder()
	for i := len(comps) - 1; i >= 0; i-- {
		if bp, ok := comps[i].(DebugBoundsProvider); ok && bp.DebugBounds().ContainsPoint(point) {
			return comps[i]
		}
	}
	return nil
}

// DrawWorld draws the scene's world (non-UI) objects under the currently applied
// camera transform, without touching the scene's own Camera. It is the editor's
// render path: the editor sets its navigation camera (and any clip rect) on the
// renderer, then calls this to draw the target scene — so it can pan and zoom
// freely while the game camera, applied when the game runs, stays untouched.
//
// When drawDebug is true, each object's DebugDrawer components are drawn on top of
// the world, marking the selection set via SetDebugSelection. The camera is left
// as the caller set it, so the caller can keep drawing editor overlays (axes, grid,
// gizmo) in the same world space before restoring its own camera.
func (s *Scene) DrawWorld(renderer Renderer, drawDebug bool) {
	if !s.Active {
		return
	}
	s.updateSortedObjects()

	for _, id := range s.SortedObjects {
		obj := s.Objects[id]
		if obj != nil && obj.Active && !obj.IsDestroyed() && !obj.UI {
			obj.Draw(renderer)
		}
	}

	if drawDebug {
		for _, id := range s.SortedObjects {
			obj := s.Objects[id]
			if obj != nil && obj.Active && !obj.IsDestroyed() && !obj.UI {
				s.drawObjectDebug(obj, renderer)
			}
		}
	}
}

// drawDebugOverlay runs the final debug pass: it walks every active object in draw
// order and calls DrawDebug on each component implementing DebugDrawer. It mirrors
// the normal draw's camera split (world objects under the camera, UI objects in raw
// screen space) so debug bounds land where their component draws.
func (s *Scene) drawDebugOverlay(renderer Renderer) {
	if s.Camera != nil {
		renderer.SetCamera(s.Camera.X, s.Camera.Y, s.Camera.Zoom)
	} else {
		renderer.SetCamera(0, 0, 0)
	}
	for _, id := range s.SortedObjects {
		obj := s.Objects[id]
		if obj != nil && obj.Active && !obj.IsDestroyed() && !obj.UI {
			s.drawObjectDebug(obj, renderer)
		}
	}

	renderer.SetCamera(0, 0, 0)
	for _, id := range s.SortedObjects {
		obj := s.Objects[id]
		if obj != nil && obj.Active && !obj.IsDestroyed() && obj.UI {
			s.drawObjectDebug(obj, renderer)
		}
	}
}

// drawObjectDebug calls DrawDebug on the object's DebugDrawer components, in draw
// order, marking the selected one.
func (s *Scene) drawObjectDebug(obj *Object, renderer Renderer) {
	for _, comp := range obj.drawComponents() {
		if dd, ok := comp.(DebugDrawer); ok {
			dd.DrawDebug(renderer, DebugInfo{Selected: comp == s.debugSelected})
		}
	}
}

// removeDestroyedObjects removes all objects marked for destruction.
func (s *Scene) removeDestroyedObjects() {
	destroyedIDs := make([]uint64, 0)
	for id, obj := range s.Objects {
		if obj.IsDestroyed() {
			destroyedIDs = append(destroyedIDs, id)
		}
	}

	for _, id := range destroyedIDs {
		s.RemoveObject(id)
	}
}

// ============================================================================
// ID and Name Generation
// ============================================================================

// generateUniqueID generates a new unique object ID.
func (s *Scene) generateUniqueID() uint64 {
	id := s.nextObjectID
	s.nextObjectID++
	return id
}

// generateUniqueName generates a unique name based on a base name.
func (s *Scene) generateUniqueName(base string) string {
	name := base
	counter := 1

	for s.nameToID[name] != 0 {
		counter++
		name = fmt.Sprintf("%s%d", base, counter)
	}

	return name
}

// ============================================================================
// JSON Serialization (Placeholders)
// ============================================================================

// LoadFromJSON loads a scene from JSON data.
func (s *Scene) LoadFromJSON(data []byte) error {
	return s.loadFromJSON(data, nil, false)
}

// LoadFromFS loads a scene from the given filesystem, resolving any referenced
// object files through the same filesystem. This is used by web builds, where
// game data is embedded rather than read from a real filesystem.
func (s *Scene) LoadFromFS(fsys fs.FS, path string) error {
	data, err := fs.ReadFile(fsys, path)
	if err != nil {
		return fmt.Errorf("failed to read scene file %s: %w", path, err)
	}
	return s.loadFromJSON(data, fsys, false)
}

// LoadForDisplay loads a scene file for editor/thumbnail display. It is lenient
// about components whose kind is not registered — they are skipped (logged)
// rather than failing the load — and it runs each loaded component's Initialize
// once so the scene renders with its defaults, while no Update ever runs (so the
// simulation stays paused). Normal game loading (LoadFromFile/LoadFromFS) stays
// strict, so a real game fails fast on a bad component reference.
func (s *Scene) LoadForDisplay(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read scene file %s: %w", path, err)
	}
	if err := s.loadFromJSON(data, nil, true); err != nil {
		return err
	}
	s.InitializeForRender()
	return nil
}

// InitializeForRender runs each object's component Initialize() exactly once, so a
// scene loaded for display renders with its defaults applied (styles resolved,
// sizes and colors filled in) — without running any Update. It is the editor's
// counterpart to Scene.Update's deferred initialization: same one-time setup, no
// simulation.
func (s *Scene) InitializeForRender() {
	for _, obj := range s.Objects {
		obj.initializeComponents()
	}
}

// loadFromJSON loads a scene from JSON data. When fsys is nil, referenced object
// files are read from the OS filesystem; otherwise they are read from fsys. When
// lenient is true, an object or component that fails to create is skipped (logged)
// instead of aborting the load; this is the editor's path for opening a scene whose
// custom components aren't compiled into the current build.
func (s *Scene) loadFromJSON(data []byte, fsys fs.FS, lenient bool) error {
	var config corejson.SceneConfig
	if err := json.Unmarshal(corejson.StripComments(data), &config); err != nil {
		return fmt.Errorf("failed to parse scene JSON: %w", err)
	}

	if config.Name != "" {
		s.Name = config.Name
	}
	if config.BackgroundColor != "" {
		if c, err := math.ParseHex(config.BackgroundColor); err == nil {
			s.BackgroundColor = c
		}
	}
	if config.Camera != nil {
		s.Camera = NewCamera()
		s.Camera.X = config.Camera.X
		s.Camera.Y = config.Camera.Y
		if config.Camera.Zoom > 0 {
			s.Camera.Zoom = config.Camera.Zoom
		}
		s.Camera.Smoothing = config.Camera.Smoothing
		s.Camera.LockX = config.Camera.LockX
		s.Camera.LockY = config.Camera.LockY
	}

	// Load objects from config
	for _, objConfig := range config.Objects {
		obj, err := createObjectFromSceneObject(objConfig, fsys, lenient)
		if err != nil {
			if lenient {
				log.Printf("scene %s: skipping object %q: %v", s.Name, objConfig.Name, err)
				continue
			}
			return fmt.Errorf("failed to create object: %w", err)
		}
		if err := s.AddObject(obj); err != nil {
			if lenient {
				log.Printf("scene %s: skipping object %q: %v", s.Name, objConfig.Name, err)
				continue
			}
			return fmt.Errorf("failed to add object to scene: %w", err)
		}
	}

	return nil
}

// LoadFromFile loads a scene from a JSON file.
func (s *Scene) LoadFromFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read scene file %s: %w", path, err)
	}
	return s.LoadFromJSON(data)
}

// createObjectFromSceneObject creates an Object from a SceneObject configuration.
// When fsys is non-nil, object template files are resolved through it. When lenient
// is true, a component whose kind isn't registered (or whose args fail to decode)
// is skipped with a log instead of failing the whole object.
func createObjectFromSceneObject(objConfig corejson.SceneObject, fsys fs.FS, lenient bool) (*Object, error) {
	var obj *Object

	// Case 1: File reference with transform override
	if objConfig.File != "" {
		// Load object template from file (or embedded filesystem).
		var objConfigFile *corejson.ObjectConfig
		var err error
		if fsys != nil {
			objConfigFile, err = corejson.LoadObjectConfigFS(fsys, objConfig.File)
		} else {
			objConfigFile, err = corejson.LoadObjectConfig(objConfig.File)
		}
		if err != nil {
			return nil, fmt.Errorf("failed to load object template %s: %w", objConfig.File, err)
		}

		// Create object from template
		obj = NewObject(objConfigFile.Name)

		// Set depth from template if specified
		if objConfigFile.Depth != 0 {
			obj.SetDepth(objConfigFile.Depth)
		}
		obj.Layer = objConfigFile.Layer
		obj.UI = objConfigFile.UI || objConfig.UI
		obj.Draggable = objConfigFile.Draggable || objConfig.Draggable

		// Add components from template
		for _, compConfig := range objConfigFile.Components {
			component, err := CreateComponentFromJSON(compConfig.Kind, compConfig.Name, compConfig.Args)
			if err != nil {
				if lenient {
					log.Printf("scene: skipping component %q on %q: %v", compConfig.Name, objConfig.Name, err)
					continue
				}
				return nil, fmt.Errorf("failed to create component %s: %w", compConfig.Kind, err)
			}
			if err := obj.AddComponent(component); err != nil {
				return nil, fmt.Errorf("failed to add component %s: %w", compConfig.Name, err)
			}
		}

		// Add tags from template
		for _, tag := range objConfigFile.Tags {
			obj.AddTag(tag)
		}

		// Apply transform override if provided
		if objConfig.Transform != nil {
			obj.Transform.Position = objConfig.Transform.Position
			if objConfig.Transform.Rotation != 0 {
				obj.Transform.Rotation = objConfig.Transform.Rotation
			}
			if objConfig.Transform.Scale.X != 0 || objConfig.Transform.Scale.Y != 0 {
				obj.Transform.Scale = objConfig.Transform.Scale
			}
		}

		// Depth override from scene (if specified)
		if objConfig.Depth != 0 {
			obj.SetDepth(objConfig.Depth)
		}
		// Layer override from scene (if specified)
		if objConfig.Layer != 0 {
			obj.SetLayer(objConfig.Layer)
		}

		return obj, nil
	}

	// Case 2: Inline object definition (no file reference)
	// Validate inline definition
	if objConfig.Name == "" {
		return nil, fmt.Errorf("inline object must have a name")
	}

	obj = NewObject(objConfig.Name)
	obj.UI = objConfig.UI
	obj.Draggable = objConfig.Draggable
	obj.Layer = objConfig.Layer

	// Add components
	for _, compConfig := range objConfig.Components {
		component, err := CreateComponentFromJSON(compConfig.Kind, compConfig.Name, compConfig.Args)
		if err != nil {
			if lenient {
				log.Printf("scene: skipping component %q on %q: %v", compConfig.Name, objConfig.Name, err)
				continue
			}
			return nil, fmt.Errorf("failed to create component %s: %w", compConfig.Kind, err)
		}
		if err := obj.AddComponent(component); err != nil {
			return nil, fmt.Errorf("failed to add component %s: %w", compConfig.Name, err)
		}
	}

	// Add tags
	for _, tag := range objConfig.Tags {
		obj.AddTag(tag)
	}

	// Apply transform if provided
	if objConfig.Transform != nil {
		obj.Transform.Position = objConfig.Transform.Position
		if objConfig.Transform.Rotation != 0 {
			obj.Transform.Rotation = objConfig.Transform.Rotation
		}
		if objConfig.Transform.Scale.X != 0 || objConfig.Transform.Scale.Y != 0 {
			obj.Transform.Scale = objConfig.Transform.Scale
		}
	}

	// Set depth if specified
	if objConfig.Depth != 0 {
		obj.SetDepth(objConfig.Depth)
	}

	return obj, nil
}

// SaveToJSON serializes the scene to indented JSON.
func (s *Scene) SaveToJSON() ([]byte, error) {
	return json.MarshalIndent(s.ToJSONConfig(), "", "  ")
}

// ToJSONConfig converts the scene to its JSON configuration. Objects are serialized
// inline (an object that was originally referenced from a .obj template file is
// written out with its components in full), which preserves its current state but
// flattens the file reference.
func (s *Scene) ToJSONConfig() *corejson.SceneConfig {
	config := &corejson.SceneConfig{
		Name:            s.Name,
		BackgroundColor: s.BackgroundColor.HexString(),
	}

	if s.Camera != nil {
		config.Camera = &corejson.CameraConfig{
			X:         s.Camera.X,
			Y:         s.Camera.Y,
			Zoom:      s.Camera.Zoom,
			Smoothing: s.Camera.Smoothing,
			LockX:     s.Camera.LockX,
			LockY:     s.Camera.LockY,
		}
	}

	// Objects in sorted order (deterministic, matching draw order).
	for _, id := range s.SortedObjects {
		obj := s.Objects[id]
		if obj == nil {
			continue
		}
		config.Objects = append(config.Objects, toSceneObject(obj))
	}

	return config
}

// SaveToFile serializes the scene and writes it to a JSON file, creating the parent
// directory if needed.
func (s *Scene) SaveToFile(path string) error {
	data, err := s.SaveToJSON()
	if err != nil {
		return fmt.Errorf("failed to marshal scene to JSON: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("failed to create directory for %s: %w", path, err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write scene file %s: %w", path, err)
	}
	return nil
}

// toSceneObject converts an object to an inline SceneObject configuration. The
// transform is omitted entirely when it is the identity (no position, rotation, or
// scale), matching how hand-written scenes leave the transform field off for
// untransformed objects.
func toSceneObject(obj *Object) corejson.SceneObject {
	so := corejson.SceneObject{
		Name:      obj.Name,
		Depth:     obj.Depth,
		Layer:     obj.Layer,
		UI:        obj.UI,
		Draggable: obj.Draggable,
	}

	t := obj.Transform
	if t.Position != math.Zero() || t.Rotation != 0 || t.Scale != math.One() {
		so.Transform = &corejson.TransformConfig{
			Position: t.Position,
			Rotation: t.Rotation,
			Scale:    t.Scale,
		}
	}

	for _, component := range obj.orderedComponents() {
		so.Components = append(so.Components, corejson.ComponentInstanceConfig{
			Kind: component.GetKind(),
			Name: component.GetName(),
			Args: ComponentArgs(component),
		})
	}

	// Tags sorted for deterministic output.
	tags := make([]string, 0, len(obj.Tags))
	for tag := range obj.Tags {
		tags = append(tags, tag)
	}
	sort.Strings(tags)
	so.Tags = tags

	return so
}

// InstantiateFromTemplate creates an object from a template file and adds it to the scene.
// Returns the created object or error.
func (s *Scene) InstantiateFromTemplate(templatePath string, transform *math.Transform) (*Object, error) {
	// Load object from template file
	obj, err := LoadObjectFromFile(templatePath)
	if err != nil {
		return nil, fmt.Errorf("failed to load template %s: %w", templatePath, err)
	}

	// Apply transform if provided
	if transform != nil {
		obj.Transform = *transform
	}

	// Add object to scene
	if err := s.AddObject(obj); err != nil {
		return nil, fmt.Errorf("failed to add object to scene: %w", err)
	}

	return obj, nil
}

// InstantiateObject creates an object from JSON data and adds it to the scene.
// Useful for runtime object creation from component scripts.
func (s *Scene) InstantiateObject(data []byte, transform *math.Transform) (*Object, error) {
	// Load object from JSON
	obj, err := LoadObjectFromJSON(data)
	if err != nil {
		return nil, fmt.Errorf("failed to load object from JSON: %w", err)
	}

	// Apply transform if provided
	if transform != nil {
		obj.Transform = *transform
	}

	// Add object to scene
	if err := s.AddObject(obj); err != nil {
		return nil, fmt.Errorf("failed to add object to scene: %w", err)
	}

	return obj, nil
}
