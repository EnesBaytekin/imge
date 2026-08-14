package ebitengine

import (
	"os"
	"path/filepath"
)

// resolveAssetPath resolves an asset path that may be relative to the working
// directory or to the assets/ directory. It returns the first path that exists,
// or the original path if neither does (the caller reports the error).
func resolveAssetPath(p string) string {
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
