package components

import (
	"testing"

	"github.com/EnesBaytekin/imge/core/math"
)

// newListFixture builds a list at offset (10,20) sized 120×60 with the given items.
func newListFixture(items ...string) *ListComponent {
	l := &ListComponent{}
	l.Items = items
	l.Offset = math.NewVector2(10, 20)
	l.Width = 120
	l.Height = 60
	l.Initialize()
	return l
}

func TestListDefaults(t *testing.T) {
	l := newListFixture("a")
	if l.itemHeight() != 22 {
		t.Fatalf("default item height = %v, want 22", l.itemHeight())
	}
	if l.scrollbarWidth() != 6 {
		t.Fatalf("default scrollbar width = %v, want 6", l.scrollbarWidth())
	}
	if l.Color == (math.Color{}) || l.HoverColor == (math.Color{}) || l.SelectedColor == (math.Color{}) {
		t.Fatal("expected non-zero default colors")
	}
	if !l.IsFocusable() {
		t.Fatal("list should be focusable by default")
	}
	if !l.BlocksPointer() {
		t.Fatal("list should block the pointer by default")
	}
}

func TestListGetSetValue(t *testing.T) {
	l := newListFixture("apple", "banana", "cherry")

	l.SetValue("banana")
	if got := l.GetValue(); got != "banana" {
		t.Fatalf("GetValue = %q, want banana", got)
	}
	if got := l.GetIndex(); got != 1 {
		t.Fatalf("GetIndex = %d, want 1", got)
	}

	// Setting a value not in the list clears the selection.
	l.SetValue("missing")
	if got := l.GetValue(); got != "" {
		t.Fatalf("GetValue after unknown set = %q, want empty", got)
	}

	// Replacing items keeps a still-present selection, drops a vanished one.
	l.SetValue("cherry")
	l.SetItems([]string{"apple", "cherry"})
	if got := l.GetValue(); got != "cherry" {
		t.Fatalf("GetValue after SetItems = %q, want cherry", got)
	}
	l.SetValue("apple")
	l.SetItems([]string{"pear"})
	if got := l.GetValue(); got != "" {
		t.Fatalf("GetValue after removing selection = %q, want empty", got)
	}
}

func TestListItemAt(t *testing.T) {
	l := newListFixture("a", "b", "c", "d")
	// Rows are 22 tall, starting at the list top (y=20).

	if got := l.itemAt(math.NewVector2(30, 20+11)); got != 0 {
		t.Fatalf("row 0 hit = %d, want 0", got)
	}
	if got := l.itemAt(math.NewVector2(30, 20+22+1)); got != 1 {
		t.Fatalf("row 1 hit = %d, want 1", got)
	}
	// Outside the list vertically.
	if got := l.itemAt(math.NewVector2(30, 19)); got != -1 {
		t.Fatalf("above-list hit = %d, want -1", got)
	}

	// After scrolling one row, the same y maps to the next row.
	l.scrollOffset = 22
	if got := l.itemAt(math.NewVector2(30, 20+11)); got != 1 {
		t.Fatalf("scrolled row hit = %d, want 1", got)
	}
}

func TestListSelectIndexScrollsIntoView(t *testing.T) {
	l := newListFixture()
	for i := 0; i < 10; i++ {
		l.Items = append(l.Items, "item"+string(rune('a'+i)))
	}

	// 10 rows × 22 = 220 content in a 60-tall list -> maxScroll 160.
	l.selectIndex(9)
	if got := l.GetValue(); got != "itemj" {
		t.Fatalf("GetValue = %q, want itemj", got)
	}
	if got := l.GetIndex(); got != 9 {
		t.Fatalf("GetIndex = %d, want 9", got)
	}
	if l.scrollOffset != 160 {
		t.Fatalf("scrollOffset after selecting last = %v, want 160", l.scrollOffset)
	}

	// Selecting the first row scrolls back to the top.
	l.selectIndex(0)
	if l.scrollOffset != 0 {
		t.Fatalf("scrollOffset after selecting first = %v, want 0", l.scrollOffset)
	}
}

