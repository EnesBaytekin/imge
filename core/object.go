// Package core contains platform-agnostic game engine logic.
// This file defines the Object system - the fundamental entity in the game world.
package core

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"sort"
	"strings"

	corejson "github.com/EnesBaytekin/imge/core/json"
	"github.com/EnesBaytekin/imge/core/math"
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

	// componentOrder records component names in insertion (JSON) order so that
	// Update/Draw/serialization are deterministic rather than map-ordered.
	componentOrder []string

	// Tags is a set of tags assigned to this object (for quick filtering)
	Tags map[string]bool

	// Transform defines the object's position, rotation, and scale in world space
	Transform math.Transform

	// Depth determines drawing order within a layer (higher depth = drawn last/on top)
	Depth float64

	// Layer is the primary drawing-order dimension: objects are sorted by layer
	// first (lower layer draws first/behind), then by depth within the layer. It
	// separates fixed chrome (e.g. an always-on-top header) from ordinary windows
	// so click-to-front can reorder windows without ever crossing a higher layer.
	Layer int

	// UI marks the object as screen-space: its position is in screen pixels, it
	// ignores the camera, and it draws after all world objects.
	UI bool

	// Draggable lets a @UIManager drag this object by its non-interactive surface
	// (the window background), used for moving UI windows. Ignored for non-UI
	// objects. Default false.
	Draggable bool

	// Active controls whether the object is updated and drawn
	Active bool

	// Scene is a reference to the parent scene (set when added to a scene)
	Scene *Scene

	// destroyed marks the object for destruction (will be removed at end of frame)
	destroyed bool

	// componentsInitialized tracks whether Initialize has run for this object's
	// components. Deferred to the first Scene.Update after the object is added so
	// Initialize sees a fully-assembled scene.
	componentsInitialized bool
}

// ============================================================================
// Object Creation
// ============================================================================

