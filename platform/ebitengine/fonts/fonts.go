// Package fonts loads TTF/OTF font files and measures text at a requested size,
// with no graphics dependency. The ebitengine renderer wraps this to rasterize and
// draw text; keeping the parsing and measurement here lets it be unit-tested
// without a display (importing Ebitengine runs a GLFW init that needs one).
package fonts

import (
	_ "embed"
	"fmt"
	"io"
	"io/fs"
	"log"

	"github.com/EnesBaytekin/imge/platform/ebitengine/assetfs"
	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
)

// DefaultSize is the size (in logical units) used when a text call asks for
// size <= 0, so "no settings" renders the built-in font at its native, crispest
// pixel grid. The built-in font is "imge-font", a 4-wide × 6-tall pixel font
// drawn on a 600 units-per-em / 100-unit pixel grid, so its design size is
// exactly 6 px — at size 6, one font pixel is exactly one game pixel. Only
// integer multiples of 6 (12, 18, …) land the glyph outlines on the device pixel
// grid; any other size rasterizes antialiased and goes soft.
const DefaultSize = 6.0

// imgeFontTTF is the built-in "imge-font" pixel font, compiled into the binary
// so a game built with the imge CLI needs no font file alongside it.
//
//go:embed imge-font.ttf
var imgeFontTTF []byte

// Library loads and caches fonts by ID and measures text at a given size. Fonts
// are loaded the same way textures are: "" (or "imge-font") is the built-in
// default, and any other value is a project-root-relative path to a .ttf/.otf
// file resolved through the asset filesystem.
type Library struct {
	// sources caches parsed (unsized) font sources by ID. "" (or "imge-font") is
	// the built-in default.
	sources map[string]*opentype.Font

	// faces caches size-specific faces keyed by "fontID\x00size". A face rasterizes
	// its glyphs at a fixed size, so each size needs its own face.
	faces map[string]font.Face

	// missing records font IDs we already warned about, so a bad path logs once.
	missing map[string]bool
}

// NewLibrary returns an empty font library.
func NewLibrary() *Library {
	return &Library{
		sources: make(map[string]*opentype.Font),
		faces:   make(map[string]font.Face),
		missing: make(map[string]bool),
	}
}

// Face returns a cached, size-specific face for fontID and size (size <= 0 selects
// DefaultSize), or nil when the font cannot be loaded.
func (l *Library) Face(fsys fs.FS, fontID string, size float64) font.Face {
	if size <= 0 {
		size = DefaultSize
	}
	key := fmt.Sprintf("%s\x00%g", fontID, size)
	if face, ok := l.faces[key]; ok {
		return face
	}

	src := l.Source(fsys, fontID)
	if src == nil {
		return nil
	}
	face, err := opentype.NewFace(src, &opentype.FaceOptions{
		Size:    size,
		DPI:     72, // 72 DPI makes Size == pixels
		Hinting: font.HintingFull,
	})
	if err != nil {
		log.Printf("fonts: failed to create face for font %q at size %g: %v", fontID, size, err)
		return nil
	}

	l.faces[key] = face
	return face
}

// Measure returns the advance width and line height (ascent+descent) of text at
// the given size, in logical units — the same box the renderer places the text
// into. Returns (0, 0) when the font can't load or the text is empty.
func (l *Library) Measure(fsys fs.FS, fontID string, size float64, text string) (float64, float64) {
	if text == "" {
		return 0, 0
	}
	face := l.Face(fsys, fontID, size)
	if face == nil {
		return 0, 0
	}
	m := face.Metrics()
	return float64(font.MeasureString(face, text)) / 64, float64(m.Ascent+m.Descent) / 64
}

// Source returns a parsed (unsized) font source for fontID, or nil when it can't
// be loaded. fontID "" (or "imge-font") selects the built-in default font.
func (l *Library) Source(fsys fs.FS, fontID string) *opentype.Font {
	if fontID == "" || fontID == "imge-font" {
		return l.defaultSource()
	}
	if src, ok := l.sources[fontID]; ok {
		return src
	}

	f, err := assetfs.Open(fsys, fontID)
	if err != nil {
		l.warnMissing(fontID, err)
		return nil
	}
	defer f.Close()

	data, err := io.ReadAll(f)
	if err != nil {
		l.warnMissing(fontID, err)
		return nil
	}
	src, err := opentype.Parse(data)
	if err != nil {
		l.warnMissing(fontID, err)
		return nil
	}
	l.sources[fontID] = src
	return src
}

// defaultSource returns the built-in default font, parsed once and cached. The
// font is compiled into the binary (see imgeFontTTF), so text with no font
// argument works out of the box with no file alongside the game.
func (l *Library) defaultSource() *opentype.Font {
	if src, ok := l.sources[""]; ok {
		return src
	}
	src, err := opentype.Parse(imgeFontTTF)
	if err != nil {
		// Should be impossible — the font is embedded at compile time.
		log.Printf("fonts: failed to parse built-in default font: %v", err)
		return nil
	}
	l.sources[""] = src
	return src
}

func (l *Library) warnMissing(fontID string, err error) {
	if !l.missing[fontID] {
		log.Printf("fonts: font not found: %q: %v", fontID, err)
		l.missing[fontID] = true
	}
}
