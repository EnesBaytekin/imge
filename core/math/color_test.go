package math

import "testing"

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
