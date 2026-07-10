package lints

import (
	"go/ast"
	"go/parser"
	"go/token"
)

// ParseFile is a test-only helper that returns a parsed AST and its
// FileSet from either a path (when content is nil) or in-memory
// bytes. It lives in a _test.go file so the production engine, which
// uses rule.LoadSource for the same job, doesn't pull it into the
// shipped binary.
func ParseFile(filename string, content []byte) (*ast.File, *token.FileSet, error) {
	fset := token.NewFileSet()
	var node *ast.File
	var err error
	if content == nil {
		node, err = parser.ParseFile(fset, filename, nil, parser.ParseComments)
	} else {
		node, err = parser.ParseFile(fset, filename, content, parser.ParseComments)
	}
	if err != nil {
		return nil, nil, err
	}
	return node, fset, nil
}
