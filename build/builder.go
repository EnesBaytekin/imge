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
	OutputDir    string // Final output directory (e.g., imge_build_mock)
	EngineSource string // Path to engine source code
}

// Build executes the full build process
func (b *Builder) Build() error {
	// Analyze project
	analysis, err := AnalyzeProject(b.ProjectDir)
	if err != nil {
		return fmt.Errorf("project analysis failed: %v", err)
	}

	// Set default output directory if not specified
	if b.OutputDir == "" {
		b.OutputDir = fmt.Sprintf("imge_build_%s", b.Platform)
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

	// Copy final output (executable + assets) to output directory
	if err := b.copyFinalOutput(); err != nil {
		fmt.Printf("Warning: Failed to copy final output: %v\n", err)
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

	// Run go mod vendor to create vendor directory
	vendorCmd := exec.Command("go", "mod", "vendor")
	vendorCmd.Dir = b.BuildDir
	vendorCmd.Stdout = os.Stdout
	vendorCmd.Stderr = os.Stderr

	fmt.Printf("Running: go mod vendor\n")
	if err := vendorCmd.Run(); err != nil {
		fmt.Printf("Warning: go mod vendor failed: %v\n", err)
		// Continue anyway, build might still work
	}

	// Build command arguments
	args := []string{
		"build",
		"-mod", "vendor",
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

func (b *Builder) copyFinalOutput() error {
	// Clean output directory before copying
	if err := os.RemoveAll(b.OutputDir); err != nil {
		fmt.Printf("Warning: Failed to clean output directory %s: %v\n", b.OutputDir, err)
		// Continue anyway
	}
	// Create output directory
	if err := os.MkdirAll(b.OutputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory %s: %v", b.OutputDir, err)
	}

	// Determine source executable path
	srcExePath := filepath.Join(b.BuildDir, b.OutputName)
	if b.OutputName == "" {
		srcExePath = filepath.Join(b.BuildDir, "game")
	}

	// Determine destination executable path
	dstExePath := filepath.Join(b.OutputDir, filepath.Base(srcExePath))

	// Copy executable
	if err := copyFile(srcExePath, dstExePath); err != nil {
		return fmt.Errorf("failed to copy executable: %v", err)
	}

	fmt.Printf("Executable copied to: %s\n", dstExePath)

	// Copy project directories (assets/, scenes/, objects/)
	dirsToCopy := []string{"assets", "scenes", "objects"}
	for _, dir := range dirsToCopy {
		srcDir := filepath.Join(b.ProjectDir, dir)
		dstDir := filepath.Join(b.OutputDir, dir)

		if _, err := os.Stat(srcDir); os.IsNotExist(err) {
			continue // Directory doesn't exist, skip
		}

		if err := copyDir(srcDir, dstDir); err != nil {
			return fmt.Errorf("failed to copy %s directory: %v", dir, err)
		}
		fmt.Printf("Copied %s/ directory to output\n", dir)
	}

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