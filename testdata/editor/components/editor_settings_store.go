package components

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// editorSettingsFile is the fixed name of the per-project editor-settings cache. It
// lives at the target project's root, next to game.imge, and holds only state that
// concerns the editor — never the built game: viewport grid spacing, the grid/axes
// colors, the last camera pan/zoom, and the last selected object. It is a hidden
// dotfile on purpose, so a developer can add the name to their own .gitignore; the
// build tool also skips it so it is never embedded or copied into a build.
const editorSettingsFile = ".imge.editor"

// editorSettings is the on-disk shape of the editor-settings cache. All fields are
// editor-only viewport state. Colors are stored as "#RRGGBB[AA]" hex strings (the
// same convention the scene/object JSON uses), so the struct keeps them as strings
// rather than math.Color and the viewport converts on apply.
type editorSettings struct {
	FormatVersion int `json:"format_version"`

	GridStepX float64 `json:"grid_step_x,omitempty"`
	GridStepY float64 `json:"grid_step_y,omitempty"`
	GridColor string  `json:"grid_color,omitempty"`
	AxesColor string  `json:"axes_color,omitempty"`

	Camera *editorCameraSettings `json:"camera,omitempty"`

	SelectedObject string `json:"selected_object,omitempty"`
}

type editorCameraSettings struct {
	X    float64 `json:"x"`
	Y    float64 `json:"y"`
	Zoom float64 `json:"zoom"`
}

// editorSettingsPath returns the path of the editor-settings cache for a project dir.
func editorSettingsPath(dir string) string {
	return filepath.Join(dir, editorSettingsFile)
}

// readEditorSettings loads the editor-settings cache for a project dir. A missing
// file returns an error (the caller treats it as "no saved settings").
func readEditorSettings(dir string) (editorSettings, error) {
	var s editorSettings
	data, err := os.ReadFile(editorSettingsPath(dir))
	if err != nil {
		return s, err
	}
	if err := json.Unmarshal(data, &s); err != nil {
		return s, err
	}
	return s, nil
}

// writeEditorSettings writes the editor-settings cache for a project dir, creating
// the project directory if needed. It follows the same indented-JSON, 0644-permission
// convention the scene/game-config savers use.
func writeEditorSettings(dir string, s editorSettings) error {
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	return os.WriteFile(editorSettingsPath(dir), data, 0644)
}
