package core

import "github.com/EnesBaytekin/imge/core/math"

// BaseUIComponent is the shared base for UI components (@Label, @Panel, @Button,
// @TextInput, and any custom UI component). A UI component is a screen-space
// element positioned relative to its owner object: the element's top-left is
// owner.Position + Offset, and its extent is Width×Height. The owner object is a
// "window" (UI=true); its components are that window's elements.
//
// It exposes the fields a @UIManager needs to route input and manage focus:
// Rect() for hit-testing and occlusion, IsEnabled for interaction, and Focusable
// for keyboard focus. In Faz 2 each widget drives itself; the manager (Faz 3) uses
// these to coordinate input across widgets without each widget depending on focus
// or selection state.
type BaseUIComponent struct {
	BaseComponent

	// Offset is the element's top-left position relative to the owner object.
	Offset math.Vector2 `json:"offset"`

	// Width and Height are the element's extent in logical units.
	Width  float64 `json:"width"`
	Height float64 `json:"height"`

	// Visible controls whether the element draws. nil means true.
	Visible *bool `json:"visible"`

	// Enabled controls whether the element receives input. nil means true.
	// A disabled element still draws but ignores pointer/keyboard events.
	Enabled *bool `json:"enabled"`

	// Focusable reports whether the element can take keyboard focus (e.g. a text
	// input). Default false.
	Focusable bool `json:"focusable"`

	// Blocking reports whether the element swallows pointer events: when it is the
	// topmost element under the cursor it is the exclusive target, so nothing drawn
	// behind it receives hover/click. A nil Blocking defaults to false here; the
	// built-in interactive components (@Panel/@Button/@TextInput) opt into blocking
	// in Initialize, and a JSON "blocking": true/false overrides any component.
	Blocking *bool `json:"blocking"`
}

// IsVisible reports whether the element draws.
func (c *BaseUIComponent) IsVisible() bool { return c.Visible == nil || *c.Visible }

// SetVisible sets whether the element draws.
func (c *BaseUIComponent) SetVisible(v bool) { b := v; c.Visible = &b }

// IsEnabled reports whether the element receives input.
func (c *BaseUIComponent) IsEnabled() bool { return c.Enabled == nil || *c.Enabled }

// SetEnabled sets whether the element receives input.
func (c *BaseUIComponent) SetEnabled(v bool) { b := v; c.Enabled = &b }

// IsFocusable reports whether the element can take keyboard focus.
func (c *BaseUIComponent) IsFocusable() bool { return c.Focusable }

// BlocksPointer reports whether the element swallows pointer events (occludes the
// elements drawn behind it). See the Blocking field.
func (c *BaseUIComponent) BlocksPointer() bool { return c.Blocking != nil && *c.Blocking }

// Position returns the element's top-left corner in screen space.
func (c *BaseUIComponent) Position() math.Vector2 {
	if c.owner == nil {
		return c.Offset
	}
	return c.owner.Transform.Position.Add(c.Offset)
}

// SetOffset sets the element's top-left position relative to the owner object.
// A layout container uses this to position sibling components.
func (c *BaseUIComponent) SetOffset(offset math.Vector2) { c.Offset = offset }

// Rect returns the element's screen-space rectangle: top-left at Position(), size
// Width×Height. A nil owner means the owner's position is treated as (0,0).
func (c *BaseUIComponent) Rect() math.Rect {
	p := c.Position()
	return math.NewRect(p.X, p.Y, c.Width, c.Height)
}

// Contains reports whether a point (in screen space) is inside the element.
func (c *BaseUIComponent) Contains(p math.Vector2) bool {
	return c.Rect().ContainsPoint(p)
}
