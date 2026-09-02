package components

import (
	"fmt"

	"github.com/EnesBaytekin/imge/core"
	"github.com/EnesBaytekin/imge/core/math"
)

// EditorLayoutComponent anchors the editor's fixed panels to the window edges and
// reflows them every frame from the current viewport size. It is what lets the
// editor stay correct when the window is resizable (game.imge `resizable: true`):
// rather than the panels holding hard-coded 960x540 coordinates, they are sized
// relative to the screen — the header and toolbar span the full width, the tree and
// inspector hug the left/right edges full-height, the console hugs the bottom, and
// the viewport fills the remainder. It draws nothing; it only positions its sibling
// panels.
type EditorLayoutComponent struct {
	core.BaseComponent

	// HeaderH is the header strip height (the title bar).
	HeaderH float64 `json:"header_h"`
	// ToolbarH is the toolbar strip height below the header.
	ToolbarH float64 `json:"toolbar_h"`
	// SidebarW is the width of the left (scene tree) and right (inspector) columns.
	SidebarW float64 `json:"sidebar_w"`
	// ConsoleH is the console strip height at the bottom.
	ConsoleH float64 `json:"console_h"`

	// fps/fpsText/fw are the frame-rate readout, recomputed in Update and drawn in
	// Draw. They are a diagnostic overlay, not serialized state.
	fps     float64
	fpsText string
	fw      float64
}

func (c *EditorLayoutComponent) Initialize() {
	if c.HeaderH <= 0 {
		c.HeaderH = 24
	}
	if c.ToolbarH <= 0 {
		c.ToolbarH = 22
	}
	if c.SidebarW <= 0 {
		c.SidebarW = 180
	}
	if c.ConsoleH <= 0 {
		c.ConsoleH = 90
	}
}

func (c *EditorLayoutComponent) Update(ctx *core.Context) {
	if ctx == nil || ctx.Renderer == nil || ctx.Time == nil {
		return
	}
	scene := c.GetScene()
	if scene == nil {
		return
	}
	w, h := ctx.Renderer.GetViewportSize()
	fw, fh := float64(w), float64(h)
	if fw <= 0 || fh <= 0 {
		return
	}

	top := c.HeaderH + c.ToolbarH
	midW := fw - 2*c.SidebarW
	if midW < 0 {
		midW = 0
	}
	midH := fh - top - c.ConsoleH
	if midH < 0 {
		midH = 0
	}

	// Edge-anchored: header and toolbar span the top; the tree/inspector hug the
	// left/right edges full-height below them; the console hugs the bottom; the
	// viewport fills the middle. Component names match editor.scene.
	setPanel(scene, "header", "bar", 0, 0, fw, c.HeaderH)
	setPanel(scene, "toolbar", "toolbar", 0, c.HeaderH, fw, c.ToolbarH)
	setPanel(scene, "scene_tree", "tree", 0, top, c.SidebarW, fh-top)
	setPanel(scene, "inspector", "inspector", fw-c.SidebarW, top, c.SidebarW, fh-top)
	setPanel(scene, "console", "console", c.SidebarW, fh-c.ConsoleH, midW, c.ConsoleH)
	setPanel(scene, "viewport", "viewport", c.SidebarW, top, midW, midH)

	// Frame-rate readout for diagnosing input latency. Raw 1/dt so an intermittent
	// stall shows up as a low number instead of being smoothed away.
	c.fw = fw
	if dt := ctx.DeltaTime(); dt > 0 {
		c.fps = 1 / dt
	}
	c.fpsText = fmt.Sprintf("%d fps", int(c.fps+0.5))
}

// Draw renders the frame-rate readout in the header's top-right corner. The layout
// object is at layer 4 (above the panels), so this overlay stays visible.
func (c *EditorLayoutComponent) Draw(r core.Renderer) {
	if c.fpsText == "" {
		return
	}
	r.DrawText(c.fpsText, "", 6, math.NewVector2(c.fw-64, 6), math.NewColor(0x8a, 0x8a, 0x9a, 0xff))
}

// sizedUI is the minimal surface the layout needs from a panel component: the
// BaseUIComponent.SetSize method every UI component inherits.
type sizedUI interface {
	SetSize(width, height float64)
}

// setPanel positions an editor panel object at (x, y) and sizes its named UI
// component to w x h. Both are in logical units. A missing object/component is a
// no-op, so a partially-authored scene still lays out the panels it has.
func setPanel(scene *core.Scene, objName, compName string, x, y, w, h float64) {
	obj := scene.GetObjectByName(objName)
	if obj == nil {
		return
	}
	obj.SetPosition(x, y)
	if comp, ok := core.GetFromNamed[core.Component](obj, compName).(sizedUI); ok {
		comp.SetSize(w, h)
	}
}
