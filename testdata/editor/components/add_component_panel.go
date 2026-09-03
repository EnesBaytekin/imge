package components

import (
	"sort"
	"strings"

	"github.com/EnesBaytekin/imge/core"
	"github.com/EnesBaytekin/imge/core/math"
)

// AddComponentPanelComponent is the modal "add component" dialog opened by the
// inspector's "+" button. It lists every available component kind — built-in and
// custom — in a @ComboBox, and adds the chosen one to the currently selected object
// when OK is clicked. Cancel (or a click outside the panel) closes it without adding.
// It is modal: while open, every other hand-rolled panel yields (see modalOpen), and
// an outside click dismisses it. Dismissal is finalized in Draw so it runs after all
// Updates, regardless of the random component update order.
type AddComponentPanelComponent struct {
	core.BaseUIComponent

	Background  math.Color `json:"background"`
	TitleText   math.Color `json:"title_text"`
	Accent      math.Color `json:"accent"`
	BorderColor math.Color `json:"border_color"`
	FontID      string     `json:"font_id"`
	FontSize    float64    `json:"font_size"`

	target   *core.Object // object the chosen component is added to
	combo    *ComboBoxComponent
	ok       *ButtonComponent
	cancel   *ButtonComponent
	dismiss  bool
	centered bool
}

// titleH returns the title-bar height.
func (c *AddComponentPanelComponent) titleH() float64 { return 18 }

func (c *AddComponentPanelComponent) Initialize() {
	if c.Background == (math.Color{}) {
		c.Background = math.NewColor(0x1d, 0x21, 0x30, 0xff)
	}
	if c.TitleText == (math.Color{}) {
		c.TitleText = math.NewColor(0xff, 0xff, 0xff, 0xff)
	}
	if c.Accent == (math.Color{}) {
		c.Accent = math.NewColor(0x2f, 0x3b, 0x54, 0xff)
	}
	if c.BorderColor == (math.Color{}) {
		c.BorderColor = math.NewColor(0x3a, 0x42, 0x57, 0xff)
	}
	if c.FontSize <= 0 {
		c.FontSize = 6
	}
	// The panel is an opaque surface: it blocks pointer events so the @UIManager
	// occludes whatever is drawn behind it.
	if c.Blocking == nil {
		c.SetBlocking(true)
	}
}

// spawnAddComponentPanel opens the add-component modal for the given target object.
// It is a no-op when there is no target, a modal is already open, or there is nothing
// to add.
func spawnAddComponentPanel(scene *core.Scene, target *core.Object) {
	if scene == nil || target == nil || modalOpen() {
		return
	}
	items := componentAddList(scene)
	if len(items) == 0 {
		return
	}

	obj := core.NewObject("add_component_panel")
	obj.UI = true
	obj.Layer = 3 // same layer as the inspector, so the modal z-orders above it
	obj.Transform.Position = math.NewVector2(160, 80) // centered lazily on first Update

	panel := &AddComponentPanelComponent{}
	panel.SetName("add_component_panel")
	panel.Width = 260
	panel.Height = 86
	panel.target = target
	obj.AddComponent(panel)

	if err := scene.AddObject(obj); err != nil {
		return
	}

	// Initialize is deferred to the first Scene.Update, but the defaults (colors,
	// fonts) are needed now to build the child widgets. Initialize is idempotent.
	panel.Initialize()
	panel.buildWidgets(items)
	setModal(panel)
	raiseToFront(scene, obj) // the modal appears on top
}

