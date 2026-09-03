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
// editable project-directory field plus LOAD, SAVE, RUN, and STOP buttons. Loading
// points the viewport at a new project directory at runtime (instead of the path baked
// into editor.scene); Save serializes the viewport's loaded scene back to its .scene
// file on disk; RUN/STOP launch and kill the target project via the imge CLI. A short
// status line reports the result of the last action.
//
// The field is a real @TextInput and the four buttons are real @Buttons — engine
// widgets added as children of this object, so caret editing and hover/press states
// come for free instead of a hand-rolled edit buffer and rect buttons. The host draws
// only the strip chrome (label, status line) and polls the widgets for commits.
type ToolbarComponent struct {
	core.BaseUIComponent

	Background math.Color `json:"background"`
	FieldBg    math.Color `json:"field_bg"`
	Text       math.Color `json:"text"`
	Dim        math.Color `json:"dim"`         // labels + placeholder
	Accent     math.Color `json:"accent"`      // LOAD button
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

	pathInput  *TextInputComponent
	loadBtn    *ButtonComponent
	saveBtn    *ButtonComponent
	runBtn     *ButtonComponent
	stopBtn    *ButtonComponent
	built      bool // widgets built once (owner/height known)
	wasFocused bool // path TextInput blur tracking

	status      string
	statusError bool

	running atomic.Bool
	runCmd  *exec.Cmd // non-nil while a RUN is active; owned by the game loop
}

// Widget geometry, in owner-relative offsets (the strip is a fixed layout, so the
// constants replace the old rect methods). Height is derived from the strip's own
// height each time it is (re)built.
const (
	toolbarFieldX = 48.0
	toolbarFieldW = 250.0
	toolbarLoadX  = 302.0
	toolbarLoadW  = 58.0
	toolbarSaveX  = 364.0
	toolbarSaveW  = 60.0
	toolbarRunX   = 428.0
	toolbarRunW   = 44.0
	toolbarStopX  = 476.0
	toolbarStopW  = 48.0
)

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
	// occludes whatever is drawn behind it.
	if c.Blocking == nil {
		c.SetBlocking(true)
	}
}

// ensureWidgets builds the path field and the four buttons once the owner and strip
// height are known. Each widget is initialized manually: the object's own Initialize
// already ran, so AddComponent will not call it.
func (c *ToolbarComponent) ensureWidgets() {
	if c.built {
		return
	}
	owner := c.GetOwner()
	if owner == nil {
		return
	}
	h := c.Rect().Height() - 6
	if h <= 0 {
		return
	}
	c.built = true

	ti := &TextInputComponent{}
	ti.FontID = c.FontID
	ti.Size = c.FontSize
	ti.TextColor = c.Text
	ti.PlaceholderColor = c.Dim
	ti.BackgroundColor = c.FieldBg
	ti.Placeholder = "type project dir..."
	ti.Width = toolbarFieldW
	ti.Height = h
	ti.DrawLayer = 1
	ti.SetName("path")
	ti.SetOffset(math.NewVector2(toolbarFieldX, 3))
	owner.AddComponent(ti)
	ti.Initialize()
	c.pathInput = ti

	c.loadBtn = makePanelButton(owner, "load", "LOAD", math.NewVector2(toolbarLoadX, 3), toolbarLoadW, h, c.FontID, c.FontSize, c.Accent)
	c.saveBtn = makePanelButton(owner, "save", "SAVE", math.NewVector2(toolbarSaveX, 3), toolbarSaveW, h, c.FontID, c.FontSize, c.SaveAccent)
	c.runBtn = makePanelButton(owner, "run", "RUN", math.NewVector2(toolbarRunX, 3), toolbarRunW, h, c.FontID, c.FontSize, c.RunAccent)
	c.stopBtn = makePanelButton(owner, "stop", "STOP", math.NewVector2(toolbarStopX, 3), toolbarStopW, h, c.FontID, c.FontSize, c.StopAccent)
}

