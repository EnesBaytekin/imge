package components

import (
	"testing"

	"github.com/EnesBaytekin/imge/core/math"
)

// TestComboBoxDropdownDirection verifies the native-combobox placement: open below
// by default, flip above when it would overflow the bottom, and when it fits
// neither side open on the side with more room.
func TestComboBoxDropdownDirection(t *testing.T) {
	c := &ComboBoxComponent{}
	c.Width = 100
	c.Height = 22
	c.Items = make([]string, 5) // list height 5*22 = 110
	c.Initialize()

	// Unknown viewport (no renderer yet) -> open below, the default.
	c.viewportH = 0
	if c.openUp() {
		t.Fatal("expected down when the viewport is unknown")
	}

	// Room below -> stays below.
	c.Offset = math.NewVector2(0, 0)
	c.viewportH = 500
	if c.openUp() {
		t.Fatal("expected down when the list fits below")
	}

	// No room below, room above -> flips up.
	c.Offset = math.NewVector2(0, 400) // header bottom 422; below = 500-422 = 78 < 110
	if !c.openUp() {
		t.Fatal("expected up when the list overflows the bottom")
	}

	// Fits neither side -> open on the side with more room.
	c.Items = make([]string, 20) // list height 440
	c.Offset = math.NewVector2(0, 150)
	c.viewportH = 400 // header bottom 172; below = 228, above = 150 -> below has more room
	if c.openUp() {
		t.Fatal("expected down when more room is below")
	}
	c.Offset = math.NewVector2(0, 250) // header bottom 272; below = 128, above = 250 -> above has more room
	if !c.openUp() {
		t.Fatal("expected up when more room is above")
	}
}

// TestComboBoxDropdownRect checks the list rectangle abuts the field on the correct
// side.
func TestComboBoxDropdownRect(t *testing.T) {
	c := &ComboBoxComponent{}
	c.Width = 100
	c.Height = 22
	c.Offset = math.NewVector2(10, 20)
	c.Items = make([]string, 3)
	c.Initialize()

	c.viewportH = 0 // down
	dr := c.dropdownRect()
	if dr.Top() != c.headerRect().Bottom() {
		t.Fatalf("dropdown should start at the header bottom: got %v", dr.Top())
	}
	if dr.Height() != 66 {
		t.Fatalf("dropdown height should be 3*22=66: got %v", dr.Height())
	}

	c.Offset = math.NewVector2(10, 70)
	c.viewportH = 120 // header top 70, bottom 92; below = 28 < 66, above = 70 >= 66 -> up
	dr = c.dropdownRect()
	if dr.Bottom() != c.headerRect().Top() {
		t.Fatalf("dropdown should end at the header top when opened up: got bottom %v vs top %v", dr.Bottom(), c.headerRect().Top())
	}
}
