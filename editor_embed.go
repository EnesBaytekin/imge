package imge

import "embed"

// EditorTemplate embeds the IMGE editor project that `imge editor <path>` builds and
// launches. Like the other templates it lives under testdata/ so the Go tool ignores
// its components/ package (those files reference built-in components that only exist
// once merged into a game's single components package at build time).
//
// The embed lists only the files the editor project actually needs (game.imge, scenes/,
// components/) rather than the whole testdata/editor tree, so that build output like
// testdata/editor/imge_build/ (created by `imge build` run in the editor dir for
// verification) is never pulled into the CLI binary. go:embed ignores .gitignore and
// only excludes dot/underscore-prefixed names, so an explicit list is the only way to
// keep the build dir out.
//
//go:embed testdata/editor/game.imge testdata/editor/scenes testdata/editor/components
var EditorTemplate embed.FS

// ExtractEditor writes the embedded editor project into dst, preserving its layout
// (game.imge, scenes/, components/).
func ExtractEditor(dst string) error {
	return extractTemplate(EditorTemplate, "testdata/editor", dst)
}
