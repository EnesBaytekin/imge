package components

import (
	"strings"

	"github.com/EnesBaytekin/imge/core"
	"github.com/EnesBaytekin/imge/core/math"
)

// Container is a layout component: it positions a set of sibling UI components on
// its own owner object, so a panel or toolbar can be arranged without computing
// every child's offset by hand.
//
// Children are addressed by name (the flat-scene model has no parent/child), in the
// order given. A missing or invisible child is skipped (and its slot collapsed), so
// toggling a child's visibility reflows the layout. The container itself draws
// nothing and is transparent to the pointer — it only assigns offsets.
//
// Layouts:
//   - "vertical"   (default) — stack children top to bottom, advancing by height.
//   - "horizontal" — lay children left to right, advancing by width.
//
// Children keep their own Width/Height; the container only moves them. Grid is not
// yet implemented.
//
// Export variables (JSON args): layout, padding {left, top, right, bottom}, gap,
// children, offset, width, height, visible, group, draw_layer.
type ContainerComponent struct {
	core.BaseUIComponent

	// Layout is "vertical" or "horizontal" (default "vertical").
	Layout string `json:"layout"`

	// Padding insets the children from the container's top-left. Only Left and Top
	// are used by the layout; Right and Bottom are reserved.
	Padding math.Border `json:"padding"`

	// Gap is the spacing between adjacent children, in logical units.
	Gap float64 `json:"gap"`

	// Children is the ordered list of sibling component names to lay out.
	Children []string `json:"children"`
}

// Initialize defaults the layout to vertical.
func (c *ContainerComponent) Initialize() {
	if strings.TrimSpace(c.Layout) == "" {
		c.Layout = "vertical"
	}
}

// Update repositions the children every frame. Layout is cheap arithmetic over a
// handful of siblings and keeps the arrangement correct when children are added,
// removed, shown, or hidden.
func (c *ContainerComponent) Update(ctx *core.Context) {
	c.layoutChildren()
}

// layoutChildren assigns each visible child's offset in layout order.
func (c *ContainerComponent) layoutChildren() {
	origin := c.Offset.Add(math.NewVector2(c.Padding.Left, c.Padding.Top))
	x, y := origin.X, origin.Y
	for _, name := range c.Children {
		child := c.child(name)
		if child == nil {
			continue
		}
		child.SetOffset(math.NewVector2(x, y))
		switch c.layout() {
		case "horizontal":
			x += child.Rect().Width() + c.Gap
		default:
			y += child.Rect().Height() + c.Gap
		}
	}
}

// child returns the named sibling as a layoutable element, or nil when missing or
// hidden.
func (c *ContainerComponent) child(name string) layoutChild {
	obj := c.GetOwner()
	if obj == nil {
		return nil
	}
	comp := obj.GetComponent(name)
	if comp == nil {
		return nil
	}
	child, ok := comp.(layoutChild)
	if !ok || !child.IsVisible() {
		return nil
	}
	return child
}

// layout returns the normalized layout name ("vertical" unless "horizontal").
func (c *ContainerComponent) layout() string {
	if strings.EqualFold(strings.TrimSpace(c.Layout), "horizontal") {
		return "horizontal"
	}
	return "vertical"
}

// layoutChild is the shape a sibling component must satisfy to be laid out: it can
// report its position/size and visibility, and accept a new offset. Any component
// embedding core.BaseUIComponent satisfies it.
type layoutChild interface {
	Rect() math.Rect
	IsVisible() bool
	SetOffset(math.Vector2)
}
