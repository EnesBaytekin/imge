package main

import (
	"crypto/sha256"
	"encoding/hex"
	"hash"
	"io"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/EnesBaytekin/imge"
	"github.com/EnesBaytekin/imge/build"
)

// handleEditor builds (or reuses a cached build of) the embedded IMGE editor, pointing
// it at a target project. `imge editor <path>` opens <path> in the editor; with no path
// it opens the current directory (if it is a project). The target path is passed to the
// editor via the IMGE_PROJECT env var (which the editor's viewport already reads), and
// the imge binary's own path via IMGE_CLI so the editor's RUN button can shell back out
// to it.
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

	exe, err := os.Executable()
	if err != nil {
		exe, _ = filepath.Abs(os.Args[0])
	}

	// Reuse the cached editor binary when one exists for this build of imge; otherwise
	// build it into a temp dir and install it into the cache for next time.
	out, err := cachedEditorBinary()
	if err != nil {
		// No usable cache (e.g. no user-cache dir): fall back to the pre-cache path of
		// building into a temp dir and running it, removed after the editor exits.
		log.Printf("editor cache unavailable: %v", err)
		var cleanup func()
		out, cleanup, err = buildEditor()
		if err != nil {
			log.Fatalf("Failed to build editor: %v", err)
		}
		defer cleanup()
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

// cachedEditorBinary returns the path of a cached editor binary for this build of imge,
// building and installing it first when it is absent. The cache is keyed by a digest of
// everything the editor is built from — the engine version plus the embedded editor and
// engine source — so a rebuilt imge CLI with any source change gets a fresh binary, and
// an unchanged CLI reuses the last one (skipping the whole extract + `go build`).
func cachedEditorBinary() (string, error) {
	dir, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}

	h := sha256.New()
	io.WriteString(h, "version "+imge.EngineVersion+"\n")
	io.WriteString(h, "goos "+runtime.GOOS+"\n")
	io.WriteString(h, "goarch "+runtime.GOARCH+"\n")
	if err := hashEmbed(h, imge.EditorTemplate); err != nil {
		return "", err
	}
	if err := hashEmbed(h, imge.EngineSource); err != nil {
		return "", err
	}
	key := hex.EncodeToString(h.Sum(nil))[:16]

	name := "imge-editor"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	cached := filepath.Join(dir, "imge", "editor", key, name)

	if _, err := os.Stat(cached); err == nil {
		return cached, nil
	}

	// Cache miss: build into a temp dir, install the binary into the cache, then drop
	// the temp dir — the built editor lives on in the cache rather than a temp dir.
	if err := os.MkdirAll(filepath.Dir(cached), 0o755); err != nil {
		return "", err
	}
	out, cleanup, err := buildEditor()
	if err != nil {
		return "", err
	}
	defer cleanup()
	if err := installFile(out, cached); err != nil {
		return "", err
	}
	return cached, nil
}

// buildEditor extracts the embedded editor project into a fresh temp dir, builds it, and
// returns the built binary path plus a cleanup func that removes the temp dir. The
// cleanup must be run after the binary is no longer needed.
func buildEditor() (out string, cleanup func(), err error) {
	tmp, err := os.MkdirTemp("", "imge-editor-")
	if err != nil {
		return "", nil, err
	}
	cleanup = func() { os.RemoveAll(tmp) }

	if err := imge.ExtractEditor(tmp); err != nil {
		cleanup()
		return "", nil, err
	}

	b := &build.Builder{ProjectDir: tmp, Target: build.TargetDesktop}
	out, err = b.Build()
	if err != nil {
		cleanup()
		return "", nil, err
	}
	return out, cleanup, nil
}

// installFile copies src to dst atomically: it writes to a sibling temp file and renames
// it over dst, so a crashed build never leaves a half-written cached binary.
func installFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	// os.Create makes the temp file 0644, so remember the source's permissions and
	// re-apply them below — a cached editor binary that lost its exec bit can't run.
	info, err := in.Stat()
	if err != nil {
		return err
	}

	tmp := dst + ".tmp"
	out, err := os.Create(tmp)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		os.Remove(tmp)
		return err
	}
	if err := out.Close(); err != nil {
		os.Remove(tmp)
		return err
	}
	if err := os.Chmod(tmp, info.Mode().Perm()); err != nil {
		os.Remove(tmp)
		return err
	}
	// Rename over any existing destination (a best-effort remove first, since Windows
	// does not replace a live file on rename).
	os.Remove(dst)
	if err := os.Rename(tmp, dst); err != nil {
		os.Remove(tmp)
		return err
	}
	return nil
}

// hashEmbed walks an embedded FS and feeds every path and file's bytes into h, so the
// digest changes whenever the embedded content changes.
func hashEmbed(h hash.Hash, fsys fs.FS) error {
	return fs.WalkDir(fsys, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		io.WriteString(h, path)
		h.Write([]byte{0})
		if d.IsDir() {
			return nil
		}
		data, err := fs.ReadFile(fsys, path)
		if err != nil {
			return err
		}
		h.Write(data)
		h.Write([]byte{0})
		return nil
	})
}
