// Package json provides JSON serialization and deserialization for game project files.
package json

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// GameConfig represents the main game configuration (game.json).
type GameConfig struct {
	Name    string       `json:"name"`
	Version string       `json:"version"`
	Window  WindowConfig `json:"window"`
	Game    GameSettings `json:"game"`
}

// WindowConfig represents window configuration.
type WindowConfig struct {
	Title  string `json:"title"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

// GameSettings represents game runtime settings.
type GameSettings struct {
	TargetFPS   int    `json:"target_fps"`
	InitialScene string `json:"initial_scene"`
}

// LoadGameConfig loads a game configuration from a JSON file.
func LoadGameConfig(path string) (*GameConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var config GameConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

// SaveGameConfig saves a game configuration to a JSON file.
func SaveGameConfig(config *GameConfig, path string) error {
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}