package fonts

import (
	"io/fs"
	"strings"
	"unicode/utf8"

	"github.com/EnesBaytekin/imge/core"
	"golang.org/x/image/font"
)

// WrappedText is the result of breaking a string into lines to fit a maximum
// width. Lines are in order; Width is the widest line's advance width, and Height
// is the total block height (len(Lines) × LineHeight), all in logical units.
type WrappedText struct {
	Lines      []string
	Width      float64
	Height     float64
	LineHeight float64
}

// Wrap breaks text into lines that each fit within maxWidth (in logical units) at
// the given font and size, honoring explicit newlines. maxWidth <= 0 disables
// width-based breaking (each "\n" still starts a new line). An empty text returns a
// zero WrappedText.
//
// ellipsis only affects WrapClip: when a line is truncated and ellipsis is true, a
// trailing "..." is appended (the result still fits maxWidth); when false, the line
// is cut with no marker.
func (l *Library) Wrap(fsys fs.FS, fontID string, size float64, text string, maxWidth float64, mode core.WrapMode, ellipsis bool) WrappedText {
	if text == "" {
		return WrappedText{}
	}
	face := l.Face(fsys, fontID, size)
	if face == nil {
		return WrappedText{}
	}

	m := face.Metrics()
	lineHeight := float64(m.Ascent+m.Descent) / 64
	measure := func(s string) float64 {
		return float64(font.MeasureString(face, s)) / 64
	}

	lines := wrapLines(text, maxWidth, mode, ellipsis, measure)
	res := WrappedText{Lines: lines, LineHeight: lineHeight}
	for _, ln := range lines {
		if w := measure(ln); w > res.Width {
			res.Width = w
		}
	}
	res.Height = float64(len(lines)) * lineHeight
	return res
}

// wrapLines is the pure line-breaking core: it takes a measure callback (advance
// width of a substring, in logical units) so the algorithm is unit-testable without
// a font. Explicit "\n" always produce hard breaks (and empty lines).
func wrapLines(text string, maxWidth float64, mode core.WrapMode, ellipsis bool, measure func(string) float64) []string {
	paragraphs := strings.Split(text, "\n")
	if maxWidth <= 0 {
		return paragraphs
	}

	var lines []string
	for _, para := range paragraphs {
		switch mode {
		case core.WrapClip:
			lines = append(lines, clipLine(para, maxWidth, ellipsis, measure))
		case core.WrapChar:
			lines = append(lines, breakChar(para, maxWidth, measure)...)
		default: // core.WrapWord
			lines = append(lines, breakWord(para, maxWidth, measure)...)
		}
	}
	return lines
}

// ellipsisMark is appended to a truncated WrapClip line when ellipsis is requested.
const ellipsisMark = "..."

// clipLine truncates text to the longest prefix (on a rune boundary) that fits
// maxWidth, dropping the rest — no wrapping. When ellipsis is true and the text is
// truncated, a trailing "..." is appended so the result still fits maxWidth.
func clipLine(text string, maxWidth float64, ellipsis bool, measure func(string) float64) string {
	if measure(text) <= maxWidth {
		return text
	}
	// Accumulate per-rune widths in one pass (O(n)); the old code re-measured the
	// growing prefix text[:end] each step (O(n²)). Summing single-rune advances is
	// exact for every face here: opentype's Kern returns 0, so MeasureString is a
	// plain sum of glyph advances.
	cut := 0
	width := 0.0
	if ellipsis {
		// Reserve room for the trailing marker up front; if even the marker alone
		// doesn't fit, nothing can be shown.
		width = measure(ellipsisMark)
		if width > maxWidth {
			return ""
		}
	}
	for end := 0; end < len(text); {
		_, size := utf8.DecodeRuneInString(text[end:])
		w := measure(text[end : end+size])
		if width+w > maxWidth {
			break
		}
		width += w
		end += size
		cut = end
	}
	if ellipsis {
		return text[:cut] + ellipsisMark
	}
	return text[:cut]
}

// breakChar breaks text into lines of at most maxWidth, splitting a rune sequence
// mid-way (terminal-style hard wrap). A single rune wider than maxWidth still gets
// its own line.
func breakChar(text string, maxWidth float64, measure func(string) float64) []string {
	if text == "" {
		return []string{""}
	}
	var lines []string
	start := 0
	width := 0.0
	for i := 0; i < len(text); {
		// Measure one rune at a time and accumulate, so the whole wrap is O(n): the
		// old code re-measured the growing prefix text[start:next] each step (O(n²)).
		_, size := utf8.DecodeRuneInString(text[i:])
		w := measure(text[i : i+size])
		// Always keep at least one rune on the line, so an oversized rune (wider than
		// maxWidth) still gets its own line and never loops forever.
		if i > start && width+w > maxWidth {
			lines = append(lines, text[start:i])
			start = i
			width = 0
		}
		width += w
		i += size
	}
	lines = append(lines, text[start:])
	return lines
}

// breakWord breaks text into lines on whitespace, keeping whole words together,
// while preserving the whitespace exactly as it appears: leading spaces (indentation),
// repeated spaces between words, and trailing spaces are all kept verbatim. Only the
// space at a wrap point is dropped. A lone word wider than maxWidth is placed on its
// own line and overflows.
func breakWord(text string, maxWidth float64, measure func(string) float64) []string {
	var lines []string
	var line strings.Builder
	var pendingSpaces string // spaces between the current line and the next word

	flush := func() {
		lines = append(lines, line.String())
		line.Reset()
	}

	for i := 0; i < len(text); {
		r, size := utf8.DecodeRuneInString(text[i:])
		if isWordSpace(r) {
			// A run of spaces: leading ones are indentation (kept on the line);
			// ones after a word are held as the separator and dropped only if the
			// next word wraps to a new line.
			j := i + size
			for j < len(text) {
				r2, sz := utf8.DecodeRuneInString(text[j:])
				if !isWordSpace(r2) {
					break
				}
				j += sz
			}
			if line.Len() == 0 {
				line.WriteString(text[i:j])
			} else {
				pendingSpaces = text[i:j]
			}
			i = j
			continue
		}

		// A run of non-space characters: one word.
		j := i + size
		for j < len(text) {
			r2, sz := utf8.DecodeRuneInString(text[j:])
			if isWordSpace(r2) {
				break
			}
			j += sz
		}
		word := text[i:j]

		switch {
		case line.Len() == 0:
			line.WriteString(word)
		case measure(line.String()+pendingSpaces+word) <= maxWidth:
			line.WriteString(pendingSpaces)
			line.WriteString(word)
		default:
			flush()
			line.WriteString(word)
		}
		pendingSpaces = ""
		i = j
	}

	// Trailing spaces (after the last word) are preserved.
	line.WriteString(pendingSpaces)
	if line.Len() > 0 || len(lines) == 0 {
		flush()
	}
	return lines
}

// isWordSpace reports whether r is whitespace that breakWord treats as a word
// boundary. Newlines are split by wrapLines before a paragraph reaches breakWord.
func isWordSpace(r rune) bool {
	return r == ' ' || r == '\t'
}
