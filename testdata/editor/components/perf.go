package components

import (
	"fmt"
	"time"
)

// frameMonitor samples the editor's frame loop and reports stalls to the console, so
// a freeze can be diagnosed from the log. It is a plain helper file (no component
// struct), like helpers.go — the build tool copies it verbatim and its codegen skips
// it (it contributes no component kind).
//
// A frame-loop stall reads in the UI as "typed characters appear all at once": while
// Update/Draw is stuck, the OS queues the key events, and Ebitengine flushes them the
// moment the loop resumes. Logging the gap here lets us tell a true stall apart from a
// steady low frame rate.
type frameMonitor struct {
	last time.Time
}

var perf frameMonitor

// stallThreshold is the frame gap (wall-clock) above which tick logs a stall line. A
// healthy editor runs every ~16ms (60 FPS); 250ms is ~4 FPS and unmistakably abnormal.
const stallThreshold = 250 * time.Millisecond

// tick is called once per frame (from the menu bar's Update, before any early return).
// It logs a console line whenever a frame takes unusually long.
func (m *frameMonitor) tick() {
	now := time.Now()
	if !m.last.IsZero() {
		if gap := now.Sub(m.last); gap >= stallThreshold {
			console.Print(fmt.Sprintf("perf: frame took %s", gap.Round(time.Millisecond)))
		}
	}
	m.last = now
}
