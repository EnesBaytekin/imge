package build

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Builder executes the build process.
type Builder struct {
	ProjectDir string
	OutputName string // Final executable name (default "game")
}

// Build analyzes the project, generates a self-contained Go module that embeds
// the pure-Go Ebitengine engine, compiles it, and copies the single executable
// into the project directory.
func (b *Builder) Build() error {
	analysis, err := AnalyzeProject(b.ProjectDir)
	if err != nil {
		return fmt.Errorf("project analysis failed: %w", err)
	}

	buildDir := filepath.Join(b.ProjectDir, ".imge_build")
	if err := os.RemoveAll(buildDir); err != nil {
		return fmt.Errorf("failed to clean build directory: %w", err)
	}
	if err := os.MkdirAll(buildDir, 0755); err != nil {
		return fmt.Errorf("failed to create build directory: %w", err)
	}
	defer os.RemoveAll(buildDir)

	generator := &Generator{
		BuildDir: buildDir,
		Analysis: analysis,
	}
	if err := generator.Generate(); err != nil {
		return fmt.Errorf("generation failed: %w", err)
	}

	outputName := b.OutputName
	if outputName == "" {
		outputName = "game"
	}

	if err := b.executeGoBuild(buildDir, outputName); err != nil {
		return fmt.Errorf("go build failed: %w", err)
	}

	// Copy the single executable to the project root.
	src := filepath.Join(buildDir, outputName)
	dst := filepath.Join(b.ProjectDir, outputName)
	if err := copyFile(src, dst); err != nil {
		return fmt.Errorf("failed to copy executable: %w", err)
	}

	fmt.Printf("Built %s\n", dst)
	return nil
}

// executeGoBuild runs `go mod tidy` and `go build` inside the build directory.
func (b *Builder) executeGoBuild(buildDir, outputName string) error {
	tidyCmd := exec.Command("go", "mod", "tidy")
	tidyCmd.Dir = buildDir
	tidyCmd.Stdout = os.Stdout
	tidyCmd.Stderr = os.Stderr
	fmt.Println("Running: go mod tidy")
	if err := tidyCmd.Run(); err != nil {
		return fmt.Errorf("go mod tidy failed: %w", err)
	}

	buildCmd := exec.Command("go", "build", "-mod=mod", "-o", outputName, ".")
	buildCmd.Dir = buildDir
	buildCmd.Stdout = os.Stdout
	var stderrBuf strings.Builder
	buildCmd.Stderr = io.MultiWriter(os.Stderr, &stderrBuf)

	fmt.Printf("Running: go build -o %s\n", outputName)
	if err := buildCmd.Run(); err != nil {
		errOutput := strings.TrimSpace(stderrBuf.String())
		if errOutput != "" {
			return fmt.Errorf("go build failed:\n%s", errOutput)
		}
		return fmt.Errorf("go build command failed: %w", err)
	}

	return nil
}
