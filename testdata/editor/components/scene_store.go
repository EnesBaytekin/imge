// This file is a plain helper file for the scene-management panel: it declares no
// component struct, only the free functions and the small sceneEntry type the scene
// list and its modals share. The build tool copies it into the generated `components`
// package verbatim, and its codegen passes over it (it contributes no component kind)
// — see build/registry.go.
package components

import (
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	imgejson "github.com/EnesBaytekin/imge/core/json"
)

// sceneEntry is one .scene file in the target project. It carries two names:
//
//   - file: the filename basename (e.g. "main") — the stable identity used for
//     selecting, creating, and deleting.
//   - name: the display name, the JSON "name" field (falling back to file). This is
//     the scene's runtime name (core.Scene.Name), which game.imge's initial_scene
//     matches against.
type sceneEntry struct {
	file string
	name string
	path string // full path to the .scene file
}

// listScenes scans the project directory for .scene files, skipping build output
// (imge_build) and hidden directories. It returns the entries sorted by display name.
func listScenes(projectDir string) []sceneEntry {
	if projectDir == "" {
		return nil
	}
	var entries []sceneEntry
	_ = filepath.WalkDir(projectDir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			name := d.Name()
			if name == "imge_build" || strings.HasPrefix(name, ".") {
				return fs.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(d.Name(), ".scene") {
			return nil
		}
		file := strings.TrimSuffix(d.Name(), ".scene")
		name := file
		if cfg, err := imgejson.LoadSceneConfig(p); err == nil && cfg.Name != "" {
			name = cfg.Name
		}
		entries = append(entries, sceneEntry{file: file, name: name, path: p})
		return nil
	})
	sort.Slice(entries, func(i, j int) bool { return entries[i].name < entries[j].name })
	return entries
}

// scenePath returns the path where a new scene named file lives (scenes/<file>.scene).
func scenePath(projectDir, file string) string {
	return filepath.Join(projectDir, "scenes", file+".scene")
}

// createSceneFile writes a fresh, empty scene named name to scenes/<name>.scene. It
// mirrors the CLI's scene template (cmd/imge/new.go): black background, a default
// identity camera, no objects.
func createSceneFile(projectDir, name string) error {
	cfg := &imgejson.SceneConfig{
		Name:            name,
		BackgroundColor: "#000000",
		Camera: &imgejson.CameraConfig{
			X: 0, Y: 0, Zoom: 1, Smoothing: 0, LockX: false, LockY: false,
		},
		Objects: []imgejson.SceneObject{},
	}
	return imgejson.SaveSceneConfig(cfg, scenePath(projectDir, name))
}

// deleteSceneFile removes a scene's .scene file at path.
func deleteSceneFile(path string) error {
	return os.Remove(path)
}

// ============================================================================
// Scene-deletion undo
// ============================================================================

// deletedScene captures everything needed to undo a scene deletion: the file's
// bytes (to rewrite it), its identity, and the surrounding state the delete altered
// (initial_scene, active scene). Unlike object edits — which undo through the
// in-scene `history` — a scene deletion is a file-level action, and the deletion
// itself clears history (it unloads/switches the active scene), so it needs its own
// single-slot undo the toolbar falls back to.
type deletedScene struct {
	projectDir string
	file       string // filename basename (the stable identity SetScene keys on)
	path       string // full .scene path
	name       string // JSON display name (for restoring initial_scene)
	wasActive  bool   // the deleted scene was loaded in the viewport
	wasInitial bool   // the deleted scene was game.imge's initial_scene
	content    []byte // the .scene file bytes as they were before deletion
}

// lastDeletedScene is the pending undo for the most recent scene deletion. It is
// package-level (like history) so the toolbar's undo shortcut can reach it after
// the delete has cleared the in-scene edit history. One slot is enough: a new
// deletion overwrites it, and restoring it consumes it.
var lastDeletedScene *deletedScene

// captureDeletedScene reads the scene file's bytes and stashes them as the pending
// undo for a deletion. It returns false when the file can't be read (nothing to
// restore), leaving lastDeletedScene unchanged.
func captureDeletedScene(entry sceneEntry, projectDir string, wasActive, wasInitial bool) bool {
	content, err := os.ReadFile(entry.path)
	if err != nil {
		return false
	}
	lastDeletedScene = &deletedScene{
		projectDir: projectDir,
		file:       entry.file,
		path:       entry.path,
		name:       entry.name,
		wasActive:  wasActive,
		wasInitial: wasInitial,
		content:    content,
	}
	return true
}

// restoreDeletedSceneFile rewrites a deleted scene's .scene file from the captured
// bytes, recreating its directory first in case the last file in it was removed.
func restoreDeletedSceneFile(d deletedScene) error {
	if err := os.MkdirAll(filepath.Dir(d.path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(d.path, d.content, 0o644)
}

// initialSceneName returns the target project's game.imge initial_scene, or "" when
// it can't be read.
func initialSceneName(projectDir string) string {
	cfg, err := imgejson.LoadGameConfig(filepath.Join(projectDir, "game.imge"))
	if err != nil {
		return ""
	}
	return cfg.Game.InitialScene
}

// setInitialScene writes game.imge's initial_scene to name.
func setInitialScene(projectDir, name string) error {
	path := filepath.Join(projectDir, "game.imge")
	cfg, err := imgejson.LoadGameConfig(path)
	if err != nil {
		return err
	}
	cfg.Game.InitialScene = name
	return imgejson.SaveGameConfig(cfg, path)
}
