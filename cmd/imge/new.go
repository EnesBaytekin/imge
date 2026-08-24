package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
)

// handleNew implements `imge new <object|component|scene> <name>`. The name may
// include a path relative to the current directory (e.g. `imge new object
// objects/player`), which places the file there and creates the directory if
// needed. It refuses to overwrite an existing file.
func handleNew() {
	if len(os.Args) < 4 {
		log.Fatalf("Usage: imge new <object|component|scene> <name>")
	}

	kind := os.Args[2]
	name := os.Args[3]

	switch kind {
	case "object", "obj":
		newObject(name)
	case "component", "comp":
		newComponent(name)
	case "scene":
		newScene(name)
	default:
		log.Fatalf("Unknown kind %q. Use `imge new object|component|scene <name>`.", kind)
	}
}

// splitName splits a name-or-path into a directory and a basename.
func splitName(namePath string) (dir, base string) {
	dir = filepath.Dir(namePath)
	base = filepath.Base(namePath)
	if base == "." || base == "" || base == string(filepath.Separator) {
		log.Fatalf("Invalid name %q.", namePath)
	}
	return dir, base
}

// trimExt removes a trailing extension (case-insensitive) from a basename, so
// `imge new object player.obj` and `imge new object player` produce the same
// file. A name that is nothing but the extension (".obj") is left untouched.
func trimExt(base, ext string) string {
	if strings.HasSuffix(strings.ToLower(base), ext) {
		if trimmed := base[:len(base)-len(ext)]; trimmed != "" {
			return trimmed
		}
	}
	return base
}

// writeNew writes content to dir/filename, creating dir if needed and refusing to
// overwrite an existing file.
func writeNew(dir, filename, content string) {
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			log.Fatalf("Failed to create directory %q: %v", dir, err)
		}
	}
	full := filepath.Join(dir, filename)
	if _, err := os.Stat(full); err == nil {
		log.Fatalf("Refusing to overwrite existing file %q.", full)
	}
	if err := os.WriteFile(full, []byte(content), 0644); err != nil {
		log.Fatalf("Failed to write %q: %v", full, err)
	}
	fmt.Printf("Created %s\n", full)
}

func newObject(namePath string) {
	dir, base := splitName(namePath)
	base = trimExt(base, ".obj")
	content := strings.ReplaceAll(objectTemplate, "{{NAME}}", base)
	writeNew(dir, fileName(base)+".obj", content)
}

func newScene(namePath string) {
	dir, base := splitName(namePath)
	base = trimExt(base, ".scene")
	content := strings.ReplaceAll(sceneTemplate, "{{NAME}}", base)
	writeNew(dir, fileName(base)+".scene", content)
}

func newComponent(namePath string) {
	dir, base := splitName(namePath)
	base = trimExt(base, ".go")
	structName := pascalCase(base)
	if structName == "" || !isLetter(structName[0]) {
		log.Fatalf("Component name %q is not a valid Go identifier — start it with a letter.", base)
	}
	content := strings.ReplaceAll(componentTemplate, "{{NAME}}", structName)
	writeNew(dir, fileName(base)+".go", content)
}

// pascalCase converts a name into an exported Go identifier. Words are split on
// non-alphanumeric characters and camelCase boundaries, so "enemy", "enemy_brain",
// "enemy-brain", and "EnemyBrain" all become "EnemyBrain".
func pascalCase(name string) string {
	var words []string
	var cur strings.Builder
	flush := func() {
		if cur.Len() > 0 {
			words = append(words, cur.String())
			cur.Reset()
		}
	}
	prevLower := false
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z':
			cur.WriteRune(r)
			prevLower = true
		case r >= 'A' && r <= 'Z':
			if prevLower {
				flush() // camelCase boundary
			}
			cur.WriteRune(r)
			prevLower = false
		case r >= '0' && r <= '9':
			cur.WriteRune(r)
			prevLower = false
		default:
			flush() // separator
			prevLower = false
		}
	}
	flush()

	var out strings.Builder
	for _, w := range words {
		// Words contain only ASCII alphanumerics, so byte slicing is safe.
		out.WriteString(strings.ToUpper(w[:1]))
		out.WriteString(strings.ToLower(w[1:]))
	}
	return out.String()
}

// fileName lowercases a name and folds non-alphanumeric characters into
// underscores, producing a filesystem-safe filename stem.
func fileName(name string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(name) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		} else {
			b.WriteByte('_')
		}
	}
	return strings.Trim(b.String(), "_")
}

func isLetter(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z')
}

const objectTemplate = `{
  "name": "{{NAME}}",
  "depth": 0,
  "ui": false,
  "tags": [],
  "components": []
}
`

// sceneTemplate doubles as the blank template's scenes/main.scene
// (testdata/blank/scenes/main.scene, with {{NAME}} → "main") — keep them in sync.
const sceneTemplate = `{
  "name": "{{NAME}}",
  "background_color": "#000000",
  "camera": {
    "x": 0,
    "y": 0,
    "zoom": 1,
    "smoothing": 0,
    "lock_x": false,
    "lock_y": false
  },
  "objects": []
}
`

const componentTemplate = `package components

import (
	"github.com/EnesBaytekin/imge/core"
)

// {{NAME}} is a custom component.
type {{NAME}} struct {
	core.BaseComponent
}

// Initialize runs once after the object is added to a scene.
func (c *{{NAME}}) Initialize() {}

// Update runs every frame.
func (c *{{NAME}}) Update(ctx *core.Context) {}

// Draw runs every frame, for custom rendering.
func (c *{{NAME}}) Draw(renderer core.Renderer) {}
`
