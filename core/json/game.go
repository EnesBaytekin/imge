package json

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// GameConfig represents the game.imge configuration file — the single source of
// truth for window and game settings.
type GameConfig struct {
	Name          string       `json:"name"`
	FormatVersion int          `json:"format_version"`
	Window        WindowConfig `json:"window"`
	Game          GameSettings `json:"game"`
}

// WindowConfig represents window settings.
type WindowConfig struct {
	Title        string `json:"title"`
	Width        int    `json:"width"`
	Height       int    `json:"height"`
	Fullscreen   bool   `json:"fullscreen"`
	PixelPerUnit int    `json:"pixel_per_unit"`
	SmoothShapes bool   `json:"smooth_shapes"`
}

// GameSettings represents game runtime settings.
type GameSettings struct {
	TargetFPS    int    `json:"target_fps"`
	InitialScene string `json:"initial_scene"`
}

// DefaultGameConfig returns a config with every field at its default value. The
// FormatVersion is left at 0: the project-format version is a build-tool concern
// (it depends on the engine's CurrentFormatVersion), so callers that write a
// game.imge set it themselves.
func DefaultGameConfig() *GameConfig {
	return &GameConfig{
		Name: "My Game",
		Window: WindowConfig{
			Title:        "My IMGE Game",
			Width:        640,
			Height:       360,
			PixelPerUnit: 1,
		},
		Game: GameSettings{
			TargetFPS:    60,
			InitialScene: "main",
		},
	}
}

// ApplyDefaults fills in zero-valued fields with their defaults, so a partial
// game.imge (one that omits fields it's happy to leave at their defaults) still
// parses to a fully-populated config. FormatVersion is deliberately left alone:
// its zero value is meaningful (the build tool interprets 0 as "original format").
func ApplyDefaults(c *GameConfig) {
	if c.Name == "" {
		c.Name = "My Game"
	}
	if c.Window.Title == "" {
		c.Window.Title = "My IMGE Game"
	}
	if c.Window.Width == 0 {
		c.Window.Width = 640
	}
	if c.Window.Height == 0 {
		c.Window.Height = 360
	}
	if c.Window.PixelPerUnit <= 0 {
		c.Window.PixelPerUnit = 1
	}
	if c.Game.TargetFPS == 0 {
		c.Game.TargetFPS = 60
	}
	if c.Game.InitialScene == "" {
		c.Game.InitialScene = "main"
	}
}

// ParseGameConfig parses a game configuration from JSON bytes (comments allowed).
func ParseGameConfig(data []byte) (*GameConfig, error) {
	var c GameConfig
	if err := json.Unmarshal(StripComments(data), &c); err != nil {
		return nil, err
	}
	ApplyDefaults(&c)
	return &c, nil
}

// LoadGameConfig loads a game configuration from a JSON file.
func LoadGameConfig(path string) (*GameConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return ParseGameConfig(data)
}

// SaveGameConfig saves a game configuration to a JSON file (indented, all fields
// present — the config structs carry no omitempty tags, so defaults are written
// out explicitly).
func SaveGameConfig(config *GameConfig, path string) error {
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}
