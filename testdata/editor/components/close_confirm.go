package components

import (
	"github.com/EnesBaytekin/imge/core"
	"github.com/EnesBaytekin/imge/core/math"
)

// CloseConfirmDialogComponent is the modal "unsaved changes" prompt shown when the
// OS window is closed (Alt+F4 / title-bar X) while the target scene has unsaved
// edits. It offers three buttons:
//
//   - Save      — write the scene, then quit
//   - Don't Save — quit without writing
//   - Cancel    — dismiss and keep editing (an outside click does the same)
//
// It is modal like the other dialogs; while it is open every other hand-rolled
// panel yields (see modalOpen) and an outside click dismisses it, finalized in Draw.
type CloseConfirmDialogComponent struct {
	core.BaseUIComponent

	Background  math.Color `json:"background"`
	TitleText   math.Color `json:"title_text"`
	Accent      math.Color `json:"accent"`       // Save button
	WarnColor   math.Color `json:"warn_color"`   // Don't Save button
	BorderColor math.Color `json:"border_color"` // Cancel button + outline

	FontID   string  `json:"font_id"`
	FontSize float64 `json:"font_size"`

	game     *core.Game
	save     *ButtonComponent
	dontSave *ButtonComponent
	cancel   *ButtonComponent
	dismiss  bool
	centered bool
}

func (c *CloseConfirmDialogComponent) Initialize() {
	if c.Background == (math.Color{}) {
		c.Background = math.NewColor(0x1d, 0x21, 0x30, 0xff)
	}
	if c.TitleText == (math.Color{}) {
		c.TitleText = math.NewColor(0xe6, 0xe6, 0xef, 0xff)
	}
	if c.Accent == (math.Color{}) {
		c.Accent = math.NewColor(0x2f, 0x3b, 0x54, 0xff)
	}
	if c.WarnColor == (math.Color{}) {
		c.WarnColor = math.NewColor(0x8a, 0x3a, 0x3a, 0xff)
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

// spawnCloseConfirm opens the unsaved-changes prompt for game. It is a no-op when a
// modal is already open (the existing modal wins; the close request is simply
// cancelled this time).
func spawnCloseConfirm(scene *core.Scene, game *core.Game) {
	if scene == nil || modalOpen() {
		return
	}

	obj := core.NewObject("close_confirm")
	obj.UI = true
	obj.Layer = 3
	obj.Transform.Position = math.NewVector2(200, 100) // centered lazily on first Update

	dlg := &CloseConfirmDialogComponent{}
	dlg.SetName("close_confirm")
	dlg.Width = 280
	dlg.Height = 78
	dlg.game = game
	obj.AddComponent(dlg)

	if err := scene.AddObject(obj); err != nil {
		return
	}

	dlg.Initialize()
	dlg.buildWidgets()
	setModal(dlg)
	raiseToFront(scene, obj)
}

// buildWidgets creates the three buttons as children of the same object.
func (c *CloseConfirmDialogComponent) buildWidgets() {
	owner := c.GetOwner()
	const bw, gap, x = 84.0, 4.0, 8.0
	c.save = makePanelButton(owner, "save", "Save", math.NewVector2(x, 46), bw, 22, c.FontID, c.FontSize, c.Accent)
	c.dontSave = makePanelButton(owner, "dont_save", "Don't Save", math.NewVector2(x+bw+gap, 46), bw, 22, c.FontID, c.FontSize, c.WarnColor)
	c.cancel = makePanelButton(owner, "cancel", "Cancel", math.NewVector2(x+2*(bw+gap), 46), bw, 22, c.FontID, c.FontSize, c.BorderColor)
}

func (c *CloseConfirmDialogComponent) Update(ctx *core.Context) {
	if ctx == nil || ctx.Input == nil {
		return
	}
	c.centerOnce(ctx)

	if c.cancel != nil && c.cancel.ConsumeClick() {
		c.dismiss = true
		return
	}
	if c.save != nil && c.save.ConsumeClick() {
		// Saving failed: keep editing so the user can retry. Otherwise quit.
		if vp := lookupViewport(c.GetScene()); vp != nil {
			if err := vp.Save(); err != nil {
				console.Print("save failed: " + err.Error())
				c.dismiss = true
				return
			}
		}
		c.dismiss = true
		if c.game != nil {
			c.game.Terminate()
		}
		return
	}
	if c.dontSave != nil && c.dontSave.ConsumeClick() {
		c.dismiss = true
		if c.game != nil {
			c.game.Terminate()
		}
		return
	}

	if modalOutsideClick(c.GetScene(), c.GetOwner(), ctx) {
		c.dismiss = true
	}
}

// centerOnce repositions the popup to the screen center on its first frame.
func (c *CloseConfirmDialogComponent) centerOnce(ctx *core.Context) {
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

func (c *CloseConfirmDialogComponent) Draw(r core.Renderer) {
	rect := c.Rect()
	if rect.Width() <= 0 || rect.Height() <= 0 {
		return
	}
	r.SetClipRect(rect)
	r.DrawRect(rect, c.Background)
	r.DrawRectOutline(rect, c.BorderColor, 1)

	r.DrawText("Unsaved changes. Save before quitting?", c.FontID, c.FontSize, math.NewVector2(rect.X()+8, rect.Y()+10), c.TitleText)

	r.ClearClip()

	if c.dismiss {
		c.close()
	}
}

// close clears the modal state and destroys the popup's object.
func (c *CloseConfirmDialogComponent) close() {
	clearModal()
	if owner := c.GetOwner(); owner != nil {
		owner.Destroy()
	}
}
