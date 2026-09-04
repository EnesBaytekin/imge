package components

import (
	"github.com/EnesBaytekin/imge/core"
)

// EditorLayoutComponent anchors the editor's fixed panels to the window edges and
// reflows them every frame from the current viewport size. It is what lets the
// editor stay correct when the window is resizable (game.imge `resizable: true`):
// rather than the panels holding hard-coded 960x540 coordinates, they are sized
// relative to the screen — the toolbar spans the full width, the tree and inspector
// hug the left/right edges full-height, the console hugs the bottom, and the
// viewport fills the remainder. It draws nothing; it only positions its sibling
// panels.
type EditorLayoutComponent struct {
	core.BaseComponent

	// ToolbarH is the menu-bar strip height at the top.
	ToolbarH float64 `json:"toolbar_h"`
	// SidebarW is the width of the left (scene tree) and right (inspector) columns.
	SidebarW float64 `json:"sidebar_w"`
	// ConsoleH is the console strip height at the bottom.
	ConsoleH float64 `json:"console_h"`
}

func (c *EditorLayoutComponent) Initialize() {
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

	top := c.ToolbarH
	midW := fw - 2*c.SidebarW
	if midW < 0 {
		midW = 0
	}
	midH := fh - top - c.ConsoleH
	if midH < 0 {
		midH = 0
	}

	// Edge-anchored: the toolbar spans the top; the tree/inspector hug the left/right
	// edges full-height below it; the console hugs the bottom; the viewport fills the
	// middle. Component names match editor.scene.
	setPanel(scene, "toolbar", "toolbar", 0, 0, fw, c.ToolbarH)
	setPanel(scene, "scene_tree", "tree", 0, top, c.SidebarW, fh-top)
	setPanel(scene, "inspector", "inspector", fw-c.SidebarW, top, c.SidebarW, fh-top)
	setPanel(scene, "console", "console", c.SidebarW, fh-c.ConsoleH, midW, c.ConsoleH)
	setPanel(scene, "viewport", "viewport", c.SidebarW, top, midW, midH)
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
