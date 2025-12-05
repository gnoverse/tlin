package lints

import (
	"go/ast"
	"strings"
)

// BuildImportAliasMap returns a mapping from import name (explicit alias or default)
// to the full import path. Aliases using blank or dot imports are skipped.
func BuildImportAliasMap(file *ast.File) map[string]string {
	aliases := make(map[string]string)

	for _, imp := range file.Imports {
		if imp.Name != nil && (imp.Name.Name == "_" || imp.Name.Name == ".") {
			continue
		}

		path := strings.Trim(imp.Path.Value, `"`)
		name := defaultImportName(path)

		if imp.Name != nil && imp.Name.Name != "" {
			name = imp.Name.Name
		}

		aliases[name] = path
	}

	return aliases
}

func defaultImportName(path string) string {
	parts := strings.Split(path, "/")
	return parts[len(parts)-1]
}
