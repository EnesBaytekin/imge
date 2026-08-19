package build

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateFileReferences(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "scenes"), 0755); err != nil {
		t.Fatal(err)
	}
	scene := `{
  "name": "main",
  "objects": [
    { "name": "ok", "components": [
      { "kind": "@Sprite", "name": "sprite", "args": { "texture": "assets/ok.png" } }
    ] },
    { "name": "missing", "components": [
      { "kind": "@Sprite", "name": "sprite", "args": { "texture": "assets/nope.png" } }
    ] },
    { "file": "objects/ghost.obj", "transform": { "position": { "x": 0, "y": 0 } } }
  ]
}`
	if err := os.WriteFile(filepath.Join(dir, "scenes", "main.scene"), []byte(scene), 0644); err != nil {
		t.Fatal(err)
	}

	g := &Generator{
		Analysis: &ProjectAnalysis{
			ProjectDir: dir,
			AssetFiles: []string{"assets/ok.png"},
			SceneFiles: []string{"scenes/main.scene"},
		},
	}

	err := g.validateFileReferences()
	if err == nil {
		t.Fatal("expected an error for the missing references")
	}
	if !strings.Contains(err.Error(), "assets/nope.png") {
		t.Errorf("error should mention the missing texture, got: %v", err)
	}
	if !strings.Contains(err.Error(), "objects/ghost.obj") {
		t.Errorf("error should mention the missing template, got: %v", err)
	}
	if strings.Contains(err.Error(), "assets/ok.png") {
		t.Errorf("error should not flag the existing file, got: %v", err)
	}
}

func TestValidateFileReferencesOk(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "scenes"), 0755); err != nil {
		t.Fatal(err)
	}
	scene := `{
  "name": "main",
  "objects": [
    { "name": "ok", "components": [
      { "kind": "@Sprite", "name": "sprite", "args": { "texture": "assets/ok.png" } },
      { "kind": "@Sound", "name": "sound", "args": { "sound": "assets/blip.wav" } }
    ] }
  ]
}`
	if err := os.WriteFile(filepath.Join(dir, "scenes", "main.scene"), []byte(scene), 0644); err != nil {
		t.Fatal(err)
	}

	g := &Generator{
		Analysis: &ProjectAnalysis{
			ProjectDir: dir,
			AssetFiles: []string{"assets/ok.png", "assets/blip.wav"},
			SceneFiles: []string{"scenes/main.scene"},
		},
	}

	if err := g.validateFileReferences(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
