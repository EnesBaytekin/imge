package components

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/EnesBaytekin/imge/core"
	imgejson "github.com/EnesBaytekin/imge/core/json"
	"github.com/EnesBaytekin/imge/core/math"
)

// ProjectPickerComponent is the modal filesystem browser opened by File → "Open
// Project..." and File → "New Project...". Unlike FileBrowserComponent (which shows
// the target project's own tree), it browses absolute filesystem paths so the user
// can pick a project anywhere on the machine: it starts in the current project's
// directory (or the home directory when no project is loaded) and navigates one
// directory at a time. Two modes share the window:
//
//   - open: directories are entered, and selecting the game.imge file (or pressing
//     Open while the current folder holds one) loads that directory as the project.
//   - new: the same directory navigation, plus a game-name text field and a
//     "create a folder with this name" checkbox. Create writes a blank project
//     (game.imge + scenes/main.scene) into <folder>/<name> (or straight into the
//     folder when the checkbox is off) and loads it.
//
// A directory that can't be read (permissions) is caught and shown as an error line
// instead of crashing, with the ".." row always available to climb back out.
//
// It is modal: while open the other hand-rolled panels yield (see modalOpen) and an
// outside click dismisses it. The list is hand-drawn; the name field, checkbox, and
// buttons are engine widget children, polled like the other modal panels.
type ProjectPickerComponent struct {
	core.BaseUIComponent

	Background  math.Color `json:"background"`
	TitleText   math.Color `json:"title_text"`
	DirText     math.Color `json:"dir_text"`  // directory rows
	FileText    math.Color `json:"file_text"` // file rows
	Dim         math.Color `json:"dim"`       // non-selectable files / path label
	Accent      math.Color `json:"accent"`    // title bar + selected row
	BorderColor math.Color `json:"border_color"`
	ScrollTrack math.Color `json:"scroll_track"`
	ScrollThumb math.Color `json:"scroll_thumb"`
	ErrorColor  math.Color `json:"error_color"` // error line

	FontID    string  `json:"font_id"`
	FontSize  float64 `json:"font_size"`
	RowHeight float64 `json:"row_height"`

	mode string // pickerOpen or pickerNew

	dir      string     // current directory (absolute)
	entries  []dirEntry // entries in dir (dirs first, then files)
	dirError string     // os.ReadDir failure for the current directory

	cursor int // index into rows (0 = "..", then entries)
	scroll float64

	nameInput        *TextInputComponent
	createDir        *CheckBoxComponent
	createDirChecked bool

	filterInput    *TextInputComponent
	filter         string
	newFolderInput *TextInputComponent
	newFolderBtn   *ButtonComponent

	cancel *ButtonComponent
	ok     *ButtonComponent

	errorText string // action error (bad name, no game.imge, create failure)
	dismiss   bool
	centered  bool

	dragging   bool
	dragGrab   math.Vector2
	closeHover bool
}

// Picker modes.
const (
	pickerOpen = "open"
	pickerNew  = "new"
)

// dirEntry is one row in the current directory listing.
type dirEntry struct {
	name  string
	isDir bool
}

// projectFileName is the single file that marks a directory as an IMGE project.
const projectFileName = "game.imge"

func (c *ProjectPickerComponent) titleH() float64 { return 18 }

func (c *ProjectPickerComponent) listH() float64 { return 180 }

// searchRect is the filter input above the listing.
func (c *ProjectPickerComponent) searchRect(rect math.Rect) math.Rect {
	return math.NewRect(rect.X()+8, rect.Y()+c.titleH()+16, rect.Width()-16, 20)
}

// listRect is the scrollable directory-listing area, below the search bar.
func (c *ProjectPickerComponent) listRect(rect math.Rect) math.Rect {
	sr := c.searchRect(rect)
	return math.NewRect(rect.X()+8, sr.Y()+24, rect.Width()-16, c.listH())
}

