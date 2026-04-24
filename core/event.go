// Package core contains platform-agnostic game engine logic.
// This file defines the Ping-Pong event system for inter-component communication.
package core

// ============================================================================
// Event
// ============================================================================

// Event represents a message sent between components via the Ping-Pong event system.
// Components emit events via Ping() and receive them via Pong().
type Event struct {
	// Name identifies the event type (e.g., "collision", "player_died").
	// Receiving components switch on this name in their Pong() method.
	Name string

	// Sender is the component that emitted this event.
	// Receiving components can use this to query the sender's owner, position, etc.
	Sender Component

	// Data holds arbitrary extra information associated with the event.
	// The interpretation depends on the event Name (user-defined).
	Data interface{}
}

// ============================================================================
// EventManager
// ============================================================================

// EventManager manages event subscriptions and queuing for a scene.
// Events are emitted by components via Ping(), queued, and processed after
// all component Update() calls complete for the frame.
type EventManager struct {
	// queue holds events emitted during the current frame.
	// Processed and cleared at the end of each frame.
	queue []*Event

	// subscriptions maps event name -> set of components interested in it.
	// Components register interest via SubscribeEvents() during initialization.
	subscriptions map[string]map[Component]bool
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
// Called by components via their Ping() method.
func (em *EventManager) Emit(event *Event) {
	em.queue = append(em.queue, event)
}

// Process delivers all queued events to their subscribers and clears the queue.
// ctx is passed to each subscriber's Pong method for engine service access.
// Called once per frame by Scene.Update() after all component Update() calls.
func (em *EventManager) Process(ctx *ComponentContext) {
	// Swap the queue so Pong handlers that call Ping() go into a fresh queue
	// and will be processed next frame (prevents infinite recursion).
	queue := em.queue
	em.queue = make([]*Event, 0)

	for _, event := range queue {
		em.deliver(event, ctx)
	}
}

// deliver sends one event to all subscribed components' Pong methods.
func (em *EventManager) deliver(event *Event, ctx *ComponentContext) {
	subscribers, exists := em.subscriptions[event.Name]
	if !exists {
		return
	}

	for subscriber := range subscribers {
		// Guard: ensure subscriber is still part of a live, active object.
		owner := subscriber.GetOwner()
		if owner == nil || owner.IsDestroyed() {
			continue
		}
		if !owner.Active {
			continue
		}

		subscriber.Pong(event, ctx)
	}
}
