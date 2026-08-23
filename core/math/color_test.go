package math

import (
	"encoding/json"
	"testing"
)

func TestParseHex(t *testing.T) {
	cases := []struct {
		in   string
		want Color
	}{
		{"#000000", Color{R: 0, G: 0, B: 0, A: 255}},
		{"#FFFFFF", Color{R: 255, G: 255, B: 255, A: 255}},
		{"#FF0000", Color{R: 255, G: 0, B: 0, A: 255}},
		{"#00FF00", Color{R: 0, G: 255, B: 0, A: 255}},
		{"#0000FF", Color{R: 0, G: 0, B: 255, A: 255}},
		{"#11223344", Color{R: 0x11, G: 0x22, B: 0x33, A: 0x44}},
		{"#0f0", Color{R: 0, G: 255, B: 0, A: 255}},
		{"#f00f", Color{R: 255, G: 0, B: 0, A: 255}},
		{"112233", Color{R: 0x11, G: 0x22, B: 0x33, A: 255}}, // no leading '#'
	}

	for _, c := range cases {
		got, err := ParseHex(c.in)
		if err != nil {
			t.Fatalf("ParseHex(%q): unexpected error %v", c.in, err)
		}
		if got != c.want {
			t.Errorf("ParseHex(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestParseHexInvalid(t *testing.T) {
	for _, in := range []string{"", "#12", "#12345", "#GGGGGG", "not-a-color"} {
		if _, err := ParseHex(in); err == nil {
			t.Errorf("ParseHex(%q): expected error, got nil", in)
		}
	}
}

func TestPremultiplied(t *testing.T) {
	almost := func(a, b float64) bool {
		d := a - b
		if d < 0 {
			d = -d
		}
		return d < 1e-6
	}

	half := 128.0 / 255.0 // alpha 128 in [0,1]

	cases := []struct {
		name  string
		color Color
		wantR float64
		wantG float64
		wantB float64
		wantA float64
	}{
		{"opaque white is identity", White, 1, 1, 1, 1},
		{"opaque red", NewColor(255, 0, 0, 255), 1, 0, 0, 1},
		{"translucent white premultiplies", NewColor(255, 255, 255, 128), half, half, half, half},
		{"translucent red premultiplies red", NewColor(255, 0, 0, 128), half, 0, 0, half},
		{"fully transparent zeroes", NewColor(255, 0, 0, 0), 0, 0, 0, 0},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r, g, b, a := c.color.Premultiplied()
			if !almost(r, c.wantR) || !almost(g, c.wantG) || !almost(b, c.wantB) || !almost(a, c.wantA) {
				t.Fatalf("Premultiplied(%v) = (%v, %v, %v, %v), want (%v, %v, %v, %v)",
					c.color, r, g, b, a, c.wantR, c.wantG, c.wantB, c.wantA)
			}
		})
	}
}

func TestColorUnmarshalJSON(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want Color
	}{
		{"object full", `{"r":255,"g":0,"b":0,"a":255}`, Red},
		{"object omits alpha, defaults to 255", `{"r":255,"g":0,"b":0}`, Red},
		{"object explicit zero alpha", `{"r":255,"g":0,"b":0,"a":0}`, Color{R: 255, A: 0}},
		{"hex full", `"#ff0000"`, Red},
		{"hex short", `"#f00"`, Red},
		{"hex with alpha", `"#ff000080"`, Color{R: 255, A: 128}},
		{"hex no hash", `"00ff00"`, Green},
		{"null is zero value", `null`, Color{}},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var got Color
			if err := json.Unmarshal([]byte(c.in), &got); err != nil {
				t.Fatalf("Unmarshal(%q): unexpected error %v", c.in, err)
			}
			if got != c.want {
				t.Errorf("Unmarshal(%q) = %v, want %v", c.in, got, c.want)
			}
		})
	}
}

func TestColorUnmarshalJSONInvalid(t *testing.T) {
	for _, in := range []string{`"#zzz"`, `"#"`, `"nope"`, `123`} {
		var c Color
		if err := json.Unmarshal([]byte(in), &c); err == nil {
			t.Errorf("Unmarshal(%s): expected error, got nil", in)
		}
	}
}

func TestColorTransformUnmarshalHueTo(t *testing.T) {
	var ct ColorTransform
	if err := json.Unmarshal([]byte(`{"hue_to":"#00ff00"}`), &ct); err != nil {
		t.Fatalf("unmarshal hue_to hex: %v", err)
	}
	if ct.HueTo == nil || *ct.HueTo != Green {
		t.Fatalf("hue_to = %v, want green", ct.HueTo)
	}

	var ctNil ColorTransform
	if err := json.Unmarshal([]byte(`{"hue_to":null}`), &ctNil); err != nil {
		t.Fatalf("unmarshal hue_to null: %v", err)
	}
	if ctNil.HueTo != nil {
		t.Fatalf("hue_to = %v, want nil", ctNil.HueTo)
	}
}
