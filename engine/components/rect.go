package components

import (
	"github.com/EnesBaytekin/imge/core"
	"github.com/EnesBaytekin/imge/core/math"
)

// Rect draws a filled rectangle: a flat solid color by default, or a nine-sliced
// texture when texture + border are given (the corners keep their natural size, the
// center and edges stretch). An optional outline is drawn over the fill.
//
// It works in both screen space (a UI window background) and world space (a
// platform block: put it on a non-UI object).
//
// Export variables (JSON args): color, texture, border {left, top, right, bottom},
// outline_color, outline_thickness, offset, width, height, visible, group,
// draw_layer.
type RectComponent struct {
	core.BaseUIComponent

	// Color fills the rect when no texture is set.
	Color math.Color `json:"color"`

	// Texture and Border opt into nine-slice rendering. Texture is the image path;
	// Border is the slice inset in texture pixels. An empty texture means a flat
	// Color fill.
	Texture string      `json:"texture"`
	Border  math.Border `json:"border"`

	// OutlineColor draws a border stroke over the fill; a fully transparent color
	// (the default) means no outline.
	OutlineColor     math.Color `json:"outline_color"`
	OutlineThickness float64    `json:"outline_thickness"`
}

// Initialize makes the rect block pointer events by default (a window background
// occludes the elements drawn behind it). Set "blocking": false in JSON to disable.
func (p *RectComponent) Initialize() {
	if p.Blocking == nil {
		b := true
		p.Blocking = &b
	}
}

func (p *RectComponent) Draw(r core.Renderer) {
	if !p.IsVisible() {
		return
	}
	rect := p.drawRect()
	if p.Texture != "" {
		core.DrawNineSlice(r, p.Texture, p.Border, rect)
	} else {
		r.DrawRect(rect, p.Color)
	}
	if p.OutlineThickness > 0 && p.OutlineColor.A > 0 {
		r.DrawRectOutline(rect, p.OutlineColor, p.OutlineThickness)
	}
}

// drawRect returns the rect's rectangle in the current draw space: the local-space
// rect (Offset × Width×Height) for a world object, which the object transform then
// places in world space, or the screen-space rect (owner.Position + Offset) for a UI
// object.
func (p *RectComponent) drawRect() math.Rect {
	owner := p.GetOwner()
	if owner != nil && !owner.UI {
		return math.NewRect(p.Offset.X, p.Offset.Y, p.Width, p.Height)
	}
	return p.Rect()
}

// LocalBounds returns the rect's local-space rectangle — the rect the owner transform
// then scales and rotates about the object origin. Transforming its four corners through
// the owner transform yields the rect's actual on-screen quad, which the editor uses to
// draw a rotated selection outline that hugs the rect instead of its axis-aligned
// enclosing box. For a UI object it returns the screen-space rect (same as Rect).
func (p *RectComponent) LocalBounds() math.Rect {
	return p.drawRect()
}

// DebugBounds reports the rect's rectangle for editor hit-testing — the same rect
// Draw fills, mapped to world space. A rect is the visual body of most world objects,
// so this makes those objects pickable in the editor even when they have no @Collider.
// UI rects also report a bounds, but scene picking skips UI objects.
func (p *RectComponent) DebugBounds() math.Rect {
	owner := p.GetOwner()
	if owner != nil && !owner.UI {
		return owner.Transform.RectBounds(p.LocalBounds())
	}
	return p.Rect()
}

// ContainsPoint reports whether a world-space point lies inside the rect's drawn quad —
// the rotated/scaled rect — rather than its axis-aligned bounds, for precise editor
// click-selection (see core.PointPicker).
func (p *RectComponent) ContainsPoint(point math.Vector2) bool {
	return shapeContainsPoint(p.GetOwner(), p.Width, p.Height, p.Offset, point)
}
