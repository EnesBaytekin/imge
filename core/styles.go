// Package core contains platform-agnostic game engine logic.
// This file defines the named style (theme) system backed by styles.imge.
package core

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"

	corejson "github.com/EnesBaytekin/imge/core/json"
)

// styleSheet is the parsed contents of a styles.imge file: component kind ->
// style name -> raw style body. The body is kept as raw JSON so it can be
// re-marshaled into any component of that kind without the engine knowing its
// shape.
type styleSheet map[string]map[string]json.RawMessage

// registeredStyles is the active style sheet, loaded once at startup from the
// project's styles.imge (if present). CreateComponent consults it when a
// component declares a `style` arg.
var registeredStyles styleSheet

// LoadStyles parses a styles.imge file's bytes and installs them as the active
// style sheet. Styles are keyed by component kind (e.g. "@Button", "@Panel", or a
// custom component's kind), then by style name:
//
//	{
//	  "@Button": { "primary": { "color": "#2e7d32" } },
//	  "@Panel":  { "window":  { "color": "#14141e", "outline_color": "#3b3b4d" } }
//	}
//
// A later LoadStyles replaces the previous sheet.
func LoadStyles(data []byte) error {
	var sheet styleSheet
	if err := json.Unmarshal(corejson.StripComments(data), &sheet); err != nil {
		return fmt.Errorf("failed to parse styles: %w", err)
	}
	registeredStyles = sheet
	return nil
}

// LoadStylesFromFile reads and installs a styles.imge file from disk.
func LoadStylesFromFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return LoadStyles(data)
}

// LoadStylesFromFS reads and installs a styles.imge file from the given
// filesystem (used by web builds where game data is embedded rather than on disk).
func LoadStylesFromFS(fsys fs.FS, path string) error {
	data, err := fs.ReadFile(fsys, path)
	if err != nil {
		return err
	}
	return LoadStyles(data)
}

// resolveStyle returns the raw style body for (kind, name), or nil if none exists.
func resolveStyle(kind, name string) json.RawMessage {
	if registeredStyles == nil {
		return nil
	}
	return registeredStyles[kind][name]
}
