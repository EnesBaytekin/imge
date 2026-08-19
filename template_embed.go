package imge

import (
	"embed"
	"io/fs"
	"os"
	"path/filepath"
)

// SampleTemplate embeds the sample platformer demo that `imge init sample`
// writes into an empty directory. It's a complete, runnable game that exercises
// the built-in component library plus a handful of custom components, doubling
// as a tutorial for how objects, scenes, and components fit together.
//
//go:embed testdata/template
var SampleTemplate embed.FS

// BlankTemplate embeds the minimal starter project that `imge init` (no
// argument) writes into an empty directory: a game.imge, one empty scene, and a
// README, so a developer can start defining objects without writing any JSON
// scaffolding from scratch.
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

// ExtractBlankTemplate writes the embedded blank project into dst and creates
// the standard empty directories a developer will drop their own files into.
// scenes/ already exists via main.scene.
func ExtractBlankTemplate(dst string) error {
	if err := extractTemplate(BlankTemplate, "testdata/blank", dst); err != nil {
		return err
	}
	for _, dir := range []string{"components", "objects", "assets"} {
		if err := os.MkdirAll(filepath.Join(dst, dir), 0755); err != nil {
			return err
		}
	}
	return nil
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