// pathLabelRect is the small line under the title showing the current directory.
func (c *ProjectPickerComponent) pathLabelRect(rect math.Rect) math.Rect {
	return math.NewRect(rect.X()+8, rect.Y()+c.titleH()+3, rect.Width()-16, 13)
}

// rows returns the rendered rows: a synthetic ".." parent entry first, then the
// directory's real entries.
func (c *ProjectPickerComponent) rows() []dirEntry {
	rows := make([]dirEntry, 0, len(c.entries)+1)
	rows = append(rows, dirEntry{name: "..", isDir: true})
	for _, e := range c.entries {
		if c.filter != "" && !strings.Contains(strings.ToLower(e.name), strings.ToLower(c.filter)) {
			continue
		}
		rows = append(rows, e)
	}
	return rows
}

// isSelectable reports whether a row is actionable: the parent, any directory, or
// (in open mode) the game.imge file. Other files are shown but inert.
func (c *ProjectPickerComponent) isSelectable(e dirEntry) bool {
	if e.isDir {
		return true
	}
	return c.mode == pickerOpen && e.name == projectFileName
}

func (c *ProjectPickerComponent) Initialize() {
	if c.Background == (math.Color{}) {
		c.Background = math.NewColor(0x1d, 0x21, 0x30, 0xff)
	}
	if c.TitleText == (math.Color{}) {
		c.TitleText = math.NewColor(0xff, 0xff, 0xff, 0xff)
	}
	if c.DirText == (math.Color{}) {
		c.DirText = math.NewColor(0x9f, 0xc7, 0xff, 0xff)
	}
	if c.FileText == (math.Color{}) {
		c.FileText = math.NewColor(0xe6, 0xe6, 0xef, 0xff)
	}
	if c.Dim == (math.Color{}) {
		c.Dim = math.NewColor(0x6b, 0x73, 0x85, 0xff)
	}
	if c.Accent == (math.Color{}) {
		c.Accent = math.NewColor(0x2f, 0x3b, 0x54, 0xff)
	}
	if c.BorderColor == (math.Color{}) {
		c.BorderColor = math.NewColor(0x3a, 0x42, 0x57, 0xff)
	}
	if c.ScrollTrack == (math.Color{}) {
		c.ScrollTrack = math.NewColor(0x2a, 0x30, 0x42, 0xff)
	}
	if c.ScrollThumb == (math.Color{}) {
		c.ScrollThumb = math.NewColor(0x4a, 0x55, 0x70, 0xff)
	}
	if c.ErrorColor == (math.Color{}) {
		c.ErrorColor = math.NewColor(0xff, 0x5a, 0x5a, 0xff)
	}
	if c.FontSize <= 0 {
		c.FontSize = 6
	}
	if c.RowHeight <= 0 {
		c.RowHeight = 14
	}
	if c.Blocking == nil {
		c.SetBlocking(true)
	}
}

// spawnProjectPicker opens the filesystem picker in the given mode. It is a no-op when
// a modal is already open or there is no viewport. The starting directory is the
// current project's directory, or the home directory when no project is loaded.
func spawnProjectPicker(scene *core.Scene, mode string) {
	if scene == nil || modalOpen() {
		return
	}
	if lookupViewport(scene) == nil {
		return
	}
	start := ""
	if vp := lookupViewport(scene); vp != nil {
		start = vp.CurrentProject()
	}
	if start == "" {
		start, _ = os.UserHomeDir()
	}
	if start == "" {
		start = string(filepath.Separator)
	}

	obj := core.NewObject("project_picker")
	obj.UI = true
	obj.Layer = 3
	obj.Transform.Position = math.NewVector2(160, 80) // centered lazily on first Update

	win := &ProjectPickerComponent{}
	win.SetName("project_picker")
	win.Width = 480
	win.Height = 400
	win.mode = mode
	win.dir = filepath.Clean(start)
	win.createDirChecked = true
	obj.AddComponent(win)

	if err := scene.AddObject(obj); err != nil {
		return
	}

	win.Initialize()
	win.reload()
	win.buildWidgets()
	setModal(win)
	raiseToFront(scene, obj)
}