// discardWidgetClicks flushes any pending button activation, so a click that landed on
// a toolbar button while a modal was open never fires once the modal closes.
func (c *ToolbarComponent) discardWidgetClicks() {
	if c.loadBtn != nil {
		c.loadBtn.ConsumeClick()
	}
	if c.saveBtn != nil {
		c.saveBtn.ConsumeClick()
	}
	if c.runBtn != nil {
		c.runBtn.ConsumeClick()
	}
	if c.stopBtn != nil {
		c.stopBtn.ConsumeClick()
	}
}

func (c *ToolbarComponent) Update(ctx *core.Context) {
	if ctx == nil || ctx.Input == nil {
		return
	}
	c.ensureWidgets()

	// A RUN that exited on its own clears its handle here and reports it, so the
	// status line stops saying "running" and STOP becomes a no-op again (the wait
	// goroutine only flips the atomic flag; runCmd stays owned by the game loop).
	if c.runCmd != nil && !c.running.Load() {
		c.runCmd = nil
		c.status = "exited"
		c.statusError = false
	}

	// A modal is open (add-component panel / confirm dialog): the toolbar is inert.
	// Swallow any widget click now; Draw swallows again for a click the @UIManager
	// delivers after this Update runs (component update order is random).
	if modalOpen() {
		c.discardWidgetClicks()
		return
	}

	// Undo/redo shortcuts (Ctrl+Z / Ctrl+Shift+Z or Ctrl+Y). Skipped while a managed
	// widget (the path field, or an inspector/args field) holds keyboard focus.
	if c.handleUndoKeys(ctx.Input) {
		return
	}

	// Ctrl+S saves the loaded scene regardless of pointer position.
	if ctx.Input.IsKeyPressed(core.KeyControl) && ctx.Input.IsKeyJustPressed(core.KeyS) {
		c.save()
		return
	}

	// Path field: commit on Enter or blur, and live-sync the field to the current
	// project whenever it is not being edited.
	if c.pathInput != nil {
		focused := c.pathInput.IsFocused()
		if focused && ctx.Input.IsKeyJustPressed(core.KeyEnter) {
			c.commit()
		}
		if c.wasFocused && !focused {
			c.commit()
		}
		c.wasFocused = focused
		if !focused {
			if vp := lookupViewport(c.GetScene()); vp != nil {
				c.pathInput.Text = vp.CurrentProject()
			}
		}
	}

	// Buttons are polled (events are never delivered to dynamically-added widgets).
	if c.loadBtn != nil && c.loadBtn.ConsumeClick() {
		c.commit()
	}
	if c.saveBtn != nil && c.saveBtn.ConsumeClick() {
		c.save()
	}
	if c.runBtn != nil && c.runBtn.ConsumeClick() {
		c.run()
	}
	if c.stopBtn != nil && c.stopBtn.ConsumeClick() {
		c.stop()
	}

	// Reflect run state: RUN is disabled while running, STOP only while running.
	if c.runBtn != nil {
		c.runBtn.SetEnabled(!c.running.Load())
	}
	if c.stopBtn != nil {
		c.stopBtn.SetEnabled(c.running.Load())
	}
}

// commit loads the project in the path field (falling back to the current project when
// the field is empty) and clears any stale status.
func (c *ToolbarComponent) commit() {
	vp := lookupViewport(c.GetScene())
	if vp == nil {
		return
	}
	path := ""
	if c.pathInput != nil {
		path = strings.TrimSpace(c.pathInput.Text)
	}
	if path == "" {
		path = vp.CurrentProject()
	}
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
// the toolbar, inspector, or args window) so the shortcut never fights in-progress
// value editing. Returns true when it consumed a shortcut.
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
	// Discard any click the @UIManager delivered to a toolbar button after this
	// panel's Update ran this frame (component update order is random), so a click
	// landing while a modal was open can't fire once the modal closes.
	if modalOpen() {
		c.discardWidgetClicks()
	}

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

	// Label (the field and buttons draw themselves as layer-1 widgets).
	r.DrawText("PROJECT", c.FontID, c.FontSize, math.NewVector2(rect.X()+10, cy(rect.Y())), c.Dim)

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
