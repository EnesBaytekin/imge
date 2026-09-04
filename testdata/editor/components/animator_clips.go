package components

import (
	"github.com/EnesBaytekin/imge/core"
	"github.com/EnesBaytekin/imge/core/math"
)

// defaultClipFPS is the frames-per-second given to a newly added clip. It matches
// Animator.Initialize's runtime fallback, so a fresh clip plays at a sane rate with
// no further input.
const defaultClipFPS = 12.0

// clipRow tracks the per-sprite widgets of one row in the clips editor: the "is a
// clip" toggle, the FPS text box, and the loop toggle, plus the FPS box's blur state
// (for commit-on-blur, matching the other field widgets).
type clipRow struct {
	sprite        string
	check         *CheckBoxComponent
	fpsInput      *TextInputComponent
	loopCheck     *CheckBoxComponent
	fpsWasFocused bool
}

// AnimatorClipsComponent is the modal "Animator Clips" editor, opened by clicking the
// read-only `clips` row in the component-args window for an @Animator. It edits the
// animator's clip list without typing sprite names by hand: every @Sprite on the
// animator's owner object (plus any sprite an existing clip already names) is listed as
// a row. Each row has a checkbox — ticking it adds a clip for that sprite, unticking it
// removes the clip — and, when a clip exists, an FPS text box and a "loop" checkbox.
//
// It is modal like the other dialogs: while it is open every hand-rolled panel yields
// (see modalOpen) and an outside click dismisses it, finalized in Draw.
type AnimatorClipsComponent struct {
	core.BaseUIComponent

	Background  math.Color `json:"background"`
	TitleText   math.Color `json:"title_text"`
	KeyText     math.Color `json:"key_text"`
	ValueText   math.Color `json:"value_text"`
	Accent      math.Color `json:"accent"` // title bar + Done button
	BorderColor math.Color `json:"border_color"`
	ErrorColor  math.Color `json:"error_color"` // FPS parse failure
	ScrollTrack math.Color `json:"scroll_track"`
	ScrollThumb math.Color `json:"scroll_thumb"`

	FontID    string  `json:"font_id"`
	FontSize  float64 `json:"font_size"`
	RowHeight float64 `json:"row_height"`

	anim *Animator

	rows    []*clipRow
	doneBtn *ButtonComponent

	scroll         float64
	scrollDragging bool
	scrollGrab     float64

	dragging bool         // the title bar is being dragged to move the window
	dragGrab math.Vector2 // mouse offset from the window's top-left when the drag began

	dismiss  bool
	centered bool
}

// Column layout of the list body, as offsets from the window's left edge. The sprite
// name is the checkbox's own label, drawn to the right of its box.
const (
	clipsXCheck = 8.0
	clipsXFps   = 158.0
	clipsXLoop  = 212.0
	clipsCheckW = 144.0
	clipsFpsW   = 44.0
	clipsLoopW  = 52.0
)

func (c *AnimatorClipsComponent) titleH() float64 { return c.RowHeight + 8 }

// bodyTop/bodyBottom are the absolute screen Y of the scrollable list body: a header
// row sits between the title bar and the list, and the Done button sits below it.
func (c *AnimatorClipsComponent) bodyTop() float64    { return c.Rect().Y() + c.titleH() + 14 }
func (c *AnimatorClipsComponent) bodyBottom() float64 { return c.Rect().Y() + c.Height - 30 }

