package components

import (
	"io"
	"os"
	"os/exec"
	"strings"
	"sync/atomic"
	"syscall"

	"github.com/EnesBaytekin/imge/core"
	"github.com/EnesBaytekin/imge/core/math"
)

// ToolbarComponent is the editor's top strip for the desktop workflow: it holds an
// editable project-directory field plus LOAD and SAVE buttons. Loading points the
// viewport at a new project directory at runtime (instead of the path baked into
// editor.scene); Save serializes the viewport's loaded scene back to its .scene file
// on disk. A short status line reports the result of the last action.
//
// Interaction is read directly from ctx.Input, like the other editor panels. Click the
// path field to edit it; Enter (or LOAD) reloads; Escape cancels. The field shows the
// viewport's current project directory when not being edited.
type ToolbarComponent struct {
	core.BaseUIComponent

	Background math.Color `json:"background"`
	FieldBg    math.Color `json:"field_bg"`
	Text       math.Color `json:"text"`
	Dim        math.Color `json:"dim"`         // labels + placeholder
	Accent     math.Color `json:"accent"`      // LOAD button / focused field
	SaveAccent math.Color `json:"save_accent"` // SAVE button
	RunAccent  math.Color `json:"run_accent"`  // RUN button
	StopAccent math.Color `json:"stop_accent"` // STOP button
	Error      math.Color `json:"error"`
	Success    math.Color `json:"success"`

	FontID   string  `json:"font_id"`
	FontSize float64 `json:"font_size"`

	// CLI is the path to the imge CLI used by RUN/STOP. Empty falls back to the
	// IMGE_CLI env var, then `imge` on PATH.
	CLI string `json:"cli"`

	focused     bool
	editBuf     string
	status      string
	statusError bool

	hover int // which toolbar control is under the cursor (0 = none); see hoverIndex

	running atomic.Bool
	runCmd  *exec.Cmd // non-nil while a RUN is active; owned by the game loop
}

// fieldRect, loadBtnRect, and saveBtnRect are the toolbar's hit regions, computed from
// the component's rect so Draw and Update never drift.
func (c *ToolbarComponent) fieldRect(rect math.Rect) math.Rect {
	return math.NewRect(rect.X()+48, rect.Y()+3, 250, rect.Height()-6)
}

func (c *ToolbarComponent) loadBtnRect(rect math.Rect) math.Rect {
	return math.NewRect(rect.X()+302, rect.Y()+3, 58, rect.Height()-6)
}

func (c *ToolbarComponent) saveBtnRect(rect math.Rect) math.Rect {
	return math.NewRect(rect.X()+364, rect.Y()+3, 60, rect.Height()-6)
}

func (c *ToolbarComponent) runBtnRect(rect math.Rect) math.Rect {
	return math.NewRect(rect.X()+428, rect.Y()+3, 44, rect.Height()-6)
}

func (c *ToolbarComponent) stopBtnRect(rect math.Rect) math.Rect {
	return math.NewRect(rect.X()+476, rect.Y()+3, 48, rect.Height()-6)
}

func (c *ToolbarComponent) Initialize() {
	if c.Background == (math.Color{}) {
		c.Background = math.NewColor(0x1a, 0x1d, 0x28, 0xff)
	}
	if c.FieldBg == (math.Color{}) {
		c.FieldBg = math.NewColor(0x10, 0x13, 0x1c, 0xff)
	}
	if c.Text == (math.Color{}) {
		c.Text = math.NewColor(0xe6, 0xe6, 0xef, 0xff)
	}
	if c.Dim == (math.Color{}) {
		c.Dim = math.NewColor(0x6b, 0x73, 0x85, 0xff)
	}
	if c.Accent == (math.Color{}) {
		c.Accent = math.NewColor(0x2f, 0x3b, 0x54, 0xff)
	}
	if c.SaveAccent == (math.Color{}) {
		c.SaveAccent = math.NewColor(0x2f, 0x5d, 0x3a, 0xff)
	}
	if c.RunAccent == (math.Color{}) {
		c.RunAccent = math.NewColor(0x2f, 0x5d, 0x8a, 0xff)
	}
	if c.StopAccent == (math.Color{}) {
		c.StopAccent = math.NewColor(0x8a, 0x3a, 0x3a, 0xff)
	}
	if c.Error == (math.Color{}) {
		c.Error = math.NewColor(0xff, 0x5a, 0x5a, 0xff)
	}
	if c.Success == (math.Color{}) {
		c.Success = math.NewColor(0x5a, 0xd0, 0x7a, 0xff)
	}
	if c.FontSize <= 0 {
		c.FontSize = 6
	}
	// The toolbar is an opaque surface: it blocks pointer events so the @UIManager
	// occludes whatever is drawn behind it (see pointerOwnedElsewhere).
	if c.Blocking == nil {
		c.SetBlocking(true)
	}
}