// NewObject creates a new object with default values.
// Note: ID must be set by the scene when adding the object.
func NewObject(name string) *Object {
	return &Object{
		ID:             0, // Will be set by scene
		Name:           name,
		Components:     make(map[string]Component),
		componentOrder: make([]string, 0),
		Tags:           make(map[string]bool),
		Transform:      math.NewTransform(),
		Depth:          0,
		Layer:          0,
		Active:         true,
		Scene:          nil,
		destroyed:      false,
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

// AddComponent adds a component to the object (appended to the end of the insertion
// order). Returns an error if a component with the same name already exists. The
// component's Initialize() is deferred until the object is in a scene and about to
// be updated (see initializeComponents).
func (obj *Object) AddComponent(component Component) error {
	return obj.AddComponentAt(component, len(obj.componentOrder))
}

// AddComponentAt adds a component and places it at the given insertion-order index
// (clamped to [0, len]). Components with the same draw layer keep insertion order,
// so re-inserting at a recorded index restores a removed component to its original
// place in ComponentsInDrawOrder. Returns an error if a component with the same name
// already exists. The component's Initialize() is deferred as in AddComponent.
func (obj *Object) AddComponentAt(component Component, index int) error {
	name := component.GetName()
	if name == "" {
		return fmt.Errorf("component must have a name")
	}

	if _, exists := obj.Components[name]; exists {
		return fmt.Errorf("component with name '%s' already exists", name)
	}

	// Set the component's owner.
	component.SetOwner(obj)

	// Store the component and record insertion order at the requested position.
	obj.Components[name] = component
	if index < 0 {
		index = 0
	}
	if index > len(obj.componentOrder) {
		index = len(obj.componentOrder)
	}
	obj.componentOrder = append(obj.componentOrder, "")
	copy(obj.componentOrder[index+1:], obj.componentOrder[index:])
	obj.componentOrder[index] = name

	return nil
}

// ComponentInsertionIndex returns the insertion-order index of the named component,
// or -1 when the object has no component with that name. Insertion order is the order
// ComponentsInDrawOrder lists components when their draw layers are equal.
func (obj *Object) ComponentInsertionIndex(name string) int {
	for i, n := range obj.componentOrder {
		if n == name {
			return i
		}
	}
	return -1
}

// AddComponentFromKind creates and adds a component from a kind identifier and args.
func (obj *Object) AddComponentFromKind(kind string, args map[string]interface{}) error {
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

// GetComponentByKind retrieves the first component matching the given kind, in
// insertion order (O(n) search). Returns nil if no component of that kind exists.
// Kind is the component identifier (e.g., "@Collider", "@Mover", "components/sprite.go").
func (obj *Object) GetComponentByKind(kind string) Component {
	for _, component := range obj.orderedComponents() {
		if component.GetKind() == kind {
			return component
		}
	}
	return nil
}

// GetComponentsByKind retrieves all components of a specific kind, in insertion
// order (O(n) search).
func (obj *Object) GetComponentsByKind(kind string) []Component {
	var result []Component
	for _, component := range obj.orderedComponents() {
		if component.GetKind() == kind {
			result = append(result, component)
		}
	}
	return result
}

// RemoveComponent removes a component by name.
// Unsubscribes the component from all events before removal.
func (obj *Object) RemoveComponent(name string) {
	component, exists := obj.Components[name]
	if !exists {
		return
	}

	// Call OnDisable if the object is active
	if obj.Active {
		component.OnDisable()
	}

	// Unsubscribe from all events
	if obj.Scene != nil && obj.Scene.EventManager != nil {
		obj.Scene.EventManager.UnsubscribeAll(component)
	}

	delete(obj.Components, name)
	obj.removeFromOrder(name)
}

// removeFromOrder removes a component name from the insertion-order slice.
func (obj *Object) removeFromOrder(name string) {
	for i, n := range obj.componentOrder {
		if n == name {
			obj.componentOrder = append(obj.componentOrder[:i], obj.componentOrder[i+1:]...)
			return
		}
	}
}

// orderedComponents returns the object's components in insertion order. It
// returns a snapshot so callers can safely mutate the object during iteration.
func (obj *Object) orderedComponents() []Component {
	comps := make([]Component, 0, len(obj.componentOrder))
	for _, name := range obj.componentOrder {
		if component, ok := obj.Components[name]; ok {
			comps = append(comps, component)
		}
	}
	return comps
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

// SetLayer sets the object's layer and marks the scene for re-sorting.
func (obj *Object) SetLayer(layer int) {
	obj.Layer = layer
	if obj.Scene != nil {
		obj.Scene.markDepthChanged(obj.ID)
	}
}

// GetLayer returns the object's layer.
func (obj *Object) GetLayer() int {
	return obj.Layer
}

// ============================================================================
// Lifecycle Methods
// ============================================================================

// initializeComponents runs each component's Initialize() exactly once, after the
// object is in a fully-loaded scene and before its first Update. It then syncs
// the component's On() handlers with the scene's event manager and fires OnEnable
// if the object is active.
func (obj *Object) initializeComponents() {
	if obj.componentsInitialized {
		return
	}
	obj.componentsInitialized = true

	for _, component := range obj.orderedComponents() {
		component.Initialize()

		if obj.Scene != nil && obj.Scene.EventManager != nil {
			obj.Scene.EventManager.SubscribeAll(component)
		}

		if obj.Active {
			component.OnEnable()
		}
	}
}

// Update calls Update on all components in insertion order.
func (obj *Object) Update(ctx *Context) {
	if !obj.Active || obj.destroyed {
		return
	}

	for _, component := range obj.orderedComponents() {
		component.Update(ctx)
	}
}

// Draw calls Draw on all components, ordered by draw layer (ascending; equal
// layers keep insertion order). A component that reports a clip rect (via
// ClipRectProvider) has its Draw restricted to that screen-space rect, so a
// partially-scrolled-out element is cut off instead of bleeding outside its host.
func (obj *Object) Draw(renderer Renderer) {
	if !obj.Active || obj.destroyed {
		return
	}

	for _, component := range obj.drawComponents() {
		if cp, ok := component.(ClipRectProvider); ok {
			if clip := cp.ClipRect(); clip != nil {
				renderer.SetClipRect(*clip)
				component.Draw(renderer)
				renderer.ClearClip()
				continue
			}
		}
		component.Draw(renderer)
	}
}

// ClipRectProvider is an optional interface a component may implement to have its Draw
// clipped to a screen-space rectangle by the object draw loop. ClipRect returns the
// rect to clip to, or nil for no clip. BaseUIComponent satisfies it; a component that
// draws outside its host (e.g. a popup) overrides ClipRect to return nil.
type ClipRectProvider interface {
	ClipRect() *math.Rect
}

// drawComponents returns the object's components in draw order: sorted by draw
// layer (ascending), stable so equal layers keep insertion order. Update order is
// unaffected — this ordering applies only to drawing.
func (obj *Object) drawComponents() []Component {
	comps := obj.orderedComponents()
	sort.SliceStable(comps, func(i, j int) bool {
		return drawLayer(comps[i]) < drawLayer(comps[j])
	})
	return comps
}

// ComponentsInDrawOrder returns the object's components in draw order (ascending
// draw layer, stable for equal layers) — the same order Draw renders them. A
// @UIManager uses this to hit-test elements back-to-front within an object.
func (obj *Object) ComponentsInDrawOrder() []Component {
	return obj.drawComponents()
}

// drawLayer returns a component's draw layer, defaulting to 0 for components that
// don't declare one.
func drawLayer(c Component) int {
	if p, ok := c.(DrawLayerProvider); ok {
		return p.GetDrawLayer()
	}
	return 0
}

// SetActive enables or disables the object. Toggling fires OnEnable/OnDisable on
// every component so they can react to activation changes (e.g. pause timers).
func (obj *Object) SetActive(active bool) {
	if obj.Active == active {
		return
	}
	obj.Active = active
	for _, component := range obj.Components {
		if active {
			component.OnEnable()
		} else {
			component.OnDisable()
		}
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

// ============================================================================
// JSON Serialization
// ============================================================================

// LoadFromJSON loads object data from JSON configuration.
// Note: This creates a new object from JSON data, it doesn't update an existing object.
func LoadObjectFromJSON(data []byte) (*Object, error) {
	var config corejson.ObjectConfig
	if err := json.Unmarshal(corejson.StripComments(data), &config); err != nil {
		return nil, fmt.Errorf("failed to parse object JSON: %w", err)
	}

	obj := NewObject(config.Name)
	obj.UI = config.UI
	obj.Draggable = config.Draggable
	obj.Layer = config.Layer

	// Set depth if specified
	if config.Depth != 0 {
		obj.SetDepth(config.Depth)
	}

	// Add components
	for _, compConfig := range config.Components {
		component, err := CreateComponentFromJSON(compConfig.Kind, compConfig.Name, compConfig.Args)
		if err != nil {
			return nil, fmt.Errorf("failed to create component %s: %w", compConfig.Kind, err)
		}
		if err := obj.AddComponent(component); err != nil {
			return nil, fmt.Errorf("failed to add component %s: %w", compConfig.Name, err)
		}
	}

	// Add tags
	for _, tag := range config.Tags {
		obj.AddTag(tag)
	}

	return obj, nil
}

// LoadObjectFromFile loads an object from a JSON file.
func LoadObjectFromFile(path string) (*Object, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read object file %s: %w", path, err)
	}
	return LoadObjectFromJSON(data)
}

// ToJSONConfig converts the object to JSON configuration.
// Note: Transform is not included in ObjectConfig (only in scene references).
func (obj *Object) ToJSONConfig() *corejson.ObjectConfig {
	config := &corejson.ObjectConfig{
		Name:      obj.Name,
		Depth:     obj.Depth,
		Layer:     obj.Layer,
		UI:        obj.UI,
		Draggable: obj.Draggable,
	}

	// Convert components (in deterministic insertion order).
	for _, component := range obj.orderedComponents() {
		config.Components = append(config.Components, corejson.ComponentInstanceConfig{
			Kind: component.GetKind(),
			Name: component.GetName(),
			Args: ComponentArgs(component),
		})
	}

	// Convert tags (sorted for deterministic output).
	tags := make([]string, 0, len(obj.Tags))
	for tag := range obj.Tags {
		tags = append(tags, tag)
	}
	sort.Strings(tags)
	config.Tags = tags

	return config
}

// ComponentArgs serializes a component's current exported, json-tagged fields back
// into an args map — the inverse of CreateComponent's injection step. It walks the
// component's visible fields (including those promoted from an embedded BaseComponent
// or BaseUIComponent) and writes each one under its json tag. Fields equal to their
// type's zero value are omitted, so a component that was configured with a small set
// of args and left the rest to Initialize defaults stays small; math.Color fields are
// written as hex strings to match the scene file format.
func ComponentArgs(component Component) map[string]interface{} {
	if p, ok := component.(RawArgsProvider); ok {
		return p.RawArgs()
	}
	args := make(map[string]interface{})
	if component == nil {
		return args
	}

	v := reflect.ValueOf(component)
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return args
		}
		v = v.Elem()
	}
	t := v.Type()

	for _, f := range reflect.VisibleFields(t) {
		if f.Anonymous {
			continue // the embedded base struct itself; its tagged fields are promoted below
		}
		tag := strings.Split(f.Tag.Get("json"), ",")[0]
		if tag == "" || tag == "-" {
			continue
		}
		fv := v.FieldByIndex(f.Index)
		if !fv.CanInterface() {
			continue
		}
		if fv.IsZero() {
			continue // rely on Initialize to re-apply the default on reload
		}
		if fv.Type() == reflect.TypeOf(math.Color{}) {
			args[tag] = fv.Interface().(math.Color).HexString()
			continue
		}
		args[tag] = fv.Interface()
	}

	return args
}

// SaveToJSON saves the object to JSON format.
func (obj *Object) SaveToJSON() ([]byte, error) {
	config := obj.ToJSONConfig()
	return json.MarshalIndent(config, "", "  ")
}

// SaveToFile saves the object to a JSON file.
func (obj *Object) SaveToFile(path string) error {
	data, err := obj.SaveToJSON()
	if err != nil {
		return fmt.Errorf("failed to marshal object to JSON: %w", err)
	}

	if err := os.MkdirAll(path, 0755); err != nil {
		return fmt.Errorf("failed to create directory for %s: %w", path, err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write object file %s: %w", path, err)
	}

	return nil
}
