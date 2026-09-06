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
