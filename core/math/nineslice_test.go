package math

import "testing"

func findSlice(s []Slice, sx, sy float64) *Slice {
	for i := range s {
		if s[i].Src.X() == sx && s[i].Src.Y() == sy {
			return &s[i]
		}
	}
	return nil
}

func TestSlice9Standard(t *testing.T) {
	dst := NewRect(0, 0, 100, 50)
	slices := Slice9(16, 16, Border{Left: 4, Top: 4, Right: 4, Bottom: 4}, dst)
	if len(slices) != 9 {
		t.Fatalf("got %d slices, want 9: %v", len(slices), slices)
	}

	tl := findSlice(slices, 0, 0)
	if tl == nil {
		t.Fatal("missing top-left corner")
	}
	if tl.Src != NewRect(0, 0, 4, 4) || tl.Dst != NewRect(0, 0, 4, 4) {
		t.Errorf("tl = %v -> %v, want 4x4 -> 4x4", tl.Src, tl.Dst)
	}

	te := findSlice(slices, 4, 0)
	if te == nil {
		t.Fatal("missing top edge")
	}
	if te.Src != NewRect(4, 0, 8, 4) || te.Dst != NewRect(4, 0, 92, 4) {
		t.Errorf("te = %v -> %v, want 8x4 -> 92x4", te.Src, te.Dst)
	}

	c := findSlice(slices, 4, 4)
	if c == nil {
		t.Fatal("missing center")
	}
	if c.Src != NewRect(4, 4, 8, 8) || c.Dst != NewRect(4, 4, 92, 42) {
		t.Errorf("center = %v -> %v, want 8x8 -> 92x42", c.Src, c.Dst)
	}

	br := findSlice(slices, 12, 12)
	if br == nil {
		t.Fatal("missing bottom-right corner")
	}
	if br.Src != NewRect(12, 12, 4, 4) || br.Dst != NewRect(96, 46, 4, 4) {
		t.Errorf("br = %v -> %v, want 4x4 -> 4x4 at (96,46)", br.Src, br.Dst)
	}
}

func TestSlice9ZeroBorder(t *testing.T) {
	s := Slice9(16, 16, Border{}, NewRect(0, 0, 100, 50))
	if len(s) != 1 {
		t.Fatalf("zero border: got %d slices, want 1", len(s))
	}
	if s[0].Src != NewRect(0, 0, 16, 16) || s[0].Dst != NewRect(0, 0, 100, 50) {
		t.Errorf("zero border slice = %v -> %v", s[0].Src, s[0].Dst)
	}
}

func TestSlice9SmallTargetScalesBorders(t *testing.T) {
	// Target width 6 but the horizontal borders sum to 8, so they scale down to
	// 3+3 and the center column vanishes. No slice may overflow the target.
	s := Slice9(16, 16, Border{Left: 4, Top: 4, Right: 4, Bottom: 4}, NewRect(0, 0, 6, 20))
	if len(s) != 6 { // 2 columns (center dropped) × 3 rows
		t.Fatalf("got %d slices, want 6: %v", len(s), s)
	}
	for _, sl := range s {
		if sl.Dst.Left() < 0 || sl.Dst.Top() < 0 || sl.Dst.Right() > 6 || sl.Dst.Bottom() > 20 {
			t.Errorf("slice overflows target: %v -> %v", sl.Src, sl.Dst)
		}
	}
}

func TestSlice9Degenerate(t *testing.T) {
	if got := Slice9(0, 0, Border{}, NewRect(0, 0, 10, 10)); got != nil {
		t.Errorf("zero texture: got %v, want nil", got)
	}
	if got := Slice9(16, 16, Border{}, NewRect(0, 0, 0, 0)); got != nil {
		t.Errorf("zero target: got %v, want nil", got)
	}
}