func (c *ToolbarComponent) Update(ctx *core.Context) {
	if ctx == nil || ctx.Input == nil {
		return
	}
	in := ctx.Input

	// A RUN that exited on its own clears its handle here and reports it, so the
	// status line stops saying "running" and STOP becomes a no-op again (the wait
	// goroutine only flips the atomic flag; runCmd stays owned by the game loop).
	if c.runCmd != nil && !c.running.Load() {
		c.runCmd = nil
		c.status = "exited"
		c.statusError = false
	}

	// Undo/redo shortcuts (Ctrl+Z / Ctrl+Shift+Z or Ctrl+Y). Skipped while the path
	// field is being edited, so the shortcut never clobbers in-progress typing.
	if !c.focused && c.handleUndoKeys(in) {
		return
	}

	// Ctrl+S saves the loaded scene regardless of pointer position.
	if in.IsKeyPressed(core.KeyControl) && in.IsKeyJustPressed(core.KeyS) {
		c.focused = false
		c.editBuf = ""
		c.save()
		return
	}

	// Text editing continues even after the pointer leaves the strip.
	if c.focused {
		switch {
		case in.IsKeyJustPressed(core.KeyEscape):
			c.focused = false
			c.editBuf = ""
			return
		case in.IsKeyJustPressed(core.KeyEnter):
			c.focused = false
			c.commit()
			return
		case in.IsKeyJustPressed(core.KeyBackspace):
			if r := []rune(c.editBuf); len(r) > 0 {
				c.editBuf = string(r[:len(r)-1])
			}
		}
		for _, ch := range in.InputChars() {
			if ch >= 0x20 {
				c.editBuf += string(ch)
			}
		}
	}

	rect := c.Rect()
	mouse := in.GetMousePosition()
	c.hover = c.hoverIndex(rect, mouse)
	if !rect.ContainsPoint(mouse) {
		return
	}
	// Yield the pointer to a floating window dragged over the toolbar. Keyboard
	// shortcuts (undo/redo, Ctrl+S, the path field) are handled above and stay global.
	if pointerOwnedElsewhere(c.GetScene(), c.GetOwner(), mouse) {
		c.hover = 0
		return
	}
	if !in.IsMouseButtonJustPressed(core.MouseButtonLeft) {
		return
	}

	switch {
	case c.fieldRect(rect).ContainsPoint(mouse):
		c.focused = true
		if vp := lookupViewport(c.GetScene()); vp != nil {
			c.editBuf = vp.CurrentProject()
		}
	case c.loadBtnRect(rect).ContainsPoint(mouse):
		c.focused = false
		c.commit()
	case c.saveBtnRect(rect).ContainsPoint(mouse):
		c.focused = false
		c.editBuf = ""
		c.save()
	case c.runBtnRect(rect).ContainsPoint(mouse):
		c.focused = false
		c.editBuf = ""
		c.run()
	case c.stopBtnRect(rect).ContainsPoint(mouse):
		c.focused = false
		c.editBuf = ""
		c.stop()
	default:
		c.focused = false
		c.editBuf = ""
	}
}

// commit loads the project in the field (falling back to the current project when the
// field was never edited) and clears any stale status.
func (c *ToolbarComponent) commit() {
	vp := lookupViewport(c.GetScene())
	if vp == nil {
		return
	}
	path := strings.TrimSpace(c.editBuf)
	if path == "" {
		path = vp.CurrentProject()
	}
	c.editBuf = ""
	c.status = ""
	if path != "" {
		vp.SetProject(path)
	}
}

