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

// consoleLog is the editor-wide capture of RUN/build stdout+stderr and editor log
// lines. The toolbar pipes the launched `imge run` process into it (on a goroutine)
// while the console panel reads it on the game loop, so it is mutex-guarded. It lives
// at package level (like the undo history) because the writer (toolbar) and reader
// (console panel) are different components in the same package.
type consoleLog struct {
	mu      sync.Mutex
	lines   []string
	partial string // trailing bytes of an in-progress line
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
	c.lines = append(c.lines, line)
	if len(c.lines) > consoleMaxLines {
		c.lines = c.lines[len(c.lines)-consoleMaxLines:]
	}
}

// Snapshot returns a copy of the captured lines (newest last) so the console panel can
// render without holding the lock during drawing.
func (c *consoleLog) Snapshot() []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]string, len(c.lines))
	copy(out, c.lines)
	return out
}

// ConsoleComponent is the bottom strip that shows captured RUN/build output plus a few
// editor log lines. It renders the most recent lines from the shared console log,
// following new output unless the user scrolls back with the wheel (scroll = lines back
// from the newest; 0 = follow the bottom).
type ConsoleComponent struct {
	core.BaseUIComponent

	Background  math.Color `json:"background"`
	BorderColor math.Color `json:"border_color"`
	Text        math.Color `json:"text"`
	Dim         math.Color `json:"dim"`

	FontID    string  `json:"font_id"`
	FontSize  float64 `json:"font_size"`
	RowHeight float64 `json:"row_height"`

	scroll int // lines scrolled back from the newest line
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
	// A modal is open (add-component panel / confirm dialog): this panel is inert.
	if modalOpen() {
		return
	}
	rect := c.Rect()
	mouse := ctx.Input.GetMousePosition()
	if !rect.ContainsPoint(mouse) {
		return
	}
	// Yield to a window drawn above the console — the @UIManager's blocking occlusion.
	if pointerOwnedElsewhere(c.GetScene(), c.GetOwner(), mouse) {
		return
	}
	// Wheel scrolls back through history; scrolling down returns to follow mode.
	if s := ctx.Input.GetMouseScroll(); s.Y != 0 {
		c.scroll += int(s.Y)
		if c.scroll < 0 {
			c.scroll = 0
		}
	}
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
	pad := 4.0
	_, th := r.MeasureText("Ag", c.FontID, c.FontSize)
	if th <= 0 {
		th = c.RowHeight // font not ready yet; keep rows readable
	}

	if len(lines) == 0 {
		r.DrawText("no output", c.FontID, c.FontSize, math.NewVector2(rect.X()+pad, rect.Y()+2), c.Dim)
		r.ClearClip()
		return
	}

	// Wrap long lines to the panel width so they break instead of clipping off the
	// right edge (the engine's `wrap: char` semantics, splitting mid-rune).
	maxWidth := rect.Width() - 2*pad
	if maxWidth < 1 {
		maxWidth = 1
	}

	// Number of visual rows that fit in the body.
	vis := int((rect.Height() - 2) / c.RowHeight)
	if vis < 1 {
		vis = 1
	}

	// Walk backward from the newest line, wrapping each into visual rows, until we
	// have enough to cover the viewport plus the scrollback. `rows` ends up
	// newest-first (the reverse of draw order).
	want := vis + c.scroll
	var rows []string
	for i := len(lines) - 1; i >= 0 && len(rows) < want; i-- {
		sub := charWrapLine(r, lines[i], c.FontID, c.FontSize, maxWidth)
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

	for k, line := range rows {
		y := rect.Y() + 2 + float64(k)*c.RowHeight
		ty := y + (c.RowHeight-th)/2
		if ty < y {
			ty = y
		}
		r.DrawText(line, c.FontID, c.FontSize, math.NewVector2(rect.X()+pad, ty), c.Text)
	}

	r.ClearClip()
}

// charWrapLine breaks one log line into sub-lines that each fit maxWidth, splitting a
// rune sequence mid-way (terminal-style hard wrap) — the same rule the engine applies
// for `wrap: char`. A single rune wider than maxWidth still gets its own line.
func charWrapLine(r core.Renderer, s, fontID string, size, maxWidth float64) []string {
	if s == "" {
		return []string{""}
	}
	var lines []string
	for start := 0; start < len(s); {
		_, sz := utf8.DecodeRuneInString(s[start:])
		end := start + sz
		for end < len(s) {
			_, z := utf8.DecodeRuneInString(s[end:])
			next := end + z
			w, _ := r.MeasureText(s[start:next], fontID, size)
			if w > maxWidth {
				break
			}
			end = next
		}
		lines = append(lines, s[start:end])
		start = end
	}
	return lines
}
