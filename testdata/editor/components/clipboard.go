package components

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

// writeClipboard puts text on the system clipboard. The editor is a desktop app, so it
// shells out to the platform's clipboard tool (the same way RUN shells out to the imge
// CLI) rather than pulling in a cgo clipboard library. On Linux it prefers the Wayland
// tool when a Wayland session is active, then falls back to the X11 tools; macOS and
// Windows use their native one-liners.
func writeClipboard(text string) error {
	switch runtime.GOOS {
	case "darwin":
		return runClipboardArgs("pbcopy", nil, text)
	case "windows":
		return runClipboardArgs("clip", nil, text)
	}

	// Linux: honor the session type first, then fall back through the common tools.
	if os.Getenv("WAYLAND_DISPLAY") != "" {
		if err := runClipboardArgs("wl-copy", nil, text); err == nil {
			return nil
		}
	}
	for _, tool := range []struct {
		name string
		args []string
	}{
		{"xclip", []string{"-selection", "clipboard", "-i"}},
		{"xsel", []string{"--clipboard", "--input"}},
		{"wl-copy", nil},
	} {
		if _, err := exec.LookPath(tool.name); err == nil {
			if err := runClipboardArgs(tool.name, tool.args, text); err == nil {
				return nil
			}
		}
	}
	return fmt.Errorf("no clipboard tool found (tried wl-copy, xclip, xsel, pbcopy, clip)")
}

// runClipboardArgs runs a clipboard tool, feeding it text on stdin. The nil
// Stdout/Stderr connect to the null device, so a failed tool doesn't spew into the
// terminal that launched the editor.
func runClipboardArgs(tool string, args []string, text string) error {
	cmd := exec.Command(tool, args...)
	cmd.Stdin = strings.NewReader(text)
	return cmd.Run()
}
