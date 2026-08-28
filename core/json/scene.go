package json

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/EnesBaytekin/imge/core/math"
)

// SceneConfig represents a scene definition (.scene file).
type SceneConfig struct {
	Name            string        `json:"name"`
	BackgroundColor string        `json:"background_color,omitempty"`
	Camera          *CameraConfig `json:"camera,omitempty"`
	Objects         []SceneObject `json:"objects"`
}

// CameraConfig represents the initial camera settings in a scene file. X/Y are
// the view's top-left corner in world coordinates; zoom is the scale factor (1 = 1:1).
type CameraConfig struct {
	X         float64 `json:"x"`
	Y         float64 `json:"y"`
	Zoom      float64 `json:"zoom"`
	Smoothing float64 `json:"smoothing"`
	LockX     bool    `json:"lock_x"`
	LockY     bool    `json:"lock_y"`
}

// SceneObject represents an object in a scene, either inline or referenced from a file.
type SceneObject struct {
	// File reference (optional)
	File string `json:"file,omitempty"`

	// Inline object definition (optional, used if File is empty)
	Name       string                    `json:"name,omitempty"`
	Transform  *TransformConfig          `json:"transform,omitempty"`
	Depth      float64                   `json:"depth,omitempty"`
	Layer      int                       `json:"layer,omitempty"`
	UI         bool                      `json:"ui,omitempty"`
	Draggable  bool                      `json:"draggable,omitempty"`
	Tags       []string                  `json:"tags,omitempty"`
	Components []ComponentInstanceConfig `json:"components,omitempty"`
}

// TransformConfig represents a 2D transform (position, rotation, scale).
type TransformConfig struct {
	Position math.Vector2 `json:"position"`
	Rotation float64      `json:"rotation,omitempty"`
	Scale    math.Vector2 `json:"scale,omitempty"`
}

// ParseSceneConfig parses a scene configuration from JSON bytes.
func ParseSceneConfig(data []byte) (*SceneConfig, error) {
	var config SceneConfig
	if err := json.Unmarshal(StripComments(data), &config); err != nil {
		return nil, err
	}
	return &config, nil
}

// LoadSceneConfig loads a scene configuration from a JSON file.
func LoadSceneConfig(path string) (*SceneConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return ParseSceneConfig(data)
}

// SaveSceneConfig saves a scene configuration to a JSON file.
func SaveSceneConfig(config *SceneConfig, path string) error {
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}
