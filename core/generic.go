package core

// GenericComponent stands in for a component whose kind is not registered in the
// current binary (e.g. a custom project component an editor cannot instantiate
// because it was not compiled in). It carries only the kind, name, and raw args —
// enough for a scene to round-trip the component unchanged, so the binary that
// *does* register the kind (the actual game) instantiates the real component with
// its real behavior. It draws nothing and updates nothing.
type GenericComponent struct {
	BaseComponent
	rawArgs map[string]interface{}
}

// NewGenericComponent creates a placeholder component for an unregistered kind.
// args is stored verbatim (already in JSON form — hex strings, numbers, bools,
// nested maps) so ComponentArgs can serialize it back identically via RawArgs.
func NewGenericComponent(kind, name string, args map[string]interface{}) *GenericComponent {
	g := &GenericComponent{}
	g.SetKind(kind)
	g.SetName(name)
	if args == nil {
		args = map[string]interface{}{}
	}
	g.rawArgs = args
	return g
}

// RawArgs returns the component's verbatim args map, for round-tripping through
// ComponentArgs without reflection (see RawArgsProvider).
func (g *GenericComponent) RawArgs() map[string]interface{} {
	return g.rawArgs
}
