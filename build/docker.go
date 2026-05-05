package build

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

const dockerImageName = "enesbaytekin/imge-sdl-builder:latest"

func checkDocker() error {
	if err := exec.Command("docker", "--version").Run(); err != nil {
		return fmt.Errorf("docker is not available. Install Docker: https://docs.docker.com/get-docker/")
	}
	return nil
}

func ensureDockerImage() error {
	if err := checkDocker(); err != nil {
		return err
	}

	checkCmd := exec.Command("docker", "image", "inspect", dockerImageName, "--format", "{{.Id}}")
	if checkCmd.Run() == nil {
		fmt.Printf("Docker image %s found locally\n", dockerImageName)
		return nil
	}

	fmt.Printf("Pulling Docker image %s...\n", dockerImageName)
	pullCmd := exec.Command("docker", "pull", dockerImageName)
	pullCmd.Stdout = os.Stdout
	pullCmd.Stderr = os.Stderr
	if err := pullCmd.Run(); err != nil {
		return fmt.Errorf("failed to pull Docker image %s: %v", dockerImageName, err)
	}

	fmt.Printf("Docker image %s pulled successfully\n", dockerImageName)
	return nil
}

func (b *Builder) executeDockerBuild() error {
	if err := ensureDockerImage(); err != nil {
		return err
	}

	outputPath := b.OutputName
	if outputPath == "" {
		outputPath = "game"
	}

	absBuildDir, err := filepath.Abs(b.BuildDir)
	if err != nil {
		return fmt.Errorf("failed to get absolute build dir: %v", err)
	}

	// Build command:
	//   1. Use replace directive so engine is picked from /imge-engine (not GitHub)
	//   2. go mod tidy resolves all dependencies (pre-cached in image)
	//   3. Build with statically linked SDL2 (via pkg-config --static wrapper)
	//      glibc stays dynamic so ALSA/PulseAudio dlopen works at runtime
	buildCmd := fmt.Sprintf(
		"cd /build && "+
			"go mod edit -replace github.com/EnesBaytekin/imge=/imge-engine && "+
			"go mod tidy && "+
			"CGO_ENABLED=1 go build -tags sdl -mod=mod -o %s .",
		outputPath,
	)

	args := []string{
		"run", "--rm",
		"-v", fmt.Sprintf("%s:/build", absBuildDir),
		dockerImageName,
		"sh", "-c", buildCmd,
	}

	cmd := exec.Command("docker", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	fmt.Println("Running Docker build...")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker build failed: %v", err)
	}

	// Fix ownership of built files (non-critical)
	if uid := os.Getuid(); uid != 0 {
		_ = exec.Command("sh", "-c", fmt.Sprintf("chown -R %d:%d %s 2>/dev/null", uid, os.Getgid(), absBuildDir)).Run()
	}

	fmt.Printf("Docker build successful! Output: %s\n", outputPath)
	return nil
}
