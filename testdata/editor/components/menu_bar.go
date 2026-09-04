package components

import (
	"io"
	"os"
	"os/exec"
	"sync/atomic"
	"syscall"

	"github.com/EnesBaytekin/imge/core"
	"github.com/EnesBaytekin/imge/core/math"
)

// MenuBarComponent is the editor's top strip reworked as a classic menu bar: a thin
// row of tabs (File / Edit / Run / Game) that each drop down a small listbox of
// entries when clicked. Clicking an entry runs its action; clicking another tab
// switches menus; clicking the open tab (or anywhere else) closes it. While a menu
// is open the rest of the editor is inert (see menusOpen), so the dismiss click is
// consumed rather than leaking into the panel beneath.
//
// The old toolbar's project-path field and LOAD/SAVE/RUN/STOP buttons move into the
// menus: File → "Open Project..." (a modal path field) and "Save"; Run → "Run" /
// "Stop". The run/stop process management, the status line, and the Ctrl+Z/Y and
// Ctrl+S shortcuts are carried over unchanged. "Game → Game Settings..." opens the
// modal that edits the target project's game.imge.
//
// The strip and dropdown are hand-drawn (not widget children); input is read directly
// from ctx.Input like the viewport. The dropdown is drawn outside the strip clip and
// covered by Contains while open, so the @UIManager's occlusion treats it as part of
// this element.
type MenuBarComponent struct {
	core.BaseUIComponent

	Background  math.Color `json:"background"`
	TabText     math.Color `json:"tab_text"` // tab label
	Hover       math.Color `json:"hover"`    // hovered / open tab highlight
	Dropdown    math.Color `json:"dropdown"` // dropdown panel fill
	BorderColor math.Color `json:"border_color"`
	ItemText    math.Color `json:"item_text"`  // menu entry label
	ItemHover   math.Color `json:"item_hover"` // hovered menu entry highlight
	Dim         math.Color `json:"dim"`        // shortcut hints / disabled entries
	Error       math.Color `json:"error"`      // status line (error)
	Success     math.Color `json:"success"`    // status line (ok)

	FontID   string  `json:"font_id"`
	FontSize float64 `json:"font_size"`

	// CLI is the path to the imge CLI used by RUN/STOP. Empty falls back to the
	// IMGE_CLI env var, then `imge` on PATH.
	CLI string `json:"cli"`

	// menus is the tab/entry table, built once in Initialize. Entry actions capture
	// this component so they can call save/run/stop/… and set the status line.
	menus []menu

	openMenu  int // index of the open menu, -1 = none
	hoverTab  int // tab under the cursor (-1 = none)
	hoverItem int // menu entry under the cursor (-1 = none)

	status      string
	statusError bool

	running atomic.Bool
	runCmd  *exec.Cmd // non-nil while a RUN is active; owned by the game loop

	closeHookRegistered bool // the OS close interceptor has been installed once
}

// menuEntry is one selectable row in a menu: its label, an optional shortcut hint,
// the action to run on click, and an optional enabled test (a disabled entry is dimmed
// and unclickable).
type menuEntry struct {
	label   string
	hint    string
	run     func()
	enabled func() bool
}

// menu is one top-level tab. A normal menu drops down a list of entries when clicked;
// a direct menu (direct: true) has no dropdown — its single entry's action runs
// immediately on click, like a small toolbar button (Run/Stop).
type menu struct {
	label  string
	direct bool
	items  []menuEntry
}

// menusOpen reports whether a menu-bar dropdown is open. It is the package-level
// counterpart to modalOpen: the non-modal panels (viewport, tree, inspector, console,
// args window) yield while it is true, so a click that dismisses the menu is consumed.
// The menu bar itself does not check it — it is the thing that is open.
var menusOpenFlag bool

func menusOpen() bool { return menusOpenFlag }

