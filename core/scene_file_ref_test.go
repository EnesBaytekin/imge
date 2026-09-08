package core

import (
	"os"
	"path/filepath"
	"testing"
)

// fileRefTestComponent is a trivial registered component used by the file-reference
// round-trip tests: it carries one json-tagged field so the tests can assert that a
// component loaded from a .obj template is preserved on the object but omitted from
// the scene's file-reference serialization (the definition lives in the .obj).
type fileRefTestComponent struct {
	BaseComponent
	Speed float64 `json:"speed"`
}

const fileRefTestKind = "fileRef.test"

func init() {
	RegisterComponent(fileRefTestKind, func() Component { return &fileRefTestComponent{} })
}

// TestFileReferencedObjectRoundTripsAsReference verifies the file-reference contract:
// an object referenced via SceneObject.File records its provenance, and the scene
// serializes it back as a reference (not a flattened inline definition), while the
// object's own config still carries the full template definition.
func TestFileReferencedObjectRoundTripsAsReference(t *testing.T) {
	dir := t.TempDir()

	objJSON := `{
		"name": "Crate",
		"depth": 2,
		"tags": ["crate"],
		"components": [
			{ "kind": "fileRef.test", "name": "c", "args": { "speed": 3 } }
		]
	}`
	if err := os.MkdirAll(filepath.Join(dir, "objects"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "objects", "crate.obj"), []byte(objJSON), 0o644); err != nil {
		t.Fatalf("write obj: %v", err)
	}

	sceneJSON := `{
		"name": "main",
		"objects": [
			{ "file": "objects/crate.obj", "transform": { "position": { "x": 260, "y": 512 } } }
		]
	}`
	if err := os.WriteFile(filepath.Join(dir, "main.scene"), []byte(sceneJSON), 0o644); err != nil {
		t.Fatalf("write scene: %v", err)
	}

	scene := NewScene("s")
	if err := scene.LoadFromFS(os.DirFS(dir), "main.scene"); err != nil {
		t.Fatalf("LoadFromFS: %v", err)
	}

	obj := scene.GetObjectByName("Crate")
	if obj == nil {
		t.Fatal("crate object missing")
	}
	if obj.File != "objects/crate.obj" {
		t.Fatalf("File = %q, want objects/crate.obj", obj.File)
	}
	if obj.Depth != 2 {
		t.Fatalf("Depth = %v, want 2", obj.Depth)
	}
	if _, ok := obj.Components["c"]; !ok {
		t.Fatal("component loaded from template is missing")
	}

	// Round-trip: the object must serialize as a file reference, not inline.
	cfg := scene.ToJSONConfig()
	if len(cfg.Objects) != 1 {
		t.Fatalf("objects = %d, want 1", len(cfg.Objects))
	}
	so := cfg.Objects[0]
	if so.File != "objects/crate.obj" {
		t.Fatalf("serialized File = %q, want objects/crate.obj", so.File)
	}
	if so.Name != "" {
		t.Fatalf("serialized Name = %q, want empty (definition lives in .obj)", so.Name)
	}
	if len(so.Components) != 0 {
		t.Fatalf("serialized Components = %d, want 0 (definition lives in .obj)", len(so.Components))
	}
	if so.Transform == nil || so.Transform.Position.X != 260 {
		t.Fatalf("transform override not preserved: %+v", so.Transform)
	}

	// The object's own config (the make-object / write-through path) still carries
	// the full template definition including the component.
	oc := obj.ToJSONConfig()
	if oc.Name != "Crate" {
		t.Fatalf("object config name = %q, want Crate", oc.Name)
	}
	if len(oc.Components) != 1 {
		t.Fatalf("object config components = %d, want 1", len(oc.Components))
	}
}

// TestInlineObjectRoundTripsInline guards that the inline path is unchanged: an
// inline object has no file provenance and still serializes its full definition.
func TestInlineObjectRoundTripsInline(t *testing.T) {
	scene := NewScene("s")
	if err := scene.LoadFromJSON([]byte(`{
		"name": "main",
		"objects": [
			{ "name": "inline", "depth": 1, "components": [ { "kind": "fileRef.test", "name": "c", "args": { "speed": 5 } } ] }
		]
	}`)); err != nil {
		t.Fatalf("LoadFromJSON: %v", err)
	}

	obj := scene.GetObjectByName("inline")
	if obj == nil {
		t.Fatal("inline object missing")
	}
	if obj.File != "" {
		t.Fatalf("File = %q, want empty", obj.File)
	}

	cfg := scene.ToJSONConfig()
	so := cfg.Objects[0]
	if so.File != "" {
		t.Fatalf("inline serialized with File %q, want empty", so.File)
	}
	if so.Name != "inline" {
		t.Fatalf("inline Name = %q, want inline", so.Name)
	}
	if len(so.Components) != 1 {
		t.Fatalf("inline components = %d, want 1", len(so.Components))
	}
}
