package components

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/EnesBaytekin/imge/core"
)

// findComponentSource returns the absolute path of the .go file in the target project's
// components/ directory that declares the given component kind (the struct type name),
// or "" when the kind is unknown or the file cannot be read. It reuses the same AST walk
// as scanProjectComponents, so the returned path always names the file that actually
// defines the component — not a guess from the kind string.
func findComponentSource(projectDir, kind string) string {
	if projectDir == "" || kind == "" {
		return ""
	}
	compDir := filepath.Join(projectDir, "components")
	entries, err := os.ReadDir(compDir)
	if err != nil {
		return ""
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		path := filepath.Join(compDir, entry.Name())
		src, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		for _, name := range componentStructNames(src) {
			if name == kind {
				return path
			}
		}
	}
	return ""
}

// componentSourceAbs resolves the absolute source location of a component given the
// editor scene (which reaches the viewport's current project directory). It returns the
// component's .go file when found, otherwise the project's components/ directory (a custom
// component with no matching file still points at the folder the user would create it in),
// or "" when there is no project or the component has no source to show. This is the path
// the "open" button launches.
func componentSourceAbs(scene *core.Scene, comp core.Component) string {
	if comp == nil || scene == nil {
		return ""
	}
	vp := lookupViewport(scene)
	if vp == nil {
		return ""
	}
	dir := vp.CurrentProject()
	if dir == "" {
		return ""
	}
	if p := findComponentSource(dir, comp.GetKind()); p != "" {
		return p
	}
	return filepath.Join(dir, "components")
}

// componentSourcePath resolves the source location for display and copy: the same path as
// componentSourceAbs but relativized to the project directory (e.g.
// "components/foo/bar.go"), falling back to the absolute path when it cannot be made
// relative.
func componentSourcePath(scene *core.Scene, comp core.Component) string {
	abs := componentSourceAbs(scene, comp)
	if abs == "" {
		return ""
	}
	vp := lookupViewport(scene)
	if vp == nil {
		return abs
	}
	if rel, err := filepath.Rel(vp.CurrentProject(), abs); err == nil {
		return rel
	}
	return abs
}

// openPathInEditor opens path with the operating system's default handler — for a .go
// file, the user's default text editor. It mirrors writeClipboard's per-OS one-liners:
// `open` on macOS, `rundll32 url.dll,FileProtocolHandler` on Windows, and `xdg-open` on
// Linux. It returns an error when the opener fails to launch.
func openPathInEditor(path string) error {
	switch runtime.GOOS {
	case "darwin":
		return exec.Command("open", path).Start()
	case "windows":
		return exec.Command("rundll32", "url.dll,FileProtocolHandler", path).Start()
	default:
		return exec.Command("xdg-open", path).Start()
	}
}