// save writes the loaded target scene back to its .scene file and reports the result.
func (c *ToolbarComponent) save() {
	vp := lookupViewport(c.GetScene())
	if vp == nil {
		c.status = "error: no viewport"
		c.statusError = true
		return
	}
	if err := vp.Save(); err != nil {
		c.status = "error: " + err.Error()
		c.statusError = true
		return
	}
	c.status = "saved"
	c.statusError = false
	console.Print("saved")
}

// run builds and launches the target project via the imge CLI, in its own process
// group so STOP can kill the whole group (`imge run` plus the game it spawns).
// Build/game output streams to the terminal that launched the editor.
func (c *ToolbarComponent) run() {
	// A second RUN while one is active would overwrite runCmd and orphan the first
	// process group (STOP would then only reach the newer one). Refuse instead.
	if c.running.Load() {
		c.status = "already running"
		c.statusError = false
		return
	}
	vp := lookupViewport(c.GetScene())
	if vp == nil {
		c.status = "error: no viewport"
		c.statusError = true
		return
	}
	dir := vp.CurrentProject()
	if dir == "" {
		c.status = "error: no project"
		c.statusError = true
		return
	}
	// Auto-save the loaded scene first so RUN reflects in-progress edits.
	if err := vp.Save(); err != nil {
		c.status = "error: " + err.Error()
		c.statusError = true
		return
	}
	console.Print("run: " + dir)
	cli, err := c.resolveCLI()
	if err != nil {
		c.status = "error: " + err.Error()
		c.statusError = true
		return
	}
	cmd := exec.Command(cli, "run")
	cmd.Dir = dir
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	// Tee build/game output to both the bottom console and the terminal. The console
	// is first so a broken or closed terminal stream can't starve it of output.
	cmd.Stdout = io.MultiWriter(&console, os.Stdout)
	cmd.Stderr = io.MultiWriter(&console, os.Stderr)
	if err := cmd.Start(); err != nil {
		c.status = "error: " + err.Error()
		c.statusError = true
		return
	}
	c.runCmd = cmd
	c.running.Store(true)
	c.status = "running"
	c.statusError = false
	go func() {
		_ = cmd.Wait()
		console.Flush()
		c.running.Store(false)
	}()
}

// stop kills the running project's process group. The negative pid reaches `imge run`
// and the game it spawned together, so neither is left orphaned.
func (c *ToolbarComponent) stop() {
	if !c.running.Load() {
		return
	}
	if c.runCmd != nil && c.runCmd.Process != nil {
		_ = syscall.Kill(-c.runCmd.Process.Pid, syscall.SIGKILL)
	}
	c.running.Store(false)
	c.runCmd = nil
	c.status = "stopped"
	c.statusError = false
	console.Print("stopped")
}

// handleUndoKeys applies the editor-wide undo/redo shortcuts and reports the result in
// the status line. Ctrl+Z undoes the last committed field edit; Ctrl+Shift+Z or Ctrl+Y
// redoes. It is skipped while any managed widget holds keyboard focus (a @TextInput in
// the inspector or args window) so the shortcut never fights in-progress value editing.
// Returns true when it consumed a shortcut.
func (c *ToolbarComponent) handleUndoKeys(in core.Input) bool {
	if !in.IsKeyPressed(core.KeyControl) {
		return false
	}
	if mgr := lookupUIManager(c.GetScene()); mgr != nil && mgr.HasFocus() {
		return false
	}
	zDown := in.IsKeyJustPressed(core.KeyZ)
	yDown := in.IsKeyJustPressed(core.KeyY)
	// Ctrl+Shift+Z redoes (shift overrides undo), as does Ctrl+Y. Shift must win over
	// plain Ctrl+Z so the combination doesn't fall through to undo.
	redo := yDown || (zDown && in.IsKeyPressed(core.KeyShift))
	undo := zDown && !in.IsKeyPressed(core.KeyShift)
	if !undo && !redo {
		return false
	}
	if redo {
		if history.redo() {
			c.status = "redone"
		} else {
			c.status = "nothing to redo"
		}
	} else {
		if history.undo() {
			c.status = "undone"
		} else {
			c.status = "nothing to undo"
		}
	}
	c.statusError = false
	return true
}

