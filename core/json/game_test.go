package json

import (
	"os"
	"path/filepath"
	"testing"
)

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

	c, err := LoadGameConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if c.Window.PixelPerUnit != 1 {
		t.Fatalf("default pixel_per_unit = %d, want 1", c.Window.PixelPerUnit)
	}
}

func TestPixelPerUnitExplicit(t *testing.T) {
	path := writeGameFile(t, `{
	  "name": "My Game",
	  "format_version": 1,
	  "window": { "width": 320, "height": 180, "pixel_per_unit": 4 }
	}`)

	c, err := LoadGameConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if c.Window.PixelPerUnit != 4 {
		t.Fatalf("pixel_per_unit = %d, want 4", c.Window.PixelPerUnit)
	}
}

func TestSmoothShapesDefaultsToFalse(t *testing.T) {
	path := writeGameFile(t, `{
	  "name": "My Game",
	  "format_version": 1,
	  "window": { "width": 320, "height": 180 }
	}`)

	c, err := LoadGameConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if c.Window.SmoothShapes {
		t.Fatalf("default smooth_shapes = true, want false")
	}
}

func TestSmoothShapesExplicit(t *testing.T) {
	path := writeGameFile(t, `{
	  "name": "My Game",
	  "format_version": 1,
	  "window": { "width": 320, "height": 180, "smooth_shapes": true }
	}`)

	c, err := LoadGameConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if !c.Window.SmoothShapes {
		t.Fatalf("smooth_shapes = false, want true")
	}
}

func TestSaveGameConfigRoundTrip(t *testing.T) {
	c := DefaultGameConfig()
	path := filepath.Join(t.TempDir(), "game.imge")
	if err := SaveGameConfig(c, path); err != nil {
		t.Fatal(err)
	}

	got, err := LoadGameConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != c.Name || got.Window.Width != 640 || got.Window.PixelPerUnit != 1 || got.Game.TargetFPS != 60 {
		t.Fatalf("round-trip mismatch: %+v", got)
	}
}