func (c *AnimatorClipsComponent) Initialize() {
	if c.Background == (math.Color{}) {
		c.Background = math.NewColor(0x1d, 0x21, 0x30, 0xff)
	}
	if c.TitleText == (math.Color{}) {
		c.TitleText = math.NewColor(0xff, 0xff, 0xff, 0xff)
	}
	if c.KeyText == (math.Color{}) {
		c.KeyText = math.NewColor(0x6b, 0x73, 0x85, 0xff)
	}
	if c.ValueText == (math.Color{}) {
		c.ValueText = math.NewColor(0xe6, 0xe6, 0xef, 0xff)
	}
	if c.Accent == (math.Color{}) {
		c.Accent = math.NewColor(0x2f, 0x3b, 0x54, 0xff)
	}
	if c.BorderColor == (math.Color{}) {
		c.BorderColor = math.NewColor(0x3a, 0x42, 0x57, 0xff)
	}
	if c.ErrorColor == (math.Color{}) {
		c.ErrorColor = math.NewColor(0xff, 0x5a, 0x5a, 0xff)
	}
	if c.ScrollTrack == (math.Color{}) {
		c.ScrollTrack = math.NewColor(0x2a, 0x30, 0x42, 0xff)
	}
	if c.ScrollThumb == (math.Color{}) {
		c.ScrollThumb = math.NewColor(0x4a, 0x55, 0x70, 0xff)
	}
	if c.FontSize <= 0 {
		c.FontSize = 6
	}
	if c.RowHeight <= 0 {
		c.RowHeight = 18
	}
	if c.Blocking == nil {
		c.SetBlocking(true)
	}
}

// spawnAnimatorClips opens the clips editor for anim. It is a no-op when the animator
// is nil or a modal is already open.
func spawnAnimatorClips(scene *core.Scene, anim *Animator) {
	if scene == nil || anim == nil || modalOpen() {
		return
	}

	obj := core.NewObject("animator_clips")
	obj.UI = true
	obj.Layer = 3
	obj.Transform.Position = math.NewVector2(200, 80) // centered lazily on first Update

	win := &AnimatorClipsComponent{}
	win.SetName("animator_clips")
	win.Width = 320
	win.Height = 240
	win.anim = anim
	obj.AddComponent(win)

	if err := scene.AddObject(obj); err != nil {
		return
	}

	win.Initialize()
	win.buildWidgets()
	setModal(win)
	raiseToFront(scene, obj)
}

// spriteNames returns the candidate clip sprites: every @Sprite on the animator's owner
// object, followed by any sprite an existing clip names that is no longer on the object
// (so an orphan clip stays visible and removable rather than silently vanishing).
func (c *AnimatorClipsComponent) spriteNames() []string {
	anim := c.anim
	if anim == nil || anim.GetOwner() == nil {
		return nil
	}
	seen := make(map[string]bool)
	var names []string
	for _, sp := range core.GetAllFrom[*Sprite](anim.GetOwner()) {
		if name := sp.GetName(); name != "" && !seen[name] {
			seen[name] = true
			names = append(names, name)
		}
	}
	for _, clip := range anim.Clips {
		if clip.Sprite != "" && !seen[clip.Sprite] {
			seen[clip.Sprite] = true
			names = append(names, clip.Sprite)
		}
	}
	return names
}

// buildWidgets creates one row per candidate sprite and the Done button. The rows are
// added as children of the same window object, then laid out (with scroll) in
// layoutRows. Called once on open; add/remove of a clip mutates the list in place.
func (c *AnimatorClipsComponent) buildWidgets() {
	owner := c.GetOwner()
	if owner == nil || c.anim == nil {
		return
	}
	names := c.spriteNames()
	c.rows = make([]*clipRow, 0, len(names))
	for i, name := range names {
		y := c.bodyTop() + float64(i)*c.RowHeight - c.scroll
		if row := c.buildRow(owner, name, y); row != nil {
			c.rows = append(c.rows, row)
		}
	}
	c.doneBtn = makePanelButton(owner, "done", "Done", math.NewVector2(8, c.Height-26), c.Width-16, 20, c.FontID, c.FontSize, c.Accent)
	c.layoutRows()
}

