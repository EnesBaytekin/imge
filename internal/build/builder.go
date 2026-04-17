package build

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Builder executes the build process
type Builder struct {
	ProjectDir   string
	BuildDir     string
	Platform     string
	OutputName   string
	EngineSource string // Path to engine source code
}

// Build executes the full build process
func (b *Builder) Build() error {
	// Analyze project
	analysis, err := AnalyzeProject(b.ProjectDir)
	if err != nil {
		return fmt.Errorf("project analysis failed: %v", err)
	}

	// Create generator
	generator := &Generator{
		BuildDir:     b.BuildDir,
		Analysis:     analysis,
		Platform:     b.Platform,
		EngineSource: b.EngineSource,
	}

	// Generate build files
	if err := generator.Generate(); err != nil {
		return fmt.Errorf("generation failed: %v", err)
	}

	// Execute go build
	if err := b.executeGoBuild(); err != nil {
		return fmt.Errorf("go build failed: %v", err)
	}

	// Copy assets to output directory (optional)
	if err := b.copyAssets(); err != nil {
		fmt.Printf("Warning: Failed to copy assets: %v\n", err)
	}

	return nil
}

func (b *Builder) executeGoBuild() error {
	// Determine output path
	outputPath := b.OutputName
	if outputPath == "" {
		outputPath = "game"
	}
	if !strings.HasSuffix(outputPath, ".exe") && (b.Platform == "windows" || strings.Contains(b.Platform, "win")) {
		outputPath += ".exe"
	}

	// First, run go mod tidy to generate go.sum
	tidyCmd := exec.Command("go", "mod", "tidy")
	tidyCmd.Dir = b.BuildDir
	tidyCmd.Stdout = os.Stdout
	tidyCmd.Stderr = os.Stderr

	fmt.Printf("Running: go mod tidy\n")
	if err := tidyCmd.Run(); err != nil {
		fmt.Printf("Warning: go mod tidy failed: %v\n", err)
		// Continue anyway, build might still work
	}

	// Build command arguments
	args := []string{
		"build",
		"-o", outputPath,
		".",
	}

	// Create command
	cmd := exec.Command("go", args...)
	cmd.Dir = b.BuildDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	fmt.Printf("Running: go %s\n", strings.Join(args, " "))
	fmt.Printf("Working directory: %s\n", b.BuildDir)

	// Execute build
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("go build command failed: %v", err)
	}

	fmt.Printf("Build successful! Output: %s\n", outputPath)
	return nil
}

func (b *Builder) copyAssets() error {
	// Source assets directory
	srcAssetsDir := filepath.Join(b.ProjectDir, "assets")
	dstAssetsDir := filepath.Join(filepath.Dir(b.OutputName), "assets")

	// Check if source exists
	if _, err := os.Stat(srcAssetsDir); os.IsNotExist(err) {
		// No assets to copy
		return nil
	}

	// Create destination directory
	if err := os.MkdirAll(dstAssetsDir, 0755); err != nil {
		return err
	}

	// Simple implementation: just create a symlink or copy
	// For now, we'll create a symlink if possible, otherwise do nothing
	// In a real implementation, we would copy files or embed them

	fmt.Printf("Assets directory: %s\n", srcAssetsDir)
	fmt.Printf("Note: Assets are not automatically copied. Place them in the same directory as the executable.\n")

	return nil
}

// Clean removes the build directory
func Clean(buildDir string) error {
	if _, err := os.Stat(buildDir); os.IsNotExist(err) {
		// Directory doesn't exist, nothing to clean
		return nil
	}

	return os.RemoveAll(buildDir)
}