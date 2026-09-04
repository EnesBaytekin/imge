package components

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/EnesBaytekin/imge/core"
)

// TestComboBoxDrawLayerRaise verifies the dropdown raises the component above its
// siblings in draw order only while it is open.
func TestComboBoxDrawLayerRaise(t *testing.T) {
	c := &ComboBoxComponent{}
	c.DrawLayer = 2
	c.Initialize()

	if got := c.GetDrawLayer(); got != 2 {
		t.Fatalf("closed draw layer = %d, want 2", got)
	}
	c.openDropdown()
	if got := c.GetDrawLayer(); got != 2+popupLayerOffset {
		t.Fatalf("open draw layer = %d, want %d", got, 2+popupLayerOffset)
	}
	c.closeDropdown()
	if got := c.GetDrawLayer(); got != 2 {
		t.Fatalf("closed-after draw layer = %d, want 2", got)
	}
}

// TestComboBoxFilter verifies the searchable-dropdown behavior: typing filters the
// list case-insensitively by substring, backspace widens it again, and Escape clears
// the filter (restoring all items) before a second Escape closes the list.
func TestComboBoxFilter(t *testing.T) {
	c := &ComboBoxComponent{Items: []string{"apple", "banana", "apricot", "cherry"}}
	c.Initialize()
	c.openDropdown()

	step := func(in *stubInput) {
		c.HandleInput(&core.Context{Input: in, Time: &stubTime{}})
	}
	check := func(want []string, filter string) {
		t.Helper()
		if c.filter != filter {
			t.Fatalf("filter = %q, want %q", c.filter, filter)
		}
		if got := c.visibleItems(); !reflect.DeepEqual(got, want) {
			t.Fatalf("visibleItems = %v, want %v", got, want)
		}
	}

	// Typing narrows the list as the filter grows.
	step(&stubInput{chars: []rune("ap")})
	check([]string{"apple", "apricot"}, "ap")

	step(&stubInput{chars: []rune("r")})
	check([]string{"apricot"}, "apr")

	// Backspace restores the wider filter.
	step(&stubInput{held: heldKeys(core.KeyBackspace)})
	check([]string{"apple", "apricot"}, "ap")

	// Escape clears the filter (list stays open, everything visible again).
	step(&stubInput{justPressed: justPressedKeys(core.KeyEscape)})
	check([]string{"apple", "banana", "apricot", "cherry"}, "")
	if !c.open {
		t.Fatal("list closed after clearing filter; want still open")
	}

	// A second Escape closes the list.
	step(&stubInput{justPressed: justPressedKeys(core.KeyEscape)})
	if c.open {
		t.Fatal("list still open after second Escape; want closed")
	}
}

// TestComboBoxKeyboardNavigation verifies the ↑/↓/Enter keyboard flow: arrows move the
// highlight (wrapping and keeping it scrolled into view), Enter selects, and ↑/↓ open
// the list when it is closed.
func TestComboBoxKeyboardNavigation(t *testing.T) {
	items := make([]string, 20)
	for i := range items {
		items[i] = fmt.Sprintf("item%02d", i)
	}
	c := &ComboBoxComponent{Items: items, ItemHeight: 10}
	c.Width = 100
	c.Height = 20
	c.Initialize()
	c.viewportH = 80 // a tiny viewport caps the 20×10 list so it must scroll

	step := func(in *stubInput) {
		c.HandleInput(&core.Context{Input: in, Time: &stubTime{}})
	}
	// The highlighted item must always be scrolled fully into the dropdown.
	inView := func() bool {
		dr := c.dropdownRect()
		rowTop := dr.Top() + float64(c.highlight)*c.itemHeight() - c.scrollOffset
		return c.highlight >= 0 && rowTop >= dr.Top() && rowTop+c.itemHeight() <= dr.Bottom()
	}

	// ↓ opens the closed list.
	step(&stubInput{justPressed: justPressedKeys(core.KeyDown)})
	if !c.open {
		t.Fatal("↓ with the list closed should open it")
	}

	// Arrow down through the whole list, checking the cursor stays visible and in range.
	for i := 0; i < len(items); i++ {
		step(&stubInput{justPressed: justPressedKeys(core.KeyDown)})
		if c.highlight < 0 || c.highlight >= len(items) {
			t.Fatalf("step %d: highlight = %d out of range", i, c.highlight)
		}
		if !inView() {
			t.Fatalf("step %d: highlight %d not scrolled into view (scroll=%v)", i, c.highlight, c.scrollOffset)
		}
	}
	// Down twice more wraps around rather than running off the end.
	before := c.highlight
	step(&stubInput{justPressed: justPressedKeys(core.KeyDown)})
	step(&stubInput{justPressed: justPressedKeys(core.KeyDown)})
	if c.highlight == before {
		t.Fatal("↓ should advance (and wrap) the highlight")
	}

	// Enter selects the highlighted item and closes the list.
	want := c.visibleItems()[c.highlight]
	step(&stubInput{justPressed: justPressedKeys(core.KeyEnter)})
	if c.Value != want {
		t.Fatalf("Enter: Value = %q, want %q", c.Value, want)
	}
	if c.open {
		t.Fatal("Enter should close the list after selecting")
	}

	// ↑ also reopens the list.
	step(&stubInput{justPressed: justPressedKeys(core.KeyUp)})
	if !c.open {
		t.Fatal("↑ with the list closed should open it")
	}
}
