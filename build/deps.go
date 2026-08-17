package build

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// checkLinuxNativeDeps verifies that the host has the C toolchain and system
// headers Ebitengine needs for a native Linux desktop build: GLFW links against
// X11/GL and oto (audio) links against ALSA via pkg-config. It returns nil when
// everything is present, or a clear, actionable error naming what's missing.
// Web and Windows cross-builds are pure Go and never reach this.
func checkLinuxNativeDeps() error {
	var missing []string

	if _, err := exec.LookPath("gcc"); err != nil {
		if _, err := exec.LookPath("cc"); err != nil {
			missing = append(missing, "a C compiler (gcc)")
		}
	}
	if _, err := exec.LookPath("pkg-config"); err != nil {
		missing = append(missing, "pkg-config")
	}
	if !hasX11Headers() {
		missing = append(missing, "X11 development headers (libx11-dev)")
	}

	if len(missing) == 0 {
		return nil
	}

	return fmt.Errorf(
		"native Linux build is missing system dependencies: %s.\n"+
			"Install them with:\n"+
			"    sudo apt update && sudo apt install -y build-essential pkg-config libgl1-mesa-dev libx11-dev libxcursor-dev libxrandr-dev libxinerama-dev libxi-dev libxxf86vm-dev libasound2-dev\n"+
			"then run `imge build` again.",
		strings.Join(missing, ", "),
	)
}

// hasX11Headers reports whether the X11 development headers are available,
// either via the standard system include path or via pkg-config.
func hasX11Headers() bool {
	if _, err := os.Stat("/usr/include/X11/Xlib.h"); err == nil {
		return true
	}
	if _, err := exec.LookPath("pkg-config"); err == nil {
		if err := exec.Command("pkg-config", "--exists", "x11").Run(); err == nil {
			return true
		}
	}
	return false
}
