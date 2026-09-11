package math

import (
	"math"
	"testing"
)

// TestRectBounds verifies that RectBounds maps a local rect through the full
// transform (scale -> rotate about the origin -> translate) and returns its
// world-space AABB, including the non-uniform scale + rotation case where a naive
// "scale the axes, then rotate" would get the aspect ratio wrong.
func TestRectBounds(t *testing.T) {
	tests := []struct {
		name string
		t    Transform
		rect Rect
		want Rect
	}{
		{
			name: "identity",
			t:    Identity(),
			rect: NewRect(10, 20, 30, 40),
			want: NewRect(10, 20, 30, 40),
		},
		{
			name: "translate only",
			t:    NewTransformWithPosition(5, 6),
			rect: NewRect(10, 20, 30, 40),
			want: NewRect(15, 26, 30, 40),
		},
		{
			name: "uniform scale",
			t:    Transform{Position: Zero(), Rotation: 0, Scale: NewVector2(2, 2)},
			rect: NewRect(10, 20, 30, 40),
			want: NewRect(20, 40, 60, 80),
		},
		{
			name: "quarter turn about origin",
			t:    Transform{Position: Zero(), Rotation: math.Pi / 2, Scale: One()},
			rect: NewRect(0, 0, 10, 5),
			want: NewRect(-5, 0, 5, 10),
		},
		{
			name: "quarter turn plus translate",
			t:    Transform{Position: NewVector2(100, 0), Rotation: math.Pi / 2, Scale: One()},
			rect: NewRect(0, 0, 10, 5),
			want: NewRect(95, 0, 5, 10),
		},
		{
			name: "non-uniform scale then quarter turn",
			t:    Transform{Position: Zero(), Rotation: math.Pi / 2, Scale: NewVector2(2, 1)},
			rect: NewRect(0, 0, 10, 5),
			want: NewRect(-5, 0, 5, 20),
		},
		{
			name: "negative scale (mirror)",
			t:    Transform{Position: Zero(), Rotation: 0, Scale: NewVector2(-1, 1)},
			rect: NewRect(10, 20, 30, 40),
			want: NewRect(-40, 20, 30, 40),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.t.RectBounds(tt.rect)
			if !rectApproxEqual(got, tt.want) {
				t.Fatalf("RectBounds(%v) = %v, want %v", tt.rect, got, tt.want)
			}
		})
	}
}

// rectApproxEqual compares two rects within a small tolerance so quarter-turn
// results (which involve cos/sin of π/2, not exactly 0 and 1) don't trip float
// inequality on the rotated corners.
func rectApproxEqual(a, b Rect) bool {
	const eps = 1e-9
	return abs(a.X()-b.X()) < eps &&
		abs(a.Y()-b.Y()) < eps &&
		abs(a.Width()-b.Width()) < eps &&
		abs(a.Height()-b.Height()) < eps
}

func abs(f float64) float64 {
	if f < 0 {
		return -f
	}
	return f
}
