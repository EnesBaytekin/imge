package components

import (
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/EnesBaytekin/imge/core"
	"github.com/EnesBaytekin/imge/core/math"
)

// consoleMaxLines bounds the captured RUN/build output so a long build can't grow the
// log unbounded. Only the most recent lines are kept (a ring buffer).
const consoleMaxLines = 500

// consolePad is the horizontal inset for the console's text and Copy button, shared by
// Draw and Update so hit-testing never drifts from the drawn layout.
const consolePad = 4.0

// consoleLine is one captured log line with a stable identity. seq is a monotonically
// increasing id, so a text selection can reference lines even as the ring buffer drops
// old lines (indices shift, ids don't).
type consoleLine struct {
	seq  uint64
	text string
}

// consoleLog is the editor-wide capture of RUN/build stdout+stderr and editor log
// lines. The toolbar pipes the launched `imge run` process into it (on a goroutine)
// while the console panel reads it on the game loop, so it is mutex-guarded. It lives
// at package level (like the undo history) because the writer (toolbar) and reader
// (console panel) are different components in the same package.
type consoleLog struct {
	mu      sync.Mutex
	lines   []consoleLine
	partial string // trailing bytes of an in-progress line
	nextSeq uint64
}

var console consoleLog

// Write implements io.Writer; it appends p to the log, splitting on newlines and
// buffering any trailing partial line. It always reports the full length written so
// it can tee into the terminal via io.MultiWriter.
func (c *consoleLog) Write(p []byte) (int, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.partial += string(p)
	for {
		i := strings.IndexByte(c.partial, '\n')
		if i < 0 {
			break
		}
		c.push(c.partial[:i])
		c.partial = c.partial[i+1:]
	}
	return len(p), nil
}

// Print appends a whole log line (used for editor actions like run/stop).
func (c *consoleLog) Print(s string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, line := range strings.Split(s, "\n") {
		c.push(line)
	}
}

// Flush emits any trailing partial line as a final line. Called when a run ends so a
// final unterminated line of output isn't dropped.
func (c *consoleLog) Flush() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.partial != "" {
		c.push(c.partial)
		c.partial = ""
	}
}

// Clear drops all captured lines.
func (c *consoleLog) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.lines = nil
	c.partial = ""
}

func (c *consoleLog) push(line string) {
	c.nextSeq++
	c.lines = append(c.lines, consoleLine{seq: c.nextSeq, text: line})
	if len(c.lines) > consoleMaxLines {
		c.lines = c.lines[len(c.lines)-consoleMaxLines:]
	}
}

// Snapshot returns a copy of the captured lines (newest last) so the console panel can
// render without holding the lock during drawing.
func (c *consoleLog) Snapshot() []consoleLine {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]consoleLine, len(c.lines))
	copy(out, c.lines)
	return out
}

// consoleCell is a caret position in the console text: a line (by seq) plus a rune
// offset into it. Selections are stored in these stable coordinates rather than line
// indices, so a selection survives scrolling and ring-buffer churn.
type consoleCell struct {
	seq  uint64
	rune int
}

// less orders cells by line, then by rune, for normalizing a selection's endpoints.
func (a consoleCell) less(b consoleCell) bool {
	if a.seq != b.seq {
		return a.seq < b.seq
	}
	return a.rune < b.rune
}

// consoleRow is one visual (wrapped) row: a fragment of a logical line plus the rune
// offset where that fragment starts, so a hit-test can map a mouse position back to a
// stable consoleCell.
type consoleRow struct {
	text      string
	seq       uint64
	startRune int
}

// ConsoleComponent is the bottom strip that shows captured RUN/build output plus a few
// editor log lines. It renders the most recent lines from the shared console log,
// following new output unless the user scrolls back with the wheel (scroll = lines back
// from the newest; 0 = follow the bottom). Text is selectable with a mouse drag (the
// selection is drawn as a highlight), and can be copied to the system clipboard with
// Ctrl+C or the Copy button; Ctrl+A selects everything.
type ConsoleComponent struct {
	core.BaseUIComponent

	Background  math.Color `json:"background"`
	BorderColor math.Color `json:"border_color"`
	Text        math.Color `json:"text"`
	Dim         math.Color `json:"dim"`
	Selection   math.Color `json:"selection"` // highlight behind selected text

	FontID    string  `json:"font_id"`
	FontSize  float64 `json:"font_size"`
	RowHeight float64 `json:"row_height"`

	scroll int // lines scrolled back from the newest line

	// Selection state, in (seq, rune) coordinates over the console snapshot. lastRows is
	// the top-to-bottom visual row list built in Draw and reused by Update's hit-testing
	// on the next frame (one frame of lag, imperceptible).
	selAnchor consoleCell
	selCaret  consoleCell
	selActive bool // a drag is in progress
	hasSel    bool // a non-empty selection exists (Copy enabled, Ctrl+C copies)
	copyHover bool // the Copy button is under the cursor
	lastRows  []consoleRow
}

