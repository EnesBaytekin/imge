package assetfs

import (
	"io"
	"testing"
	"testing/fstest"
)

// TestOpenDirectPath verifies a root-relative path (here one that includes
// assets/) is opened exactly as given.
func TestOpenDirectPath(t *testing.T) {
	fsys := fstest.MapFS{
		"assets/pixel.png": {Data: []byte("DIRECT")},
	}

	r, err := Open(fsys, "assets/pixel.png")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer r.Close()

	data, _ := io.ReadAll(r)
	if string(data) != "DIRECT" {
		t.Fatalf("got %q, want DIRECT", data)
	}
}

// TestOpenNoAssetsFallback verifies that a bare filename does NOT fall back to
// assets/: "pixel.png" is resolved only at the project root, so when the file
// lives in assets/ it must be referenced as "assets/pixel.png".
func TestOpenNoAssetsFallback(t *testing.T) {
	fsys := fstest.MapFS{
		"assets/pixel.png": {Data: []byte("PNGDATA")},
	}

	if _, err := Open(fsys, "pixel.png"); err == nil {
		t.Fatal("expected bare path pixel.png to fail (no assets/ fallback), got success")
	}
}
