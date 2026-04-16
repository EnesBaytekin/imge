// Package core contains platform-agnostic game engine logic.
// This file defines the Component system - the building blocks of game objects.
package core

import "github.com/EnesBaytekin/imge/internal/core/math"

// ============================================================================
// Component Interface
// ============================================================================

// Component is the interface that all game components must implement.
// Both built-in and user-defined components use this same interface.
type Component interface {
	// Initialize is called when the component is created from JSON data.
	// Args can be of any type (string, number, boolean, array, object).
	Initialize(args []interface{}) error

	// Update is called every frame for logic updates.
	Update(deltaTime float64)

	// Draw is called every frame for rendering.
	Draw(renderer Renderer)

	// SetOwner sets the parent object that owns this component.
	SetOwner(obj *Object)

	// GetOwner returns the parent object that owns this component.
	GetOwner() *Object

	// OnEnable is called when the component becomes active.
	OnEnable()

	// OnDisable is called when the component becomes inactive.
	OnDisable()

	// GetName returns the component's name (unique within the object).
	GetName() string

	// GetKind returns the component's kind identifier (file path).
	// For built-in: "@Hitbox", "@Image", etc.
	// For user-defined: "scripts/custom.lua", etc.
	GetKind() string
}

// ============================================================================
// BaseComponent
// ============================================================================

// BaseComponent provides default implementations for the Component interface.
// All components should embed BaseComponent to get common functionality.
type BaseComponent struct {
	owner *Object
	name  string
	kind  string // component kind (file identifier)
}

// SetOwner sets the parent object that owns this component.
func (c *BaseComponent) SetOwner(obj *Object) {
	c.owner = obj
}

// GetOwner returns the parent object that owns this component.
func (c *BaseComponent) GetOwner() *Object {
	return c.owner
}

// SetName sets the component's name (unique within the object).
func (c *BaseComponent) SetName(name string) {
	c.name = name
}

// GetName returns the component's name.
func (c *BaseComponent) GetName() string {
	return c.name
}

// SetKind sets the component's kind identifier (file path).
func (c *BaseComponent) SetKind(kind string) {
	c.kind = kind
}

// GetKind returns the component's kind identifier.
func (c *BaseComponent) GetKind() string {
	return c.kind
}

// Update is a default empty implementation.
// Components should override this method if they need update logic.
func (c *BaseComponent) Update(deltaTime float64) {}

// Draw is a default empty implementation.
// Components should override this method if they need rendering logic.
func (c *BaseComponent) Draw(renderer Renderer) {}

// Initialize is a default empty implementation.
// Components should override this method if they need initialization from JSON args.
func (c *BaseComponent) Initialize(args []interface{}) error {
	return nil
}

// OnEnable is a default empty implementation.
// Components should override this method if they need activation logic.
func (c *BaseComponent) OnEnable() {}

// OnDisable is a default empty implementation.
// Components should override this method if they need deactivation logic.
func (c *BaseComponent) OnDisable() {}

// ============================================================================
// Component Factory and Registry
// ============================================================================

// ComponentFactory is a function that creates a component instance.
// Used for both built-in and user-defined components.
type ComponentFactory func(args []interface{}) (Component, error)

// componentRegistry stores factory functions for all components.
// Key: component file identifier (e.g., "@Hitbox", "@Image", "scripts/custom.lua")
// Value: factory function that creates the component
var componentRegistry = make(map[string]ComponentFactory)

// RegisterComponent registers a component factory.
// This should be called for all components (built-in and user-defined).
func RegisterComponent(kind string, factory ComponentFactory) {
	componentRegistry[kind] = factory
}

// UnregisterComponent removes a component factory from the registry.
func UnregisterComponent(kind string) {
	delete(componentRegistry, kind)
}

// CreateComponent creates a component from a kind identifier and args.
// Looks up the component factory in the registry.
// Returns error if the component kind is not registered.
func CreateComponent(kind string, args []interface{}) (Component, error) {
	factory, exists := componentRegistry[kind]
	if !exists {
		return nil, &ComponentError{Kind: kind, Reason: "component kind not registered"}
	}

	component, err := factory(args)
	if err != nil {
		return nil, &ComponentError{Kind: kind, Reason: "factory failed: " + err.Error()}
	}

	// Set the component's kind
	component.SetKind(kind)
	return component, nil
}

// IsComponentRegistered checks if a component kind is registered.
func IsComponentRegistered(kind string) bool {
	_, exists := componentRegistry[kind]
	return exists
}

// ============================================================================
// Component Error Handling
// ============================================================================

// ComponentError represents an error that occurred during component creation.
type ComponentError struct {
	Kind   string
	Reason string
}

func (e *ComponentError) Error() string {
	return "component error [" + e.Kind + "]: " + e.Reason
}

// ============================================================================
// Helper Functions
// ============================================================================

// GetTransform is a helper method for components to access their owner's transform.
// Returns nil if the component has no owner.
func GetTransform(component Component) *math.Transform {
	owner := component.GetOwner()
	if owner == nil {
		return nil
	}
	return &owner.Transform
}

// GetPosition is a helper method for components to access their owner's position.
// Returns (0, 0) if the component has no owner.
func GetPosition(component Component) math.Vector2 {
	owner := component.GetOwner()
	if owner == nil {
		return math.Vector2{}
	}
	return owner.Transform.Position
}

// GetDepth is a helper method for components to access their owner's depth.
// Returns 0 if the component has no owner.
func GetDepth(component Component) float64 {
	owner := component.GetOwner()
	if owner == nil {
		return 0
	}
	return owner.Depth
}

// CreateComponentFromJSON creates a component from JSON configuration.
func CreateComponentFromJSON(kind, name string, args map[string]interface{}) (Component, error) {
	// Convert map to slice for Initialize
	argSlice := []interface{}{args}
	component, err := CreateComponent(kind, argSlice)
	if err != nil {
		return nil, err
	}
	// Set component name
	component.SetName(name)
	return component, nil
}

// ResolveComponentKind resolves a component kind string.
// If kind starts with '@', it's a built-in component.
// Currently returns the kind as-is (registration handles mapping).
func ResolveComponentKind(kind string) string {
	// For now, just return the kind as-is.
	// Built-in components should be registered with '@' prefix.
	return kind
}

// ============================================================================
// Runtime Helper Functions for Components
// ============================================================================

// GetSceneFromComponent returns the scene that contains the component's owner object.
// Returns nil if the component has no owner or the owner is not in a scene.
func GetSceneFromComponent(component Component) *Scene {
	owner := component.GetOwner()
	if owner == nil {
		return nil
	}
	return owner.Scene
}

// InstantiateFromTemplateInScene is a helper for components to instantiate objects.
// If scene is nil, it tries to get the scene from the component's owner.
func InstantiateFromTemplateInScene(component Component, templatePath string, transform *math.Transform) (*Object, error) {
	var scene *Scene

	// Try to get scene from component if not provided
	if component != nil {
		scene = GetSceneFromComponent(component)
	}

	if scene == nil {
		return nil, fmt.Errorf("no scene available for instantiation")
	}

	return scene.InstantiateFromTemplate(templatePath, transform)
}