package json

import (
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
)

// ObjectConfig represents an object template (.obj file).
// Contains all object properties except transform.
type ObjectConfig struct {
	Name       string                    `json:"name"`
	Depth      float64                   `json:"depth,omitempty"`
	UI         bool                      `json:"ui,omitempty"`
	Draggable  bool                      `json:"draggable,omitempty"`
	Components []ComponentInstanceConfig `json:"components"`
	Tags       []string                  `json:"tags,omitempty"`
}

// ParseObjectConfig parses an object configuration from JSON bytes.
func ParseObjectConfig(data []byte) (*ObjectConfig, error) {
	var config ObjectConfig
	if err := json.Unmarshal(StripComments(data), &config); err != nil {
		return nil, err
	}
	return &config, nil
}

// LoadObjectConfig loads an object configuration from a JSON file.
func LoadObjectConfig(path string) (*ObjectConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return ParseObjectConfig(data)
}

// LoadObjectConfigFS loads an object configuration from the given filesystem.
// Used by web builds where game data is embedded rather than read from disk.
func LoadObjectConfigFS(fsys fs.FS, path string) (*ObjectConfig, error) {
	data, err := fs.ReadFile(fsys, path)
	if err != nil {
		return nil, err
	}
	return ParseObjectConfig(data)
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