// buildRow creates the three widgets for one sprite. The checkbox is ticked when a clip
// for that sprite already exists; the FPS box and loop toggle are enabled only then.
func (c *AnimatorClipsComponent) buildRow(owner *core.Object, name string, y float64) *clipRow {
	row := &clipRow{sprite: name}
	has := c.clipIndex(name) >= 0

	cb := &CheckBoxComponent{}
	cb.Text = name
	cb.BoxSize = 14
	cb.FontID = c.FontID
	cb.Size = c.FontSize
	cb.TextColor = c.ValueText
	cb.Width = clipsCheckW
	cb.Height = 16
	cb.DrawLayer = 1
	cb.SetName("clip_" + name)
	cb.SetOffset(math.NewVector2(c.Rect().X()+clipsXCheck, y).Subtract(owner.Transform.Position))
	if err := owner.AddComponent(cb); err != nil {
		return nil
	}
	cb.Initialize()
	cb.SetChecked(has)
	row.check = cb

	ti := &TextInputComponent{}
	ti.FontID = c.FontID
	ti.Size = c.FontSize
	ti.TextColor = c.ValueText
	ti.BackgroundColor = fieldBackground
	ti.OutlineColor = fieldOutline
	ti.OutlineThickness = 1
	ti.Width = clipsFpsW
	ti.Height = 16
	ti.DrawLayer = 1
	ti.SetName("fps_" + name)
	ti.SetOffset(math.NewVector2(c.Rect().X()+clipsXFps, y).Subtract(owner.Transform.Position))
	if err := owner.AddComponent(ti); err != nil {
		return nil
	}
	ti.Initialize()
	ti.Text = formatFloat(c.clipFPS(name))
	ti.SetEnabled(has)
	row.fpsInput = ti

	lc := &CheckBoxComponent{}
	lc.Text = "loop"
	lc.BoxSize = 14
	lc.FontID = c.FontID
	lc.Size = c.FontSize
	lc.TextColor = c.ValueText
	lc.Width = clipsLoopW
	lc.Height = 16
	lc.DrawLayer = 1
	lc.SetName("loop_" + name)
	lc.SetOffset(math.NewVector2(c.Rect().X()+clipsXLoop, y).Subtract(owner.Transform.Position))
	if err := owner.AddComponent(lc); err != nil {
		return nil
	}
	lc.Initialize()
	lc.SetChecked(c.clipLoop(name))
	lc.SetEnabled(has)
	row.loopCheck = lc

	return row
}

// clipIndex returns the index of the clip naming sprite in anim.Clips, or -1.
func (c *AnimatorClipsComponent) clipIndex(sprite string) int {
	for i := range c.anim.Clips {
		if c.anim.Clips[i].Sprite == sprite {
			return i
		}
	}
	return -1
}

// clipFPS returns the clip's frames-per-second, falling back to defaultClipFPS when the
// clip is absent or its FPS is unset (matching Animator.Initialize's runtime fallback).
func (c *AnimatorClipsComponent) clipFPS(sprite string) float64 {
	if i := c.clipIndex(sprite); i >= 0 && c.anim.Clips[i].FPS > 0 {
		return c.anim.Clips[i].FPS
	}
	return defaultClipFPS
}

// clipLoop returns whether the clip loops (false when absent).
func (c *AnimatorClipsComponent) clipLoop(sprite string) bool {
	if i := c.clipIndex(sprite); i >= 0 {
		return c.anim.Clips[i].Loop
	}
	return false
}

// cloneClips copies a clip slice so undo/redo snapshots stay independent of the live
// slice's later reassignments.
func cloneClips(clips []Clip) []Clip {
	out := make([]Clip, len(clips))
	copy(out, clips)
	return out
}

// applyClips replaces the animator's clips and re-initializes it, so the viewport
// reflects the change immediately (Initialize re-resolves the clip→sprite map).
func (c *AnimatorClipsComponent) applyClips(clips []Clip) {
	c.anim.Clips = cloneClips(clips)
	c.anim.Initialize()
}

