package components

import (
	"strings"

	"github.com/EnesBaytekin/imge/core"
)

// Transition is a single rule for leaving a state. It fires when the machine is
// in the owning state and the given event occurs.
type Transition struct {
	// Event is the event name that triggers this transition.
	Event string `json:"event"`

	// From scopes which event source may trigger this transition:
	//
	//	""                -> manual only: reachable only via Trigger(event)
	//	"component"       -> the named component on this same object
	//	"object.component"-> the named component on the named object
	//	"scene"           -> any component in the scene (source ignored)
	From string `json:"from"`

	// To is the state to enter when the transition fires.
	To string `json:"to"`

	// Delay, in seconds, before the transition actually runs after being
	// triggered (0 = immediate). A new transition or SetState cancels a pending
	// delayed one.
	Delay float64 `json:"delay"`
}

// State is a named machine state and its outgoing transitions.
type State struct {
	Name        string       `json:"name"`
	Transitions []Transition `json:"transitions,omitempty"`
}

// StateMachine drives an object's behavior through a set of named states. Each
// state lists transitions: when a transition's event occurs (from an allowed
// source), the machine switches to the transition's "to" state.
//
// It emits "state_entered" (Data = new state name) and "state_exited" (Data =
// previous state name) on every change, so other components can react to state
// changes without polling.
//
// Export variables (JSON args): initial, states [{name, transitions
// [{event, from, to, delay}]}].
type StateMachine struct {
	core.BaseComponent

	Initial string  `json:"initial"`
	States  []State `json:"states"`

	states       map[string]State
	current      string
	previous     string
	timeInState  float64
	enteredFrame uint64
	pending      *pendingTransition
}

// pendingTransition is a delayed transition waiting to fire.
type pendingTransition struct {
	to    string
	timer float64
}

// Initialize builds the state lookup and enters the initial state.
func (sm *StateMachine) Initialize() {
	sm.stateMap()

	if sm.current == "" {
		initial := sm.Initial
		if initial == "" && len(sm.States) > 0 {
			initial = sm.States[0].Name
		}
		if initial != "" {
			sm.current = initial
			sm.enteredFrame = sm.frame()
		}
	}
}

// Update advances time-in-state and fires any pending delayed transition.
func (sm *StateMachine) Update(ctx *core.Context) {
	sm.timeInState += ctx.DeltaTime()

	if sm.pending != nil {
		sm.pending.timer -= ctx.DeltaTime()
		if sm.pending.timer <= 0 {
			to := sm.pending.to
			sm.pending = nil
			sm.setState(to, false)
		}
	}
}

// EventNames returns the event names this machine auto-listens for: the event
// names of transitions that declare a "from" scope. Called by the event manager
// after Initialize.
func (sm *StateMachine) EventNames() []string {
	seen := make(map[string]bool)
	var names []string
	for _, st := range sm.States {
		for _, tr := range st.Transitions {
			if tr.From == "" || tr.Event == "" {
				continue
			}
			if !seen[tr.Event] {
				seen[tr.Event] = true
				names = append(names, tr.Event)
			}
		}
	}
	return names
}

// HandleEvent delivers a scene event and fires any transition whose event name
// and "from" scope match. This replaces BaseComponent.HandleEvent, so the state
// machine matches by source scope rather than the plain On() handler map.
func (sm *StateMachine) HandleEvent(event *core.Event) {
	sm.dispatch(event.Name, event.Source, true)
}

// Trigger manually fires an event on the current state, ignoring "from" scopes.
// It reaches both manual (from "") and auto transitions.
func (sm *StateMachine) Trigger(event string) {
	sm.dispatch(event, nil, false)
}

// dispatch runs the matching transitions of the current state. When duringEvent
// is true, only transitions whose "from" scope matches the source fire.
func (sm *StateMachine) dispatch(eventName string, source core.Component, duringEvent bool) {
	st, ok := sm.stateMap()[sm.current]
	if !ok {
		return
	}
	for _, tr := range st.Transitions {
		if tr.Event == "" || tr.Event != eventName {
			continue
		}
		if duringEvent {
			if tr.From == "" || !matchSource(tr.From, source, sm.GetOwner()) {
				continue
			}
		}
		sm.fire(tr, duringEvent)
	}
}

// fire schedules a delayed transition or performs an immediate one.
func (sm *StateMachine) fire(tr Transition, duringEvent bool) {
	if tr.Delay > 0 {
		sm.pending = &pendingTransition{to: tr.To, timer: tr.Delay}
		return
	}
	sm.setState(tr.To, duringEvent)
}

// SetState forces the machine into the given state (must be a declared state).
func (sm *StateMachine) SetState(name string) {
	sm.setState(name, false)
}

// setState performs a state change and emits the enter/exit events. When
// duringEvent is true the change happens during event processing (after this
// frame's Updates), so the "just entered" frame is advanced by one.
func (sm *StateMachine) setState(name string, duringEvent bool) {
	if name == "" || name == sm.current {
		return
	}
	if _, ok := sm.stateMap()[name]; !ok {
		return
	}

	sm.previous = sm.current
	sm.current = name
	sm.timeInState = 0
	sm.enteredFrame = sm.frame()
	if duringEvent {
		sm.enteredFrame++
	}
	sm.pending = nil

	if sm.previous != "" {
		sm.Emit("state_exited", sm.previous)
	}
	sm.Emit("state_entered", sm.current)
}

// Current returns the current state name ("" before the machine is initialized).
func (sm *StateMachine) Current() string { return sm.current }

// Previous returns the state before the current one ("" on the first state).
func (sm *StateMachine) Previous() string { return sm.previous }

// TimeInState returns how many seconds the machine has been in the current state.
func (sm *StateMachine) TimeInState() float64 { return sm.timeInState }

// JustEntered reports whether the current state was entered during this update
// cycle. It is true once after each transition (and for the initial state).
func (sm *StateMachine) JustEntered() bool {
	return sm.enteredFrame != 0 && sm.enteredFrame == sm.frame()
}

// stateMap lazily builds the name -> State lookup from the JSON config.
func (sm *StateMachine) stateMap() map[string]State {
	if sm.states == nil {
		sm.states = make(map[string]State, len(sm.States))
		for _, st := range sm.States {
			sm.states[st.Name] = st
		}
	}
	return sm.states
}

// frame returns the scene's current update frame count (0 when not in a scene).
func (sm *StateMachine) frame() uint64 {
	owner := sm.GetOwner()
	if owner == nil || owner.Scene == nil {
		return 0
	}
	return owner.Scene.FrameNumber()
}

// matchSource reports whether an event emitted by source matches a "from" scope
// string. The reserved scope "scene" matches any source.
func matchSource(from string, source core.Component, owner *core.Object) bool {
	if from == "scene" {
		return true
	}
	if source == nil {
		return false
	}
	if i := strings.IndexByte(from, '.'); i >= 0 {
		objName := from[:i]
		compName := from[i+1:]
		srcOwner := source.GetOwner()
		return srcOwner != nil && srcOwner.Name == objName && source.GetName() == compName
	}
	return source.GetOwner() == owner && source.GetName() == from
}
