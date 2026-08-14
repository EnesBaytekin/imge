package build

import (
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/EnesBaytekin/imge"
)

// engineModule is the Go module path of the engine. The generated game module
// references it and points it at the embedded source via a replace directive.
const engineModule = "github.com/EnesBaytekin/imge"

// ebitenVersion is pinned so game builds resolve a deterministic, known-good
// Ebitengine release. Bump alongside the engine's own go.mod when upgrading.
const ebitenVersion = "v2.9.9"

// Build targets.
const (
	TargetDesktop = "desktop" // native executable for the current OS
	TargetWeb     = "web"     // WebAssembly build
)

// Generator creates a self-contained Go module inside a temporary build
// directory. The module embeds the pure-Go engine source (core, engine, and the
// Ebitengine platform), the user's components, and the game data, so `go build`
// produces a single executable (or WebAssembly bundle) with no CGO or Docker.
type Generator struct {
	BuildDir string
	Analysis *ProjectAnalysis
	Target   string // TargetDesktop or TargetWeb
}

// Generate creates all necessary build files.
func (g *Generator) Generate() error {
	if err := g.extractEngine(); err != nil {
		return fmt.Errorf("failed to extract engine source: %w", err)
	}
	if err := g.writeEngineGoMod(); err != nil {
		return fmt.Errorf("failed to write engine go.mod: %w", err)
	}
	if err := g.copyUserComponents(); err != nil {
		return fmt.Errorf("failed to copy user components: %w", err)
	}
	if err := g.copyProjectData(); err != nil {
		return fmt.Errorf("failed to copy project data: %w", err)
	}
	if err := g.generateMainGo(); err != nil {
		return fmt.Errorf("failed to generate main.go: %w", err)
	}
	if err := g.generateGoMod(); err != nil {
		return fmt.Errorf("failed to generate go.mod: %w", err)
	}
	return nil
}

// extractEngine copies the embedded pure-Go engine source into <buildDir>/_engine.
func (g *Generator) extractEngine() error {
	dst := filepath.Join(g.BuildDir, "_engine")
	return copyEmbedFS(imge.EngineSource, ".", dst)
}

// writeEngineGoMod writes a minimal go.mod for the embedded engine so the game
// module's replace directive has a valid module to point at.
func (g *Generator) writeEngineGoMod() error {
	content := fmt.Sprintf("module %s\n\ngo 1.24\n\nrequire github.com/hajimehoshi/ebiten/v2 %s\n", engineModule, ebitenVersion)
	return os.WriteFile(filepath.Join(g.BuildDir, "_engine", "go.mod"), []byte(content), 0644)
}

// copyUserComponents copies the user's component .go files into the build dir.
// If the project has no components, a placeholder package is written so the
// generated main.go's blank import remains valid.
func (g *Generator) copyUserComponents() error {
	if len(g.Analysis.ComponentFiles) == 0 {
		placeholder := filepath.Join(g.BuildDir, "components", "placeholder.go")
		if err := os.MkdirAll(filepath.Dir(placeholder), 0755); err != nil {
			return err
		}
		return os.WriteFile(placeholder, []byte("// Package components holds user-defined components.\npackage components\n"), 0644)
	}

	for _, compFile := range g.Analysis.ComponentFiles {
		srcPath := filepath.Join(g.Analysis.ProjectDir, compFile)
		dstPath := filepath.Join(g.BuildDir, compFile)
		if err := os.MkdirAll(filepath.Dir(dstPath), 0755); err != nil {
			return fmt.Errorf("failed to create component directory %s: %w", filepath.Dir(dstPath), err)
		}
		if err := copyFile(srcPath, dstPath); err != nil {
			return fmt.Errorf("failed to copy component %s: %w", compFile, err)
		}
	}
	return nil
}

// copyProjectData copies the assets/, scenes/, and objects/ directories (any
// that exist) into the build dir so they can be embedded into the final binary.
func (g *Generator) copyProjectData() error {
	for _, dir := range []string{"assets", "scenes", "objects"} {
		src := filepath.Join(g.Analysis.ProjectDir, dir)
		if _, err := os.Stat(src); os.IsNotExist(err) {
			continue
		}
		if err := copyDir(src, filepath.Join(g.BuildDir, dir)); err != nil {
			return fmt.Errorf("failed to copy %s: %w", dir, err)
		}
	}
	return nil
}

// generateMainGo writes the Ebitengine entrypoint for the selected target. The
// generated program embeds the project data, loads every scene, and hands
// control to the Ebitengine platform's Run loop.
func (g *Generator) generateMainGo() error {
	// Determine which data directories exist so we only embed what's present.
	var patterns []string
	for _, dir := range []string{"assets", "scenes", "objects"} {
		if dirHasFiles(filepath.Join(g.BuildDir, dir)) {
			patterns = append(patterns, dir)
		}
	}

	if g.Target == TargetWeb {
		return g.generateWebMainGo(patterns)
	}
	return g.generateDesktopMainGo(patterns)
}