// Dropdown geometry. The tab strip is a fixed layout (constant tab width), so these
// constants replace per-label measurement in Update (Draw still measures text to
// center it, but hit-testing must not depend on the renderer).
const (
	menuTabWidth    = 48.0
	menuTabPadX     = 8.0
	menuWidth       = 160.0
	menuRowHeight   = 18.0
	menuItemPadX    = 8.0
	menuDropdownPad = 4.0 // vertical padding at the top/bottom of the dropdown
)

func (c *MenuBarComponent) tabRect(i int) math.Rect {
	rect := c.Rect()
	return math.NewRect(rect.X()+menuTabPadX+float64(i)*menuTabWidth, rect.Y(), menuTabWidth, rect.Height())
}

// dropdownRect returns the open menu's panel rect, below the open tab.
func (c *MenuBarComponent) dropdownRect() math.Rect {
	if c.openMenu < 0 || c.openMenu >= len(c.menus) {
		return math.Rect{}
	}
	tr := c.tabRect(c.openMenu)
	h := float64(len(c.menus[c.openMenu].items))*menuRowHeight + 2*menuDropdownPad
	return math.NewRect(tr.X()-4, c.Rect().Bottom(), menuWidth, h)
}

// itemRect returns one menu entry's row rect within the open dropdown.
func (c *MenuBarComponent) itemRect(i int) math.Rect {
	dr := c.dropdownRect()
	return math.NewRect(dr.X(), dr.Top()+menuDropdownPad+float64(i)*menuRowHeight, dr.Width(), menuRowHeight)
}

// Contains reports whether p is over the strip or, when a menu is open, its dropdown —
// so the manager hit-tests the whole dropdown as one element (mirrors @ComboBox).
func (c *MenuBarComponent) Contains(p math.Vector2) bool {
	if c.Rect().ContainsPoint(p) {
		return true
	}
	if c.openMenu >= 0 {
		return c.dropdownRect().ContainsPoint(p)
	}
	return false
}

