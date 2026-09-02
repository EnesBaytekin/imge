package main

import (
	"log"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/EnesBaytekin/imge"
	"github.com/EnesBaytekin/imge/build"
)

// handleEditor builds and launches the embedded IMGE editor, pointing it at a target
// project. `imge editor <path>` opens <path> in the editor; with no path it opens the
// current directory (if it is a project). The target path is passed to the editor via
// the IMGE_PROJECT env var (which the editor's viewport already reads), and the imge
// binary's own path via IMGE_CLI so the editor's RUN button can shell back out to it.
func handleEditor() {
	args := os.Args[2:]
	target := "."
	if len(args) > 0 {
		target = args[0]
	}

	abs, err := filepath.Abs(target)
	if err != nil {
		log.Fatalf("Failed to resolve path %q: %v", target, err)
	}
	if _, err := build.FindGameFile(abs); err != nil {
		log.Fatalf("Not an IMGE project: %v (run `imge init` first)", err)
	}

	// Build the embedded editor into a fresh temp dir, then run it pointed at abs.
	tmp, err := os.MkdirTemp("", "imge-editor-")
	if err != nil {
		log.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmp)

	if err := imge.ExtractEditor(tmp); err != nil {
		log.Fatalf("Failed to extract editor: %v", err)
	}

	b := &build.Builder{ProjectDir: tmp, Target: build.TargetDesktop}
	out, err := b.Build()
	if err != nil {
		log.Fatalf("Failed to build editor: %v", err)
	}

	exe, err := os.Executable()
	if err != nil {
		exe, _ = filepath.Abs(os.Args[0])
	}

	cmd := exec.Command(out)
	cmd.Env = append(os.Environ(), "IMGE_PROJECT="+abs, "IMGE_CLI="+exe)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	if err := cmd.Run(); err != nil {
		log.Fatalf("Editor exited with error: %v", err)
	}
}
