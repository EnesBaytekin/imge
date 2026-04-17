package build

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ProjectAnalysis holds information about a game project
type ProjectAnalysis struct {
	ProjectDir    string
	GameConfig    GameConfig
	ComponentFiles []string // Paths to component .go files
	AssetFiles    []string // Paths to asset files
	SceneFiles    []string // Paths to .scene files
	ObjectFiles   []string // Paths to .obj files
}

// GameConfig represents the game.json configuration
type GameConfig struct {
	Name    string         `json:"name"`
	Version string         `json:"version"`
	Window  WindowConfig  `json:"window"`
	Game    GameSettings  `json:"game"`
}

// WindowConfig represents window settings
type WindowConfig struct {
	Title  string `json:"title"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

// GameSettings represents game runtime settings
type GameSettings struct {
	TargetFPS    int    `json:"target_fps"`
	InitialScene string `json:"initial_scene"`
}

// AnalyzeProject analyzes a game project directory and returns its structure
func AnalyzeProject(projectDir string) (*ProjectAnalysis, error) {
	analysis := &ProjectAnalysis{
		ProjectDir: projectDir,
	}

	// Check if project directory exists
	if _, err := os.Stat(projectDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("project directory does not exist: %s", projectDir)
	}

	// Load game.json
	gameJSONPath := filepath.Join(projectDir, "game.json")
	if err := analysis.loadGameConfig(gameJSONPath); err != nil {
		return nil, fmt.Errorf("failed to load game.json: %v", err)
	}

	// Find component files
	componentsDir := filepath.Join(projectDir, "components")
	if err := analysis.findComponentFiles(componentsDir); err != nil {
		return nil, fmt.Errorf("failed to find component files: %v", err)
	}

	// Find asset files
	assetsDir := filepath.Join(projectDir, "assets")
	if err := analysis.findAssetFiles(assetsDir); err != nil {
		// Assets directory is optional
		fmt.Printf("Note: assets directory not found or empty: %s\n", assetsDir)
	}

	// Find scene files
	scenesDir := filepath.Join(projectDir, "scenes")
	if err := analysis.findSceneFiles(scenesDir); err != nil {
		// Scenes directory is optional
		fmt.Printf("Note: scenes directory not found or empty: %s\n", scenesDir)
	}

	// Find object files
	objectsDir := filepath.Join(projectDir, "objects")
	if err := analysis.findObjectFiles(objectsDir); err != nil {
		// Objects directory is optional
		fmt.Printf("Note: objects directory not found or empty: %s\n", objectsDir)
	}

	return analysis, nil
}

func (a *ProjectAnalysis) loadGameConfig(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	// Parse JSON using encoding/json
	var config GameConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("failed to parse game.json: %v", err)
	}

	// Set defaults if fields are empty
	if config.Name == "" {
		config.Name = "My Game"
	}
	if config.Version == "" {
		config.Version = "1.0.0"
	}
	if config.Window.Title == "" {
		config.Window.Title = "My IMGE Game"
	}
	if config.Window.Width == 0 {
		config.Window.Width = 800
	}
	if config.Window.Height == 0 {
		config.Window.Height = 600
	}
	if config.Game.TargetFPS == 0 {
		config.Game.TargetFPS = 60
	}
	if config.Game.InitialScene == "" {
		config.Game.InitialScene = "main"
	}

	a.GameConfig = config
	return nil
}

func (a *ProjectAnalysis) findComponentFiles(dir string) error {
	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && strings.HasSuffix(info.Name(), ".go") {
			relPath, err := filepath.Rel(a.ProjectDir, path)
			if err != nil {
				return err
			}
			a.ComponentFiles = append(a.ComponentFiles, relPath)
		}
		return nil
	})
}

func (a *ProjectAnalysis) findAssetFiles(dir string) error {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return nil
	}

	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() {
			relPath, err := filepath.Rel(a.ProjectDir, path)
			if err != nil {
				return err
			}
			a.AssetFiles = append(a.AssetFiles, relPath)
		}
		return nil
	})
}

func (a *ProjectAnalysis) findSceneFiles(dir string) error {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return nil
	}

	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && strings.HasSuffix(info.Name(), ".scene") {
			relPath, err := filepath.Rel(a.ProjectDir, path)
			if err != nil {
				return err
			}
			a.SceneFiles = append(a.SceneFiles, relPath)
		}
		return nil
	})
}

func (a *ProjectAnalysis) findObjectFiles(dir string) error {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return nil
	}

	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && strings.HasSuffix(info.Name(), ".obj") {
			relPath, err := filepath.Rel(a.ProjectDir, path)
			if err != nil {
				return err
			}
			a.ObjectFiles = append(a.ObjectFiles, relPath)
		}
		return nil
	})
}