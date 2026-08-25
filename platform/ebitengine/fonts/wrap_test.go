package fonts

import (
	"testing"

	"github.com/EnesBaytekin/imge/core"
)

// The pure wrapLines tests use a synthetic measure (byte length) so the wrapping
// algorithm is tested deterministically without depending on the built-in font's
// exact advance widths. The integration tests below exercise Library.Wrap against
// the real "imge-font".

func TestWrapWordBreaksOnSpaces(t *testing.T) {
	measure := func(s string) float64 { return float64(len(s)) }
	got := wrapLines("aaa bbb ccc", 7, core.WrapWord, false, measure)
	assertLines(t, got, []string{"aaa bbb", "ccc"})
}

func TestWrapWordLoneOversizedWordOverflows(t *testing.T) {
	measure := func(s string) float64 { return float64(len(s)) }
	// A word wider than maxWidth still gets its own line (and overflows).
	got := wrapLines("ok supercalifragilistic end", 4, core.WrapWord, false, measure)
	assertLines(t, got, []string{"ok", "supercalifragilistic", "end"})
}

func TestWrapWordPreservesWhitespace(t *testing.T) {
	measure := func(s string) float64 { return float64(len(s)) }
	// Leading spaces, repeated internal spaces, and trailing spaces are kept verbatim.
	got := wrapLines("   a  b   ", 100, core.WrapWord, false, measure)
	assertLines(t, got, []string{"   a  b   "})
}

func TestWrapWordKeepsLeadingSpaces(t *testing.T) {
	measure := func(s string) float64 { return float64(len(s)) }
	// Two leading spaces stay on the first line; the break-space is dropped.
	got := wrapLines("  ab cd", 4, core.WrapWord, false, measure)
	assertLines(t, got, []string{"  ab", "cd"})
}

func TestWrapWordPreservesLeadingSpacesBuiltin(t *testing.T) {
	l := NewLibrary()
	w := l.Wrap(nil, "", 6, "     selam", 280, core.WrapWord, false)
	if len(w.Lines) != 1 || w.Lines[0] != "     selam" {
		t.Fatalf("Wrap(word) = %q, want [\"     selam\"]", w.Lines)
	}
}

func TestWrapCharSplitsWords(t *testing.T) {
	measure := func(s string) float64 { return float64(len(s)) }
	got := wrapLines("abcdefgh", 3, core.WrapChar, false, measure)
	assertLines(t, got, []string{"abc", "def", "gh"})
}

func TestWrapCharOversizedRuneGetsOwnLine(t *testing.T) {
	measure := func(s string) float64 { return float64(len(s)) }
	got := wrapLines("ab", 1, core.WrapChar, false, measure)
	assertLines(t, got, []string{"a", "b"})
}

func TestWrapClipTruncates(t *testing.T) {
	measure := func(s string) float64 { return float64(len(s)) }
	got := wrapLines("abcdef", 4, core.WrapClip, false, measure)
	assertLines(t, got, []string{"abcd"})
}

func TestWrapClipFitsWhole(t *testing.T) {
	measure := func(s string) float64 { return float64(len(s)) }
	got := wrapLines("abc", 4, core.WrapClip, true, measure)
	assertLines(t, got, []string{"abc"})
}

func TestWrapClipEllipsisAppendsAndFits(t *testing.T) {
	measure := func(s string) float64 { return float64(len(s)) }
	// "abc" + "..." = 6 fits; "abcd" + "..." = 7 exceeds maxWidth 6.
	got := wrapLines("abcdefgh", 6, core.WrapClip, true, measure)
	assertLines(t, got, []string{"abc..."})
	if measure(got[0]) > 6 {
		t.Fatalf("ellipsized line %q width %g exceeds maxWidth 6", got[0], measure(got[0]))
	}
}

func TestWrapClipEllipsisNoMarkerWhenFits(t *testing.T) {
	measure := func(s string) float64 { return float64(len(s)) }
	got := wrapLines("abc", 10, core.WrapClip, true, measure)
	assertLines(t, got, []string{"abc"})
}

