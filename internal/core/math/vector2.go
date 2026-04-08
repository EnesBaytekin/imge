// Package math provides mathematical utilities for the game engine.
package math

import (
	"math"
)

// Vector2 represents a 2D vector with X and Y coordinates.
// We use float64 for precision in game calculations.
type Vector2 struct {
	X, Y float64
}

// NewVector2 creates a new Vector2 with the given coordinates.
func NewVector2(x, y float64) Vector2 {
	return Vector2{X: x, Y: y}
}

// Zero returns a zero vector (0, 0).
func Zero() Vector2 {
	return Vector2{X: 0, Y: 0}
}

// One returns a vector with both components set to 1.
func One() Vector2 {
	return Vector2{X: 1, Y: 1}
}

// Add returns the sum of this vector and another.
func (v Vector2) Add(other Vector2) Vector2 {
	return Vector2{X: v.X + other.X, Y: v.Y + other.Y}
}

// Subtract returns the difference between this vector and another.
func (v Vector2) Subtract(other Vector2) Vector2 {
	return Vector2{X: v.X - other.X, Y: v.Y - other.Y}
}

// Multiply scales the vector by a scalar value.
func (v Vector2) Multiply(scalar float64) Vector2 {
	return Vector2{X: v.X * scalar, Y: v.Y * scalar}
}

// Divide scales the vector by dividing by a scalar value.
// Returns zero vector if scalar is zero to avoid division by zero.
func (v Vector2) Divide(scalar float64) Vector2 {
	if scalar == 0 {
		return Zero()
	}
	return Vector2{X: v.X / scalar, Y: v.Y / scalar}
}

// Length returns the magnitude (length) of the vector.
func (v Vector2) Length() float64 {
	return math.Sqrt(v.X*v.X + v.Y*v.Y)
}

// LengthSquared returns the squared length of the vector (faster than Length).
func (v Vector2) LengthSquared() float64 {
	return v.X*v.X + v.Y*v.Y
}

// Normalize returns a unit vector (length 1) in the same direction.
// If the vector is zero, returns zero vector.
func (v Vector2) Normalize() Vector2 {
	length := v.Length()
	if length == 0 {
		return Zero()
	}
	return Vector2{X: v.X / length, Y: v.Y / length}
}

// Dot returns the dot product of this vector and another.
func (v Vector2) Dot(other Vector2) float64 {
	return v.X*other.X + v.Y*other.Y
}

// Distance returns the distance between this vector and another.
func (v Vector2) Distance(other Vector2) float64 {
	dx := v.X - other.X
	dy := v.Y - other.Y
	return math.Sqrt(dx*dx + dy*dy)
}

// DistanceSquared returns the squared distance (faster than Distance).
func (v Vector2) DistanceSquared(other Vector2) float64 {
	dx := v.X - other.X
	dy := v.Y - other.Y
	return dx*dx + dy*dy
}

// Lerp linearly interpolates between this vector and another by t (0 to 1).
func (v Vector2) Lerp(target Vector2, t float64) Vector2 {
	return Vector2{
		X: v.X + (target.X-v.X)*t,
		Y: v.Y + (target.Y-v.Y)*t,
	}
}

// Equals checks if two vectors are approximately equal (within epsilon).
func (v Vector2) Equals(other Vector2, epsilon float64) bool {
	return math.Abs(v.X-other.X) <= epsilon && math.Abs(v.Y-other.Y) <= epsilon
}

// String returns a string representation of the vector.
func (v Vector2) String() string {
	return fmt.Sprintf("Vector2(%f, %f)", v.X, v.Y)
}

// Note: We're keeping the math package simple and focused on 2D game needs.
// This will be used for positions, velocities, scales, and other 2D calculations.