// generateDesktopMainGo writes the native entrypoint. It extracts the embedded
// data to a temp directory and chdirs into it so relative paths (and asset
// loading via the OS filesystem) work exactly as they do during development.
func (g *Generator) generateDesktopMainGo(patterns []string) error {
	embedDirective := ""
	hasData := len(patterns) > 0
	if hasData {
		embedDirective = "//go:embed " + strings.Join(patterns, " ") + "\n"
	}

	data := struct {
		ModuleName     string
		WindowTitle    string
		WindowWidth    int
		WindowHeight   int
		TargetFPS      int
		InitialScene   string
		EmbedDirective string
		HasData        bool
	}{
		ModuleName:     fmt.Sprintf("%s_build", filepath.Base(g.Analysis.ProjectDir)),
		WindowTitle:    g.Analysis.GameConfig.Window.Title,
		WindowWidth:    g.Analysis.GameConfig.Window.Width,
		WindowHeight:   g.Analysis.GameConfig.Window.Height,
		TargetFPS:      g.Analysis.GameConfig.Game.TargetFPS,
		InitialScene:   g.Analysis.GameConfig.Game.InitialScene,
		EmbedDirective: embedDirective,
		HasData:        hasData,
	}

	return g.renderTemplate(mainTemplateDesktop, data)
}

// generateWebMainGo writes the WebAssembly entrypoint. It loads scenes and their
// referenced object files directly from the embedded filesystem (no OS file I/O,
// which is unavailable in the browser).
func (g *Generator) generateWebMainGo(patterns []string) error {
	if len(patterns) == 0 {
		return fmt.Errorf("web build requires at least one of scenes/, objects/, or assets/")
	}

	data := struct {
		ModuleName    string
		WindowTitle   string
		WindowWidth   int
		WindowHeight  int
		TargetFPS     int
		InitialScene  string
		EmbedPatterns string
	}{
		ModuleName:    fmt.Sprintf("%s_build", filepath.Base(g.Analysis.ProjectDir)),
		WindowTitle:   g.Analysis.GameConfig.Window.Title,
		WindowWidth:   g.Analysis.GameConfig.Window.Width,
		WindowHeight:  g.Analysis.GameConfig.Window.Height,
		TargetFPS:     g.Analysis.GameConfig.Game.TargetFPS,
		InitialScene:  g.Analysis.GameConfig.Game.InitialScene,
		EmbedPatterns: strings.Join(patterns, " "),
	}

	return g.renderTemplate(mainTemplateWeb, data)
}

// renderTemplate renders the given template with data and writes main.go.
func (g *Generator) renderTemplate(tmplText string, data interface{}) error {
	tmpl, err := template.New("main").Parse(tmplText)
	if err != nil {
		return err
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(g.BuildDir, "main.go"), buf.Bytes(), 0644)
}

// generateGoMod writes the game module's go.mod, pointing the engine module at
// the extracted source with a replace directive.
func (g *Generator) generateGoMod() error {
	modName := fmt.Sprintf("%s_build", filepath.Base(g.Analysis.ProjectDir))
	content := fmt.Sprintf("module %s\n\ngo 1.24\n\nrequire %s v0.0.0\n\nreplace %s => ./_engine\n", modName, engineModule, engineModule)
	return os.WriteFile(filepath.Join(g.BuildDir, "go.mod"), []byte(content), 0644)
}

