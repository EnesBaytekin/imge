package build

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// Builder executes the build process.
type Builder struct {
	ProjectDir string
	Target     string // TargetDesktop or TargetWeb
	GOOS       string // target OS ("" = native host)
	GOARCH     string // target architecture ("" = native host)
}

// Build analyzes the project, generates a self-contained Go module that embeds
// the pure-Go Ebitengine engine, compiles it for the selected target, and writes
// the result into <project>/imge_build/. It returns the built artifact path (the
// executable for desktop, the bundle directory for web).
func (b *Builder) Build() (string, error) {
	analysis, err := AnalyzeProject(b.ProjectDir)
	if err != nil {
		return "", fmt.Errorf("project analysis failed: %w", err)
	}

	goos := b.GOOS
	if goos == "" {
		goos = runtime.GOOS
	}
	goarch := b.GOARCH
	if goarch == "" {
		goarch = runtime.GOARCH
	}
	if b.Target != TargetWeb {
		if err := ValidateTarget(goos, goarch); err != nil {
			return "", err
		}
	}

	buildDir := filepath.Join(b.ProjectDir, ".imge_build")
	if err := os.RemoveAll(buildDir); err != nil {
		return "", fmt.Errorf("failed to clean build directory: %w", err)
	}
	if err := os.MkdirAll(buildDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create build directory: %w", err)
	}
	defer os.RemoveAll(buildDir)

	generator := &Generator{
		BuildDir: buildDir,
		Analysis: analysis,
		Target:   b.Target,
	}
	if err := generator.Generate(); err != nil {
		return "", fmt.Errorf("generation failed: %w", err)
	}

	if b.Target == TargetWeb {
		return b.buildWeb(buildDir)
	}
	return b.buildDesktop(buildDir, analysis, goos, goarch)
}

// buildDesktop compiles a native executable into imge_build/<name>_<os>-<arch>.
func (b *Builder) buildDesktop(buildDir string, analysis *ProjectAnalysis, goos, goarch string) (string, error) {
	if err := b.goModTidy(buildDir); err != nil {
		return "", err
	}

	outDir := filepath.Join(b.ProjectDir, "imge_build")
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create output directory: %w", err)
	}

	outName := fmt.Sprintf("%s_%s-%s", slugify(analysis.GameConfig.Name), goos, goarch)
	if goos == "windows" {
		outName += ".exe"
	}

	cmd := exec.Command("go", "build", "-mod=mod", "-o", outName, ".")
	cmd.Dir = buildDir
	if goos != runtime.GOOS || goarch != runtime.GOARCH {
		cmd.Env = append(os.Environ(), "GOOS="+goos, "GOARCH="+goarch)
	}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	fmt.Printf("Running: go build -o %s (%s/%s)\n", outName, goos, goarch)
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("go build failed: %w", err)
	}

	src := filepath.Join(buildDir, outName)
	dst := filepath.Join(outDir, outName)
	if err := copyFile(src, dst); err != nil {
		return "", fmt.Errorf("failed to copy executable: %w", err)
	}

	fmt.Printf("Built %s\n", dst)
	return dst, nil
}

// buildWeb compiles a WebAssembly bundle into <project>/imge_build/web/.
func (b *Builder) buildWeb(buildDir string) (string, error) {
	if err := b.goModTidy(buildDir); err != nil {
		return "", err
	}

	outDir := filepath.Join(b.ProjectDir, "imge_build", "web")
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create web output directory: %w", err)
	}

	wasmPath := filepath.Join(outDir, "game.wasm")
	cmd := exec.Command("go", "build", "-mod=mod", "-o", wasmPath, ".")
	cmd.Dir = buildDir
	cmd.Env = append(os.Environ(), "GOOS=js", "GOARCH=wasm")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	fmt.Println("Running: GOOS=js GOARCH=wasm go build -o imge_build/web/game.wasm")
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("web build failed: %w", err)
	}

	if err := b.copyWasmExec(outDir); err != nil {
		return "", fmt.Errorf("failed to copy wasm_exec.js: %w", err)
	}
	if err := b.writeIndexHTML(outDir); err != nil {
		return "", fmt.Errorf("failed to write index.html: %w", err)
	}

	fmt.Printf("Built web bundle in %s/\n", outDir)
	fmt.Println("Serve it with:  cd imge_build/web && python3 -m http.server 8000")
	fmt.Println("Then open:       http://localhost:8000/")
	return outDir, nil
}

// goModTidy runs `go mod tidy` inside the build directory.
func (b *Builder) goModTidy(buildDir string) error {
	tidyCmd := exec.Command("go", "mod", "tidy")
	tidyCmd.Dir = buildDir
	tidyCmd.Stdout = os.Stdout
	tidyCmd.Stderr = os.Stderr
	fmt.Println("Running: go mod tidy")
	if err := tidyCmd.Run(); err != nil {
		return fmt.Errorf("go mod tidy failed: %w", err)
	}
	return nil
}

// copyWasmExec copies the Go wasm runtime helper into the web output directory.
// The location moved from $GOROOT/misc/wasm to $GOROOT/lib/wasm in Go 1.25, so
// both are tried.
func (b *Builder) copyWasmExec(outDir string) error {
	goroot, err := exec.Command("go", "env", "GOROOT").Output()
	if err != nil {
		return err
	}
	root := strings.TrimSpace(string(goroot))

	candidates := []string{
		filepath.Join(root, "lib", "wasm", "wasm_exec.js"),
		filepath.Join(root, "misc", "wasm", "wasm_exec.js"),
	}
	for _, src := range candidates {
		if _, err := os.Stat(src); err == nil {
			return copyFile(src, filepath.Join(outDir, "wasm_exec.js"))
		}
	}
	return fmt.Errorf("wasm_exec.js not found under %s (checked lib/wasm and misc/wasm)", root)
}

// writeIndexHTML writes a minimal loader page for the wasm bundle. The loader
// fetches game.wasm as bytes and instantiates it manually (rather than using
// instantiateStreaming) so it works with any static file server regardless of
// whether it sends the application/wasm MIME type.
func (b *Builder) writeIndexHTML(outDir string) error {
	html := `<!doctype html>
<html>
<head>
	<meta charset="utf-8">
	<title>IMGE Game</title>
	<style>html, body { margin: 0; height: 100%; background: #000; overflow: hidden; }</style>
</head>
<body>
	<script src="wasm_exec.js"></script>
	<script>
		const go = new Go();
		fetch("game.wasm")
			.then((resp) => resp.arrayBuffer())
			.then((bytes) => WebAssembly.instantiate(bytes, go.importObject))
			.then((result) => { go.run(result.instance); })
			.catch((err) => console.error("failed to load game:", err));
	</script>
</body>
</html>
`
	return os.WriteFile(filepath.Join(outDir, "index.html"), []byte(html), 0644)
}
