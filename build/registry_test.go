package build

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFindComponentTypeName(t *testing.T) {
	cases := []struct {
		name          string
		src           string
		want          string
		wantComponent bool
		wantErr       bool
	}{
		{
			name: "single component struct",
			src: `package components

import "github.com/EnesBaytekin/imge/core"

type Foo struct {
	core.BaseComponent
	Speed float64
}
`,
			want:          "Foo",
			wantComponent: true,
		},
		{
			name: "ui component struct",
			src: `package components

import "github.com/EnesBaytekin/imge/core"

type Button struct {
	core.BaseUIComponent
	Text string
}
`,
			want:          "Button",
			wantComponent: true,
		},
		{
			name: "multiple component structs",
			src: `package components

import "github.com/EnesBaytekin/imge/core"

type Foo struct {
	core.BaseComponent
}
type Bar struct {
	core.BaseComponent
}
`,
			wantErr: true,
		},
		{
			name: "helper file with no component struct",
			src: `package components

const maxRows = 20

func helper(x int) int { return x }
`,
			wantComponent: false,
		},
		{
			name: "unexported struct is ignored",
			src: `package components

import "github.com/EnesBaytekin/imge/core"

type foo struct {
	core.BaseComponent
}
`,
			wantComponent: false,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, isComponent, err := findComponentTypeName([]byte(c.src))
			if c.wantErr {
				if err == nil {
					t.Fatalf("expected error, got %q", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if isComponent != c.wantComponent {
				t.Errorf("isComponent = %v, want %v", isComponent, c.wantComponent)
			}
			if got != c.want {
				t.Errorf("got %q, want %q", got, c.want)
			}
		})
	}
}

func TestBuiltinKind(t *testing.T) {
	cases := map[string]string{
		"Collider":     "@Collider",
		"Mover":        "@Mover",
		"Sprite":       "@Sprite",
		"Sound":        "@Sound",
		"FooComponent": "@Foo", // a trailing "Component" suffix is still stripped
	}
	for in, want := range cases {
		if got := builtinKind(in); got != want {
			t.Errorf("builtinKind(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestGenerateRegistry(t *testing.T) {
	dir := t.TempDir()
	compDir := filepath.Join(dir, "components")
	if err := os.MkdirAll(compDir, 0755); err != nil {
		t.Fatal(err)
	}

	g := &Generator{BuildDir: dir}
	kinds := []componentKind{
		{kind: "@Collider", typeName: "Collider", source: "built-in:collider.go"},
		{kind: "PlayerComponent", typeName: "PlayerComponent", source: "components/player.go"},
	}
	if err := g.generateRegistry(kinds); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(filepath.Join(compDir, "registry.go"))
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)
	for _, want := range []string{
		"package components",
		`core.RegisterComponent("@Collider", func() core.Component { return &Collider{} })`,
		`core.RegisterComponent("PlayerComponent", func() core.Component { return &PlayerComponent{} })`,
	} {
		if !strings.Contains(content, want) {
			t.Errorf("registry missing %q\n---\n%s", want, content)
		}
	}
}

// TestComponentKinds verifies that componentKinds discovers both built-ins (from
// the embedded engine source) and custom components (from disk), deriving the
// right kind identifier for each.
func TestComponentKinds(t *testing.T) {
	dir := t.TempDir()
	compDir := filepath.Join(dir, "components")
	if err := os.MkdirAll(compDir, 0755); err != nil {
		t.Fatal(err)
	}
	playerSrc := `package components

import "github.com/EnesBaytekin/imge/core"

type PlayerComponent struct {
	core.BaseComponent
	Speed float64
}
`
	if err := os.WriteFile(filepath.Join(compDir, "player.go"), []byte(playerSrc), 0644); err != nil {
		t.Fatal(err)
	}

	g := &Generator{
		Analysis: &ProjectAnalysis{
			ProjectDir:     dir,
			ComponentFiles: []string{"components/player.go"},
		},
	}

	kinds, err := g.componentKinds()
	if err != nil {
		t.Fatal(err)
	}

	got := make(map[string]string) // kind -> typeName
	for _, k := range kinds {
		got[k.kind] = k.typeName
	}

	for _, want := range []struct{ kind, typeName string }{
		{"@Animator", "Animator"},
		{"@Bounce", "Bounce"},
		{"@Chase", "Chase"},
		{"@Collider", "Collider"},
		{"@Damage", "Damage"},
		{"@Follow", "Follow"},
		{"@Friction", "Friction"},
		{"@Gravity", "Gravity"},
		{"@Health", "Health"},
		{"@Mover", "Mover"},
		{"@Patrol", "Patrol"},
		{"@PlayerController", "PlayerController"},
		{"@Sound", "Sound"},
		{"@Spin", "Spin"},
		{"@Sprite", "Sprite"},
		{"@TimedDespawn", "TimedDespawn"},
		{"@Trigger", "Trigger"},
		{"@Velocity", "Velocity"},
		{"@Wander", "Wander"},
		{"@Button", "ButtonComponent"},
		{"@CheckBox", "CheckBoxComponent"},
		{"@ColorPicker", "ColorPickerComponent"},
		{"@ComboBox", "ComboBoxComponent"},
		{"@Container", "ContainerComponent"},
		{"@Label", "LabelComponent"},
		{"@List", "ListComponent"},
		{"@Panel", "PanelComponent"},
		{"@Slider", "SliderComponent"},
		{"@TextInput", "TextInputComponent"},
		{"@UIManager", "UIManagerComponent"},
		{"PlayerComponent", "PlayerComponent"},
	} {
		if got[want.kind] != want.typeName {
			t.Errorf("kind %q = %q, want %q (all: %v)", want.kind, got[want.kind], want.typeName, got)
		}
	}
}
