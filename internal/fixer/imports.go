package fixer

import (
	"bytes"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"

	tt "github.com/gnolang/tlin/internal/types"
	"golang.org/x/tools/go/ast/astutil"
)

// EnsureImports adds missing imports to the source code.
// It parses the source, checks which imports are missing, adds them,
// and returns the modified source.
func EnsureImports(src []byte, imports []string) ([]byte, error) {
	if len(imports) == 0 {
		return src, nil
	}

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "", src, parser.ParseComments)
	if err != nil {
		return src, err
	}

	modified := false
	for _, importPath := range imports {
		if !hasImport(file, importPath) {
			astutil.AddImport(fset, file, importPath)
			modified = true
		}
	}

	if !modified {
		return src, nil
	}

	var buf bytes.Buffer
	if err := format.Node(&buf, fset, file); err != nil {
		return src, err
	}

	return buf.Bytes(), nil
}

// hasImport checks if the file already has the specified import.
func hasImport(file *ast.File, importPath string) bool {
	for _, imp := range file.Imports {
		// imp.Path.Value includes quotes, e.g., `"errors"`
		path := imp.Path.Value
		if len(path) >= 2 {
			path = path[1 : len(path)-1] // remove quotes
		}
		if path == importPath {
			return true
		}
	}
	return false
}

// CollectRequiredImports gathers all unique required imports from issues.
func CollectRequiredImports(issues []tt.Issue) []string {
	seen := make(map[string]bool)
	var imports []string

	for _, issue := range issues {
		for _, imp := range issue.RequiredImports {
			if !seen[imp] {
				seen[imp] = true
				imports = append(imports, imp)
			}
		}
	}

	return imports
}
