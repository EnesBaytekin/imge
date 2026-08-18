// Package core contains platform-agnostic game engine logic.
// This file defines the Component system - the building blocks of game objects.
package core

import (
	"encoding/json"
	"fmt"

	"github.com/EnesBaytekin/imge/core/math"
)

// ============================================================================
// Component Interface
// ============================================================================

// Component is the interface that all game components must implement.
// Both built-in and user-defined components use this same interface.
//
// A component's exported, JSON-tagged fields are its "export variables": they are
// populated from the component's `args` object in .obj/.scene files. Unexported
// fields stay private to the component (local state).
type Component interface {
	// Initialize is called exactly once, after the object is in a fully-loaded
	// scene and before its first Update. This is where defaults are set and any
	// scene-dependent setup happens (c.Scene() is available here).
	Initialize()

	// Update is called every frame for logic updates.
	// ctx provides access to engine services (Input, Audio, Time, Scene, etc.)
	Update(ctx *Context)

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

	// SetName sets the component's name.
	SetName(name string)

	// GetKind returns the component's kind identifier (file path).
	// For built-in: "@Collider", "@Mover", etc.
	// For user-defined: "components/player.go", etc.
	GetKind() string

	// SetKind sets the component's kind identifier.
	SetKind(kind string)
}

// Dependable is an optional interface a component may implement to declare the
// component kinds it needs to function (e.g. @Animator requires @Sprite). The
// declaration is informational: the build tool reads it to warn when an object
// uses a component without also giving it the components it declares it needs.
type Dependable interface {
	// Requires returns the component kinds this component depends on.
	Requires() []string
}

// ============================================================================
// BaseComponent
// ============================================================================

