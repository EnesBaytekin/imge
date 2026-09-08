// This file is a plain helper file for the file-browser window: it declares no
// component struct, only the free functions and the small fileEntry type the browser
// and its preview share. The build tool copies it into the generated `components`
// package verbatim, and its codegen passes over it (it contributes no component kind)
// — see build/registry.go.
package components

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// fileEntry is one file or directory in the target project. `rel` is the
// project-relative path (the same form a sprite `texture` or a SceneObject `file`
// uses); `parent` is the rel path of its containing directory ("" for top-level).
type fileEntry struct {
	rel    string
	name   string
	isDir  bool
	depth  int    // indentation level (0 = project root)
	parent string // rel path of the parent directory
}

// listProjectFiles walks the project directory into a depth-first, dirs-first list.
// Build output (imge_build) and dot-directories/files are skipped, matching
// listScenes. Entries are returned sorted: directories before files, then by name.
func listProjectFiles(projectDir string) []fileEntry {
	if projectDir == "" {
		return nil
	}
	var out []fileEntry
	var walk func(rel string, depth int)
	walk = func(rel string, depth int) {
		base := projectDir
		if rel != "" {
			base = filepath.Join(projectDir, rel)
		}
		entries, err := os.ReadDir(base)
		if err != nil {
			return
		}
		sort.Slice(entries, func(i, j int) bool {
			a, b := entries[i], entries[j]
			if a.IsDir() != b.IsDir() {
				return a.IsDir()
			}
			return strings.ToLower(a.Name()) < strings.ToLower(b.Name())
		})
		for _, e := range entries {
			name := e.Name()
			if name == "imge_build" || strings.HasPrefix(name, ".") {
				continue
			}
			childRel := name
			if rel != "" {
				childRel = filepath.Join(rel, name)
			}
			out = append(out, fileEntry{rel: childRel, name: name, isDir: e.IsDir(), depth: depth, parent: rel})
			if e.IsDir() {
				walk(childRel, depth+1)
			}
		}
	}
	walk("", 0)
	return out
}

// fileTypeOf classifies a filename by extension into one of the preview categories:
// "image", "obj", "scene", or "" (no preview).
func fileTypeOf(name string) string {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".png", ".jpg", ".jpeg", ".gif":
		return "image"
	case ".obj":
		return "obj"
	case ".scene":
		return "scene"
	}
	return ""
}