// reload re-reads the current directory, keeping ".." available even on failure so the
// user can always climb back out of an unreadable directory.
func (c *ProjectPickerComponent) reload() {
	des, err := os.ReadDir(c.dir)
	if err != nil {
		c.entries = nil
		c.dirError = err.Error()
		c.cursor = 0
		c.scroll = 0
		return
	}
	c.dirError = ""
	sort.Slice(des, func(i, j int) bool {
		a, b := des[i], des[j]
		if a.IsDir() != b.IsDir() {
			return a.IsDir()
		}
		return strings.ToLower(a.Name()) < strings.ToLower(b.Name())
	})
	out := make([]dirEntry, 0, len(des))
	for _, d := range des {
		name := d.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}
		out = append(out, dirEntry{name: name, isDir: d.IsDir()})
	}
	c.entries = out
	c.cursor = 0
	c.scroll = 0
}

// buildWidgets creates the mode-specific widgets (name field + checkbox for new mode)
// and the shared Cancel/Open-Create buttons, all children of this window object.
func (c *ProjectPickerComponent) buildWidgets() {
	owner := c.GetOwner()
	btnY := c.Height - 30

	// Filter bar (both modes): filters the directory listing as you type.
	fi := &TextInputComponent{}
	fi.FontID = c.FontID
	fi.Size = c.FontSize
	fi.TextColor = c.TitleText
	fi.PlaceholderColor = c.Dim
	fi.BackgroundColor = fieldBackground
	fi.OutlineColor = fieldOutline
	fi.OutlineThickness = 1
	fi.Placeholder = "search..."
	fi.Width = c.Width - 16
	fi.Height = 20
	fi.DrawLayer = 1
	fi.SetName("filter")
	fi.SetOffset(math.NewVector2(8, 34))
	owner.AddComponent(fi)
	fi.Initialize()
	c.filterInput = fi

	// New-folder row (both modes): a folder-name field plus a button that creates
	// the directory and navigates into it.
	nfi := &TextInputComponent{}
	nfi.FontID = c.FontID
	nfi.Size = c.FontSize
	nfi.TextColor = c.TitleText
	nfi.PlaceholderColor = c.Dim
	nfi.BackgroundColor = fieldBackground
	nfi.OutlineColor = fieldOutline
	nfi.OutlineThickness = 1
	nfi.Placeholder = "folder name..."
	nfi.Width = c.Width - 108
	nfi.Height = 20
	nfi.DrawLayer = 1
	nfi.SetName("new_folder_name")
	nfi.SetOffset(math.NewVector2(8, 242))
	owner.AddComponent(nfi)
	nfi.Initialize()
	c.newFolderInput = nfi

	c.newFolderBtn = makePanelButton(owner, "new_folder", "New Folder", math.NewVector2(c.Width-92, 242), 84, 20, c.FontID, c.FontSize, c.BorderColor)

	if c.mode == pickerNew {
		ti := &TextInputComponent{}
		ti.FontID = c.FontID
		ti.Size = c.FontSize
		ti.TextColor = c.TitleText
		ti.PlaceholderColor = c.Dim
		ti.BackgroundColor = fieldBackground
		ti.OutlineColor = fieldOutline
		ti.OutlineThickness = 1
		ti.Placeholder = "game name..."
		ti.Width = c.Width - 16
		ti.Height = 20
		ti.DrawLayer = 1
		ti.SetName("name")
		ti.SetOffset(math.NewVector2(8, 268))
		owner.AddComponent(ti)
		ti.Initialize()
		ti.Text = "My Game"
		c.nameInput = ti

		cb := &CheckBoxComponent{}
		cb.Text = "create a folder with this name"
		cb.FontID = c.FontID
		cb.Size = c.FontSize
		cb.TextColor = c.FileText
		cb.BoxSize = 16
		cb.DrawLayer = 1
		cb.SetName("create_dir")
		cb.SetOffset(math.NewVector2(8, 294))
		owner.AddComponent(cb)
		cb.Initialize()
		cb.SetChecked(true)
		c.createDir = cb
	}

	c.cancel = makePanelButton(owner, "cancel", "Cancel", math.NewVector2(c.Width-200, btnY), 90, 22, c.FontID, c.FontSize, c.BorderColor)
	okLabel := "Open"
	if c.mode == pickerNew {
		okLabel = "Create"
	}
	c.ok = makePanelButton(owner, "ok", okLabel, math.NewVector2(c.Width-100, btnY), 92, 22, c.FontID, c.FontSize, c.Accent)
}

