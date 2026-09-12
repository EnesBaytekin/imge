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
	Resizable    bool   `json:"resizable"`
	PixelPerUnit int    `json:"pixel_per_unit"`
	Scale        int    `json:"scale"`
	SmoothShapes bool   `json:"smooth_shapes"`
	// SmoothRotation opts texture rotation into framebuffer-resolution (smooth,
	// sub-unit) rasterization. The default (true) rotates at pixel_per_unit
	// resolution — the historical behavior. When false, rotation is quantized to
	// logical pixels ("chunky"), so a rotated sprite stays pixel-perfect like the
	// rest of the chunky shape pipeline.
	SmoothRotation bool `json:"smooth_rotation"`
	// Vsync controls whether the platform waits for the display's vertical blank
	// before presenting. Defaults to true (no tearing); set false for the lowest
	// input latency.
	Vsync bool `json:"vsync"`
}

// UnmarshalJSON defaults SmoothRotation and Vsync to true when the fields are absent,
// so a game.imge written before the flags existed keeps the historical behavior
// (smooth rotation, vsync on) rather than silently flipping to chunky/tearing. When
// present, the written value wins.
func (w *WindowConfig) UnmarshalJSON(data []byte) error {
	type windowAlias WindowConfig
	aux := struct {
		*windowAlias
		SmoothRotation *bool `json:"smooth_rotation"`
		Vsync          *bool `json:"vsync"`
	}{windowAlias: (*windowAlias)(w)}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	if aux.SmoothRotation != nil {
		w.SmoothRotation = *aux.SmoothRotation
	} else {
		w.SmoothRotation = true
	}
	if aux.Vsync != nil {
		w.Vsync = *aux.Vsync
	} else {
		w.Vsync = true
	}
	return nil
}

// GameSettings represents game runtime settings.
//
// TargetFPS caps the update rate. The default (when the field is absent) is 60 —
// a fixed step that keeps CPU use flat on high-refresh panels and gives
// deterministic physics. 0 opts back into "sync updates with the display refresh
// rate" (lowest input latency, but the loop runs as fast as the panel refreshes).
type GameSettings struct {
	TargetFPS    int    `json:"target_fps"`
	InitialScene string `json:"initial_scene"`

	// targetFPSSet records whether the "game" object was present in the JSON. It lets
	// ApplyDefaults distinguish "target_fps omitted" (→ default 60) from "target_fps
	// explicitly 0" (→ sync with refresh). It is unexported, so it never serializes.
	targetFPSSet bool
}

// UnmarshalJSON defaults TargetFPS to 60 when the field is absent, so a game.imge
// that omits target_fps runs at a fixed 60 rather than syncing to the display
// refresh. An explicit 0 still means "sync with the refresh rate". Mirrors
// WindowConfig.UnmarshalJSON's pointer-default pattern.
func (g *GameSettings) UnmarshalJSON(data []byte) error {
	type gameSettingsAlias GameSettings
	aux := struct {
		*gameSettingsAlias
		TargetFPS *int `json:"target_fps"`
	}{gameSettingsAlias: (*gameSettingsAlias)(g)}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	g.targetFPSSet = true
	if aux.TargetFPS != nil {
		g.TargetFPS = *aux.TargetFPS
	} else {
		g.TargetFPS = 60
	}
	return nil
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
			// SmoothRotation defaults to true: rotation is smooth (sub-unit) at any
			// pixel_per_unit, matching the historical texture-rotation behavior.
			SmoothRotation: true,
			// Vsync defaults to true (no tearing); set false for lowest input latency.
			Vsync: true,
		},
		Game: GameSettings{
			// TargetFPS defaults to 60: a fixed step keeps CPU flat on high-refresh
			// panels and gives deterministic physics (0 = sync with the refresh rate).
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
	// A game.imge with no "game" object at all never invokes GameSettings.UnmarshalJSON,
	// so target_fps is still 0 here. Default it to 60 in that case; an explicit
	// target_fps: 0 (the "sync with refresh" opt-out) is preserved because
	// UnmarshalJSON records that the object was present.
	if !c.Game.targetFPSSet {
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
