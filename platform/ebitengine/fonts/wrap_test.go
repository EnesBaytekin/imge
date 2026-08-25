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
	got := wrapLines("aaa bbb ccc", 7, core.WrapWord, measure)
	assertLines(t, got, []string{"aaa bbb", "ccc"})
}

func TestWrapWordLoneOversizedWordOverflows(t *testing.T) {
	measure := func(s string) float64 { return float64(len(s)) }
	// A word wider than maxWidth still gets its own line (and overflows).
	got := wrapLines("ok supercalifragilistic end", 4, core.WrapWord, measure)
	assertLines(t, got, []string{"ok", "supercalifragilistic", "end"})
}

func TestWrapCharSplitsWords(t *testing.T) {
	measure := func(s string) float64 { return float64(len(s)) }
	got := wrapLines("abcdefgh", 3, core.WrapChar, measure)
	assertLines(t, got, []string{"abc", "def", "gh"})
}

func TestWrapCharOversizedRuneGetsOwnLine(t *testing.T) {
	measure := func(s string) float64 { return float64(len(s)) }
	got := wrapLines("ab", 1, core.WrapChar, measure)
	assertLines(t, got, []string{"a", "b"})
}

func TestWrapClipTruncates(t *testing.T) {
	measure := func(s string) float64 { return float64(len(s)) }
	got := wrapLines("abcdef", 4, core.WrapClip, measure)
	assertLines(t, got, []string{"abcd"})
}

func TestWrapClipFitsWhole(t *testing.T) {
	measure := func(s string) float64 { return float64(len(s)) }
	got := wrapLines("abc", 4, core.WrapClip, measure)
	assertLines(t, got, []string{"abc"})
}

func TestWrapPreservesNewlines(t *testing.T) {
	measure := func(s string) float64 { return float64(len(s)) }
	got := wrapLines("a\n\nb", 100, core.WrapWord, measure)
	assertLines(t, got, []string{"a", "", "b"})
}

func TestWrapNoMaxWidthKeepsOnlyNewlines(t *testing.T) {
	measure := func(s string) float64 { return float64(len(s)) }
	got := wrapLines("hello world\nsecond line", 0, core.WrapWord, measure)
	assertLines(t, got, []string{"hello world", "second line"})
}

func TestWrapMeasuresWithBuiltinFont(t *testing.T) {
	l := NewLibrary()
	_, lineH := l.Measure(nil, "", 6, "X")
	if lineH <= 0 {
		t.Fatal("built-in line height must be positive")
	}

	w := l.Wrap(nil, "", 6, "the quick brown fox jumps over", 40, core.WrapWord)
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

func TestWrapEmptyText(t *testing.T) {
	l := NewLibrary()
	w := l.Wrap(nil, "", 6, "", 100, core.WrapWord)
	if len(w.Lines) != 0 || w.Width != 0 || w.Height != 0 {
		t.Fatalf("Wrap(empty) = %+v, want zero", w)
	}
}

func TestWrapMissingFont(t *testing.T) {
	l := NewLibrary()
	w := l.Wrap(nil, "definitely/missing/font.ttf", 6, "hello", 100, core.WrapWord)
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