// mainTemplateDesktop is the generated native entrypoint. .EmbedDirective is
// either a "//go:embed <dirs>" line (with trailing newline) or empty, so it must
// sit directly above the projectData declaration with no blank line between them.
const mainTemplateDesktop = `// GENERATED CODE - DO NOT EDIT
package main

import (
	"embed"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/EnesBaytekin/imge/core"
	_ "github.com/EnesBaytekin/imge/engine/components"
	"github.com/EnesBaytekin/imge/platform/ebitengine"
	_ "{{.ModuleName}}/components"
)

{{.EmbedDirective}}var projectData embed.FS

func main() {
	// Extract embedded game data to a temp directory so all relative paths
	// (objects/, scenes/, assets/) resolve exactly as they did in the project.
	dataDir, err := extractProjectData()
	if err != nil {
		log.Fatalf("failed to extract project data: %v", err)
	}
	defer os.RemoveAll(dataDir)

	if err := os.Chdir(dataDir); err != nil {
		log.Fatalf("failed to change working directory: %v", err)
	}

	platform, err := ebitengine.New()
	if err != nil {
		log.Fatalf("failed to create platform: %v", err)
	}
	defer platform.Cleanup()

	game := core.NewGameWithConfig(core.Config{
		WindowWidth:  {{.WindowWidth}},
		WindowHeight: {{.WindowHeight}},
		WindowTitle:  "{{.WindowTitle}}",
		TargetFPS:    {{.TargetFPS}},
		InitialScene: "{{.InitialScene}}",
	})
	game.SetPlatform(platform)

	if err := game.Init(); err != nil {
		log.Fatalf("failed to initialize game: %v", err)
	}

	loadScenes(game)

	if err := platform.Run(game); err != nil {
		log.Fatalf("game error: %v", err)
	}
}

// loadScenes loads every .scene file under scenes/ so the initial scene (and any
// scene the game switches to later) is available.
func loadScenes(game *core.Game) {
	entries, err := os.ReadDir("scenes")
	if err != nil {
		log.Printf("warning: no scenes directory found: %v", err)
		return
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".scene") {
			continue
		}
		name := strings.TrimSuffix(entry.Name(), ".scene")
		scene := core.NewScene(name)
		if err := scene.LoadFromFile(filepath.Join("scenes", entry.Name())); err != nil {
			log.Printf("warning: failed to load scene %q: %v", name, err)
			continue
		}
		game.AddScene(scene)
	}
}

// extractProjectData writes the embedded game data to a fresh temp directory.
func extractProjectData() (string, error) {
	dir, err := os.MkdirTemp("", "imge-*")
	if err != nil {
		return "", err
	}
{{if .HasData}}
	if err := writeFS(projectData, ".", dir); err != nil {
		os.RemoveAll(dir)
		return "", err
	}
{{end}}
	return dir, nil
}

// writeFS copies every file in fsys from src to dst, preserving relative paths.
func writeFS(fsys fs.FS, src, dst string) error {
	return fs.WalkDir(fsys, src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return err
		}
		data, err := fs.ReadFile(fsys, path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0644)
	})
}
`

// mainTemplateWeb is the generated WebAssembly entrypoint. It has no OS
// filesystem calls — scenes and object files are read from the embedded FS.
const mainTemplateWeb = `// GENERATED CODE - DO NOT EDIT
package main

import (
	"embed"
	"io/fs"
	"log"
	"strings"

	"github.com/EnesBaytekin/imge/core"
	_ "github.com/EnesBaytekin/imge/engine/components"
	"github.com/EnesBaytekin/imge/platform/ebitengine"
	_ "{{.ModuleName}}/components"
)

//go:embed {{.EmbedPatterns}}
var projectData embed.FS

func main() {
	platform, err := ebitengine.New()
	if err != nil {
		log.Fatalf("failed to create platform: %v", err)
	}
	defer platform.Cleanup()

	// Load textures and audio from the embedded filesystem (no OS file I/O in
	// the browser).
	platform.SetAssetFS(projectData)

	game := core.NewGameWithConfig(core.Config{
		WindowWidth:  {{.WindowWidth}},
		WindowHeight: {{.WindowHeight}},
		WindowTitle:  "{{.WindowTitle}}",
		TargetFPS:    {{.TargetFPS}},
		InitialScene: "{{.InitialScene}}",
	})
	game.SetPlatform(platform)

	if err := game.Init(); err != nil {
		log.Fatalf("failed to initialize game: %v", err)
	}

	loadScenes(game)

	if err := platform.Run(game); err != nil {
		log.Fatalf("game error: %v", err)
	}
}

// loadScenes loads every .scene file from the embedded filesystem.
func loadScenes(game *core.Game) {
	entries, err := fs.ReadDir(projectData, "scenes")
	if err != nil {
		log.Printf("warning: no scenes directory found: %v", err)
		return
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".scene") {
			continue
		}
		name := strings.TrimSuffix(entry.Name(), ".scene")
		scene := core.NewScene(name)
		if err := scene.LoadFromFS(projectData, "scenes/"+entry.Name()); err != nil {
			log.Printf("warning: failed to load scene %q: %v", name, err)
			continue
		}
		game.AddScene(scene)
	}
}
`

// copyEmbedFS copies every file under src in fsys into dst, preserving paths.
func copyEmbedFS(fsys fs.FS, src, dst string) error {
	return fs.WalkDir(fsys, src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return err
		}
		data, err := fs.ReadFile(fsys, path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0644)
	})
}

// dirHasFiles reports whether a directory exists and contains at least one file.
func dirHasFiles(dir string) bool {
	found := false
	_ = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() {
			found = true
			return filepath.SkipAll
		}
		return nil
	})
	return found
}

// copyDir copies all files in src into dst, preserving relative paths. Test
// files are skipped so they don't leak into the final binary's module graph.
func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if strings.HasSuffix(info.Name(), "_test.go") {
			return nil
		}
		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		dstPath := filepath.Join(dst, relPath)
		if err := os.MkdirAll(filepath.Dir(dstPath), 0755); err != nil {
			return err
		}
		return copyFile(path, dstPath)
	})
}

// copyFile copies a single file, preserving its permissions.
func copyFile(src, dst string) error {
	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return err
	}
	return os.Chmod(dst, srcInfo.Mode())
}
