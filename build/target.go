package build

import (
	"fmt"
	"runtime"
	"strings"
)

// slugify converts a game name into a filesystem-safe identifier: lowercase
// ASCII alphanumerics, with everything else folded into dashes. Non-ASCII
// characters are dropped so the resulting filename is safe on Windows too.
func slugify(s string) string {
	s = strings.ToLower(s)
	var b strings.Builder
	lastDash := false
	for _, r := range s {
		switch {
		case (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9'):
			b.WriteRune(r)
			lastDash = false
		case !lastDash && b.Len() > 0:
			b.WriteByte('-')
			lastDash = true
		}
	}
	out := strings.Trim(b.String(), "-")
	if out == "" {
		out = "game"
	}
	return out
}

// Platform identifies a desktop build target (OS + architecture).
type Platform struct {
	GOOS   string
	GOARCH string
}

// HostPlatform returns the OS/architecture this CLI is running on.
func HostPlatform() Platform {
	return Platform{GOOS: runtime.GOOS, GOARCH: runtime.GOARCH}
}

// canBuildDesktop reports whether IMGE can build a desktop executable for the
// given target from the given host, using only the host's native C toolchain.
// Ebitengine desktop needs Cgo (GLFW/OpenGL) on Linux and macOS, while Windows
// is pure Go. So: native builds work, Windows cross-compiles from anywhere, and
// macOS builds both of its architectures natively (clang produces universal
// binaries). Everything else needs a C cross toolchain we don't assume exists.
func canBuildDesktop(hostGOOS, hostGOARCH, goos, goarch string) bool {
	if goos == hostGOOS && goarch == hostGOARCH {
		return true
	}
	if goos == "windows" && (goarch == "amd64" || goarch == "arm64") {
		return true
	}
	if goos == "darwin" && hostGOOS == "darwin" && (goarch == "amd64" || goarch == "arm64") {
		return true
	}
	return false
}

// ValidateTarget reports whether IMGE can build a desktop executable for the
// given target from the current host, returning a helpful error if not.
func ValidateTarget(goos, goarch string) error {
	if canBuildDesktop(runtime.GOOS, runtime.GOARCH, goos, goarch) {
		return nil
	}
	switch {
	case goos == "windows":
		return fmt.Errorf("windows target only supports amd64 and arm64 (got %s)", goarch)
	case goos == "darwin":
		return fmt.Errorf("cannot cross-compile to macOS from %s: Ebitengine uses Cgo (GLFW/Metal), which needs osxcross + the Apple SDK. Build on a Mac or use macOS CI", runtime.GOOS)
	default:
		return fmt.Errorf("cannot cross-compile to %s/%s from %s/%s: Ebitengine desktop uses Cgo (GLFW), which needs a C cross toolchain + sysroot", goos, goarch, runtime.GOOS, runtime.GOARCH)
	}
}
