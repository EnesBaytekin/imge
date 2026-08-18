package build

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFindComponentTypeName(t *testing.T) {
	cases := []struct {
		name    string
		src     string
		want    string
		wantErr bool
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
			want: "Foo",
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
			name: "no component struct",
			src: `package components

type NotAComponent struct {
	X int
}
`,
			wantErr: true,
		},
		{
			name: "unexported struct is ignored",
			src: `package components

import "github.com/EnesBaytekin/imge/core"

type foo struct {
	core.BaseComponent
}
`,
			wantErr: true,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := findComponentTypeName([]byte(c.src))
			if c.wantErr {
				if err == nil {
					t.Fatalf("expected error, got %q", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
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
		{kind: "components/player.go", typeName: "PlayerComponent", source: "components/player.go"},
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
		`core.RegisterComponent("components/player.go", func() core.Component { return &PlayerComponent{} })`,
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
		{"@Velocity", "Velocity"},
		{"@Wander", "Wander"},
		{"components/player.go", "PlayerComponent"},
	} {
		if got[want.kind] != want.typeName {
			t.Errorf("kind %q = %q, want %q (all: %v)", want.kind, got[want.kind], want.typeName, got)
		}
	}
}
