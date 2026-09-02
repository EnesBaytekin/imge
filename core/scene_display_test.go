package core

import (
	"os"
	"path/filepath"
	"testing"
)

// displayTestComponent records Initialize/Update calls so tests can assert that
// LoadForDisplay runs Initialize once and never runs Update.
type displayTestComponent struct {
	BaseComponent
	initialized int
	updated     int
}

func (c *displayTestComponent) Initialize()     { c.initialized++ }
func (c *displayTestComponent) Update(*Context) { c.updated++ }

const displayTestKind = "display.test"

func init() {
	RegisterComponent(displayTestKind, func() Component { return &displayTestComponent{} })
}

func TestInitializeForRenderRunsInitializeNotUpdate(t *testing.T) {
	scene := NewScene("test")
	obj := NewObject("obj")
	comp := &displayTestComponent{}
	comp.SetName("c")
	_ = obj.AddComponent(comp)
	_ = scene.AddObject(obj)

	scene.InitializeForRender()

	if comp.initialized != 1 {
		t.Fatalf("Initialize ran %d times, want 1", comp.initialized)
	}
	if comp.updated != 0 {
		t.Fatalf("Update ran %d times, want 0", comp.updated)
	}

	// A second call must be a no-op (one-time initialization).
	scene.InitializeForRender()
	if comp.initialized != 1 {
		t.Fatalf("Initialize ran %d times after second call, want 1", comp.initialized)
	}
}

// TestLoadForDisplaySkipsUnknownKind verifies the lenient load path keeps known
// components, drops unknown kinds, and still runs Initialize on what survived.
func TestLoadForDisplaySkipsUnknownKind(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "s.scene")
	json := `{
		"name": "t",
		"objects": [
			{ "name": "mixed", "components": [
				{ "kind": "display.test", "name": "keep" },
				{ "kind": "display.unknown", "name": "drop" }
			] }
		]
	}`
	if err := os.WriteFile(path, []byte(json), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}

	scene := NewScene("s")
	if err := scene.LoadForDisplay(path); err != nil {
		t.Fatalf("LoadForDisplay: %v", err)
	}

	obj := scene.GetObjectByName("mixed")
	if obj == nil {
		t.Fatal("mixed object missing")
	}
	if len(obj.Components) != 1 {
		t.Fatalf("mixed object has %d components, want 1 (unknown skipped)", len(obj.Components))
	}
	comp := obj.Components["keep"].(*displayTestComponent)
	if comp.initialized != 1 {
		t.Fatalf("surviving component Initialize ran %d times, want 1", comp.initialized)
	}
}

// TestLoadFromJSONStrictFailsOnUnknownKind guards that normal (strict) loading
// still fails fast on an unknown kind — leniency is opt-in via LoadForDisplay.
func TestLoadFromJSONStrictFailsOnUnknownKind(t *testing.T) {
	scene := NewScene("t")
	err := scene.LoadFromJSON([]byte(`{
		"name": "t",
		"objects": [
			{ "name": "x", "components": [ { "kind": "display.unknown", "name": "c" } ] }
		]
	}`))
	if err == nil {
		t.Fatal("strict load accepted an unknown component kind")
	}
}
