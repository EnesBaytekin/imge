package components

import (
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
