// Package core contains platform-agnostic game engine logic.
// This file defines the event system for inter-component communication.
package core

// ============================================================================
// Event
// ============================================================================

// Event represents a message sent between components. Components emit events via
// BaseComponent.Emit(name, data) and receive them via BaseComponent.On(name, handler).
type Event struct {
	// Name identifies the event type (e.g., "collision", "player_died").
	Name string

	// Data holds arbitrary extra information associated with the event.
	// The interpretation depends on the event Name (user-defined).
	Data interface{}

	// Source is the component that emitted the event, or nil for engine-generated
	// events. Listeners can filter by it (e.g. a @StateMachine's "from" scope).
	Source Component
}

// ============================================================================
// EventManager
// ============================================================================

// EventManager manages event subscriptions and queuing for a scene.
// Events are emitted by components via Emit(), queued, and processed after all
// component Update() calls complete for the frame.
type EventManager struct {
	// queue holds events emitted during the current frame.
	// Processed and cleared at the end of each frame.
	queue []*Event

	// subscriptions maps event name -> set of components interested in it.
	// Populated from each component's On() handlers after Initialize.
	subscriptions map[string]map[Component]bool
}

// eventReceiver is the internal interface a component must satisfy to receive
// events. BaseComponent implements it; the EventManager uses it to discover a
// component's handler names and deliver events, without polluting the public
// Component interface with event plumbing.
type eventReceiver interface {
	EventNames() []string
	HandleEvent(*Event)
}

// NewEventManager creates a new EventManager with empty queue and subscriptions.
func NewEventManager() *EventManager {
	return &EventManager{
		queue:         make([]*Event, 0),
		subscriptions: make(map[string]map[Component]bool),
	}
}

// Subscribe registers a component's interest in an event name.
// Multiple calls with the same component+name are idempotent.
func (em *EventManager) Subscribe(component Component, eventName string) {
	if em.subscriptions[eventName] == nil {
		em.subscriptions[eventName] = make(map[Component]bool)
	}
	em.subscriptions[eventName][component] = true
}

// SubscribeAll registers a component for every event name it has On() handlers
// for. Called once per component after its Initialize runs.
func (em *EventManager) SubscribeAll(component Component) {
	receiver, ok := component.(eventReceiver)
	if !ok {
		return
	}
	for _, name := range receiver.EventNames() {
		em.Subscribe(component, name)
	}
}

// Unsubscribe removes a component's interest in an event name.
func (em *EventManager) Unsubscribe(component Component, eventName string) {
	if subscribers, exists := em.subscriptions[eventName]; exists {
		delete(subscribers, component)
		if len(subscribers) == 0 {
			delete(em.subscriptions, eventName)
		}
	}
}

// UnsubscribeAll removes a component from ALL event subscriptions.
func (em *EventManager) UnsubscribeAll(component Component) {
	for eventName, subscribers := range em.subscriptions {
		delete(subscribers, component)
		if len(subscribers) == 0 {
			delete(em.subscriptions, eventName)
		}
	}
}

// Emit adds an event to the processing queue.
// Called by components via their Emit() method.
func (em *EventManager) Emit(event *Event) {
	em.queue = append(em.queue, event)
}

// Process delivers all queued events to their subscribers and clears the queue.
// Called once per frame by Scene.Update() after all component Update() calls.
func (em *EventManager) Process() {
	// Swap the queue so handlers that call Emit() go into a fresh queue and will
	// be processed next frame (prevents infinite recursion).
	queue := em.queue
	em.queue = make([]*Event, 0)

	for _, event := range queue {
		em.deliver(event)
	}
}

// deliver sends one event to all subscribed components' handlers.
func (em *EventManager) deliver(event *Event) {
	subscribers, exists := em.subscriptions[event.Name]
	if !exists {
		return
	}

	for subscriber := range subscribers {
		// Guard: ensure subscriber is still part of a live, active object.
		owner := subscriber.GetOwner()
		if owner == nil || owner.IsDestroyed() || !owner.Active {
			continue
		}

		if receiver, ok := subscriber.(eventReceiver); ok {
			receiver.HandleEvent(event)
		}
	}
}