func (c *AnimatorClipsComponent) addClip(sprite string) {
	if c.clipIndex(sprite) >= 0 {
		return
	}
	old := cloneClips(c.anim.Clips)
	c.anim.Clips = append(c.anim.Clips, Clip{Sprite: sprite, FPS: defaultClipFPS, Loop: false})
	c.anim.Initialize()
	new := cloneClips(c.anim.Clips)
	history.record("added clip "+sprite,
		func() { c.applyClips(old) },
		func() { c.applyClips(new) },
		true)
}

func (c *AnimatorClipsComponent) removeClip(sprite string) {
	i := c.clipIndex(sprite)
	if i < 0 {
		return
	}
	old := cloneClips(c.anim.Clips)
	c.anim.Clips = append(c.anim.Clips[:i], c.anim.Clips[i+1:]...)
	c.anim.Initialize()
	new := cloneClips(c.anim.Clips)
	history.record("removed clip "+sprite,
		func() { c.applyClips(old) },
		func() { c.applyClips(new) },
		true)
}

func (c *AnimatorClipsComponent) setClipFPS(sprite string, fps float64) {
	i := c.clipIndex(sprite)
	if i < 0 {
		return
	}
	old := cloneClips(c.anim.Clips)
	c.anim.Clips[i].FPS = fps
	c.anim.Initialize()
	new := cloneClips(c.anim.Clips)
	history.record("clip "+sprite+" fps",
		func() { c.applyClips(old) },
		func() { c.applyClips(new) },
		true)
}

func (c *AnimatorClipsComponent) setClipLoop(sprite string, loop bool) {
	i := c.clipIndex(sprite)
	if i < 0 {
		return
	}
	old := cloneClips(c.anim.Clips)
	c.anim.Clips[i].Loop = loop
	c.anim.Initialize()
	new := cloneClips(c.anim.Clips)
	history.record("clip "+sprite+" loop",
		func() { c.applyClips(old) },
		func() { c.applyClips(new) },
		true)
}

// pollRows applies widget changes each frame: the tick toggle adds/removes a clip, the
// FPS box commits on Enter or blur, and the loop toggle updates the clip's loop flag.
func (c *AnimatorClipsComponent) pollRows(ctx *core.Context) {
	for _, row := range c.rows {
		if row == nil || row.check == nil {
			continue
		}
		idx := c.clipIndex(row.sprite)
		checked := row.check.GetChecked()

		if checked && idx < 0 {
			c.addClip(row.sprite)
			row.fpsInput.Text = formatFloat(c.clipFPS(row.sprite))
			row.loopCheck.SetChecked(false)
			row.fpsInput.SetEnabled(true)
			row.loopCheck.SetEnabled(true)
			idx = c.clipIndex(row.sprite)
		} else if !checked && idx >= 0 {
			c.removeClip(row.sprite)
			row.fpsInput.SetEnabled(false)
			row.loopCheck.SetEnabled(false)
			idx = -1
		}

		// FPS: commit on Enter or blur, matching the args-window field widgets.
		ti := row.fpsInput
		focused := ti.IsFocused()
		if focused && ctx.Input.IsKeyJustPressed(core.KeyEnter) {
			if c.commitFPS(row) {
				ti.TextColor = c.ValueText
			} else {
				ti.TextColor = c.ErrorColor
			}
		}
		if row.fpsWasFocused && !focused {
			if !c.commitFPS(row) {
				ti.Text = formatFloat(c.clipFPS(row.sprite))
			}
			ti.TextColor = c.ValueText
		}
		row.fpsWasFocused = focused

		// Loop toggle (only meaningful while a clip exists).
		if idx >= 0 && row.loopCheck.GetChecked() != c.anim.Clips[idx].Loop {
			c.setClipLoop(row.sprite, row.loopCheck.GetChecked())
		}
	}
}

// commitFPS parses the row's FPS box and writes it to the clip; it reports whether the
// value parsed to a positive number.
func (c *AnimatorClipsComponent) commitFPS(row *clipRow) bool {
	f, err := parseFloat(row.fpsInput.Text)
	if err != nil || f <= 0 {
		return false
	}
	if idx := c.clipIndex(row.sprite); idx >= 0 && c.anim.Clips[idx].FPS != f {
		c.setClipFPS(row.sprite, f)
	}
	return true
}

