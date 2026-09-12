package components

import (
	"github.com/EnesBaytekin/imge/core"
	"github.com/EnesBaytekin/imge/core/math"
)

// Rect draws a filled rectangle in one of two modes: "color" (a flat solid fill)
// or "nine_slice" (a nine-sliced texture whose corners keep their natural size
// while the center and edges stretch). An optional outline is drawn over the fill.
//
// The active mode is chosen explicitly via Mode; both modes' data (color *and*
// texture+border) are always kept, so switching modes never discards the other's
// settings — an editor can offer Color / 9-Slice tabs without wiping values.
//
// It works in both screen space (a UI window background) and world space (a
// platform block: put it on a non-UI object).
//
// Export variables (JSON args): mode, color, texture, border {left, top, right,
// bottom}, outline_color, outline_thickness, offset, width, height, visible,
// enabled, blocking, group, draw_layer.
type RectComponent struct {
	core.BaseUIComponent

	// Mode selects the fill: "color" (the default) draws a flat Color fill, while
	// "nine_slice" draws Texture as a nine-slice using Border. An empty Mode is
	// normalized in Initialize: "nine_slice" when a Texture is set (backward
	// compatible), otherwise "color".
	Mode string `json:"mode"`

	// Color fills the rect in color mode (and when a nine-slice has no texture yet).
	Color math.Color `json:"color"`

	// Texture and Border opt into nine-slice rendering. Texture is the image path;
	// Border is the slice inset in texture pixels. They only apply when Mode is
	// "nine_slice".
	Texture string      `json:"texture"`
	Border  math.Border `json:"border"`

	// OutlineColor draws a border stroke over the fill in color mode; a fully
	// transparent color (the default) means no outline. It does not apply in
	// nine-slice mode.
	OutlineColor     math.Color `json:"outline_color"`
	OutlineThickness float64    `json:"outline_thickness"`
}

// Initialize makes the rect block pointer events by default (a window background
// occludes the elements drawn behind it). Set "blocking": false in JSON to disable.
// It also normalizes an empty Mode for backward compatibility: an old scene that
// set a texture keeps rendering nine-slice, while a blank rect defaults to color.
func (p *RectComponent) Initialize() {
	if p.Blocking == nil {
		b := true
		p.Blocking = &b
	}
	if p.Mode == "" {
		if p.Texture != "" {
			p.Mode = "nine_slice"
		} else {
			p.Mode = "color"
		}
	}
}

// IsNineSlice reports whether the rect renders in nine-slice mode.
func (p *RectComponent) IsNineSlice() bool { return p.Mode == "nine_slice" }

func (p *RectComponent) Draw(r core.Renderer) {
	if !p.IsVisible() {
		return
	}
	rect := p.drawRect()
	if p.IsNineSlice() {
		// Nine-slice mode applies only the texture+border settings shown on its tab;
		// the Color-tab settings (flat fill and outline) do not apply. An empty texture
		// still shows the flat color as a placeholder so the rect stays visible while a
		// texture is being picked.
		if p.Texture != "" {
			core.DrawNineSlice(r, p.Texture, p.Border, rect)
		} else {
			r.DrawRect(rect, p.Color)
		}
		return
	}
	// Color mode: flat fill plus the outline (a Color-tab styling concern, so it is
	// drawn only here, never in nine-slice mode).
	r.DrawRect(rect, p.Color)
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

// DrawDebug draws the rect's bounds as an outline when the scene has debug drawing
// enabled (imge build --debug). The outline is translucent cyan; the current
// selection is drawn brighter so it stands out. In nine-slice mode it additionally
// outlines each of the 9 slice cells (using the same math DrawNineSlice uses), so the
// corner/edge/center split is visible and updates live as border/width change.
func (p *RectComponent) DrawDebug(r core.Renderer, info core.DebugInfo) {
	color := math.NewColor(90, 200, 255, 170)
	if info.Selected {
		color = math.NewColor(150, 230, 255, 255)
	}

	// Draw in the space Draw uses: local space (mapped through the object transform)
	// for a world object, or screen space for a UI object — the same split as the
	// collider's hitbox.
	owner := p.GetOwner()
	rect := math.NewRect(p.Offset.X, p.Offset.Y, p.Width, p.Height)
	if owner == nil || owner.UI {
		rect = p.Rect()
	}
	r.DrawRectOutlineScreen(rect, color, 1)

	if !p.IsNineSlice() {
		return
	}
	tw, th := r.GetTextureSize(p.Texture)
	if tw <= 0 || th <= 0 {
		return
	}
	for _, s := range math.Slice9(tw, th, p.Border, rect) {
		r.DrawRectOutlineScreen(s.Dst, color, 1)
	}
}

// ContainsPoint reports whether a world-space point lies inside the rect's drawn quad —
// the rotated/scaled rect — rather than its axis-aligned bounds, for precise editor
// click-selection (see core.PointPicker).
func (p *RectComponent) ContainsPoint(point math.Vector2) bool {
	return shapeContainsPoint(p.GetOwner(), p.Width, p.Height, p.Offset, point)
}