func TestListMoveSelectionWraps(t *testing.T) {
	l := newListFixture("a", "b", "c")

	// No selection + Down -> first item.
	l.moveSelection(1)
	if got := l.GetValue(); got != "a" {
		t.Fatalf("Down with no selection = %q, want a", got)
	}

	// Up from the first wraps to the last.
	l.moveSelection(-1)
	if got := l.GetValue(); got != "c" {
		t.Fatalf("Up from first = %q, want c", got)
	}

	// Down from the last wraps to the first.
	l.moveSelection(1)
	if got := l.GetValue(); got != "a" {
		t.Fatalf("Down from last = %q, want a", got)
	}
}

func TestListScrollClamps(t *testing.T) {
	l := newListFixture()
	for i := 0; i < 10; i++ {
		l.Items = append(l.Items, "item")
	}

	l.Scroll(math.NewVector2(0, -1000)) // scroll down far past the end
	if l.scrollOffset != 160 {
		t.Fatalf("scrollOffset over-scrolled = %v, want 160", l.scrollOffset)
	}
	l.Scroll(math.NewVector2(0, 1000)) // scroll up far past the top
	if l.scrollOffset != 0 {
		t.Fatalf("scrollOffset under-scrolled = %v, want 0", l.scrollOffset)
	}
}

func TestListScrollbarDrag(t *testing.T) {
	l := newListFixture()
	for i := 0; i < 10; i++ {
		l.Items = append(l.Items, "item"+string(rune('a'+i)))
	}
	// 10 rows × 22 = 220 in a 60-tall list -> maxScroll 160. The scrollbar sits on
	// the right edge: x in [124,130), y in [20,80).
	over := math.NewVector2(127, 50)

	// A press on the scrollbar starts a drag; a press on a row does not.
	l.BeginAdjust(over)
	if !l.draggingBar {
		t.Fatal("press on scrollbar should start a drag")
	}
	if l.scrollOffset <= 0 {
		t.Fatalf("drag should move the scroll: offset %v", l.scrollOffset)
	}

	// Dragging to the top scrolls to the first row.
	l.Adjust(math.NewVector2(127, 20))
	if l.scrollOffset != 0 {
		t.Fatalf("drag to top = %v, want 0", l.scrollOffset)
	}
	l.EndAdjust()
	if l.draggingBar {
		t.Fatal("EndAdjust should end the drag")
	}

	// A press over a row (not the scrollbar) is not a drag.
	l.BeginAdjust(math.NewVector2(50, 31))
	if l.draggingBar {
		t.Fatal("press on a row should not start a scrollbar drag")
	}
	l.EndAdjust()
}

func TestListScrollbarGeometry(t *testing.T) {
	l := newListFixture("a", "b", "c") // 3 rows = 66 > 60, so it overflows
	if !l.scrollbarVisible() {
		t.Fatal("3-row list in a 60-tall rect should show a scrollbar")
	}
	if got := l.maxScroll(); got != 6 {
		t.Fatalf("maxScroll = %v, want 6", got)
	}

	// The thumb is proportional to the visible fraction and sits within the track.
	thumb := l.thumbRect()
	track := l.scrollbarRect()
	if thumb.Top() < track.Top() || thumb.Bottom() > track.Bottom() {
		t.Fatalf("thumb %v outside track %v", thumb, track)
	}
	if thumb.Width() != l.scrollbarWidth() {
		t.Fatalf("thumb width = %v, want %v", thumb.Width(), l.scrollbarWidth())
	}

	// A list that fits has no scrollbar.
	short := newListFixture("a") // 1 row = 22 < 60
	if short.scrollbarVisible() {
		t.Fatal("1-row list should not show a scrollbar")
	}
}
