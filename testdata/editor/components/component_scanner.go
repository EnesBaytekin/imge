package components

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// scanProjectComponents returns the sorted struct names of every custom component in
// the target project: exported struct types in projectDir/components/*.go that embed
// core.BaseComponent (or core.BaseUIComponent, which itself embeds BaseComponent).
//
// It replicates the build tool's AST scan (build/registry.go: findComponentTypeName /
// embedsBaseComponent / isComponentBase), which is not linked into the editor binary.
// The struct name IS the component kind for a user component — the build registers
// user components under their type name, not their file path (build/registry.go), so
// the names returned here are exactly what to hand to addComponentTo and what a
// GenericComponent stores to round-trip through Save.
//
// It is a plain helper file: it declares no component struct, so the build tool copies
// it verbatim and its codegen passes over it (it contributes no component kind).
func scanProjectComponents(projectDir string) []string {
	if projectDir == "" {
		return nil
	}
	compDir := filepath.Join(projectDir, "components")
	entries, err := os.ReadDir(compDir)
	if err != nil {
		return nil // no components dir (or unreadable): just no custom components
	}

	var names []string
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		src, err := os.ReadFile(filepath.Join(compDir, entry.Name()))
		if err != nil {
			continue
		}
		names = append(names, componentStructNames(src)...)
	}
	sort.Strings(names)
	return names
}

// componentStructNames parses one component source file and returns the names of the
// exported struct types that embed a component base type. A helper file (free
// functions/consts/vars only) returns nil.
func componentStructNames(src []byte) []string {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "component.go", src, 0)
	if err != nil {
		return nil
	}

	var names []string
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
			if !ok || !ts.Name.IsExported() {
				continue
			}
			if embedsComponentBase(st) {
				names = append(names, ts.Name.Name)
			}
		}
	}
	return names
}

// embedsComponentBase reports whether a struct embeds a component base type
// (core.BaseComponent, or core.BaseUIComponent which itself embeds BaseComponent).
func embedsComponentBase(st *ast.StructType) bool {
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
