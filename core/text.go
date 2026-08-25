package core

// WrapMode controls how wrapped text (DrawTextWrapped) fits a line into maxWidth.
type WrapMode int

const (
	// WrapWord breaks lines on whitespace so whole words stay together. A single
	// word wider than maxWidth is still placed on its own line (and overflows) —
	// the standard word-wrap behavior for dialogue and UI text.
	WrapWord WrapMode = iota

	// WrapChar breaks lines exactly at maxWidth, splitting a word mid-way if it
	// doesn't fit — the terminal/log style of hard line wrapping.
	WrapChar

	// WrapClip does not wrap at all: the text is truncated to a single line that
	// fits within maxWidth, and anything past that is dropped. A trailing "..." is
	// appended by default (ellipsis=true) so the cut is visible; pass ellipsis=false
	// to truncate with no marker.
	WrapClip
)
