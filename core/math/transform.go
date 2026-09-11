// Package math provides mathematical utilities for the game engine.
package math

import (
	"fmt"
	"math"
)

// Transform represents a 2D transformation (position, rotation, scale).
// This is a core component for positioning game objects in the world.
type Transform struct {
	Position Vector2 // Translation in world space
	Rotation float64 // Rotation in radians (0 = facing right, positive = counter-clockwise)
	Scale    Vector2 // Scale factors (1,1 = original size)
}

// NewTransform creates a new transform with default values.
func NewTransform() Transform {
	return Transform{
		Position: Zero(),
		Rotation: 0,
		Scale:    One(),
	}
}

// NewTransformWithPosition creates a new transform with the given position.
func NewTransformWithPosition(x, y float64) Transform {
	return Transform{
		Position: NewVector2(x, y),
		Rotation: 0,
		Scale:    One(),
	}
}

// Identity returns an identity transform (no translation, rotation, or scale).
func Identity() Transform {
	return Transform{
		Position: Zero(),
		Rotation: 0,
		Scale:    One(),
	}
}

// Translate moves the transform by the given offset.
func (t Transform) Translate(offset Vector2) Transform {
	return Transform{
		Position: t.Position.Add(offset),
		Rotation: t.Rotation,
		Scale:    t.Scale,
	}
}

// Rotate rotates the transform by the given angle (in radians).
func (t Transform) Rotate(angle float64) Transform {
	return Transform{
		Position: t.Position,
		Rotation: t.Rotation + angle,
		Scale:    t.Scale,
	}
}

// ScaleBy scales the transform by the given factors.
func (t Transform) ScaleBy(scaleX, scaleY float64) Transform {
	return Transform{
		Position: t.Position,
		Rotation: t.Rotation,
		Scale: Vector2{
			X: t.Scale.X * scaleX,
			Y: t.Scale.Y * scaleY,
		},
	}
}

// UniformScale scales the transform uniformly (same factor for X and Y).
func (t Transform) UniformScale(factor float64) Transform {
	return t.ScaleBy(factor, factor)
}

// LocalToWorld transforms a local point to world space.
func (t Transform) LocalToWorld(localPoint Vector2) Vector2 {
	// Apply scale
	scaled := Vector2{
		X: localPoint.X * t.Scale.X,
		Y: localPoint.Y * t.Scale.Y,
	}

	// Apply rotation
	cos := math.Cos(t.Rotation)
	sin := math.Sin(t.Rotation)
	rotated := Vector2{
		X: scaled.X*cos - scaled.Y*sin,
		Y: scaled.X*sin + scaled.Y*cos,
	}

	// Apply translation
	return rotated.Add(t.Position)
}

// WorldToLocal transforms a world point to local space (inverse transform).
func (t Transform) WorldToLocal(worldPoint Vector2) Vector2 {
	// Subtract translation
	translated := worldPoint.Subtract(t.Position)

	// Apply inverse rotation
	cos := math.Cos(-t.Rotation)
	sin := math.Sin(-t.Rotation)
	rotated := Vector2{
		X: translated.X*cos - translated.Y*sin,
		Y: translated.X*sin + translated.Y*cos,
	}

	// Apply inverse scale (avoid division by zero)
	invScaleX := 1.0
	invScaleY := 1.0
	if t.Scale.X != 0 {
		invScaleX = 1.0 / t.Scale.X
	}
	if t.Scale.Y != 0 {
		invScaleY = 1.0 / t.Scale.Y
	}

	return Vector2{
		X: rotated.X * invScaleX,
		Y: rotated.Y * invScaleY,
	}
}

