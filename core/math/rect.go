// Package math provides mathematical utilities for the game engine.
package math

import "fmt"

// Rect represents an axis-aligned rectangle (AABB) in 2D space.
// Commonly used for collision detection, drawing areas, and UI elements.
type Rect struct {
	Position Vector2 // Top-left corner position
	Size     Vector2 // Width and height
}

// NewRect creates a new rectangle with given position and size.
func NewRect(x, y, width, height float64) Rect {
	return Rect{
		Position: NewVector2(x, y),
		Size:     NewVector2(width, height),
	}
}

// NewRectFromVectors creates a new rectangle from Vector2 position and size.
func NewRectFromVectors(position, size Vector2) Rect {
	return Rect{
		Position: position,
		Size:     size,
	}
}

// X returns the X coordinate of the left edge.
func (r Rect) X() float64 {
	return r.Position.X
}

// Y returns the Y coordinate of the top edge.
func (r Rect) Y() float64 {
	return r.Position.Y
}

// Width returns the width of the rectangle.
func (r Rect) Width() float64 {
	return r.Size.X
}

// Height returns the height of the rectangle.
func (r Rect) Height() float64 {
	return r.Size.Y
}

// Left returns the X coordinate of the left edge.
func (r Rect) Left() float64 {
	return r.Position.X
}

// Right returns the X coordinate of the right edge.
func (r Rect) Right() float64 {
	return r.Position.X + r.Size.X
}

// Top returns the Y coordinate of the top edge.
func (r Rect) Top() float64 {
	return r.Position.Y
}

// Bottom returns the Y coordinate of the bottom edge.
func (r Rect) Bottom() float64 {
	return r.Position.Y + r.Size.Y
}

// Center returns the center point of the rectangle.
func (r Rect) Center() Vector2 {
	return Vector2{
		X: r.Position.X + r.Size.X/2,
		Y: r.Position.Y + r.Size.Y/2,
	}
}

// ContainsPoint checks if a point is inside the rectangle (inclusive).
func (r Rect) ContainsPoint(point Vector2) bool {
	return point.X >= r.Left() && point.X <= r.Right() &&
		point.Y >= r.Top() && point.Y <= r.Bottom()
}

// ContainsRect checks if this rectangle completely contains another rectangle.
func (r Rect) ContainsRect(other Rect) bool {
	return r.Left() <= other.Left() && r.Right() >= other.Right() &&
		r.Top() <= other.Top() && r.Bottom() >= other.Bottom()
}

// Overlaps checks if this rectangle overlaps with another rectangle.
func (r Rect) Overlaps(other Rect) bool {
	return r.Left() < other.Right() && r.Right() > other.Left() &&
		r.Top() < other.Bottom() && r.Bottom() > other.Top()
}

// Intersection returns the overlapping area between two rectangles.
// Returns zero rectangle if they don't overlap.
func (r Rect) Intersection(other Rect) Rect {
	left := max(r.Left(), other.Left())
	right := min(r.Right(), other.Right())
	top := max(r.Top(), other.Top())
	bottom := min(r.Bottom(), other.Bottom())

	if left >= right || top >= bottom {
		return NewRect(0, 0, 0, 0)
	}

	return NewRect(left, top, right-left, bottom-top)
}

// Union returns the smallest rectangle that contains both rectangles.
func (r Rect) Union(other Rect) Rect {
	left := min(r.Left(), other.Left())
	right := max(r.Right(), other.Right())
	top := min(r.Top(), other.Top())
	bottom := max(r.Bottom(), other.Bottom())

	return NewRect(left, top, right-left, bottom-top)
}

// Translate moves the rectangle by the given offset.
func (r Rect) Translate(offset Vector2) Rect {
	return Rect{
		Position: r.Position.Add(offset),
		Size:     r.Size,
	}
}

// Scale scales the rectangle around its center by the given factors.
func (r Rect) Scale(scaleX, scaleY float64) Rect {
	center := r.Center()
	newWidth := r.Width() * scaleX
	newHeight := r.Height() * scaleY

	return Rect{
		Position: Vector2{
			X: center.X - newWidth/2,
			Y: center.Y - newHeight/2,
		},
		Size: NewVector2(newWidth, newHeight),
	}
}

// Area returns the area of the rectangle.
func (r Rect) Area() float64 {
	return r.Width() * r.Height()
}

// Perimeter returns the perimeter (circumference) of the rectangle.
func (r Rect) Perimeter() float64 {
	return 2 * (r.Width() + r.Height())
}

// String returns a string representation of the rectangle.
func (r Rect) String() string {
	return fmt.Sprintf("Rect(Pos: %s, Size: %s)", r.Position.String(), r.Size.String())
}

// Helper functions (not exported, for internal use)
func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func max(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}