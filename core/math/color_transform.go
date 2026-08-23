package math

import (
	"image"
	"image/color"
	"math"
)

// ColorTransform groups the knobs that recolor a sprite's texture. A sprite's
// `color` JSON arg maps onto this struct. The knobs compose in a fixed pipeline
// (see Matrix), so results are predictable.
type ColorTransform struct {
	// Tint is the multiply color applied after any grayscale/hue work. It can only
	// darken, never brighten. Default is white (identity). When Solid is true, Tint
	// is instead the flat fill color of the silhouette.
	Tint Color `json:"tint"`

	// Hue rotates the hue forward by this many degrees (0 = none).
	Hue float64 `json:"hue"`

	// HueTo, when set, rotates the texture's dominant hue to this color's hue. Hue
	// (if non-zero) is added on top as an extra offset. Works best on sprites
	// dominated by a single hue family; a grayscale texture has no dominant hue.
	HueTo *Color `json:"hue_to"`

	// Grayscale desaturates fully, keeping only perceived brightness (luma).
	Grayscale bool `json:"grayscale"`

	// Solid replaces the opaque pixels' RGB with Tint, keeping the source alpha
	// (the sprite's shape). Hue/Grayscale are ignored in this mode.
	Solid bool `json:"solid"`
}

// IsIdentity reports whether the transform does nothing (a plain draw).
func (t ColorTransform) IsIdentity() bool {
	if t.Grayscale || t.Solid || t.Hue != 0 || t.HueTo != nil {
		return false
	}
	return t.Tint == (Color{}) || t.Tint == White
}

// tintOrDefault returns Tint, defaulting to White when unset.
func (t ColorTransform) tintOrDefault() Color {
	if t.Tint == (Color{}) {
		return White
	}
	return t.Tint
}

// hueRotation returns the total hue rotation in degrees: the manual Hue offset
// plus, when HueTo is set, the rotation that maps the source's dominant hue to the
// target hue.
func (t ColorTransform) hueRotation(dominantHue float64) float64 {
	rot := t.Hue
	if t.HueTo != nil {
		hr, hg, hb, _ := t.HueTo.ToFloats()
		rot += HueOf(hr, hg, hb) - dominantHue
	}
	return rot
}

// Matrix resolves the transform into a single ColorMatrix, applying the knobs in
// this fixed order: grayscale -> hue -> tint (multiply) -> solid.
//
// dominantHue is the source texture's dominant hue in degrees and is used only by
// HueTo; pass 0 when it is unknown. Callers with the texture compute it with
// DominantHue.
func (t ColorTransform) Matrix(dominantHue float64) ColorMatrix {
	m := IdentityColorMatrix()

	if t.Grayscale {
		m = grayscaleMatrix().Mul(m)
	}

	if rot := t.hueRotation(dominantHue); rot != 0 {
		m = hueRotateMatrix(rot).Mul(m)
	}

	if t.Solid {
		m = solidMatrix(t.tintOrDefault()).Mul(m)
	} else {
		r, g, b, a := t.tintOrDefault().ToFloats()
		m = scaleMatrix(r, g, b, a).Mul(m)
	}

	return m
}

// ColorMatrix is a 4x5 affine transform on straight-alpha RGBA colors in [0,1].
// Rows are the output channels R,G,B,A; columns are the input channels R,G,B,A plus
// a constant. Applying computes out = M*in + T and clamps each channel to [0,1].
type ColorMatrix struct {
	M [4][4]float64 // M[out][in]
	T [4]float64    // T[out] constant
}

// IdentityColorMatrix returns the identity color matrix.
func IdentityColorMatrix() ColorMatrix {
	return ColorMatrix{
		M: [4][4]float64{
			{1, 0, 0, 0},
			{0, 1, 0, 0},
			{0, 0, 1, 0},
			{0, 0, 0, 1},
		},
	}
}

// Apply applies the matrix to a straight-alpha color in [0,1], clamping to [0,1].
func (m ColorMatrix) Apply(r, g, b, a float64) (float64, float64, float64, float64) {
	or := m.M[0][0]*r + m.M[0][1]*g + m.M[0][2]*b + m.M[0][3]*a + m.T[0]
	og := m.M[1][0]*r + m.M[1][1]*g + m.M[1][2]*b + m.M[1][3]*a + m.T[1]
	ob := m.M[2][0]*r + m.M[2][1]*g + m.M[2][2]*b + m.M[2][3]*a + m.T[2]
	oa := m.M[3][0]*r + m.M[3][1]*g + m.M[3][2]*b + m.M[3][3]*a + m.T[3]
	return clamp01(or), clamp01(og), clamp01(ob), clamp01(oa)
}

// Mul composes two matrices: (m.Mul(n)).Apply(v) == m.Apply(n.Apply(v)). In other
// words, n is applied first, then m.
func (m ColorMatrix) Mul(n ColorMatrix) ColorMatrix {
	var out ColorMatrix
	for i := 0; i < 4; i++ {
		for j := 0; j < 4; j++ {
			var s float64
			for k := 0; k < 4; k++ {
				s += m.M[i][k] * n.M[k][j]
			}
			out.M[i][j] = s
		}
		out.T[i] = m.M[i][0]*n.T[0] + m.M[i][1]*n.T[1] + m.M[i][2]*n.T[2] + m.M[i][3]*n.T[3] + m.T[i]
	}
	return out
}

