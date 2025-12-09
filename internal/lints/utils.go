package lints

import "go/ast"

func extractImports[T any](node *ast.File, valueFunc func(string) T) map[string]T {
	imports := make(map[string]T)

	for _, imp := range node.Imports {
		path := imp.Path.Value[1 : len(imp.Path.Value)-1]
		imports[path] = valueFunc(path)
	}

	return imports
}