func (c *ConsoleComponent) Initialize() {
	if c.Background == (math.Color{}) {
		c.Background = math.NewColor(0x10, 0x13, 0x1c, 0xff)
	}
	if c.BorderColor == (math.Color{}) {
		c.BorderColor = math.NewColor(0x3a, 0x42, 0x57, 0xff)
	}
	if c.Text == (math.Color{}) {
		c.Text = math.NewColor(0xc9, 0xcf, 0xdd, 0xff)
	}
	if c.Dim == (math.Color{}) {
		c.Dim = math.NewColor(0x6b, 0x73, 0x85, 0xff)
	}
	if c.Selection == (math.Color{}) {
		c.Selection = math.NewColor(0x2f, 0x5d, 0x8a, 0xa0)
	}
	if c.FontSize <= 0 {
		c.FontSize = 6
	}
	if c.RowHeight <= 0 {
		c.RowHeight = 11
	}
	// The console is an opaque surface: it blocks pointer events so the @UIManager
	// occludes whatever is drawn behind it (see pointerOwnedElsewhere).
	if c.Blocking == nil {
		c.SetBlocking(true)
	}
}

func (c *ConsoleComponent) Update(ctx *core.Context) {
	if ctx == nil || ctx.Input == nil {
		return
	}
	// A modal or an open menu bar is up: this panel is inert.
	if modalOpen() || menusOpen() {
		return
	}
	in := ctx.Input
	mouse := in.GetMousePosition()
	rect := c.Rect()

	// Keyboard: Ctrl+A selects all, Ctrl+C copies the selection. These act on the
	// console's selection even when the pointer is elsewhere (a selection made by drag
	// stays copyable after the mouse moves off the panel). Skipped while a managed
	// widget holds keyboard focus, matching the editor's undo/save shortcuts.
	if mgr := lookupUIManager(c.GetScene()); mgr == nil || !mgr.HasFocus() {
		if in.IsKeyPressed(core.KeyControl) {
			if in.IsKeyJustPressed(core.KeyA) {
				c.selectAll()
				return
			}
			if in.IsKeyJustPressed(core.KeyC) && c.hasSel {
				c.copySelection()
				return
			}
		}
	}

	c.reconcileSelection()

	over := rect.ContainsPoint(mouse)
	owned := over && pointerOwnedElsewhere(c.GetScene(), c.GetOwner(), mouse)

	// Copy button: hover + click (dimmed/disabled when there is no selection).
	btn := c.copyButtonRect()
	c.copyHover = over && !owned && btn.ContainsPoint(mouse)
	if c.copyHover && in.IsMouseButtonJustPressed(core.MouseButtonLeft) {
		if c.hasSel {
			c.copySelection()
		}
		return
	}

	// Selection drag over the text: a fresh press anchors, holding extends the caret,
	// releasing ends the drag. Dragging continues even after the pointer leaves the
	// panel (cellAt just clamps to the last valid row).
	if over && !owned && in.IsMouseButtonJustPressed(core.MouseButtonLeft) {
		if cell, ok := c.cellAt(ctx.Renderer, mouse); ok {
			c.selAnchor = cell
			c.selCaret = cell
			c.selActive = true
			c.hasSel = false
		} else {
			c.clearSelection()
		}
		return
	}
	if c.selActive && in.IsMouseButtonPressed(core.MouseButtonLeft) {
		if cell, ok := c.cellAt(ctx.Renderer, mouse); ok {
			c.selCaret = cell
			c.hasSel = !(c.selAnchor == c.selCaret)
		}
		return
	}
	if c.selActive {
		c.selActive = false
		c.hasSel = !(c.selAnchor == c.selCaret)
	}

	if !over || owned {
		return
	}
	// Wheel scrolls back through history; scrolling down returns to follow mode.
	if s := in.GetMouseScroll(); s.Y != 0 {
		c.scroll += int(s.Y)
		if c.scroll < 0 {
			c.scroll = 0
		}
	}
}

