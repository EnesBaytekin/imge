// Package core contains platform-agnostic game engine logic.
// This file defines the Object system - the fundamental entity in the game world.
package core

import (
	"fmt"

	"github.com/EnesBaytekin/imge/internal/core/math"
)

// ============================================================================
// Object Definition
// ============================================================================

// Object represents an entity in the game world.
// Objects are composed of components and can be positioned, rotated, and scaled.
type Object struct {
	// ID is a unique integer identifier within the scene (runtime-generated)
	ID uint64

	// Name is a unique human-readable identifier within the scene (auto-generated if duplicate)
	Name string

	// Components stores all components attached to this object.
	// Key: component name (unique within object), Value: component instance
	Components map[string]Component

	// Tags is a set of tags assigned to this object (for quick filtering)
	Tags map[string]bool

	// Transform defines the object's position, rotation, and scale in world space
	Transform math.Transform

	// Depth determines drawing order (higher depth = drawn last/on top)
	Depth float64

	// Active controls whether the object is updated and drawn
	Active bool

	// Scene is a reference to the parent scene (set when added to a scene)
	Scene *Scene

	// destroyed marks the object for destruction (will be removed at end of frame)
	destroyed bool
}

// ============================================================================
// Object Creation
// ============================================================================

// NewObject creates a new object with default values.
// Note: ID must be set by the scene when adding the object.
func NewObject(name string) *Object {
	return &Object{
		ID:         0, // Will be set by scene
		Name:       name,
		Components: make(map[string]Component),
		Tags:       make(map[string]bool),
		Transform:  math.NewTransform(),
		Depth:      0,
		Active:     true,
		Scene:      nil,
		destroyed:  false,
	}
}

// NewObjectWithTransform creates a new object with a specific transform.
func NewObjectWithTransform(name string, transform math.Transform) *Object {
	obj := NewObject(name)
	obj.Transform = transform
	return obj
}

// ============================================================================
// ID and Name Management
// ============================================================================

// GetID returns the object's unique integer ID.
func (obj *Object) GetID() uint64 {
	return obj.ID
}

// SetID sets the object's unique integer ID.
// Should only be called by the scene when adding the object.
func (obj *Object) SetID(id uint64) {
	obj.ID = id
}

// GetName returns the object's human-readable name.
func (obj *Object) GetName() string {
	return obj.Name
}

// SetName sets the object's name and updates scene mapping if in a scene.
func (obj *Object) SetName(name string) error {
	if obj.Scene != nil {
		// Scene will handle name uniqueness and update mapping
		return obj.Scene.renameObject(obj.ID, name)
	}
	obj.Name = name
	return nil
}

// ============================================================================
// Component Management
// ============================================================================

// AddComponent adds a component to the object.
// Returns an error if a component with the same name already exists.
func (obj *Object) AddComponent(component Component) error {
	name := component.GetName()
	if name == "" {
		return fmt.Errorf("component must have a name")
	}

	if _, exists := obj.Components[name]; exists {
		return fmt.Errorf("component with name '%s' already exists", name)
	}

	// Set the component's owner
	component.SetOwner(obj)

	// Store the component
	obj.Components[name] = component

	// Call OnEnable if the object is active
	if obj.Active {
		component.OnEnable()
	}

	return nil
}

// AddComponentFromKind creates and adds a component from a kind identifier and args.
func (obj *Object) AddComponentFromKind(kind string, args []interface{}) error {
	component, err := CreateComponent(kind, args)
	if err != nil {
		return fmt.Errorf("failed to create component from kind %s: %w", kind, err)
	}

	return obj.AddComponent(component)
}

// GetComponent retrieves a component by name (O(1) lookup).
// Returns nil if the component doesn't exist.
func (obj *Object) GetComponent(name string) Component {
	return obj.Components[name]
}

// GetComponentsByKind retrieves all components of a specific kind (O(n) search).
func (obj *Object) GetComponentsByKind(kind string) []Component {
	var result []Component
	for _, component := range obj.Components {
		if component.GetKind() == kind {
			result = append(result, component)
		}
	}
	return result
}