func (c *AnimatorClipsComponent) maxScroll() float64 {
	content := float64(len(c.rows)) * c.RowHeight
	body := c.bodyBottom() - c.bodyTop()
	if content > body {
		return content - body
	}
	return 0
}

func (c *AnimatorClipsComponent) clampScroll() {
	if max := c.maxScroll(); c.scroll > max {
		c.scroll = max
	}
	if c.scroll < 0 {
		c.scroll = 0
	}
}

func (c *AnimatorClipsComponent) scrollTrack(rect math.Rect) math.Rect {
	const w = 6.0
	return math.NewRect(rect.X()+rect.Width()-w-2, c.bodyTop(), w, c.bodyBottom()-c.bodyTop())
}

// layoutRows repositions each row's widgets for the current scroll and hides rows
// scrolled out of the body, clipping the rest to the body.
func (c *AnimatorClipsComponent) layoutRows() {
	owner := c.GetOwner()
	if owner == nil {
		return
	}
	rect := c.Rect()
	clip := math.NewRect(rect.X(), c.bodyTop(), rect.Width(), c.bodyBottom()-c.bodyTop())
	for i, row := range c.rows {
		if row == nil {
			continue
		}
		y := c.bodyTop() + float64(i)*c.RowHeight - c.scroll
		visible := y+c.RowHeight > c.bodyTop() && y < c.bodyBottom()
		for _, w := range []fieldWidget{row.check, row.fpsInput, row.loopCheck} {
			if w == nil {
				continue
			}
			if !visible {
				w.SetVisible(false)
				w.ClearClipRect()
				continue
			}
			w.SetVisible(true)
			w.SetClipRect(clip)
		}
		if !visible {
			continue
		}
		row.check.SetOffset(math.NewVector2(rect.X()+clipsXCheck, y).Subtract(owner.Transform.Position))
		row.fpsInput.SetOffset(math.NewVector2(rect.X()+clipsXFps, y).Subtract(owner.Transform.Position))
		row.loopCheck.SetOffset(math.NewVector2(rect.X()+clipsXLoop, y).Subtract(owner.Transform.Position))
	}
}

func (c *AnimatorClipsComponent) handleScrollbarPress(mouse math.Vector2, rect math.Rect) {
	track := c.scrollTrack(rect)
	contentH := float64(len(c.rows)) * c.RowHeight
	max := c.maxScroll()
	thumb, ok := scrollThumb(track, contentH, c.scroll, max)
	if !ok {
		return
	}
	if thumb.ContainsPoint(mouse) {
		c.scrollDragging = true
		c.scrollGrab = mouse.Y - thumb.Y()
		return
	}
	if track.ContainsPoint(mouse) {
		c.scroll = scrollFromThumb(track, contentH, max, mouse.Y, thumb.Height()/2)
		c.layoutRows()
	}
}

