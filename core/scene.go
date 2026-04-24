// Package core contains platform-agnostic game engine logic.
// This file defines the Scene system - a container for game objects.
package core

import (
	"encoding/json"
	"fmt"
	"os"
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

	// Name is the scene's identifier
	Name string

	// Active controls whether the scene is updated and drawn
	Active bool

	// EventManager handles the Ping-Pong event queue and subscriptions for this scene.
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

	// Subscribe all components to events now that they have a scene
	for _, comp := range obj.Components {
		comp.SubscribeEvents()
	}

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

	depth := obj.Depth

	// Find insertion index using binary search
	insertIndex := sort.Search(len(s.SortedObjects), func(i int) bool {
		otherID := s.SortedObjects[i]
		otherObj := s.Objects[otherID]
		if otherObj == nil {
			return true // Shouldn't happen, but treat missing objects as infinite depth
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

// ============================================================================
// Lifecycle Methods
// ============================================================================

// Update calls Update on all active objects in the scene.
// After all component updates, processes the event queue (Ping -> Pong delivery).
// Depth order doesn't matter for updates.
func (s *Scene) Update(ctx *ComponentContext) {
	if !s.Active {
		return
	}

	for _, obj := range s.Objects {
		if obj.Active && !obj.IsDestroyed() {
			obj.Update(ctx)
		}
	}

	// Process Ping-Pong event queue (deliver events to subscribed components)
	s.EventManager.Process(ctx)

	// Remove destroyed objects
	s.removeDestroyedObjects()
}

// Draw calls Draw on all active objects in the scene, sorted by depth.
func (s *Scene) Draw(renderer Renderer) {
	if !s.Active {
		return
	}

	// Ensure sorted list is up-to-date
	s.updateSortedObjects()

	for _, id := range s.SortedObjects {
		obj := s.Objects[id]
		if obj != nil && obj.Active && !obj.IsDestroyed() {
			obj.Draw(renderer)
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
	var config corejson.SceneConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("failed to parse scene JSON: %w", err)
	}

	s.Name = config.Name
	// TODO: Parse background color

	// Load objects from config
	for _, objConfig := range config.Objects {
		obj, err := createObjectFromSceneObject(objConfig)
		if err != nil {
			return fmt.Errorf("failed to create object: %w", err)
		}
		if err := s.AddObject(obj); err != nil {
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
func createObjectFromSceneObject(objConfig corejson.SceneObject) (*Object, error) {
	var obj *Object

	// Case 1: File reference with transform override
	if objConfig.File != "" {
		// Load object template from file
		objConfigFile, err := corejson.LoadObjectConfig(objConfig.File)
		if err != nil {
			return nil, fmt.Errorf("failed to load object template %s: %w", objConfig.File, err)
		}

		// Create object from template
		obj = NewObject(objConfigFile.Name)

		// Set depth from template if specified
		if objConfigFile.Depth != 0 {
			obj.SetDepth(objConfigFile.Depth)
		}

		// Add components from template
		for _, compConfig := range objConfigFile.Components {
			component, err := CreateComponentFromJSON(compConfig.Kind, compConfig.Name, compConfig.Args)
			if err != nil {
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

		return obj, nil
	}

	// Case 2: Inline object definition (no file reference)
	// Validate inline definition
	if objConfig.Name == "" {
		return nil, fmt.Errorf("inline object must have a name")
	}

	obj = NewObject(objConfig.Name)

	// Add components
	for _, compConfig := range objConfig.Components {
		component, err := CreateComponentFromJSON(compConfig.Kind, compConfig.Name, compConfig.Args)
		if err != nil {
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

// SaveToJSON saves the scene to JSON format.
// TODO: Implement JSON serialization based on the defined format.
func (s *Scene) SaveToJSON() ([]byte, error) {
	return nil, fmt.Errorf("SaveToJSON not yet implemented")
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