// RectBounds returns the axis-aligned bounding box of a local-space rectangle after
// this transform is applied (scale -> rotate about the origin -> translate). It is
// the world-space AABB a component's local rect occupies, used for hit-testing and
// editor selection of rotated/scaled objects. A zero rotation and unit scale return
// the rect translated by Position.
func (t Transform) RectBounds(rect Rect) Rect {
	c0 := t.LocalToWorld(NewVector2(rect.X(), rect.Y()))
	c1 := t.LocalToWorld(NewVector2(rect.X()+rect.Width(), rect.Y()))
	c2 := t.LocalToWorld(NewVector2(rect.X()+rect.Width(), rect.Y()+rect.Height()))
	c3 := t.LocalToWorld(NewVector2(rect.X(), rect.Y()+rect.Height()))
	minX := min(c0.X, min(c1.X, min(c2.X, c3.X)))
	minY := min(c0.Y, min(c1.Y, min(c2.Y, c3.Y)))
	maxX := max(c0.X, max(c1.X, max(c2.X, c3.X)))
	maxY := max(c0.Y, max(c1.Y, max(c2.Y, c3.Y)))
	return NewRect(minX, minY, maxX-minX, maxY-minY)
}

// GetForward returns the forward direction vector (facing direction based on rotation).
func (t Transform) GetForward() Vector2 {
	return Vector2{
		X: math.Cos(t.Rotation),
		Y: math.Sin(t.Rotation),
	}
}

// GetRight returns the right direction vector (perpendicular to forward).
func (t Transform) GetRight() Vector2 {
	return Vector2{
		X: -math.Sin(t.Rotation),
		Y: math.Cos(t.Rotation),
	}
}

// LookAt rotates the transform to look at a target point.
func (t Transform) LookAt(target Vector2) Transform {
	direction := target.Subtract(t.Position)
	if direction.LengthSquared() == 0 {
		return t // No change if already at target
	}

	angle := math.Atan2(direction.Y, direction.X)
	return Transform{
		Position: t.Position,
		Rotation: angle,
		Scale:    t.Scale,
	}
}

// Combine combines this transform with another (applies other transform first, then this one).
func (t Transform) Combine(other Transform) Transform {
	// Combined position: apply other's transform to this position
	combinedPos := other.LocalToWorld(t.Position)

	// Combined rotation
	combinedRot := t.Rotation + other.Rotation

	// Combined scale (component-wise multiplication)
	combinedScale := Vector2{
		X: t.Scale.X * other.Scale.X,
		Y: t.Scale.Y * other.Scale.Y,
	}

	return Transform{
		Position: combinedPos,
		Rotation: combinedRot,
		Scale:    combinedScale,
	}
}

// Inverse returns the inverse transform.
func (t Transform) Inverse() Transform {
	// Inverse scale
	invScaleX := 1.0
	invScaleY := 1.0
	if t.Scale.X != 0 {
		invScaleX = 1.0 / t.Scale.X
	}
	if t.Scale.Y != 0 {
		invScaleY = 1.0 / t.Scale.Y
	}

	// Inverse rotation
	invRot := -t.Rotation

	// Inverse position: apply inverse rotation and scale to negative position
	negPos := t.Position.Multiply(-1)
	cos := math.Cos(invRot)
	sin := math.Sin(invRot)
	invPos := Vector2{
		X: (negPos.X*cos - negPos.Y*sin) * invScaleX,
		Y: (negPos.X*sin + negPos.Y*cos) * invScaleY,
	}

	return Transform{
		Position: invPos,
		Rotation: invRot,
		Scale:    Vector2{X: invScaleX, Y: invScaleY},
	}
}

// String returns a string representation of the transform.
func (t Transform) String() string {
	return fmt.Sprintf("Transform(Pos: %s, Rot: %.2f rad, Scale: %s)",
		t.Position.String(), t.Rotation, t.Scale.String())
}

// DegreesToRadians converts degrees to radians.
func DegreesToRadians(degrees float64) float64 {
	return degrees * math.Pi / 180.0
}

// RadiansToDegrees converts radians to degrees.
func RadiansToDegrees(radians float64) float64 {
	return radians * 180.0 / math.Pi
}
