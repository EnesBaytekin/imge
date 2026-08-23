// Package math provides mathematical utilities for the game engine.
package math

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"
)

// Color represents a 32-bit RGBA color with 8 bits per channel.
// This is a common format for graphics APIs and image processing.
//
// The json tags let a color be configured from component args as
// {"r":255,"g":0,"b":0,"a":255}.
type Color struct {
	R uint8 `json:"r"`
	G uint8 `json:"g"`
	B uint8 `json:"b"`
	A uint8 `json:"a"`
}

// UnmarshalJSON lets a Color be configured from either an RGBA object
// ({"r":255,"g":0,"b":0,"a":255}) or a hex string ("#RRGGBB"). The alpha channel
// defaults to 255 (opaque) when omitted. A JSON null is a no-op, matching the
// encoding/json convention.
func (c *Color) UnmarshalJSON(data []byte) error {
	s := strings.TrimSpace(string(data))
	if s == "null" {
		return nil
	}

	// Hex string form.
	if len(s) > 0 && s[0] == '"' {
		var str string
		if err := json.Unmarshal(data, &str); err != nil {
			return err
		}
		parsed, err := ParseHex(str)
		if err != nil {
			return err
		}
		*c = parsed
		return nil
	}

	// Object form. Decode into a plain struct (so this method isn't re-entered),
	// with A as a pointer so an omitted key is distinguishable from an explicit 0.
	var obj struct {
		R uint8  `json:"r"`
		G uint8  `json:"g"`
		B uint8  `json:"b"`
		A *uint8 `json:"a"`
	}
	if err := json.Unmarshal(data, &obj); err != nil {
		return err
	}
	a := uint8(255)
	if obj.A != nil {
		a = *obj.A
	}
	*c = Color{R: obj.R, G: obj.G, B: obj.B, A: a}
	return nil
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

// ParseHex parses a color from a hex string. Accepts "#RGB", "#RGBA",
// "#RRGGBB", and "#RRGGBBAA" (the leading "#" is optional). Returns an error on
// invalid input.
func ParseHex(s string) (Color, error) {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "#")

	// Expand a short form (#RGB / #RGBA) to its full-length equivalent.
	switch len(s) {
	case 3, 4:
		var b strings.Builder
		for i := 0; i < len(s); i++ {
			b.WriteByte(s[i])
			b.WriteByte(s[i])
		}
		s = b.String()
	case 6, 8:
		// already full-length
	default:
		return Color{}, fmt.Errorf("invalid hex color %q: expected #RGB, #RGBA, #RRGGBB, or #RRGGBBAA", s)
	}

	pair := func(i int) (uint8, error) {
		hi, ok := hexDigit(s[i])
		if !ok {
			return 0, fmt.Errorf("invalid hex digit %q", s[i])
		}
		lo, ok := hexDigit(s[i+1])
		if !ok {
			return 0, fmt.Errorf("invalid hex digit %q", s[i+1])
		}
		return hi<<4 | lo, nil
	}

	r, err := pair(0)
	if err != nil {
		return Color{}, err
	}
	g, err := pair(2)
	if err != nil {
		return Color{}, err
	}
	b, err := pair(4)
	if err != nil {
		return Color{}, err
	}

	a := uint8(255)
	if len(s) == 8 {
		a, err = pair(6)
		if err != nil {
			return Color{}, err
		}
	}

	return Color{R: r, G: g, B: b, A: a}, nil
}

func hexDigit(c byte) (uint8, bool) {
	switch {
	case c >= '0' && c <= '9':
		return c - '0', true
	case c >= 'a' && c <= 'f':
		return c - 'a' + 10, true
	case c >= 'A' && c <= 'F':
		return c - 'A' + 10, true
	}
	return 0, false
}

// ToFloats converts the color to float values (0.0 to 1.0).
func (c Color) ToFloats() (r, g, b, a float64) {
	return uint8ToFloat(c.R), uint8ToFloat(c.G), uint8ToFloat(c.B), uint8ToFloat(c.A)
}

// Premultiplied returns the color's RGBA premultiplied by alpha, as floats in
// [0, 1]. Straight RGBA is (r, g, b, a); premultiplied is (r*a, g*a, b*a, a).
// Pipelines that operate on premultiplied-alpha colors (e.g. a color scale
// applied to a texture) need this form.
func (c Color) Premultiplied() (r, g, b, a float64) {
	r = uint8ToFloat(c.R)
	g = uint8ToFloat(c.G)
	b = uint8ToFloat(c.B)
	a = uint8ToFloat(c.A)
	r *= a
	g *= a
	b *= a
	return
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
