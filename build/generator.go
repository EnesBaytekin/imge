package build

import (
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
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
	Debug    bool   // enable the debug overlay pass in the generated game
}

// Generate creates all necessary build files.
func (g *Generator) Generate() error {
	if err := g.extractEngine(); err != nil {
		return fmt.Errorf("failed to extract engine source: %w", err)
	}
	if err := g.writeEngineGoMod(); err != nil {
		return fmt.Errorf("failed to write engine go.mod: %w", err)
	}
	if err := g.copyComponents(); err != nil {
		return fmt.Errorf("failed to copy components: %w", err)
	}
	kinds, err := g.componentKinds()
	if err != nil {
		return err
	}
	warnings, err := g.validateComponents(kinds)
	for _, w := range warnings {
		fmt.Fprintf(os.Stderr, "warning: %s\n", w)
	}
	if err != nil {
		return err
	}
	if err := g.validateFileReferences(); err != nil {
		return err
	}
	if err := g.generateRegistry(kinds); err != nil {
		return fmt.Errorf("failed to generate component registry: %w", err)
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

// copyComponents merges the built-in component files (from the embedded engine)
// and the user's component files into a single `components` package in the build
// dir. This single-package model lets every component call every other
// component's methods directly, with no capability interfaces between them.
func (g *Generator) copyComponents() error {
	dstDir := filepath.Join(g.BuildDir, "components")
	if err := os.MkdirAll(dstDir, 0755); err != nil {
		return err
	}

	// Track flattened filenames so collisions are reported clearly.
	used := make(map[string]string) // flattened name -> source description

	// 1. Built-in components from the extracted engine.
	builtinDir := filepath.Join(g.BuildDir, "_engine", "engine", "components")
	entries, err := os.ReadDir(builtinDir)
	if err != nil {
		return fmt.Errorf("failed to read built-in components: %w", err)
	}
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		if err := copyFile(filepath.Join(builtinDir, name), filepath.Join(dstDir, name)); err != nil {
			return fmt.Errorf("failed to copy built-in component %s: %w", name, err)
		}
		used[name] = "built-in:" + name
	}

	// 2. User components, flattened into the single package directory.
	for _, compFile := range g.Analysis.ComponentFiles {
		flattened := flattenComponentName(compFile)
		if prev, exists := used[flattened]; exists {
			return fmt.Errorf("component file collision: %s and %s both flatten to %s", prev, compFile, flattened)
		}
		used[flattened] = compFile

		srcPath := filepath.Join(g.Analysis.ProjectDir, compFile)
		if err := copyFile(srcPath, filepath.Join(dstDir, flattened)); err != nil {
			return fmt.Errorf("failed to copy component %s: %w", compFile, err)
		}
	}

	return nil
}

// flattenComponentName maps a project-relative component path to a filename in
// the single components/ package. Files directly under components/ keep their
// basename; nested files fold their directory into the filename.
func flattenComponentName(rel string) string {
	rel = filepath.ToSlash(rel)
	base := path.Base(rel)
	dir := path.Dir(rel)
	if dir == "." || dir == "/" || dir == "components" {
		return base
	}
	return strings.ReplaceAll(dir, "/", "_") + "_" + base
}

// copyProjectData copies the entire project tree (minus source and build output)
// into <buildDir>/project/ so it can be embedded as a single unit. Embedding the
// whole project — rather than only assets/, scenes/, and objects/ — is what lets
// developers organize data files anywhere under the project root and reference
// them by their root-relative path.
func (g *Generator) copyProjectData() error {
	dst := filepath.Join(g.BuildDir, "project")
	if err := os.MkdirAll(dst, 0755); err != nil {
		return err
	}

	return filepath.WalkDir(g.Analysis.ProjectDir, func(path string, d fs.DirEntry, err error) error {
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
		// Component source is compiled into the binary, not embedded as data.
		if strings.HasSuffix(d.Name(), ".go") {
			return nil
		}

		relPath, err := filepath.Rel(g.Analysis.ProjectDir, path)
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

// generateMainGo writes the Ebitengine entrypoint for the selected target. The
// generated program embeds the project data, loads every scene, and hands
// control to the Ebitengine platform's Run loop.
func (g *Generator) generateMainGo() error {
	hasData := dirHasFiles(filepath.Join(g.BuildDir, "project"))
	if g.Target == TargetWeb {
		return g.generateWebMainGo(hasData)
	}
	return g.generateDesktopMainGo(hasData)
}

// generateDesktopMainGo writes the native entrypoint. It extracts the embedded
// data to a temp directory and chdirs into it so root-relative paths (and asset
// loading via the OS filesystem) work exactly as they do during development.
func (g *Generator) generateDesktopMainGo(hasData bool) error {
	embedDirective := ""
	if hasData {
		embedDirective = "//go:embed all:project\n"
	}

	data := struct {
		ModuleName         string
		WindowTitle        string
		WindowWidth        int
		WindowHeight       int
		WindowFullscreen   bool
		WindowResizable    bool
		WindowPixelPerUnit int
		WindowScale        int
		WindowSmoothShapes bool
		TargetFPS          int
		InitialScene       string
		EmbedDirective     string
		HasData            bool
		Debug              bool
	}{
		ModuleName:         fmt.Sprintf("%s_build", filepath.Base(g.Analysis.ProjectDir)),
		WindowTitle:        g.Analysis.GameConfig.Window.Title,
		WindowWidth:        g.Analysis.GameConfig.Window.Width,
		WindowHeight:       g.Analysis.GameConfig.Window.Height,
		WindowFullscreen:   g.Analysis.GameConfig.Window.Fullscreen,
		WindowResizable:    g.Analysis.GameConfig.Window.Resizable,
		WindowPixelPerUnit: g.Analysis.GameConfig.Window.PixelPerUnit,
		WindowScale:        g.Analysis.GameConfig.Window.Scale,
		WindowSmoothShapes: g.Analysis.GameConfig.Window.SmoothShapes,
		TargetFPS:          g.Analysis.GameConfig.Game.TargetFPS,
		InitialScene:       g.Analysis.GameConfig.Game.InitialScene,
		EmbedDirective:     embedDirective,
		HasData:            hasData,
		Debug:              g.Debug,
	}

	return g.renderTemplate(mainTemplateDesktop, data)
}

// generateWebMainGo writes the WebAssembly entrypoint. It loads scenes and their
// referenced object files directly from the embedded filesystem (no OS file I/O,
// which is unavailable in the browser).
func (g *Generator) generateWebMainGo(hasData bool) error {
	if !hasData {
		return fmt.Errorf("web build requires project data (scenes, objects, or assets)")
	}

	data := struct {
		ModuleName         string
		WindowTitle        string
		WindowWidth        int
		WindowHeight       int
		WindowFullscreen   bool
		WindowPixelPerUnit int
		WindowSmoothShapes bool
		TargetFPS          int
		InitialScene       string
		HasData            bool
		Debug              bool
	}{
		ModuleName:         fmt.Sprintf("%s_build", filepath.Base(g.Analysis.ProjectDir)),
		WindowTitle:        g.Analysis.GameConfig.Window.Title,
		WindowWidth:        g.Analysis.GameConfig.Window.Width,
		WindowHeight:       g.Analysis.GameConfig.Window.Height,
		WindowFullscreen:   g.Analysis.GameConfig.Window.Fullscreen,
		WindowPixelPerUnit: g.Analysis.GameConfig.Window.PixelPerUnit,
		WindowSmoothShapes: g.Analysis.GameConfig.Window.SmoothShapes,
		TargetFPS:          g.Analysis.GameConfig.Game.TargetFPS,
		InitialScene:       g.Analysis.GameConfig.Game.InitialScene,
		HasData:            hasData,
		Debug:              g.Debug,
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
	"github.com/EnesBaytekin/imge/platform/ebitengine"
	_ "{{.ModuleName}}/components"
)

{{.EmbedDirective}}var projectData embed.FS

func main() {
	// Extract embedded game data to a temp directory so every root-relative path
	// (objects, scenes, assets, or any free-form layout) resolves exactly as it
	// did in the project.
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
		Window: core.WindowConfig{
			Title:      "{{.WindowTitle}}",
			Width:      {{.WindowWidth}},
			Height:     {{.WindowHeight}},
			Fullscreen: {{.WindowFullscreen}},
			Resizable:  {{.WindowResizable}},
			PixelPerUnit: {{.WindowPixelPerUnit}},
			Scale:      {{.WindowScale}},
			SmoothShapes: {{.WindowSmoothShapes}},
		},
		TargetFPS:    {{.TargetFPS}},
		InitialScene: "{{.InitialScene}}",
	})
	game.SetPlatform(platform)

	if err := game.Init(); err != nil {
		log.Fatalf("failed to initialize game: %v", err)
	}

	// styles.imge is optional: load it before scenes so components can resolve
	// their style args during scene loading.
	if data, err := os.ReadFile("styles.imge"); err == nil {
		if err := core.LoadStyles(data); err != nil {
			log.Printf("warning: failed to load styles.imge: %v", err)
		}
	}

	loadScenes(game)

	if err := platform.Run(game); err != nil {
		log.Fatalf("game error: %v", err)
	}
}

// loadScenes walks the extracted project root and loads every .scene file found,
// at any depth, so the initial scene (and any scene the game switches to later)
// is available.
func loadScenes(game *core.Game) {
	_ = filepath.WalkDir(".", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() || !strings.HasSuffix(d.Name(), ".scene") {
			return nil
		}
		name := strings.TrimSuffix(d.Name(), ".scene")
		scene := core.NewScene(name)
		if err := scene.LoadFromFile(p); err != nil {
			log.Printf("warning: failed to load scene %q: %v", p, err)
			return nil
		}
{{if .Debug}}		scene.SetDebugDraw(true)
{{end}}		game.AddScene(scene)
		return nil
	})
}

// extractProjectData writes the embedded game data to a fresh temp directory.
func extractProjectData() (string, error) {
	dir, err := os.MkdirTemp("", "imge-*")
	if err != nil {
		return "", err
	}
{{if .HasData}}
	if err := writeFS(projectData, "project", dir); err != nil {
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
	"github.com/EnesBaytekin/imge/platform/ebitengine"
	_ "{{.ModuleName}}/components"
)

//go:embed all:project
var projectData embed.FS

func main() {
	platform, err := ebitengine.New()
	if err != nil {
		log.Fatalf("failed to create platform: %v", err)
	}
	defer platform.Cleanup()

	// Re-root the embedded FS to the project/ prefix so every path is
	// project-relative, matching the OS filesystem on desktop builds.
	projectFS, err := fs.Sub(projectData, "project")
	if err != nil {
		log.Fatalf("failed to resolve embedded project data: %v", err)
	}

	// Load textures and audio from the embedded filesystem (no OS file I/O in
	// the browser).
	platform.SetAssetFS(projectFS)

	game := core.NewGameWithConfig(core.Config{
		Window: core.WindowConfig{
			Title:      "{{.WindowTitle}}",
			Width:      {{.WindowWidth}},
			Height:     {{.WindowHeight}},
			Fullscreen: {{.WindowFullscreen}},
			PixelPerUnit: {{.WindowPixelPerUnit}},
			SmoothShapes: {{.WindowSmoothShapes}},
		},
		TargetFPS:    {{.TargetFPS}},
		InitialScene: "{{.InitialScene}}",
	})
	game.SetPlatform(platform)

	if err := game.Init(); err != nil {
		log.Fatalf("failed to initialize game: %v", err)
	}

	// styles.imge is optional: load it before scenes so components can resolve
	// their style args during scene loading.
	if data, err := fs.ReadFile(projectFS, "styles.imge"); err == nil {
		if err := core.LoadStyles(data); err != nil {
			log.Printf("warning: failed to load styles.imge: %v", err)
		}
	}

	loadScenes(game, projectFS)

	if err := platform.Run(game); err != nil {
		log.Fatalf("game error: %v", err)
	}
}

// loadScenes walks the embedded project FS and loads every .scene file found, at
// any depth, so the initial scene (and any scene the game switches to later) is
// available.
func loadScenes(game *core.Game, projectFS fs.FS) {
	_ = fs.WalkDir(projectFS, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() || !strings.HasSuffix(d.Name(), ".scene") {
			return nil
		}
		name := strings.TrimSuffix(d.Name(), ".scene")
		scene := core.NewScene(name)
		if err := scene.LoadFromFS(projectFS, p); err != nil {
			log.Printf("warning: failed to load scene %q: %v", p, err)
			return nil
		}
{{if .Debug}}		scene.SetDebugDraw(true)
{{end}}		game.AddScene(scene)
		return nil
	})
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
