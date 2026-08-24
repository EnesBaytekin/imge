package imge

import (
	"embed"
	"io/fs"
	"os"
	"path/filepath"

	corejson "github.com/EnesBaytekin/imge/core/json"
)

// SampleTemplate embeds the sample platformer demo that `imge init sample`
// writes into an empty directory. It's a complete, runnable game that exercises
// the built-in component library plus a handful of custom components, doubling
// as a tutorial for how objects, scenes, and components fit together.
//
//go:embed testdata/template
var SampleTemplate embed.FS

// BlankTemplate embeds the starter scene that `imge init` (no argument) writes
// into an empty directory. The game.imge itself is not embedded — it's generated
// from the config structs (core/json.DefaultGameConfig) by ExtractBlankTemplate,
// so the structs stay the single source of truth for its fields.
//
//go:embed testdata/blank
var BlankTemplate embed.FS

// Both templates live under testdata/ so the Go tool ignores their components/
// packages — those files reference built-in components that only exist once
// merged into a game's single components package at build time.

// ExtractSampleTemplate writes the embedded sample project into dst, preserving
// its layout (components/, scenes/, objects/, assets/, game.imge, README.md).
func ExtractSampleTemplate(dst string) error {
	return extractTemplate(SampleTemplate, "testdata/template", dst)
}

// ExtractBlankTemplate writes the blank project into dst: a game.imge with every
// field set to its default (generated from core/json.DefaultGameConfig) plus
// scenes/main.scene (kept identical to what `imge new scene` generates — see
// sceneTemplate in cmd/imge). It creates no empty directories and no README.
func ExtractBlankTemplate(dst string) error {
	if err := extractTemplate(BlankTemplate, "testdata/blank", dst); err != nil {
		return err
	}
	cfg := corejson.DefaultGameConfig()
	cfg.FormatVersion = CurrentFormatVersion
	return corejson.SaveGameConfig(cfg, filepath.Join(dst, "game.imge"))
}

// extractTemplate writes every file under root in fsys into dst, preserving the
// relative layout.
func extractTemplate(fsys fs.FS, root, dst string) error {
	return fs.WalkDir(fsys, root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(root, p)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return err
		}
		data, err := fs.ReadFile(fsys, p)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0644)
	})
}
