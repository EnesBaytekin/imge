// Package assetfs opens game assets from either an embedded filesystem (web
// builds) or the OS filesystem (desktop builds). It has no Ebitengine
// dependency, so its resolution logic can be unit-tested headlessly.
package assetfs

import (
	"io"
	"io/fs"
	"os"
)

// Open opens a game asset by its project-root-relative path. When fsys is
// non-nil (web builds) the asset is read from the embedded filesystem; otherwise
// it is read from the OS filesystem (desktop builds). The path is taken exactly
// as given — there is no assets/ prefix fallback: `player.png` means
// <project root>/player.png, and `assets/player.png` means the assets directory.
func Open(fsys fs.FS, p string) (io.ReadCloser, error) {
	if fsys != nil {
		return fsys.Open(p)
	}
	return os.Open(p)
}
