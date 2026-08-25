package fonts

import "testing"

func TestMeasureDefaultFont(t *testing.T) {
	l := NewLibrary()

	w, h := l.Measure(nil, "", 8, "Hello, IMGE!")
	if w <= 0 || h <= 0 {
		t.Fatalf("Measure(default) = (%g, %g), want positive width and height", w, h)
	}
}

func TestBuiltinFontID(t *testing.T) {
	l := NewLibrary()
	if l.Source(nil, "imge-font") == nil {
		t.Fatal(`Source("imge-font") = nil, want the built-in font`)
	}
	// "" and "imge-font" must resolve to the same built-in face.
	w0, h0 := l.Measure(nil, "", 6, "Hello, imge!")
	w1, h1 := l.Measure(nil, "imge-font", 6, "Hello, imge!")
	if w0 != w1 || h0 != h1 {
		t.Fatalf(`Measure("") = (%g, %g), Measure("imge-font") = (%g, %g); want equal`, w0, h0, w1, h1)
	}
}

func TestMeasureEmpty(t *testing.T) {
	l := NewLibrary()
	if w, h := l.Measure(nil, "", 8, ""); w != 0 || h != 0 {
		t.Fatalf("Measure(empty) = (%g, %g), want (0, 0)", w, h)
	}
}

func TestMeasureMissingFont(t *testing.T) {
	l := NewLibrary()
	if w, h := l.Measure(nil, "definitely/missing/font.ttf", 8, "Hello"); w != 0 || h != 0 {
		t.Fatalf("Measure(missing font) = (%g, %g), want (0, 0)", w, h)
	}
}

func TestMeasureDefaultSize(t *testing.T) {
	l := NewLibrary()
	w0, h0 := l.Measure(nil, "", 0, "Hello") // size <= 0 selects the default
	w8, h8 := l.Measure(nil, "", DefaultSize, "Hello")
	if w0 != w8 || h0 != h8 {
		t.Fatalf("Measure(size 0) = (%g, %g), size %g = (%g, %g); want equal", w0, h0, DefaultSize, w8, h8)
	}
}

func TestMeasureScalesWithSize(t *testing.T) {
	l := NewLibrary()
	w8, _ := l.Measure(nil, "", 8, "Hello")
	w16, _ := l.Measure(nil, "", 16, "Hello")
	// Doubling the size should roughly double the advance width.
	if w16 < w8*1.5 || w16 > w8*2.5 {
		t.Fatalf("width at size 16 = %g, width at size 8 = %g; want ~2x", w16, w8)
	}
}