// copyButtonRect is the small "Copy" button in the console's top-right corner.
func (c *ConsoleComponent) copyButtonRect() math.Rect {
	rect := c.Rect()
	const bw = 46.0
	return math.NewRect(rect.X()+rect.Width()-8-bw, rect.Y()+2, bw, c.RowHeight+2)
}

// cellAt maps a screen position to a consoleCell over the visual rows drawn last
// frame, or ok=false when the position is outside the text. It needs the renderer to
// measure rune widths.
func (c *ConsoleComponent) cellAt(r core.Renderer, pos math.Vector2) (consoleCell, bool) {
	if r == nil || len(c.lastRows) == 0 {
		return consoleCell{}, false
	}
	rect := c.Rect()
	rowIdx := int((pos.Y - (rect.Y() + 2)) / c.RowHeight)
	if rowIdx < 0 || rowIdx >= len(c.lastRows) {
		return consoleCell{}, false
	}
	row := c.lastRows[rowIdx]
	relX := pos.X - (rect.X() + consolePad)
	idx := runeIndexAtWidth(r, row.text, c.FontID, c.FontSize, relX)
	return consoleCell{seq: row.seq, rune: row.startRune + idx}, true
}

// selectAll selects every retained log line, start to end.
func (c *ConsoleComponent) selectAll() {
	lines := console.Snapshot()
	if len(lines) == 0 {
		c.clearSelection()
		return
	}
	last := lines[len(lines)-1]
	c.selAnchor = consoleCell{seq: lines[0].seq, rune: 0}
	c.selCaret = consoleCell{seq: last.seq, rune: utf8.RuneCountInString(last.text)}
	c.selActive = false
	c.hasSel = true
}

// clearSelection drops any selection (and an in-progress drag).
func (c *ConsoleComponent) clearSelection() {
	c.selAnchor = consoleCell{}
	c.selCaret = consoleCell{}
	c.selActive = false
	c.hasSel = false
}

// reconcileSelection clears a selection whose lines have already dropped out of the
// ring buffer, so a stale selection doesn't leave Copy enabled for nothing.
func (c *ConsoleComponent) reconcileSelection() {
	if !c.hasSel {
		return
	}
	lines := console.Snapshot()
	if len(lines) == 0 || (c.selAnchor.seq < lines[0].seq && c.selCaret.seq < lines[0].seq) {
		c.clearSelection()
	}
}

// copySelection writes the current selection to the system clipboard. A successful
// copy is silent: logging "copied" back into the console would pollute the very log
// the user is copying from. Only a failure is reported.
func (c *ConsoleComponent) copySelection() {
	if !c.hasSel {
		return
	}
	if err := writeClipboard(c.selectedText()); err != nil {
		console.Print("copy: " + err.Error())
	}
}

// selectedText returns the selected text, joining the spanned logical lines with
// newlines and slicing the first/last partial lines by rune offset.
func (c *ConsoleComponent) selectedText() string {
	if !c.hasSel {
		return ""
	}
	a, b := c.selAnchor, c.selCaret
	if b.less(a) {
		a, b = b, a
	}
	var parts []string
	for _, ln := range console.Snapshot() {
		if ln.seq < a.seq || ln.seq > b.seq {
			continue
		}
		from, to := 0, utf8.RuneCountInString(ln.text)
		if ln.seq == a.seq {
			from = a.rune
		}
		if ln.seq == b.seq {
			to = b.rune
		}
		parts = append(parts, consoleSliceRunes(ln.text, from, to))
	}
	return strings.Join(parts, "\n")
}

// consoleSliceRunes returns the [from, to) rune range of s, clamped to the string's
// bounds. (Prefixed to avoid clashing with the built-in TextInput's sliceRunes, which
// shares this single-package build.)
func consoleSliceRunes(s string, from, to int) string {
	n := utf8.RuneCountInString(s)
	if from < 0 {
		from = 0
	}
	if to < from {
		to = from
	}
	if from >= n {
		return ""
	}
	if to > n {
		to = n
	}
	bi := 0
	r := 0
	for ; r < from && bi < len(s); r++ {
		_, sz := utf8.DecodeRuneInString(s[bi:])
		bi += sz
	}
	start := bi
	for ; r < to && bi < len(s); r++ {
		_, sz := utf8.DecodeRuneInString(s[bi:])
		bi += sz
	}
	return s[start:bi]
}