// targetDir returns the directory a new project would be created in: <dir>/<name> when
// the create-folder checkbox is on, otherwise <dir> itself.
func (c *ProjectPickerComponent) targetDir() string {
	if c.mode != pickerNew || c.nameInput == nil {
		return c.dir
	}
	name := strings.TrimSpace(c.nameInput.Text)
	if !c.createDirChecked || name == "" {
		return c.dir
	}
	return filepath.Join(c.dir, name)
}

// createBlankProject writes an empty project (game.imge + scenes/main.scene) into dir,
// naming the game `name`. It mirrors `imge init`'s blank template using the JSON
// config package directly (the editor can't import the CLI's template embedder).
func createBlankProject(dir, name string) error {
	if err := os.MkdirAll(filepath.Join(dir, "scenes"), 0o755); err != nil {
		return err
	}
	cfg := imgejson.DefaultGameConfig()
	if name != "" {
		cfg.Name = name
	}
	if err := imgejson.SaveGameConfig(cfg, filepath.Join(dir, projectFileName)); err != nil {
		return err
	}
	return createSceneFile(dir, "main")
}

// openCurrent opens the current directory as a project, failing (with an error line)
// when it does not contain a game.imge.
func (c *ProjectPickerComponent) openCurrent() {
	if _, err := os.Stat(filepath.Join(c.dir, projectFileName)); err != nil {
		c.errorText = "no " + projectFileName + " in this folder"
		return
	}
	c.loadProject(c.dir)
}

// loadProject switches the viewport to dir and closes this window. SetProject tears
// down the modal (this window) and clears the undo history — the only thing that does.
func (c *ProjectPickerComponent) loadProject(dir string) {
	if vp := lookupViewport(c.GetScene()); vp != nil {
		vp.SetProject(dir)
	}
	c.dismiss = true
}

// navigateInto enters a subdirectory of the current one.
func (c *ProjectPickerComponent) navigateInto(name string) {
	next := filepath.Join(c.dir, name)
	if info, err := os.Stat(next); err != nil || !info.IsDir() {
		c.errorText = "cannot open " + name
		return
	}
	c.dir = next
	c.errorText = ""
	c.reload()
}

// navigateUp climbs to the parent directory, if there is one.
func (c *ProjectPickerComponent) navigateUp() {
	parent := filepath.Dir(c.dir)
	if parent == c.dir {
		return // already at the filesystem root
	}
	c.dir = parent
	c.errorText = ""
	c.reload()
}

