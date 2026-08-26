package build

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/EnesBaytekin/imge"
)

// componentKind associates a registered component kind with its Go type name, the
// source it came from (a built-in file or a project-relative path), and the kinds
// the component declares it needs via Requires().
type componentKind struct {
	kind     string
	typeName string
	source   string
	requires []string
}

// findComponentTypeName parses a component source file and returns the name of
// the exported struct type that embeds core.BaseComponent.
func findComponentTypeName(src []byte) (string, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "component.go", src, 0)
	if err != nil {
		return "", err
	}

	var found []string
	for _, decl := range f.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok || gen.Tok != token.TYPE {
			continue
		}
		for _, spec := range gen.Specs {
			ts, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}
			st, ok := ts.Type.(*ast.StructType)
			if !ok {
				continue
			}
			if !ts.Name.IsExported() {
				continue
			}
			if embedsBaseComponent(st) {
				found = append(found, ts.Name.Name)
			}
		}
	}

	switch len(found) {
	case 1:
		return found[0], nil
	case 0:
		return "", fmt.Errorf("no exported struct embeds core.BaseComponent")
	default:
		return "", fmt.Errorf("multiple component structs found: %v", found)
	}
}

// findRequires parses a component source file and returns the string literals in
// its `Requires() []string` method, if any. Components use this to declare the
// kinds they depend on (e.g. @Animator requires "@Sprite").
func findRequires(src []byte) []string {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "component.go", src, 0)
	if err != nil {
		return nil
	}

	var requires []string
	ast.Inspect(f, func(n ast.Node) bool {
		fn, ok := n.(*ast.FuncDecl)
		if !ok || fn.Name.Name != "Requires" || fn.Body == nil {
			return true
		}
		ast.Inspect(fn.Body, func(n2 ast.Node) bool {
			lit, ok := n2.(*ast.BasicLit)
			if !ok || lit.Kind != token.STRING {
				return true
			}
			if s, err := strconv.Unquote(lit.Value); err == nil {
				requires = append(requires, s)
			}
			return true
		})
		return true
	})
	return requires
}

// embedsBaseComponent reports whether a struct embeds a component base type
// (core.BaseComponent, or core.BaseUIComponent which itself embeds BaseComponent).
func embedsBaseComponent(st *ast.StructType) bool {
	for _, field := range st.Fields.List {
		if len(field.Names) != 0 {
			continue // named field, not an embedding
		}
		switch t := field.Type.(type) {
		case *ast.SelectorExpr:
			if isComponentBase(t.Sel.Name) {
				return true
			}
		case *ast.Ident:
			if isComponentBase(t.Name) {
				return true
			}
		}
	}
	return false
}

// isComponentBase reports whether an embedded type name marks a component.
func isComponentBase(name string) bool {
	return name == "BaseComponent" || name == "BaseUIComponent"
}

// componentKinds derives the (kind, typeName, source) triples for every component
// — built-ins from the embedded engine, customs from the project. Used by both
// validation and registry codegen so the two can never disagree on a kind.
func (g *Generator) componentKinds() ([]componentKind, error) {
	var kinds []componentKind

	// Built-in components from the embedded engine source.
	entries, err := fs.ReadDir(imge.EngineSource, "engine/components")
	if err != nil {
		return nil, fmt.Errorf("failed to read built-in components: %w", err)
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		data, err := fs.ReadFile(imge.EngineSource, "engine/components/"+entry.Name())
		if err != nil {
			return nil, err
		}
		typeName, err := findComponentTypeName(data)
		if err != nil {
			return nil, fmt.Errorf("built-in component %s: %w", entry.Name(), err)
		}
		kinds = append(kinds, componentKind{
			kind:     builtinKind(typeName),
			typeName: typeName,
			source:   "built-in:" + entry.Name(),
			requires: findRequires(data),
		})
	}

	// User components, from disk. Their kind is the Go struct type name: every
	// component — built-in and custom — is merged into one `components` package,
	// so type names are already unique. Referencing the type (not the file path)
	// keeps the JSON stable if the file is later moved or renamed.
	for _, compFile := range g.Analysis.ComponentFiles {
		src, err := os.ReadFile(filepath.Join(g.Analysis.ProjectDir, compFile))
		if err != nil {
			return nil, fmt.Errorf("failed to read component %s: %w", compFile, err)
		}
		typeName, err := findComponentTypeName(src)
		if err != nil {
			return nil, fmt.Errorf("component file %s: %w", compFile, err)
		}
		kinds = append(kinds, componentKind{
			kind:     typeName,
			typeName: typeName,
			source:   compFile,
			requires: findRequires(src),
		})
	}

	return kinds, nil
}

// builtinKind maps a built-in component type name to its kind identifier
// (e.g. HitboxComponent -> @Hitbox).
func builtinKind(typeName string) string {
	return "@" + strings.TrimSuffix(typeName, "Component")
}

// generateRegistry writes components/registry.go, which auto-registers every
// component (built-in and custom) in a single init() via core.RegisterComponent.
func (g *Generator) generateRegistry(kinds []componentKind) error {
	var b strings.Builder
	b.WriteString("// GENERATED CODE - DO NOT EDIT\n")
	b.WriteString("// Auto-registers every component in the project.\n")
	b.WriteString("package components\n\n")
	b.WriteString("import \"github.com/EnesBaytekin/imge/core\"\n\n")
	b.WriteString("func init() {\n")
	for _, k := range kinds {
		fmt.Fprintf(&b, "\tcore.RegisterComponent(%q, func() core.Component { return &%s{} })\n", k.kind, k.typeName)
	}
	b.WriteString("}\n")

	return os.WriteFile(filepath.Join(g.BuildDir, "components", "registry.go"), []byte(b.String()), 0644)
}