func (c *ConsoleComponent) Draw(r core.Renderer) {
	rect := c.Rect()
	if rect.Width() <= 0 || rect.Height() <= 0 {
		return
	}

	r.SetClipRect(rect)
	r.DrawRect(rect, c.Background)
	r.DrawRectOutline(rect, c.BorderColor, 1)

	lines := console.Snapshot()
	_, th := r.MeasureText("Ag", c.FontID, c.FontSize)
	if th <= 0 {
		th = c.RowHeight // font not ready yet; keep rows readable
	}

	c.lastRows = nil
	if len(lines) == 0 {
		r.DrawText("no output", c.FontID, c.FontSize, math.NewVector2(rect.X()+consolePad, rect.Y()+2), c.Dim)
	} else {
		c.drawRows(r, rect, lines, th)
	}

	// Copy button, drawn on top so it reads as a floating control over the text.
	c.drawCopyButton(r, rect, th)

	r.ClearClip()
}

// drawRows wraps the log into visual rows, records them for hit-testing, and draws the
// selection highlight behind the text.
func (c *ConsoleComponent) drawRows(r core.Renderer, rect math.Rect, lines []consoleLine, th float64) {
	rows := c.buildRows(r, rect, lines)
	c.lastRows = rows

	var selFrom, selTo consoleCell
	if c.hasSel {
		selFrom, selTo = c.selAnchor, c.selCaret
		if selTo.less(selFrom) {
			selFrom, selTo = selTo, selFrom
		}
	}

	textX := rect.X() + consolePad
	for k, row := range rows {
		y := rect.Y() + 2 + float64(k)*c.RowHeight

		// Highlight the selected rune range of this row, drawn behind the text.
		if c.hasSel && row.seq >= selFrom.seq && row.seq <= selTo.seq {
			rowLen := utf8.RuneCountInString(row.text)
			relFrom := 0
			relTo := rowLen
			if row.seq == selFrom.seq {
				relFrom = selFrom.rune - row.startRune
			}
			if row.seq == selTo.seq {
				relTo = selTo.rune - row.startRune
			}
			if relFrom < 0 {
				relFrom = 0
			}
			if relTo > rowLen {
				relTo = rowLen
			}
			if relTo > relFrom {
				x0 := textX + textWidth(r, row.text, c.FontID, c.FontSize, 0, relFrom)
				x1 := textX + textWidth(r, row.text, c.FontID, c.FontSize, 0, relTo)
				r.DrawRect(math.NewRect(x0, y, x1-x0, c.RowHeight), c.Selection)
			}
		}

		ty := y + (c.RowHeight-th)/2
		if ty < y {
			ty = y
		}
		r.DrawText(row.text, c.FontID, c.FontSize, math.NewVector2(textX, ty), c.Text)
	}
}

// drawCopyButton draws the Copy button, dimmed when there is nothing to copy.
func (c *ConsoleComponent) drawCopyButton(r core.Renderer, rect math.Rect, th float64) {
	btn := c.copyButtonRect()
	bg := c.Background
	label := c.Dim
	if c.hasSel {
		label = c.Text
		if c.copyHover {
			bg = c.Selection
		}
	}
	r.DrawRect(btn, bg)
	r.DrawRectOutline(btn, c.BorderColor, 1)
	tw, _ := r.MeasureText("Copy", c.FontID, c.FontSize)
	ty := btn.Y() + (btn.Height()-th)/2
	if ty < btn.Y() {
		ty = btn.Y()
	}
	r.DrawText("Copy", c.FontID, c.FontSize, math.NewVector2(btn.X()+(btn.Width()-tw)/2, ty), label)
}