// createNewFolder creates a new directory inside the current one from the folder-name
// field, then navigates into it (matching "make a folder, then work inside it").
func (c *ProjectPickerComponent) createNewFolder() {
	name := strings.TrimSpace(c.newFolderInput.Text)
	switch {
	case name == "":
		c.errorText = "folder name is required"
		return
	case strings.ContainsAny(name, `/\`):
		c.errorText = "name can't contain / or \\"
		return
	}
	path := filepath.Join(c.dir, name)
	if err := os.Mkdir(path, 0o755); err != nil {
		if os.IsExist(err) {
			c.errorText = "folder already exists"
		} else {
			c.errorText = err.Error()
		}
		return
	}
	c.errorText = ""
	c.newFolderInput.Text = ""
	c.dir = path
	c.reload()
}

func (c *ProjectPickerComponent) contentHeight() float64 { return float64(len(c.rows())) * c.RowHeight }

func (c *ProjectPickerComponent) maxScroll(rect math.Rect) float64 {
	if m := c.contentHeight() - c.listRect(rect).Height(); m > 0 {
		return m
	}
	return 0
}

func (c *ProjectPickerComponent) clampScroll(rect math.Rect) {
	if max := c.maxScroll(rect); c.scroll > max {
		c.scroll = max
	}
	if c.scroll < 0 {
		c.scroll = 0
	}
}

// rowAt returns the row index under mouseY (screen space), or -1.
func (c *ProjectPickerComponent) rowAt(rect math.Rect, mouseY float64) int {
	lr := c.listRect(rect)
	rows := c.rows()
	for i := range rows {
		y := lr.Y() + float64(i)*c.RowHeight - c.scroll
		if mouseY >= y && mouseY < y+c.RowHeight {
			return i
		}
	}
	return -1
}

// activateRow acts on a row: ".." climbs up, a directory is entered, and (in open mode)
// game.imge loads the project. Non-selectable files are ignored.
func (c *ProjectPickerComponent) activateRow(i int) {
	rows := c.rows()
	if i < 0 || i >= len(rows) {
		return
	}
	e := rows[i]
	c.errorText = ""
	switch {
	case e.name == "..":
		c.navigateUp()
	case e.isDir:
		c.navigateInto(e.name)
	case c.mode == pickerOpen && e.name == projectFileName:
		c.loadProject(c.dir)
	}
}

// moveCursor shifts the cursor by delta rows, keeping it in view.
func (c *ProjectPickerComponent) moveCursor(rect math.Rect, delta int) {
	rows := c.rows()
	if len(rows) == 0 {
		return
	}
	idx := c.cursor
	if idx < 0 || idx >= len(rows) {
		idx = 0
	}
	idx += delta
	if idx < 0 {
		idx = 0
	}
	if idx >= len(rows) {
		idx = len(rows) - 1
	}
	c.cursor = idx
	// Scroll the cursor row into view.
	lr := c.listRect(rect)
	y := lr.Y() + float64(idx)*c.RowHeight - c.scroll
	if y < lr.Y() {
		c.scroll = float64(idx) * c.RowHeight
	} else if y+c.RowHeight > lr.Y()+lr.Height() {
		c.scroll = float64(idx)*c.RowHeight + c.RowHeight - lr.Height()
	}
	c.clampScroll(rect)
}

func (c *ProjectPickerComponent) Update(ctx *core.Context) {
	if ctx == nil || ctx.Input == nil {
		return
	}
	c.centerOnce(ctx)

	mouse := ctx.Input.GetMousePosition()
	rect := c.Rect()

	if c.dragging {
		if ctx.Input.IsMouseButtonPressed(core.MouseButtonLeft) {
			c.GetOwner().SetPosition(mouse.X-c.dragGrab.X, mouse.Y-c.dragGrab.Y)
		} else {
			c.dragging = false
		}
	}

	// Poll the new-mode checkbox state each frame.
	if c.createDir != nil {
		c.createDirChecked = c.createDir.GetChecked()
	}

	// Poll the filter field each frame; reset the cursor when it changes.
	if c.filterInput != nil {
		f := c.filterInput.Text
		if f != c.filter {
			c.filter = f
			c.cursor = 0
			c.scroll = 0
		}
	}

	// Buttons.
	if c.cancel != nil && c.cancel.ConsumeClick() {
		c.dismiss = true
		return
	}
	if c.ok != nil && c.ok.ConsumeClick() {
		if c.mode == pickerNew {
			c.commitNew()
		} else {
			c.openCurrent()
		}
		if c.dismiss {
			return
		}
	}
	if c.newFolderBtn != nil && c.newFolderBtn.ConsumeClick() {
		c.createNewFolder()
	}

	// Enter in the name field creates (new mode), matching a typed-in name.
	if c.nameInput != nil && c.nameInput.IsFocused() && ctx.Input.IsKeyJustPressed(core.KeyEnter) {
		c.commitNew()
		if c.dismiss {
			return
		}
	}

	// Enter in the folder-name field creates a directory and enters it.
	if c.newFolderInput != nil && c.newFolderInput.IsFocused() && ctx.Input.IsKeyJustPressed(core.KeyEnter) {
		c.createNewFolder()
	}

	// Keyboard list navigation, only when no text field holds focus.
	kbFree := (c.nameInput == nil || !c.nameInput.IsFocused()) &&
		(c.filterInput == nil || !c.filterInput.IsFocused()) &&
		(c.newFolderInput == nil || !c.newFolderInput.IsFocused())
	if kbFree {
		if ctx.Input.IsKeyJustPressed(core.KeyUp) {
			c.moveCursor(rect, -1)
		}
		if ctx.Input.IsKeyJustPressed(core.KeyDown) {
			c.moveCursor(rect, 1)
		}
		if ctx.Input.IsKeyJustPressed(core.KeyEnter) {
			c.activateRow(c.cursor)
		}
		if ctx.Input.IsKeyJustPressed(core.KeyBackspace) {
			c.navigateUp()
		}
	}

	// Wheel scrolls the list.
	if s := ctx.Input.GetMouseScroll(); s.Y != 0 && c.listRect(rect).ContainsPoint(mouse) {
		c.scroll -= s.Y * c.RowHeight * 2
		c.clampScroll(rect)
	}

	c.closeHover = c.closeRect(rect).ContainsPoint(mouse)

	if !ctx.Input.IsMouseButtonJustPressed(core.MouseButtonLeft) {
		return
	}

	// Close button.
	if c.closeHover {
		c.dismiss = true
		return
	}

	// List row click.
	if i := c.rowAt(rect, mouse.Y); i >= 0 && c.listRect(rect).ContainsPoint(mouse) {
		rows := c.rows()
		if i < len(rows) && c.isSelectable(rows[i]) {
			c.cursor = i
			c.activateRow(i)
		}
		return
	}

	// Title-bar press starts a drag.
	if math.NewRect(rect.X(), rect.Y(), rect.Width(), c.titleH()).ContainsPoint(mouse) {
		c.dragging = true
		c.dragGrab = mouse.Subtract(rect.Position)
		return
	}

	if modalOutsideClick(c.GetScene(), c.GetOwner(), ctx) {
		c.dismiss = true
	}
}

// commitNew validates the name and creates + loads the blank project.
func (c *ProjectPickerComponent) commitNew() {
	name := strings.TrimSpace(c.nameInput.Text)
	switch {
	case name == "":
		c.errorText = "name is required"
		return
	case strings.ContainsAny(name, `/\`):
		c.errorText = "name can't contain / or \\"
		return
	}
	target := c.targetDir()
	if err := createBlankProject(target, name); err != nil {
		c.errorText = err.Error()
		return
	}
	c.loadProject(target)
}

func (c *ProjectPickerComponent) closeRect(rect math.Rect) math.Rect {
	const s = 12.0
	return math.NewRect(rect.X()+rect.Width()-s-4, rect.Y()+(c.titleH()-s)/2, s, s)
}

func (c *ProjectPickerComponent) centerOnce(ctx *core.Context) {
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

func (c *ProjectPickerComponent) Draw(r core.Renderer) {
	rect := c.Rect()
	if rect.Width() <= 0 || rect.Height() <= 0 {
		return
	}
	r.SetClipRect(rect)
	r.DrawRect(rect, c.Background)
	r.DrawRectOutline(rect, c.BorderColor, 1)

	_, th := r.MeasureText("Ag", c.FontID, c.FontSize)

	// Title bar + close "x".
	r.DrawRect(math.NewRect(rect.X(), rect.Y(), rect.Width(), c.titleH()), c.Accent)
	title := "OPEN PROJECT"
	if c.mode == pickerNew {
		title = "NEW PROJECT"
	}
	r.DrawText(title, c.FontID, c.FontSize, math.NewVector2(rect.X()+6, rect.Y()+(c.titleH()-th)/2), c.TitleText)
	cr := c.closeRect(rect)
	if c.closeHover {
		r.DrawRect(cr, c.Background.Lerp(math.White, 0.12))
	}
	xw, xh := r.MeasureText("x", c.FontID, c.FontSize)
	r.DrawText("x", c.FontID, c.FontSize, math.NewVector2(cr.X()+(cr.Width()-xw)/2, cr.Y()+(cr.Height()-xh)/2), c.TitleText)

	// Current-directory label.
	pl := c.pathLabelRect(rect)
	r.DrawText(c.dir, c.FontID, c.FontSize, math.NewVector2(pl.X(), pl.Y()), c.Dim)

	// Directory listing.
	lr := c.listRect(rect)
	r.DrawRect(lr, c.Background.Lerp(math.White, 0.04))
	r.SetClipRect(lr)
	rows := c.rows()
	top := lr.Y()
	for i, e := range rows {
		y := top + float64(i)*c.RowHeight - c.scroll
		if y+c.RowHeight < lr.Y() || y > lr.Y()+lr.Height() {
			continue
		}
		if i == c.cursor {
			r.DrawRect(math.NewRect(lr.X(), y, lr.Width(), c.RowHeight), c.Accent)
		}
		label := e.name
		color := c.FileText
		if e.name == ".." {
			label = "▸ .."
			color = c.DirText
		} else if e.isDir {
			label = "▸ " + e.name
			color = c.DirText
		} else if !c.isSelectable(e) {
			color = c.Dim
		} else {
			label = "▸ " + e.name
			color = c.FileText
		}
		ty := y + (c.RowHeight-th)/2
		if ty < y {
			ty = y
		}
		r.DrawText(label, c.FontID, c.FontSize, math.NewVector2(lr.X()+4, ty), color)
	}
	r.SetClipRect(rect)
	if thumb, ok := scrollThumb(c.scrollTrack(rect), c.contentHeight(), c.scroll, c.maxScroll(rect)); ok {
		drawScrollbar(r, c.scrollTrack(rect), thumb, c.ScrollTrack, c.ScrollThumb)
	}

	// New-mode extras: the live target path preview sits under the name/checkbox.
	if c.mode == pickerNew {
		ty := rect.Y() + 316
		r.DrawText("create in: "+c.targetDir(), c.FontID, c.FontSize, math.NewVector2(rect.X()+8, ty), c.Dim)
	}

	// Error line (read failure or action error).
	errText := c.dirError
	if errText == "" {
		errText = c.errorText
	}
	if errText != "" {
		ey := rect.Y() + 268
		if c.mode == pickerNew {
			ey = rect.Y() + 332
		}
		r.DrawText(errText, c.FontID, c.FontSize, math.NewVector2(rect.X()+8, ey), c.ErrorColor)
	}

	r.ClearClip()

	if c.dismiss {
		c.closeSelf()
	}
}

func (c *ProjectPickerComponent) scrollTrack(rect math.Rect) math.Rect {
	lr := c.listRect(rect)
	const sbW = 6.0
	return math.NewRect(lr.X()+lr.Width()-sbW-2, lr.Y(), sbW, lr.Height())
}

// closeSelf clears the modal state and destroys the window's object.
func (c *ProjectPickerComponent) closeSelf() {
	clearModal()
	if owner := c.GetOwner(); owner != nil {
		owner.Destroy()
	}
}
