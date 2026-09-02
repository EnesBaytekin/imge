package imge

import "embed"

// EditorTemplate embeds the IMGE editor project that `imge editor <path>` builds and
// launches. Like the other templates it lives under testdata/ so the Go tool ignores
// its components/ package (those files reference built-in components that only exist
// once merged into a game's single components package at build time).
//
//go:embed testdata/editor
var EditorTemplate embed.FS

// ExtractEditor writes the embedded editor project into dst, preserving its layout
// (game.imge, scenes/, components/).
func ExtractEditor(dst string) error {
	return extractTemplate(EditorTemplate, "testdata/editor", dst)
}