// BaseComponent provides default implementations for the Component interface,
// plus the event helpers (On/Emit) and scene access. All components should embed
// BaseComponent to get common functionality.
type BaseComponent struct {
	owner *Object
	name  string
	kind  string // component kind (file identifier)

	// handlers maps event name -> registered handler functions, populated via On().
	handlers map[string][]func(any)
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

// Scene returns the scene that contains this component's owner, or nil if the
// owner isn't in a scene yet.
func (c *BaseComponent) Scene() *Scene {
	if c.owner == nil {
		return nil
	}
	return c.owner.Scene
}

// Initialize is a default empty implementation.
// Components should override this method if they need initialization or defaults.
func (c *BaseComponent) Initialize() {}

// Update is a default empty implementation.
// Components should override this method if they need update logic.
func (c *BaseComponent) Update(ctx *Context) {}

// Draw is a default empty implementation.
// Components should override this method if they need rendering logic.
func (c *BaseComponent) Draw(renderer Renderer) {}

// OnEnable is a default empty implementation.
// Components should override this method if they need activation logic.
func (c *BaseComponent) OnEnable() {}

// OnDisable is a default empty implementation.
// Components should override this method if they need deactivation logic.
func (c *BaseComponent) OnDisable() {}

// ============================================================================
// Event Helpers
// ============================================================================

// On registers a handler for an event name. Handlers are typically registered in
// Initialize, before the first Update. Multiple handlers may be registered for
// the same name; they run in registration order when the event is delivered.
//
//	c.On("damaged", func(data any) {
//	    amount := data.(float64)
//	    ...
//	})
func (c *BaseComponent) On(name string, handler func(any)) {
	if c.handlers == nil {
		c.handlers = make(map[string][]func(any))
	}
	c.handlers[name] = append(c.handlers[name], handler)
}

// Emit broadcasts an event to the scene's event queue. It is delivered to every
// component that registered a handler for `name` via On(), after all Update()
// calls for this frame complete.
//
//	c.Emit("damaged", 10.0)
func (c *BaseComponent) Emit(name string, data any) {
	if c.owner == nil || c.owner.Scene == nil || c.owner.Scene.EventManager == nil {
		return
	}
	c.owner.Scene.EventManager.Emit(&Event{Name: name, Data: data})
}

// EventNames returns the event names this component has handlers for.
// Used by the EventManager to sync subscriptions after Initialize.
func (c *BaseComponent) EventNames() []string {
	names := make([]string, 0, len(c.handlers))
	for name := range c.handlers {
		names = append(names, name)
	}
	return names
}

// HandleEvent delivers an event to this component's registered handlers.
// Used internally by the EventManager.
func (c *BaseComponent) HandleEvent(event *Event) {
	for _, handler := range c.handlers[event.Name] {
		handler(event.Data)
	}
}

// ============================================================================
// Component Factory and Registry
// ============================================================================

// ComponentFactory is a function that creates a new, zero-valued component
// instance. Config (args) is injected afterward by json unmarshaling into the
// component's exported fields.
type ComponentFactory func() Component

// componentRegistry stores factory functions for all components.
// Key: component kind identifier (e.g., "@Collider", "components/player.go")
// Value: factory function that creates the component
var componentRegistry = make(map[string]ComponentFactory)

// RegisterComponent registers a component factory. This is called automatically
// by the generated components/registry.go for every component in the project, so
// user component files do not need an init() of their own.
func RegisterComponent(kind string, factory ComponentFactory) {
	componentRegistry[kind] = factory
}

// UnregisterComponent removes a component factory from the registry.
func UnregisterComponent(kind string) {
	delete(componentRegistry, kind)
}

// CreateComponent creates a component from a kind identifier and its JSON args.
// It looks up the factory, constructs the component, then injects the args by
// unmarshaling them into the component's exported (json-tagged) fields.
// Returns error if the kind is not registered or the args fail to decode.
func CreateComponent(kind string, args map[string]interface{}) (Component, error) {
	factory, exists := componentRegistry[kind]
	if !exists {
		return nil, &ComponentError{Kind: kind, Reason: "component kind not registered"}
	}

	component := factory()

	if len(args) > 0 {
		data, err := json.Marshal(args)
		if err != nil {
			return nil, &ComponentError{Kind: kind, Reason: "failed to encode args: " + err.Error()}
		}
		if err := json.Unmarshal(data, component); err != nil {
			return nil, &ComponentError{Kind: kind, Reason: "failed to decode args: " + err.Error()}
		}
	}

	component.SetKind(kind)
	return component, nil
}

// CreateComponentFromJSON creates a named component from its JSON configuration.
func CreateComponentFromJSON(kind, name string, args map[string]interface{}) (Component, error) {
	component, err := CreateComponent(kind, args)
	if err != nil {
		return nil, err
	}
	component.SetName(name)
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

// GetFrom returns the first component of type T attached to obj, in insertion
// order. It lets a component reach a sibling component's methods directly:
//
//	if collider := core.GetFrom[*Collider](owner); collider != nil { ... }
//
// It returns the zero value of T (a nil pointer for pointer types) when obj is
// nil or has no component of that type.
func GetFrom[T Component](obj *Object) T {
	var zero T
	if obj == nil {
		return zero
	}
	for _, component := range obj.orderedComponents() {
		if t, ok := component.(T); ok {
			return t
		}
	}
	return zero
}

// GetAllFrom returns every component of type T attached to obj, in insertion
// order. Returns nil if obj is nil or has no component of that type.
func GetAllFrom[T Component](obj *Object) []T {
	if obj == nil {
		return nil
	}
	var result []T
	for _, component := range obj.orderedComponents() {
		if t, ok := component.(T); ok {
			result = append(result, t)
		}
	}
	return result
}

// GetFromNamed returns the component of type T attached to obj with the given
// name, or the zero value of T if no such component exists (or it is of a
// different type). Unlike GetFrom, this is a direct O(1) name lookup and ignores
// insertion order.
func GetFromNamed[T Component](obj *Object, name string) T {
	var zero T
	if obj == nil {
		return zero
	}
	if component, ok := obj.Components[name].(T); ok {
		return component
	}
	return zero
}

// GetTransform is a helper for components to access their owner's transform.
// Returns nil if the component has no owner.
func GetTransform(component Component) *math.Transform {
	owner := component.GetOwner()
	if owner == nil {
		return nil
	}
	return &owner.Transform
}

// GetPosition is a helper for components to access their owner's position.
// Returns (0, 0) if the component has no owner.
func GetPosition(component Component) math.Vector2 {
	owner := component.GetOwner()
	if owner == nil {
		return math.Vector2{}
	}
	return owner.Transform.Position
}

// GetDepth is a helper for components to access their owner's depth.
// Returns 0 if the component has no owner.
func GetDepth(component Component) float64 {
	owner := component.GetOwner()
	if owner == nil {
		return 0
	}
	return owner.Depth
}

// ResolveComponentKind resolves a component kind string.
// If kind starts with '@', it's a built-in component.
// Currently returns the kind as-is (registration handles mapping).
func ResolveComponentKind(kind string) string {
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
