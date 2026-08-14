package assetfs

import (
	"io"
	"testing"
	"testing/fstest"
)

// TestOpenResolvesAssetsPrefix verifies that Open reads an asset from an
// embedded filesystem, falling back to the assets/ prefix when the bare path
// isn't present (the same resolution desktop builds do against the OS).
func TestOpenResolvesAssetsPrefix(t *testing.T) {
	fsys := fstest.MapFS{
		"assets/pixel.png": {Data: []byte("PNGDATA")},
	}

	r, err := Open(fsys, "pixel.png")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer r.Close()

	data, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if string(data) != "PNGDATA" {
		t.Fatalf("got %q, want PNGDATA", data)
	}
}

// TestOpenDirectPath verifies a path that already includes assets/ is opened
// directly without double-prefixing.
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
