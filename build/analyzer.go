package build

import (
	"fmt"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/EnesBaytekin/imge"
	corejson "github.com/EnesBaytekin/imge/core/json"
)

// ProjectAnalysis holds information about a game project
type ProjectAnalysis struct {
	ProjectDir     string
	GameConfig     corejson.GameConfig
	ComponentFiles []string // Paths to component .go files
	AssetFiles     []string // Paths to asset files
	SceneFiles     []string // Paths to .scene files
	ObjectFiles    []string // Paths to .obj files
}

// FindGameFile returns the path to the project's marker file, `game.imge`. It
// returns an error if the file does not exist.
func FindGameFile(dir string) (string, error) {
	path := filepath.Join(dir, "game.imge")
	if _, err := os.Stat(path); err != nil {
		return "", fmt.Errorf("no game.imge found in %s (it marks the project root)", dir)
	}
	return path, nil
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

	// Load the project marker (game.imge).
	gameFilePath, err := FindGameFile(projectDir)
	if err != nil {
		return nil, err
	}
	config, err := corejson.LoadGameConfig(gameFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to load %s: %v", filepath.Base(gameFilePath), err)
	}
	if err := validateFormatVersion(config); err != nil {
		return nil, fmt.Errorf("invalid %s: %v", filepath.Base(gameFilePath), err)
	}
	analysis.GameConfig = *config

	// Scan the whole project root once, classifying every file by extension and
	// (for .go) package. This makes the directory layout fully free-form: a
	// developer may nest scenes, objects, assets, and components anywhere, or keep
	// everything flat at the root.
	if err := analysis.scanProject(projectDir); err != nil {
		return nil, fmt.Errorf("failed to scan project files: %v", err)
	}

	return analysis, nil
}

// validateFormatVersion checks the project's declared format version against the
// one this build understands. An absent (0) format version means "the original
// format", so it defaults to the current one; a newer version is rejected, and an
// older one needs a migration we don't have yet. On success the config's
// FormatVersion is normalized to the current version.
func validateFormatVersion(config *corejson.GameConfig) error {
	if config.FormatVersion == 0 {
		config.FormatVersion = imge.CurrentFormatVersion
	}
	if config.FormatVersion > imge.CurrentFormatVersion {
		return fmt.Errorf("format_version %d is newer than this imge supports (up to %d); update the imge tool", config.FormatVersion, imge.CurrentFormatVersion)
	}
	if config.FormatVersion < imge.CurrentFormatVersion {
		return fmt.Errorf("format_version %d is older than this imge supports (up to %d); the project needs migrating", config.FormatVersion, imge.CurrentFormatVersion)
	}
	return nil
}

// scanProject walks the whole project root once and classifies every file into
// components, scenes, objects, and data assets. Directories that are not part of
// the game (VCS, build output, dependency trees) are skipped. This single walk is
// what makes the directory layout free-form: any file can live anywhere under the
// project root and still be found by the build.
func (a *ProjectAnalysis) scanProject(root string) error {
	return filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			switch d.Name() {
			case ".git", ".imge_build", "imge_build", "node_modules":
				return filepath.SkipDir
			}
			return nil
		}

		relPath, err := filepath.Rel(a.ProjectDir, path)
		if err != nil {
			return err
		}

		name := d.Name()
		switch {
		case strings.HasSuffix(name, ".go"):
			// Component source (only `package components` files; other .go files
			// are neither compiled nor embedded — the engine's model is a single
			// components package).
			if strings.HasSuffix(name, "_test.go") {
				return nil
			}
			if isComponentFile(path) {
				a.ComponentFiles = append(a.ComponentFiles, relPath)
			}
		case strings.HasSuffix(name, ".scene"):
			a.SceneFiles = append(a.SceneFiles, relPath)
		case strings.HasSuffix(name, ".obj"):
			a.ObjectFiles = append(a.ObjectFiles, relPath)
		default:
			a.AssetFiles = append(a.AssetFiles, relPath)
		}
		return nil
	})
}

// isComponentFile reports whether a .go file declares `package components`.
// PackageClauseOnly stops after the package clause, so a file with a syntax error
// elsewhere (e.g. mid-edit) is still classified correctly.
func isComponentFile(path string) bool {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, nil, parser.PackageClauseOnly)
	if err != nil {
		return false
	}
	return f.Name.Name == "components"
}
