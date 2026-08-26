package core

import (
	"encoding/json"
	"fmt"
	"strings"
)

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

// UnmarshalJSON accepts a WrapMode written as a string name ("word", "char", or
// "clip"; case-insensitive) so config files stay readable instead of forcing magic
// numbers.
func (w *WrapMode) UnmarshalJSON(data []byte) error {
	var str string
	if err := json.Unmarshal(data, &str); err != nil {
		return fmt.Errorf("invalid wrap mode: expected \"word\", \"char\", or \"clip\"")
	}
	switch strings.ToLower(str) {
	case "word":
		*w = WrapWord
	case "char":
		*w = WrapChar
	case "clip":
		*w = WrapClip
	default:
		return fmt.Errorf("invalid wrap mode %q: expected \"word\", \"char\", or \"clip\"", str)
	}
	return nil
}
