package core

import "testing"

// styleTestComponent exercises the generic style mechanism with a minimal
// component whose JSON-tagged fields mirror how built-in UI components work.
type styleTestComponent struct {
	BaseComponent
	Color string `json:"color"`
	Width int    `json:"width"`
}

func TestCreateComponentStyle(t *testing.T) {
	RegisterComponent("StyleTest", func() Component { return &styleTestComponent{} })
	defer UnregisterComponent("StyleTest")
	defer func() { registeredStyles = nil }()

	sheet := `{
	  "StyleTest": {
	    "primary": { "color": "#ff0000", "width": 10 },
	    "danger":  { "color": "#00ff00" }
	  }
	}`
	if err := LoadStyles([]byte(sheet)); err != nil {
		t.Fatalf("LoadStyles: %v", err)
	}

	// A style supplies defaults; inline args override them.
	c, err := CreateComponent("StyleTest", map[string]any{"style": "primary", "width": 42})
	if err != nil {
		t.Fatalf("CreateComponent: %v", err)
	}
	sc := c.(*styleTestComponent)
	if sc.Color != "#ff0000" || sc.Width != 42 {
		t.Fatalf("merge = color %q width %d, want #ff0000 / 42", sc.Color, sc.Width)
	}

	// An unknown style is an error, not a silent no-op.
	if _, err := CreateComponent("StyleTest", map[string]any{"style": "nope"}); err == nil {
		t.Fatal("expected an error for an unknown style")
	}

	// No style: zero values, no error.
	c2, err := CreateComponent("StyleTest", map[string]any{"color": "#0000ff"})
	if err != nil {
		t.Fatalf("CreateComponent (no style): %v", err)
	}
	if c2.(*styleTestComponent).Width != 0 {
		t.Fatalf("no-style width = %d, want 0", c2.(*styleTestComponent).Width)
	}
}