// buildWidgets creates the combobox and OK/Cancel buttons as children of the same
// object, positioned by offset. Each is initialized manually (the object's own
// Initialize already ran).
func (c *AddComponentPanelComponent) buildWidgets(items []string) {
	owner := c.GetOwner()

	combo := &ComboBoxComponent{}
	combo.Items = items
	combo.FontID = c.FontID
	combo.Size = c.FontSize
	combo.Width = 244
	combo.Height = 20
	combo.DrawLayer = 1
	combo.SetName("combo")
	combo.SetOffset(math.NewVector2(8, 26))
	owner.AddComponent(combo)
	combo.Initialize()
	c.combo = combo

	c.ok = makePanelButton(owner, "ok", "OK", math.NewVector2(8, 56), 118, 20, c.FontID, c.FontSize, c.Accent)
	c.cancel = makePanelButton(owner, "cancel", "Cancel", math.NewVector2(132, 56), 120, 20, c.FontID, c.FontSize, c.BorderColor)
}

func (c *AddComponentPanelComponent) Update(ctx *core.Context) {
	if ctx == nil || ctx.Input == nil {
		return
	}
	c.centerOnce(ctx)

	// OK/Cancel are polled (events are never delivered to dynamically-added widgets,
	// since event subscriptions are snapshotted at Initialize).
	if c.cancel != nil && c.cancel.ConsumeClick() {
		c.dismiss = true
		return
	}
	if c.ok != nil && c.ok.ConsumeClick() {
		if c.combo != nil && c.combo.GetValue() != "" {
			addComponentTo(c.target, c.combo.GetValue())
		}
		c.dismiss = true
		return
	}

	if modalOutsideClick(c.GetScene(), c.GetOwner(), ctx) {
		c.dismiss = true
	}
}

// centerOnce repositions the panel to the screen center on its first frame, once the
// viewport size is known.
func (c *AddComponentPanelComponent) centerOnce(ctx *core.Context) {
	if c.centered || c.GetOwner() == nil {
		return
	}
	c.centered = true
	if ctx.Renderer == nil {
		return
	}
	vw, vh := ctx.Renderer.GetViewportSize()
	if vw <= 0 || vh <= 0 {
		return
	}
	c.GetOwner().SetPosition((float64(vw)-c.Width)/2, (float64(vh)-c.Height)/2)
}

func (c *AddComponentPanelComponent) Draw(r core.Renderer) {
	rect := c.Rect()
	if rect.Width() <= 0 || rect.Height() <= 0 {
		return
	}
	r.SetClipRect(rect)
	r.DrawRect(rect, c.Background)
	r.DrawRectOutline(rect, c.BorderColor, 1)

	_, th := r.MeasureText("Ag", c.FontID, c.FontSize)
	r.DrawRect(math.NewRect(rect.X(), rect.Y(), rect.Width(), c.titleH()), c.Accent)
	r.DrawText("ADD COMPONENT", c.FontID, c.FontSize, math.NewVector2(rect.X()+6, rect.Y()+(c.titleH()-th)/2), c.TitleText)

	r.ClearClip()

	// Finalize dismissal after drawing (and after all Updates), so a dismiss click
	// never leaks through to a panel beneath it this frame.
	if c.dismiss {
		c.close()
	}
}

// close clears the modal state and destroys the panel's object (reaped next Update).
func (c *AddComponentPanelComponent) close() {
	clearModal()
	if owner := c.GetOwner(); owner != nil {
		owner.Destroy()
	}
}

// componentAddList builds the sorted, deduplicated list of component kinds the add
// panel offers: every @-prefixed built-in (from the engine registry) plus every
// custom project component struct (scanned from the project's components/*.go). The
// custom struct name is itself the component kind, matching how the build registers
// user components (build/registry.go).
func componentAddList(scene *core.Scene) []string {
	seen := make(map[string]bool)
	var list []string
	for _, kind := range core.ComponentKinds() {
		if strings.HasPrefix(kind, "@") {
			seen[kind] = true
			list = append(list, kind)
		}
	}
	projectDir := ""
	if vp := lookupViewport(scene); vp != nil {
		projectDir = vp.CurrentProject()
	}
	for _, name := range scanProjectComponents(projectDir) {
		if !seen[name] {
			seen[name] = true
			list = append(list, name)
		}
	}
	sort.Strings(list)
	return list
}
