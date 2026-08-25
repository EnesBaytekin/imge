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
func (l *Library) Wrap(fsys fs.FS, fontID string, size float64, text string, maxWidth float64, mode core.WrapMode) WrappedText {
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

	lines := wrapLines(text, maxWidth, mode, measure)
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
func wrapLines(text string, maxWidth float64, mode core.WrapMode, measure func(string) float64) []string {
	paragraphs := strings.Split(text, "\n")
	if maxWidth <= 0 {
		return paragraphs
	}

	var lines []string
	for _, para := range paragraphs {
		switch mode {
		case core.WrapClip:
			lines = append(lines, clipLine(para, maxWidth, measure))
		case core.WrapChar:
			lines = append(lines, breakChar(para, maxWidth, measure)...)
		default: // core.WrapWord
			lines = append(lines, breakWord(para, maxWidth, measure)...)
		}
	}
	return lines
}

// clipLine truncates text to the longest prefix (on a rune boundary) that fits
// maxWidth, dropping the rest — no wrapping.
func clipLine(text string, maxWidth float64, measure func(string) float64) string {
	if measure(text) <= maxWidth {
		return text
	}
	cut := 0
	for end := 0; end < len(text); {
		_, size := utf8.DecodeRuneInString(text[end:])
		end += size
		if measure(text[:end]) > maxWidth {
			break
		}
		cut = end
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
	for start := 0; start < len(text); {
		// Always take at least one rune, so an oversized rune doesn't loop forever.
		_, size := utf8.DecodeRuneInString(text[start:])
		end := start + size
		for end < len(text) {
			_, sz := utf8.DecodeRuneInString(text[end:])
			next := end + sz
			if measure(text[start:next]) > maxWidth {
				break
			}
			end = next
		}
		lines = append(lines, text[start:end])
		start = end
	}
	return lines
}

// breakWord breaks text into lines on whitespace, keeping whole words together
// (single spaces between them). A lone word wider than maxWidth is placed on its own
// line and overflows.
func breakWord(text string, maxWidth float64, measure func(string) float64) []string {
	words := strings.Fields(text)
	if len(words) == 0 {
		return []string{""}
	}
	var lines []string
	line := words[0]
	for _, word := range words[1:] {
		if candidate := line + " " + word; measure(candidate) <= maxWidth {
			line = candidate
			continue
		}
		lines = append(lines, line)
		line = word
	}
	return append(lines, line)
}