// resolveCLI returns the path to the imge CLI: the explicit CLI field, else the
// IMGE_CLI env var (set by `imge editor`), else `imge` on PATH.
func (c *ToolbarComponent) resolveCLI() (string, error) {
	if c.CLI != "" {
		return c.CLI, nil
	}
	if v := os.Getenv("IMGE_CLI"); v != "" {
		return v, nil
	}
	return exec.LookPath("imge")
}

func (c *ToolbarComponent) Draw(r core.Renderer) {
	rect := c.Rect()
	if rect.Width() <= 0 || rect.Height() <= 0 {
		return
	}

	r.SetClipRect(rect)
	r.DrawRect(rect, c.Background)

	_, th := r.MeasureText("Ag", c.FontID, c.FontSize)
	cy := func(y float64) float64 { // vertical center of a row
		return y + (rect.Height()-th)/2
	}

	// Label + editable path field.
	r.DrawText("PROJECT", c.FontID, c.FontSize, math.NewVector2(rect.X()+10, cy(rect.Y())), c.Dim)
	field := c.fieldRect(rect)
	fieldBg := c.FieldBg
	if c.hover == 1 {
		fieldBg = fieldBg.Lerp(math.White, 0.06)
	}
	r.DrawRect(field, fieldBg)
	if c.focused {
		r.DrawRectOutline(field, c.Accent, 1)
	}

	shown := ""
	shownColor := c.Text
	if c.focused {
		shown = c.editBuf + "_"
	} else if vp := lookupViewport(c.GetScene()); vp != nil {
		shown = vp.CurrentProject()
	}
	if shown == "" {
		shown = "type project dir..."
		shownColor = c.Dim
	}
	r.DrawText(shown, c.FontID, c.FontSize, math.NewVector2(field.X()+4, cy(field.Y())), shownColor)

	// LOAD and SAVE buttons.
	c.drawButton(r, c.loadBtnRect(rect), "LOAD", c.Accent, c.Text, th, c.hover == 2)
	c.drawButton(r, c.saveBtnRect(rect), "SAVE", c.SaveAccent, c.Text, th, c.hover == 3)
	c.drawButton(r, c.runBtnRect(rect), "RUN", c.RunAccent, c.Text, th, c.hover == 4)
	c.drawButton(r, c.stopBtnRect(rect), "STOP", c.StopAccent, c.Text, th, c.hover == 5)

	// Status line (right side).
	if c.status != "" {
		sc := c.Success
		if c.statusError {
			sc = c.Error
		}
		r.DrawText(c.status, c.FontID, c.FontSize, math.NewVector2(rect.X()+528, cy(rect.Y())), sc)
	}

	r.ClearClip()
}

func (c *ToolbarComponent) drawButton(r core.Renderer, btn math.Rect, label string, bg, fg math.Color, textH float64, hovered bool) {
	if hovered {
		bg = bg.Lerp(math.White, 0.12)
	}
	r.DrawRect(btn, bg)
	r.DrawText(label, c.FontID, c.FontSize, math.NewVector2(btn.X()+5, btn.Y()+(btn.Height()-textH)/2), fg)
}

// hoverIndex reports which toolbar control the cursor is over, or 0 for none. It maps
// to the constants used in Draw (1=field, 2=LOAD, 3=SAVE, 4=RUN, 5=STOP).
func (c *ToolbarComponent) hoverIndex(rect math.Rect, mouse math.Vector2) int {
	switch {
	case c.fieldRect(rect).ContainsPoint(mouse):
		return 1
	case c.loadBtnRect(rect).ContainsPoint(mouse):
		return 2
	case c.saveBtnRect(rect).ContainsPoint(mouse):
		return 3
	case c.runBtnRect(rect).ContainsPoint(mouse):
		return 4
	case c.stopBtnRect(rect).ContainsPoint(mouse):
		return 5
	}
	return 0
}