// RemoveComponent removes a component by name.
func (obj *Object) RemoveComponent(name string) {
	component, exists := obj.Components[name]
	if !exists {
		return
	}

	// Call OnDisable if the object is active
	if obj.Active {
		component.OnDisable()
	}

	delete(obj.Components, name)
}

// ============================================================================
// Tag Management
// ============================================================================

// AddTag adds a tag to the object.
// Also updates the scene's tag mapping if the object is in a scene.
func (obj *Object) AddTag(tag string) {
	if obj.Tags[tag] {
		return // Tag already exists
	}

	obj.Tags[tag] = true

	// Update scene tag mapping if we're in a scene
	if obj.Scene != nil {
		obj.Scene.addObjectToTag(obj.ID, tag)
	}
}

// RemoveTag removes a tag from the object.
// Also updates the scene's tag mapping if the object is in a scene.
func (obj *Object) RemoveTag(tag string) {
	if !obj.Tags[tag] {
		return // Tag doesn't exist
	}

	delete(obj.Tags, tag)

	// Update scene tag mapping if we're in a scene
	if obj.Scene != nil {
		obj.Scene.removeObjectFromTag(obj.ID, tag)
	}
}

// HasTag checks if the object has a specific tag (O(1) lookup).
func (obj *Object) HasTag(tag string) bool {
	return obj.Tags[tag]
}

// ============================================================================
// Depth Management
// ============================================================================

// SetDepth sets the object's depth value and marks the scene for re-sorting.
// Returns an error if depth is NaN or Infinity.
func (obj *Object) SetDepth(depth float64) error {
	// TODO: Validate depth (not NaN, not Infinity)
	obj.Depth = depth

	// Notify scene that depth changed
	if obj.Scene != nil {
		obj.Scene.markDepthChanged(obj.ID)
	}

	return nil
}

// GetDepth returns the object's depth value.
func (obj *Object) GetDepth() float64 {
	return obj.Depth
}

// ============================================================================
// Lifecycle Methods
// ============================================================================

// Update calls Update on all components.
func (obj *Object) Update(deltaTime float64) {
	if !obj.Active || obj.destroyed {
		return
	}

	for _, component := range obj.Components {
		component.Update(deltaTime)
	}
}

// Draw calls Draw on all components.
func (obj *Object) Draw(renderer Renderer) {
	if !obj.Active || obj.destroyed {
		return
	}

	for _, component := range obj.Components {
		component.Draw(renderer)
	}
}

// Destroy marks the object for destruction.
// The object will be removed from the scene at the end of the frame.
func (obj *Object) Destroy() {
	obj.destroyed = true

	// Call OnDisable on all components
	for _, component := range obj.Components {
		component.OnDisable()
	}

	// Clear scene reference
	obj.Scene = nil
}

// IsDestroyed returns true if the object is marked for destruction.
func (obj *Object) IsDestroyed() bool {
	return obj.destroyed
}

// ============================================================================
// Transform Helpers
// ============================================================================

// SetPosition sets the object's position.
func (obj *Object) SetPosition(x, y float64) {
	obj.Transform.Position = math.NewVector2(x, y)
}

// GetPosition returns the object's position.
func (obj *Object) GetPosition() math.Vector2 {
	return obj.Transform.Position
}

// SetRotation sets the object's rotation (in radians).
func (obj *Object) SetRotation(rotation float64) {
	obj.Transform.Rotation = rotation
}

// GetRotation returns the object's rotation (in radians).
func (obj *Object) GetRotation() float64 {
	return obj.Transform.Rotation
}

// SetScale sets the object's scale factors.
func (obj *Object) SetScale(x, y float64) {
	obj.Transform.Scale = math.NewVector2(x, y)
}

// GetScale returns the object's scale factors.
func (obj *Object) GetScale() math.Vector2 {
	return obj.Transform.Scale
}