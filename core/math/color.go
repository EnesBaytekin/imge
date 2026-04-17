// Package math provides mathematical utilities for the game engine.
package math

import (
	"fmt"
	"math"
)

// Color represents a 32-bit RGBA color with 8 bits per channel.
// This is a common format for graphics APIs and image processing.
type Color struct {
	R, G, B, A uint8
}

// NewColor creates a new color from RGBA values (0-255).
func NewColor(r, g, b, a uint8) Color {
	return Color{R: r, G: g, B: b, A: a}
}

// NewColorFromFloats creates a new color from float values (0.0 to 1.0).
func NewColorFromFloats(r, g, b, a float64) Color {
	return Color{
		R: floatToUint8(r),
		G: floatToUint8(g),
		B: floatToUint8(b),
		A: floatToUint8(a),
	}
}

// NewColorFromHex creates a color from a hexadecimal value (0xRRGGBBAA or 0xRRGGBB).
func NewColorFromHex(hex uint32) Color {
	if hex <= 0xFFFFFF {
		// No alpha specified, assume fully opaque
		return Color{
			R: uint8((hex >> 16) & 0xFF),
			G: uint8((hex >> 8) & 0xFF),
			B: uint8(hex & 0xFF),
			A: 255,
		}
	}
	// With alpha
	return Color{
		R: uint8((hex >> 24) & 0xFF),
		G: uint8((hex >> 16) & 0xFF),
		B: uint8((hex >> 8) & 0xFF),
		A: uint8(hex & 0xFF),
	}
}

// ToFloats converts the color to float values (0.0 to 1.0).
func (c Color) ToFloats() (r, g, b, a float64) {
	return uint8ToFloat(c.R), uint8ToFloat(c.G), uint8ToFloat(c.B), uint8ToFloat(c.A)
}

// Lerp linearly interpolates between this color and another by t (0 to 1).
func (c Color) Lerp(target Color, t float64) Color {
	r1, g1, b1, a1 := c.ToFloats()
	r2, g2, b2, a2 := target.ToFloats()

	return NewColorFromFloats(
		r1+(r2-r1)*t,
		g1+(g2-g1)*t,
		b1+(b2-b1)*t,
		a1+(a2-a1)*t,
	)
}

// Multiply multiplies the color by another color (component-wise multiplication).
func (c Color) Multiply(other Color) Color {
	r1, g1, b1, a1 := c.ToFloats()
	r2, g2, b2, a2 := other.ToFloats()

	return NewColorFromFloats(
		r1*r2,
		g1*g2,
		b1*b2,
		a1*a2,
	)
}

// Scale scales the color by a scalar factor (multiplies all components).
func (c Color) Scale(factor float64) Color {
	r, g, b, a := c.ToFloats()
	return NewColorFromFloats(
		r*factor,
		g*factor,
		b*factor,
		a,
	)
}

// WithAlpha returns a new color with the specified alpha value.
func (c Color) WithAlpha(alpha uint8) Color {
	return Color{R: c.R, G: c.G, B: c.B, A: alpha}
}

// WithAlphaFloat returns a new color with the specified alpha value (0.0 to 1.0).
func (c Color) WithAlphaFloat(alpha float64) Color {
	return NewColorFromFloats(
		uint8ToFloat(c.R),
		uint8ToFloat(c.G),
		uint8ToFloat(c.B),
		alpha,
	)
}

// Equals checks if two colors are exactly equal.
func (c Color) Equals(other Color) bool {
	return c.R == other.R && c.G == other.G && c.B == other.B && c.A == other.A
}

// Hex returns the color as a 32-bit hexadecimal value (0xRRGGBBAA).
func (c Color) Hex() uint32 {
	return (uint32(c.R) << 24) | (uint32(c.G) << 16) | (uint32(c.B) << 8) | uint32(c.A)
}

// String returns a string representation of the color.
func (c Color) String() string {
	return fmt.Sprintf("Color(R:%d, G:%d, B:%d, A:%d)", c.R, c.G, c.B, c.A)
}

// Predefined colors for convenience
var (
	Black       = NewColor(0, 0, 0, 255)
	White       = NewColor(255, 255, 255, 255)
	Red         = NewColor(255, 0, 0, 255)
	Green       = NewColor(0, 255, 0, 255)
	Blue        = NewColor(0, 0, 255, 255)
	Yellow      = NewColor(255, 255, 0, 255)
	Magenta     = NewColor(255, 0, 255, 255)
	Cyan        = NewColor(0, 255, 255, 255)
	Transparent = NewColor(0, 0, 0, 0)
	Gray        = NewColor(128, 128, 128, 255)
	LightGray   = NewColor(192, 192, 192, 255)
	DarkGray    = NewColor(64, 64, 64, 255)
)

// Helper functions for float/uint8 conversion
func floatToUint8(f float64) uint8 {
	// Clamp to 0-1 range
	f = math.Max(0, math.Min(1, f))
	return uint8(f * 255)
}

func uint8ToFloat(u uint8) float64 {
	return float64(u) / 255.0
}