package build

import (
	"os"
	"path/filepath"
	"testing"
)

// writeGameFile writes a game.imge to a fresh temp dir and returns its path.
func writeGameFile(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "game.imge")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestPixelPerUnitDefaultsToOne(t *testing.T) {
	path := writeGameFile(t, `{
	  "name": "My Game",
	  "format_version": 1,
	  "window": { "width": 320, "height": 180 }
	}`)

	var a ProjectAnalysis
	if err := a.loadGameConfig(path); err != nil {
		t.Fatal(err)
	}
	if got := a.GameConfig.Window.PixelPerUnit; got != 1 {
		t.Fatalf("default pixel_per_unit = %d, want 1", got)
	}
}

func TestPixelPerUnitExplicit(t *testing.T) {
	path := writeGameFile(t, `{
	  "name": "My Game",
	  "format_version": 1,
	  "window": { "width": 320, "height": 180, "pixel_per_unit": 4 }
	}`)

	var a ProjectAnalysis
	if err := a.loadGameConfig(path); err != nil {
		t.Fatal(err)
	}
	if got := a.GameConfig.Window.PixelPerUnit; got != 4 {
		t.Fatalf("pixel_per_unit = %d, want 4", got)
	}
}
