package components

import (
	"testing"

	"github.com/EnesBaytekin/imge/core/math"
)

// TestCheckBoxToggle verifies Activate flips the state on every click.
func TestCheckBoxToggle(t *testing.T) {
	c := &CheckBoxComponent{}
	c.Initialize()

	if c.GetChecked() {
		t.Fatal("default state should be unchecked")
	}
	c.Activate()
	if !c.GetChecked() {
		t.Fatal("Activate should check the box")
	}
	c.Activate()
	if c.GetChecked() {
		t.Fatal("Activate should uncheck the box")
	}
}

// TestCheckBoxSetChecked verifies SetChecked is a silent setter.
func TestCheckBoxSetChecked(t *testing.T) {
	c := &CheckBoxComponent{}
	c.Initialize()

	c.SetChecked(true)
	if !c.GetChecked() {
		t.Fatal("SetChecked(true) should check the box")
	}
	c.SetChecked(false)
	if c.GetChecked() {
		t.Fatal("SetChecked(false) should uncheck the box")
	}
}

// TestCheckBoxDefaults verifies the flat-fallback defaults and flags after Initialize.
func TestCheckBoxDefaults(t *testing.T) {
	c := &CheckBoxComponent{}
	c.Initialize()

	if c.BoxSize != 16 {
		t.Fatalf("default BoxSize = %v, want 16", c.BoxSize)
	}
	if c.Gap != 6 {
		t.Fatalf("default Gap = %v, want 6", c.Gap)
	}
	if c.Size != 12 {
		t.Fatalf("default Size = %v, want 12", c.Size)
	}
	if c.OutlineThickness != 1 {
		t.Fatalf("default OutlineThickness = %v, want 1", c.OutlineThickness)
	}
	if c.BoxColor != (math.NewColor(30, 30, 42, 255)) {
		t.Fatalf("default BoxColor = %v", c.BoxColor)
	}
	if c.CheckedColor != c.BoxColor {
		t.Fatalf("CheckedColor should default to BoxColor: got %v vs %v", c.CheckedColor, c.BoxColor)
	}
	if c.CheckColor != math.White {
		t.Fatalf("default CheckColor = %v, want white", c.CheckColor)
	}
	if c.OutlineColor != (math.NewColor(74, 74, 106, 255)) {
		t.Fatalf("default OutlineColor = %v", c.OutlineColor)
	}
	if c.TextColor != math.White {
		t.Fatalf("default TextColor = %v, want white", c.TextColor)
	}
	if !c.BlocksPointer() {
		t.Fatal("checkbox should block the pointer")
	}
}

// TestCheckBoxBoxRect verifies the box is a square of BoxSize, vertically centered at
// the element's left edge.
func TestCheckBoxBoxRect(t *testing.T) {
	c := &CheckBoxComponent{}
	c.Width = 100
	c.Height = 24
	c.Offset = math.NewVector2(10, 30)
	c.Initialize()

	br := c.boxRect()
	if br.Left() != c.Rect().Left() {
		t.Fatalf("box should start at the element's left edge: got %v vs %v", br.Left(), c.Rect().Left())
	}
	if br.Width() != 16 || br.Height() != 16 {
		t.Fatalf("box should be a 16x16 square, got %vx%v", br.Width(), br.Height())
	}
	wantTop := c.Rect().Center().Y - 8
	if br.Top() != wantTop {
		t.Fatalf("box should be vertically centered: top %v, want %v", br.Top(), wantTop)
	}
}

// TestCheckBoxCheckedBorder verifies the checked border falls back to Border when
// CheckedBorder is left zero.
func TestCheckBoxCheckedBorder(t *testing.T) {
	c := &CheckBoxComponent{}
	c.Border = math.Border{Left: 2, Top: 2, Right: 2, Bottom: 2}
	if got := c.checkedBorder(); got != c.Border {
		t.Fatalf("checked border should fall back to Border: got %v", got)
	}

	c.CheckedBorder = math.Border{Left: 4, Top: 4, Right: 4, Bottom: 4}
	if got := c.checkedBorder(); got != c.CheckedBorder {
		t.Fatalf("checked border should use CheckedBorder when set: got %v", got)
	}
}