func TestWrapClipEllipsisTooNarrow(t *testing.T) {
	measure := func(s string) float64 { return float64(len(s)) }
	// "..." (3) is wider than maxWidth 2, so nothing can be shown with the mark.
	got := wrapLines("abcdef", 2, core.WrapClip, true, measure)
	assertLines(t, got, []string{""})
}

func TestWrapPreservesNewlines(t *testing.T) {
	measure := func(s string) float64 { return float64(len(s)) }
	got := wrapLines("a\n\nb", 100, core.WrapWord, false, measure)
	assertLines(t, got, []string{"a", "", "b"})
}

func TestWrapNoMaxWidthKeepsOnlyNewlines(t *testing.T) {
	measure := func(s string) float64 { return float64(len(s)) }
	got := wrapLines("hello world\nsecond line", 0, core.WrapWord, false, measure)
	assertLines(t, got, []string{"hello world", "second line"})
}

func TestWrapMeasuresWithBuiltinFont(t *testing.T) {
	l := NewLibrary()
	_, lineH := l.Measure(nil, "", 6, "X")
	if lineH <= 0 {
		t.Fatal("built-in line height must be positive")
	}

	w := l.Wrap(nil, "", 6, "the quick brown fox jumps over", 40, core.WrapWord, false)
	if len(w.Lines) < 2 {
		t.Fatalf("Wrap(word, maxWidth 40) produced %d lines, want >= 2", len(w.Lines))
	}
	if w.LineHeight != lineH {
		t.Fatalf("LineHeight = %g, want %g", w.LineHeight, lineH)
	}
	if want := float64(len(w.Lines)) * lineH; w.Height != want {
		t.Fatalf("Height = %g, want %g", w.Height, want)
	}

	// Every non-overflowing line fits maxWidth; Width is the widest line.
	var widest float64
	for _, ln := range w.Lines {
		lw, _ := l.Measure(nil, "", 6, ln)
		if lw > widest {
			widest = lw
		}
		if lw > 40+1e-6 {
			t.Fatalf("line %q width %g exceeds maxWidth 40", ln, lw)
		}
	}
	if w.Width != widest {
		t.Fatalf("Width = %g, want widest line %g", w.Width, widest)
	}
}

func TestWrapClipEllipsisFitsBuiltinFont(t *testing.T) {
	l := NewLibrary()
	const maxW = 60.0
	w := l.Wrap(nil, "", 6, "a very long title that overflows the box", maxW, core.WrapClip, true)
	if len(w.Lines) != 1 {
		t.Fatalf("WrapClip lines = %d, want 1", len(w.Lines))
	}
	line := w.Lines[0]
	if len(line) < 3 || line[len(line)-3:] != "..." {
		t.Fatalf("line = %q, want it to end with \"...\"", line)
	}
	lw, _ := l.Measure(nil, "", 6, line)
	if lw > maxW+1e-6 {
		t.Fatalf("ellipsized line %q width %g exceeds maxWidth %g", line, lw, maxW)
	}
}

func TestWrapEmptyText(t *testing.T) {
	l := NewLibrary()
	w := l.Wrap(nil, "", 6, "", 100, core.WrapWord, false)
	if len(w.Lines) != 0 || w.Width != 0 || w.Height != 0 {
		t.Fatalf("Wrap(empty) = %+v, want zero", w)
	}
}

func TestWrapMissingFont(t *testing.T) {
	l := NewLibrary()
	w := l.Wrap(nil, "definitely/missing/font.ttf", 6, "hello", 100, core.WrapWord, false)
	if len(w.Lines) != 0 || w.Width != 0 || w.Height != 0 {
		t.Fatalf("Wrap(missing font) = %+v, want zero", w)
	}
}

func assertLines(t *testing.T, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("got %d lines %q, want %d lines %q", len(got), got, len(want), want)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Fatalf("line %d = %q, want %q (all: %q)", i, got[i], want[i], got)
		}
	}
}