func clamp01(x float64) float64 {
	if x < 0 {
		return 0
	}
	if x > 1 {
		return 1
	}
	return x
}

// grayscaleMatrix collapses RGB to luma (BT.601) while keeping alpha.
func grayscaleMatrix() ColorMatrix {
	return ColorMatrix{
		M: [4][4]float64{
			{0.2990, 0.5870, 0.1140, 0},
			{0.2990, 0.5870, 0.1140, 0},
			{0.2990, 0.5870, 0.1140, 0},
			{0, 0, 0, 1},
		},
	}
}

// scaleMatrix is a per-channel multiply (tint when not solid).
func scaleMatrix(r, g, b, a float64) ColorMatrix {
	return ColorMatrix{
		M: [4][4]float64{
			{r, 0, 0, 0},
			{0, g, 0, 0},
			{0, 0, b, 0},
			{0, 0, 0, a},
		},
	}
}

// solidMatrix replaces RGB with a constant fill and scales alpha by the fill's
// alpha, so a semi-transparent tint yields a semi-transparent silhouette.
func solidMatrix(fill Color) ColorMatrix {
	r, g, b, a := fill.ToFloats()
	return ColorMatrix{
		M: [4][4]float64{
			{0, 0, 0, 0},
			{0, 0, 0, 0},
			{0, 0, 0, 0},
			{0, 0, 0, a},
		},
		T: [4]float64{r, g, b, 0},
	}
}

// hueRotateMatrix rotates hue forward by angleDeg using the same YCrCb (BT.601)
// conversion Ebitengine's ChangeHSV uses: RGB -> YCrCb -> rotate Cb/Cr -> RGB.
// Luma (perceived brightness) is preserved, so near-black shadows and white
// highlights keep their brightness while the hue shifts; neutral (gray) pixels are
// unchanged.
func hueRotateMatrix(angleDeg float64) ColorMatrix {
	// Ebitengine's ChangeHSV rotates hue backward with a positive theta, so negate
	// the angle to make a positive hue offset mean "forward" on the color wheel.
	sin, cos := math.Sincos(degToRad(-angleDeg))

	rgbToYCbCr := ColorMatrix{
		M: [4][4]float64{
			{0.2990, 0.5870, 0.1140, 0},
			{-0.1687, -0.3313, 0.5000, 0},
			{0.5000, -0.4187, -0.0813, 0},
			{0, 0, 0, 1},
		},
	}
	rotate := ColorMatrix{
		M: [4][4]float64{
			{1, 0, 0, 0},
			{0, cos, sin, 0},
			{0, -sin, cos, 0},
			{0, 0, 0, 1},
		},
	}
	yCbCrToRgb := ColorMatrix{
		M: [4][4]float64{
			{1, 0, 1.40200, 0},
			{1, -0.34414, -0.71414, 0},
			{1, 1.77200, 0, 0},
			{0, 0, 0, 1},
		},
	}
	return yCbCrToRgb.Mul(rotate).Mul(rgbToYCbCr)
}

func degToRad(d float64) float64 { return d * math.Pi / 180 }

// HueOf returns the hue of an RGB color in degrees [0, 360). Neutral (gray) colors
// have no hue; it returns 0 for them.
func HueOf(r, g, b float64) float64 {
	maxC := math.Max(r, math.Max(g, b))
	minC := math.Min(r, math.Min(g, b))
	delta := maxC - minC
	if delta == 0 {
		return 0
	}
	var h float64
	switch maxC {
	case r:
		h = 60 * ((g - b) / delta)
	case g:
		h = 60 * ((b-r)/delta + 2)
	default: // b
		h = 60 * ((r-g)/delta + 4)
	}
	if h < 0 {
		h += 360
	}
	return h
}

// DominantHue returns the dominant hue of an image in degrees [0, 360), computed
// as the circular mean of each pixel's hue weighted by its saturation and alpha.
// Transparent and near-gray pixels contribute little, so a sprite's background and
// shadows don't skew the result. Returns 0 for an image with no colorful pixels
// (e.g. a grayscale texture).
func DominantHue(img image.Image) float64 {
	if img == nil {
		return 0
	}
	bounds := img.Bounds()
	var sumSin, sumCos, sumWeight float64
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			c := color.NRGBAModel.Convert(img.At(x, y)).(color.NRGBA)
			r := uint8ToFloat(c.R)
			g := uint8ToFloat(c.G)
			b := uint8ToFloat(c.B)
			a := uint8ToFloat(c.A)
			if a == 0 {
				continue
			}
			maxC := math.Max(r, math.Max(g, b))
			minC := math.Min(r, math.Min(g, b))
			sat := 0.0
			if maxC > 0 {
				sat = (maxC - minC) / maxC
			}
			w := sat * a
			if w == 0 {
				continue
			}
			rad := HueOf(r, g, b) * math.Pi / 180
			sumSin += w * math.Sin(rad)
			sumCos += w * math.Cos(rad)
			sumWeight += w
		}
	}
	if sumWeight == 0 {
		return 0
	}
	deg := math.Atan2(sumSin, sumCos) * 180 / math.Pi
	if deg < 0 {
		deg += 360
	}
	return deg
}