func (c *MenuBarComponent) Initialize() {
	// Start with no menu open and nothing hovered. openMenu's Go zero value is 0
	// (the File tab), so without this the File dropdown renders open at startup.
	c.openMenu = -1
	c.hoverTab = -1
	c.hoverItem = -1

	if c.Background == (math.Color{}) {
		c.Background = math.NewColor(0x1a, 0x1d, 0x28, 0xff)
	}
	if c.TabText == (math.Color{}) {
		c.TabText = math.NewColor(0xe6, 0xe6, 0xef, 0xff)
	}
	if c.Hover == (math.Color{}) {
		c.Hover = math.NewColor(0x2f, 0x3b, 0x54, 0xff)
	}
	if c.Dropdown == (math.Color{}) {
		c.Dropdown = math.NewColor(0x1d, 0x21, 0x30, 0xff)
	}
	if c.BorderColor == (math.Color{}) {
		c.BorderColor = math.NewColor(0x3a, 0x42, 0x57, 0xff)
	}
	if c.ItemText == (math.Color{}) {
		c.ItemText = math.NewColor(0xe6, 0xe6, 0xef, 0xff)
	}
	if c.ItemHover == (math.Color{}) {
		c.ItemHover = math.NewColor(0x2f, 0x5d, 0x8a, 0xff)
	}
	if c.Dim == (math.Color{}) {
		c.Dim = math.NewColor(0x6b, 0x73, 0x85, 0xff)
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

	c.menus = []menu{
		{
			label: "File",
			items: []menuEntry{
				{label: "Open Project...", run: c.openProject},
				{label: "Save", hint: "Ctrl+S", run: c.save},
			},
		},
		{
			label: "Edit",
			items: []menuEntry{
				{label: "Undo", hint: "Ctrl+Z", run: c.undo},
				{label: "Redo", hint: "Ctrl+Y", run: c.redo},
				{label: "Editor Settings...", run: c.openEditorSettings},
			},
		},
		{
			label: "Game",
			items: []menuEntry{
				{label: "Game Settings...", run: c.openGameSettings},
			},
		},
		{
			label:  "Run",
			direct: true,
			items:  []menuEntry{{label: "Run", run: c.run, enabled: func() bool { return !c.running.Load() }}},
		},
		{
			label:  "Stop",
			direct: true,
			items:  []menuEntry{{label: "Stop", run: c.stop, enabled: func() bool { return c.running.Load() }}},
		},
	}

	// The menu bar is an opaque surface: it blocks pointer events so the @UIManager
	// occludes whatever is drawn behind it.
	if c.Blocking == nil {
		c.SetBlocking(true)
	}
}

// openMenuAt opens the given tab's dropdown and marks the menu as open.
func (c *MenuBarComponent) openMenuAt(i int) {
	c.openMenu = i
	c.hoverItem = -1
	menusOpenFlag = true
}

// activateTab handles a click on tab i: a direct tab runs its action immediately, a
// normal tab opens its dropdown.
func (c *MenuBarComponent) activateTab(i int) {
	if i < 0 || i >= len(c.menus) {
		return
	}
	m := c.menus[i]
	if m.direct {
		c.closeMenu()
		item := m.items[0]
		if (item.enabled == nil || item.enabled()) && item.run != nil {
			item.run()
		}
		return
	}
	c.openMenuAt(i)
}

// closeMenu closes the dropdown and clears the menu-open flag.
func (c *MenuBarComponent) closeMenu() {
	c.openMenu = -1
	c.hoverTab = -1
	c.hoverItem = -1
	menusOpenFlag = false
}

// tabAt returns the tab under pos, or -1.
func (c *MenuBarComponent) tabAt(pos math.Vector2) int {
	for i := range c.menus {
		if c.tabRect(i).ContainsPoint(pos) {
			return i
		}
	}
	return -1
}

// itemAt returns the menu entry under pos (screen space), or -1.
func (c *MenuBarComponent) itemAt(pos math.Vector2) int {
	if c.openMenu < 0 || c.openMenu >= len(c.menus) {
		return -1
	}
	dr := c.dropdownRect()
	if !dr.ContainsPoint(pos) {
		return -1
	}
	idx := int((pos.Y - dr.Top() - menuDropdownPad) / menuRowHeight)
	if idx < 0 || idx >= len(c.menus[c.openMenu].items) {
		return -1
	}
	return idx
}

// activateItem runs the clicked entry's action (if enabled) and closes the menu.
func (c *MenuBarComponent) activateItem(i int) {
	if c.openMenu < 0 || c.openMenu >= len(c.menus) {
		return
	}
	items := c.menus[c.openMenu].items
	if i < 0 || i >= len(items) {
		return
	}
	item := items[i]
	c.closeMenu()
	if item.enabled != nil && !item.enabled() {
		return
	}
	if item.run != nil {
		item.run()
	}
}

func (c *MenuBarComponent) Update(ctx *core.Context) {
	if ctx == nil || ctx.Input == nil {
		return
	}

	// Sample the frame loop for stalls (logged to the console). Runs every frame the
	// menu bar updates, before any modal/shortcut early return, so a freeze anywhere
	// in the editor still shows up here.
	perf.tick()

	// Install the OS window-close interceptor once. Closing (Alt+F4 / title-bar X)
	// with unsaved scene edits opens the save/don't-save/cancel prompt; a clean
	// document closes immediately.
	if !c.closeHookRegistered && ctx.Game != nil {
		game := ctx.Game
		game.SetCloseHandler(func() bool {
			// Flush the editor settings before any close path (clean quit or the
			// save/don't-save prompt). A later Cancel just re-saves on the next close.
			if vp := lookupViewport(c.GetScene()); vp != nil {
				vp.saveEditorPrefs()
			}
			if !history.isDirty() {
				return true
			}
			spawnCloseConfirm(c.GetScene(), game)
			return false
		})
		c.closeHookRegistered = true
	}

	// A RUN that exited on its own clears its handle here and reports it, so the
	// status line stops saying "running" and STOP becomes a no-op again.
	if c.runCmd != nil && !c.running.Load() {
		c.runCmd = nil
		c.status = "exited"
		c.statusError = false
	}

	// A modal is open (game settings / open-project / confirm): the menu bar is inert
	// and any open menu closes.
	if modalOpen() {
		c.closeMenu()
		return
	}

	// Undo/redo and save shortcuts (skipped while a managed widget holds keyboard focus).
	if c.handleUndoKeys(ctx.Input) {
		return
	}
	if ctx.Input.IsKeyPressed(core.KeyControl) && ctx.Input.IsKeyJustPressed(core.KeyS) {
		c.save()
		return
	}
	// F5 builds and runs the target project (the standard "play" hotkey).
	if ctx.Input.IsKeyJustPressed(core.KeyF5) {
		c.run()
		return
	}

	mouse := ctx.Input.GetMousePosition()
	rect := c.Rect()

	if c.openMenu >= 0 {
		// Menu open: track hover, and act only on a fresh left press.
		c.hoverItem = c.itemAt(mouse)
		c.hoverTab = -1
		if rect.ContainsPoint(mouse) {
			c.hoverTab = c.tabAt(mouse)
		}
		if !ctx.Input.IsMouseButtonJustPressed(core.MouseButtonLeft) {
			return
		}
		if i := c.itemAt(mouse); i >= 0 {
			c.activateItem(i)
			return
		}
		if rect.ContainsPoint(mouse) {
			if t := c.tabAt(mouse); t >= 0 {
				if t == c.openMenu {
					c.closeMenu()
				} else {
					c.activateTab(t)
				}
				return
			}
		}
		// Click outside the strip and dropdown: close, consuming the press.
		c.closeMenu()
		return
	}

	// Menu closed: hover a tab, open it on a fresh press.
	c.hoverItem = -1
	if rect.ContainsPoint(mouse) {
		c.hoverTab = c.tabAt(mouse)
	} else {
		c.hoverTab = -1
	}
	if !ctx.Input.IsMouseButtonJustPressed(core.MouseButtonLeft) {
		return
	}
	if c.hoverTab >= 0 {
		c.activateTab(c.hoverTab)
	}
}

func (c *MenuBarComponent) Draw(r core.Renderer) {
	rect := c.Rect()
	if rect.Width() <= 0 || rect.Height() <= 0 {
		return
	}

	r.SetClipRect(rect)
	r.DrawRect(rect, c.Background)

	_, th := r.MeasureText("Ag", c.FontID, c.FontSize)
	cy := rect.Y() + (rect.Height()-th)/2

	// Tabs, plus the direct Run/Stop buttons (which act immediately and never open a
	// menu). Direct buttons dim when their action is disabled.
	for i := range c.menus {
		m := c.menus[i]
		tr := c.tabRect(i)
		open := !m.direct && i == c.openMenu
		hovered := i == c.hoverTab
		enabled := !m.direct || len(m.items) == 0 || m.items[0].enabled == nil || m.items[0].enabled()
		if open || (hovered && enabled) {
			r.DrawRect(tr, c.Hover)
		}
		color := c.TabText
		if !enabled {
			color = c.Dim
		}
		r.DrawText(m.label, c.FontID, c.FontSize, math.NewVector2(tr.X()+menuItemPadX, cy), color)
	}

	// Status line (right side).
	if c.status != "" {
		sc := c.Success
		if c.statusError {
			sc = c.Error
		}
		sw, _ := r.MeasureText(c.status, c.FontID, c.FontSize)
		r.DrawText(c.status, c.FontID, c.FontSize, math.NewVector2(rect.X()+rect.Width()-sw-8, cy), sc)
	}

	r.ClearClip()

	// Dropdown, drawn outside the strip clip so it overlays the panels below.
	if c.openMenu >= 0 {
		c.drawDropdown(r)
	}
}

// drawDropdown draws the open menu's panel and entries with their hover/hint state.
func (c *MenuBarComponent) drawDropdown(r core.Renderer) {
	if c.openMenu < 0 || c.openMenu >= len(c.menus) {
		return
	}
	dr := c.dropdownRect()
	r.SetClipRect(dr)
	r.DrawRect(dr, c.Dropdown)
	r.DrawRectOutline(dr, c.BorderColor, 1)

	items := c.menus[c.openMenu].items
	for i := range items {
		item := items[i]
		ir := c.itemRect(i)
		enabled := item.enabled == nil || item.enabled()
		if i == c.hoverItem && enabled {
			r.DrawRect(ir, c.ItemHover)
		}
		color := c.ItemText
		if !enabled {
			color = c.Dim
		}
		_, th := r.MeasureText(item.label, c.FontID, c.FontSize)
		r.DrawText(item.label, c.FontID, c.FontSize, math.NewVector2(ir.X()+menuItemPadX, ir.Center().Y-th/2), color)
		if item.hint != "" {
			hw, _ := r.MeasureText(item.hint, c.FontID, c.FontSize)
			r.DrawText(item.hint, c.FontID, c.FontSize, math.NewVector2(ir.Right()-hw-menuItemPadX, ir.Center().Y-th/2), c.Dim)
		}
	}

	r.ClearClip()
}

// openProject opens the modal project-path input.
func (c *MenuBarComponent) openProject() {
	spawnOpenProject(c.GetScene())
}

// openGameSettings opens the modal game.imge editor.
func (c *MenuBarComponent) openGameSettings() {
	spawnGameSettings(c.GetScene())
}

// openEditorSettings opens the modal editor-only settings window (grid spacing, …).
func (c *MenuBarComponent) openEditorSettings() {
	spawnEditorSettings(c.GetScene())
}

// undo/redo apply the editor history and report the result in the status line.
func (c *MenuBarComponent) undo() {
	if history.undo() {
		c.status = "undone"
	} else {
		c.status = "nothing to undo"
	}
	c.statusError = false
}

func (c *MenuBarComponent) redo() {
	if history.redo() {
		c.status = "redone"
	} else {
		c.status = "nothing to redo"
	}
	c.statusError = false
}

// save writes the loaded target scene back to its .scene file and reports the result.
func (c *MenuBarComponent) save() {
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
func (c *MenuBarComponent) run() {
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

// stop kills the running project's process group.
func (c *MenuBarComponent) stop() {
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
// redoes. It is skipped while any managed widget holds keyboard focus. Returns true when
// it consumed a shortcut.
func (c *MenuBarComponent) handleUndoKeys(in core.Input) bool {
	if !in.IsKeyPressed(core.KeyControl) {
		return false
	}
	if mgr := lookupUIManager(c.GetScene()); mgr != nil && mgr.HasFocus() {
		return false
	}
	zDown := in.IsKeyJustPressed(core.KeyZ)
	yDown := in.IsKeyJustPressed(core.KeyY)
	redo := yDown || (zDown && in.IsKeyPressed(core.KeyShift))
	undo := zDown && !in.IsKeyPressed(core.KeyShift)
	if !undo && !redo {
		return false
	}
	if redo {
		c.redo()
	} else {
		c.undo()
	}
	return true
}

// resolveCLI returns the path to the imge CLI: the explicit CLI field, else the
// IMGE_CLI env var (set by `imge editor`), else `imge` on PATH.
func (c *MenuBarComponent) resolveCLI() (string, error) {
	if c.CLI != "" {
		return c.CLI, nil
	}
	if v := os.Getenv("IMGE_CLI"); v != "" {
		return v, nil
	}
	return exec.LookPath("imge")
}