func (c *AnimatorClipsComponent) Update(ctx *core.Context) {
	if ctx == nil || ctx.Input == nil {
		return
	}
	c.centerOnce(ctx)

	mouse := ctx.Input.GetMousePosition()

	// Drag-to-move: while the title bar is held, follow the cursor (even outside the
	// window). Moving the owner carries the row widgets with it, since their offsets
	// are relative to the owner's transform.
	if c.dragging {
		if ctx.Input.IsMouseButtonPressed(core.MouseButtonLeft) {
			c.GetOwner().SetPosition(mouse.X-c.dragGrab.X, mouse.Y-c.dragGrab.Y)
		} else {
			c.dragging = false
		}
	}

	c.pollRows(ctx)

	if c.doneBtn != nil && c.doneBtn.ConsumeClick() {
		c.dismiss = true
		return
	}

	rect := c.Rect()

	// A scrollbar drag keeps following the cursor even outside the window.
	if c.scrollDragging {
		if ctx.Input.IsMouseButtonPressed(core.MouseButtonLeft) {
			c.scroll = scrollFromThumb(c.scrollTrack(rect), float64(len(c.rows))*c.RowHeight, c.maxScroll(), mouse.Y, c.scrollGrab)
			c.clampScroll()
			c.layoutRows()
		} else {
			c.scrollDragging = false
		}
	}

	if !rect.ContainsPoint(mouse) {
		return
	}
	// Yield to a window drawn above this one.
	if pointerOwnedElsewhere(c.GetScene(), c.GetOwner(), mouse) {
		return
	}

	// Wheel scrolls the list, unless a widget holds focus (so a focused FPS box never
	// scrolls out from under the caret).
	if s := ctx.Input.GetMouseScroll(); s.Y != 0 {
		if mgr := lookupUIManager(c.GetScene()); mgr == nil || !mgr.HasFocus() {
			c.scroll -= s.Y * c.RowHeight * 2
			c.clampScroll()
			c.layoutRows()
		}
	}

	if ctx.Input.IsMouseButtonJustPressed(core.MouseButtonLeft) {
		if math.NewRect(rect.X(), rect.Y(), rect.Width(), c.titleH()).ContainsPoint(mouse) {
			c.dragging = true
			c.dragGrab = mouse.Subtract(rect.Position)
			return
		}
		c.handleScrollbarPress(mouse, rect)
	}

	if modalOutsideClick(c.GetScene(), c.GetOwner(), ctx) {
		c.dismiss = true
	}
}

func (c *AnimatorClipsComponent) centerOnce(ctx *core.Context) {
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

func (c *AnimatorClipsComponent) Draw(r core.Renderer) {
	rect := c.Rect()
	if rect.Width() <= 0 || rect.Height() <= 0 {
		return
	}
	r.SetClipRect(rect)
	r.DrawRect(rect, c.Background)
	r.DrawRectOutline(rect, c.BorderColor, 1)

	_, th := r.MeasureText("Ag", c.FontID, c.FontSize)

	// Title bar.
	r.DrawRect(math.NewRect(rect.X(), rect.Y(), rect.Width(), c.titleH()), c.Accent)
	title := "ANIMATOR CLIPS"
	if c.anim != nil {
		title += " — " + c.anim.GetName()
	}
	r.DrawText(title, c.FontID, c.FontSize, math.NewVector2(rect.X()+6, rect.Y()+(c.titleH()-th)/2), c.TitleText)

	// Column headers between the title bar and the first row.
	hy := rect.Y() + c.titleH() + (14-th)/2
	r.DrawText("sprite", c.FontID, c.FontSize, math.NewVector2(rect.X()+clipsXCheck, hy), c.KeyText)
	r.DrawText("fps", c.FontID, c.FontSize, math.NewVector2(rect.X()+clipsXFps, hy), c.KeyText)
	r.DrawText("loop", c.FontID, c.FontSize, math.NewVector2(rect.X()+clipsXLoop, hy), c.KeyText)

	if len(c.rows) == 0 {
		r.DrawText("no sprites on this object", c.FontID, c.FontSize, math.NewVector2(rect.X()+8, c.bodyTop()+2), c.KeyText)
	}

	r.ClearClip()

	if thumb, ok := scrollThumb(c.scrollTrack(rect), float64(len(c.rows))*c.RowHeight, c.scroll, c.maxScroll()); ok {
		drawScrollbar(r, c.scrollTrack(rect), thumb, c.ScrollTrack, c.ScrollThumb)
	}

	if c.dismiss {
		c.closeSelf()
	}
}

// closeSelf clears the modal state and destroys the window's object.
func (c *AnimatorClipsComponent) closeSelf() {
	clearModal()
	if owner := c.GetOwner(); owner != nil {
		owner.Destroy()
	}
}
