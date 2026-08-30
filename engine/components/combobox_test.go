package components

import "testing"

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
