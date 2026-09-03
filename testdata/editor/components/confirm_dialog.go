package components

import (
	"github.com/EnesBaytekin/imge/core"
	"github.com/EnesBaytekin/imge/core/math"
)

// ConfirmDialogComponent is a small modal confirmation popup: it draws a message and
// two buttons — "evet sil" (confirm) and "hayır iptal" (cancel). Confirm runs the
// onConfirm callback (captured at spawn) and closes; cancel, or a click outside the
// popup, closes without running it. It is generic: the inspector uses it to confirm a
// component removal, but any confirm/cancel action can reuse it.
//
// Like AddComponentPanelComponent it is modal: while open every other hand-rolled
// panel yields (see modalOpen) and an outside click dismisses it, finalized in Draw.
type ConfirmDialogComponent struct {
	core.BaseUIComponent

	Background  math.Color `json:"background"`
	TitleText   math.Color `json:"title_text"`
	Accent      math.Color `json:"accent"`      // confirm ("evet sil") button
	BorderColor math.Color `json:"border_color"` // cancel ("hayır iptal") button
	FontID      string     `json:"font_id"`
	FontSize    float64    `json:"font_size"`

	message   string
	onConfirm func()
	confirm   *ButtonComponent
	cancel    *ButtonComponent
	dismiss   bool
	centered  bool
}

func (c *ConfirmDialogComponent) Initialize() {
	if c.Background == (math.Color{}) {
		c.Background = math.NewColor(0x1d, 0x21, 0x30, 0xff)
	}
	if c.TitleText == (math.Color{}) {
		c.TitleText = math.NewColor(0xe6, 0xe6, 0xef, 0xff)
	}
	if c.Accent == (math.Color{}) {
		c.Accent = math.NewColor(0x8a, 0x3a, 0x3a, 0xff) // destructive: red
	}
	if c.BorderColor == (math.Color{}) {
		c.BorderColor = math.NewColor(0x3a, 0x42, 0x57, 0xff)
	}
	if c.FontSize <= 0 {
		c.FontSize = 6
	}
	if c.Blocking == nil {
		c.SetBlocking(true)
	}
}

// spawnConfirmDialog opens a confirm modal showing message. onConfirm runs when the
// "evet sil" button is clicked. It is a no-op when a modal is already open.
func spawnConfirmDialog(scene *core.Scene, message string, onConfirm func()) {
	if scene == nil || modalOpen() {
		return
	}

	obj := core.NewObject("confirm_dialog")
	obj.UI = true
	obj.Layer = 3
	obj.Transform.Position = math.NewVector2(200, 100) // centered lazily on first Update

	dlg := &ConfirmDialogComponent{}
	dlg.SetName("confirm_dialog")
	dlg.Width = 220
	dlg.Height = 72
	dlg.message = message
	dlg.onConfirm = onConfirm
	obj.AddComponent(dlg)

	if err := scene.AddObject(obj); err != nil {
		return
	}

	dlg.Initialize()
	dlg.buildWidgets()
	setModal(dlg)
	raiseToFront(scene, obj)
}

// buildWidgets creates the confirm/cancel buttons as children of the same object.
func (c *ConfirmDialogComponent) buildWidgets() {
	owner := c.GetOwner()
	c.confirm = makePanelButton(owner, "confirm", "evet sil", math.NewVector2(8, 40), 98, 20, c.FontID, c.FontSize, c.Accent)
	c.cancel = makePanelButton(owner, "cancel", "hayır iptal", math.NewVector2(112, 40), 100, 20, c.FontID, c.FontSize, c.BorderColor)
}

func (c *ConfirmDialogComponent) Update(ctx *core.Context) {
	if ctx == nil || ctx.Input == nil {
		return
	}
	c.centerOnce(ctx)

	if c.cancel != nil && c.cancel.ConsumeClick() {
		c.dismiss = true
		return
	}
	if c.confirm != nil && c.confirm.ConsumeClick() {
		if c.onConfirm != nil {
			c.onConfirm()
		}
		c.dismiss = true
		return
	}

	if modalOutsideClick(c.GetScene(), c.GetOwner(), ctx) {
		c.dismiss = true
	}
}

// centerOnce repositions the popup to the screen center on its first frame.
func (c *ConfirmDialogComponent) centerOnce(ctx *core.Context) {
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

func (c *ConfirmDialogComponent) Draw(r core.Renderer) {
	rect := c.Rect()
	if rect.Width() <= 0 || rect.Height() <= 0 {
		return
	}
	r.SetClipRect(rect)
	r.DrawRect(rect, c.Background)
	r.DrawRectOutline(rect, c.BorderColor, 1)

	r.DrawText(c.message, c.FontID, c.FontSize, math.NewVector2(rect.X()+8, rect.Y()+12), c.TitleText)

	r.ClearClip()

	if c.dismiss {
		c.close()
	}
}

// close clears the modal state and destroys the popup's object.
func (c *ConfirmDialogComponent) close() {
	clearModal()
	if owner := c.GetOwner(); owner != nil {
		owner.Destroy()
	}
}