// buildRows computes the top-to-bottom visual rows currently in view, applying wrap
// and scrollback, and reconciles c.scroll when the history is shorter than the
// scroll offset. It mirrors the original Draw-time row walk but returns consoleRows
// (with their line seq and rune offset) instead of bare strings.
func (c *ConsoleComponent) buildRows(r core.Renderer, rect math.Rect, lines []consoleLine) []consoleRow {
	maxWidth := rect.Width() - 2*consolePad
	if maxWidth < 1 {
		maxWidth = 1
	}
	vis := int((rect.Height() - 2) / c.RowHeight)
	if vis < 1 {
		vis = 1
	}

	// Walk backward from the newest line, wrapping each into visual rows, until we
	// have enough to cover the viewport plus the scrollback. `rows` ends up
	// newest-first (the reverse of draw order).
	want := vis + c.scroll
	var rows []consoleRow
	for i := len(lines) - 1; i >= 0 && len(rows) < want; i-- {
		sub := wrapConsoleLine(r, lines[i], c.FontID, c.FontSize, maxWidth)
		for j := len(sub) - 1; j >= 0; j-- {
			rows = append(rows, sub[j])
		}
	}

	// If we ran out of history before filling `want`, the whole log is in view and
	// scroll can't be valid beyond its top: clamp it back.
	if len(rows) < want {
		if m := len(rows) - vis; m > 0 {
			if c.scroll > m {
				c.scroll = m
			}
		} else {
			c.scroll = 0
		}
	}

	// Drop the newest `scroll` rows (scrollback), keep the first `vis` of the rest,
	// then reverse into top-to-bottom draw order.
	if len(rows) > c.scroll {
		rows = rows[c.scroll:]
	} else {
		rows = nil
	}
	if len(rows) > vis {
		rows = rows[:vis]
	}
	for a, b := 0, len(rows)-1; a < b; a, b = a+1, b-1 {
		rows[a], rows[b] = rows[b], rows[a]
	}
	return rows
}

// wrapConsoleLine breaks one log line into visual rows that each fit maxWidth,
// splitting a rune sequence mid-way (terminal-style hard wrap) — the same rule the
// engine applies for `wrap: char`. Each row records the logical line's seq and the
// rune offset where its fragment starts, so a hit-test can map back to a stable
// consoleCell.
func wrapConsoleLine(r core.Renderer, line consoleLine, fontID string, size, maxWidth float64) []consoleRow {
	s := line.text
	if s == "" {
		return []consoleRow{{text: "", seq: line.seq, startRune: 0}}
	}
	var rows []consoleRow
	start := 0
	width := 0.0
	consumed := 0
	runeStart := 0
	for i := 0; i < len(s); {
		// Measure one rune at a time and accumulate so the whole wrap is O(n); the
		// old code re-measured the growing prefix s[start:next] each step (O(n²)).
		_, sz := utf8.DecodeRuneInString(s[i:])
		w, _ := r.MeasureText(s[i:i+sz], fontID, size)
		if i > start && width+w > maxWidth {
			rows = append(rows, consoleRow{text: s[start:i], seq: line.seq, startRune: runeStart})
			start = i
			runeStart = consumed
			width = 0
		}
		width += w
		consumed++
		i += sz
	}
	rows = append(rows, consoleRow{text: s[start:], seq: line.seq, startRune: runeStart})
	return rows
}

// runeIndexAtWidth returns how many whole runes of s fit within x logical units — the
// caret position a click at x maps to (0 at the left edge, the rune count past the
// last rune). Used by the selection hit-test.
func runeIndexAtWidth(r core.Renderer, s, fontID string, size, x float64) int {
	n := utf8.RuneCountInString(s)
	if x <= 0 {
		return 0
	}
	width := 0.0
	idx := 0
	for i := 0; i < len(s) && idx < n; {
		_, sz := utf8.DecodeRuneInString(s[i:])
		w, _ := r.MeasureText(s[i:i+sz], fontID, size)
		width += w
		if width > x {
			break
		}
		i += sz
		idx++
	}
	return idx
}

// textWidth measures the width of the [from, to) rune range of s (clamped), used to
// place the selection highlight's x span.
func textWidth(r core.Renderer, s, fontID string, size float64, from, to int) float64 {
	n := utf8.RuneCountInString(s)
	if from < 0 {
		from = 0
	}
	if to > n {
		to = n
	}
	if to <= from {
		return 0
	}
	bi := 0
	for i := 0; i < from && bi < len(s); i++ {
		_, sz := utf8.DecodeRuneInString(s[bi:])
		bi += sz
	}
	end := bi
	for i := from; i < to && end < len(s); i++ {
		_, sz := utf8.DecodeRuneInString(s[end:])
		end += sz
	}
	w, _ := r.MeasureText(s[bi:end], fontID, size)
	return w
}
