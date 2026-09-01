package core

import "github.com/EnesBaytekin/imge/core/math"

// DebugInfo is passed to a component's DrawDebug so it can render its debug
// overlay according to the editor's current state.
type DebugInfo struct {
	// Selected reports whether the component (or its owner) is the current
	// selection, so it can draw its bounds in a highlighted color.
	Selected bool
}

// DebugDrawer is an optional interface a component may implement to draw a debug
// overlay — hitboxes, bounds, anchors — on top of the finished frame. Unlike Draw,
// which runs in object order, DrawDebug is a final pass the Scene runs after every
// normal draw, so debug visuals always sit on top of the rendered game regardless
// of depth.
//
// The overlay is only rendered when the scene has debug drawing enabled (see
// Scene.SetDebugDraw / the imge build --debug flag). It is off by default in a
// normal build.
type DebugDrawer interface {
	DrawDebug(r Renderer, info DebugInfo)
}

// DebugBoundsProvider is an optional interface a component may implement to report
// a screen-space rectangle for editor hit-testing and selection: the editor uses it
// to pick objects in the viewport by their debug bounds. A component that draws a
// debug box should also report the same rectangle here so click-selection matches
// what the user sees.
type DebugBoundsProvider interface {
	DebugBounds() math.Rect
}
