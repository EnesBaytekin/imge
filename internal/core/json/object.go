package json

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// ObjectConfig represents an object template (.obj file).
type ObjectConfig struct {
	Name       string                    `json:"name"`
	Components []ComponentInstanceConfig `json:"components"`
	DefaultTags []string                 `json:"tags,omitempty"`
}

// LoadObjectConfig loads an object configuration from a JSON file.
func LoadObjectConfig(path string) (*ObjectConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var config ObjectConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

// SaveObjectConfig saves an object configuration to a JSON file.
func SaveObjectConfig(config *ObjectConfig, path string) error {
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}