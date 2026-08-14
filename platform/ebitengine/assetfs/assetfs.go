// Package assetfs resolves and opens game assets from either an embedded
// filesystem (web builds) or the OS filesystem (desktop builds). It has no
// Ebitengine dependency, so its resolution logic can be unit-tested headlessly.
package assetfs

import (
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
)

// Resolve resolves an OS path that may be relative to the working directory or
// to the assets/ directory. It returns the first path that exists, or the
// original path if neither does (the caller reports the error).
func Resolve(p string) string {
	if p == "" {
		return p
	}
	if _, err := os.Stat(p); err == nil {
		return p
	}
	alt := filepath.Join("assets", p)
	if _, err := os.Stat(alt); err == nil {
		return alt
	}
	return p
}

// ResolveFS is Resolve for an fs.FS. Web builds embed their assets, so paths use
// "/" separators (as required by fs.FS) instead of the OS filesystem's.
func ResolveFS(fsys fs.FS, p string) string {
	if p == "" {
		return p
	}
	if _, err := fs.Stat(fsys, p); err == nil {
		return p
	}
	alt := path.Join("assets", p)
	if _, err := fs.Stat(fsys, alt); err == nil {
		return alt
	}
	return p
}

// Open opens an asset, resolving the "assets/" prefix as a fallback. When fsys
// is non-nil (web builds) the asset is read from the embedded filesystem;
// otherwise it is read from the OS filesystem (desktop builds).
func Open(fsys fs.FS, p string) (io.ReadCloser, error) {
	if fsys != nil {
		return fsys.Open(ResolveFS(fsys, p))
	}
	return os.Open(Resolve(p))
}
