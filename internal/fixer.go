package internal

import (
	"bytes"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"strings"
)

// TODO: Must flattening the nested unnecessary if-else blocks.

// improveCode refactors the input source code and returns the formatted version.
func improveCode(src []byte) (string, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "", src, parser.ParseComments)
	if err != nil {
		return "", err
	}

	err = refactorAST(file)
	if err != nil {
		return "", err
	}

	return formatSource(fset, file)
}

// refactorAST processes the AST to modify specific patterns.
func refactorAST(file *ast.File) error {
	ast.Inspect(file, func(n ast.Node) bool {
		ifStmt, ok := n.(*ast.IfStmt)
		if !ok || ifStmt.Else == nil {
			return true
		}

		blockStmt, ok := ifStmt.Else.(*ast.BlockStmt)
		if !ok || len(ifStmt.Body.List) == 0 {
			return true
		}

		_, isReturn := ifStmt.Body.List[len(ifStmt.Body.List)-1].(*ast.ReturnStmt)
		if !isReturn {
			return true
		}

		mergeElseIntoIf(file, ifStmt, blockStmt)
		ifStmt.Else = nil

		return true
	})
	return nil
}

// mergeElseIntoIf merges the statements of an 'else' block into the enclosing function body.
func mergeElseIntoIf(file *ast.File, ifStmt *ast.IfStmt, blockStmt *ast.BlockStmt) {
	for _, list := range file.Decls {
		decl, ok := list.(*ast.FuncDecl)
		if !ok {
			continue
		}
		for i, stmt := range decl.Body.List {
			if ifStmt != stmt {
				continue
			}
			decl.Body.List = append(decl.Body.List[:i+1], append(blockStmt.List, decl.Body.List[i+1:]...)...)
			break
		}
	}
}

// formatSource formats the AST back to source code.
func formatSource(fset *token.FileSet, file *ast.File) (string, error) {
	var buf bytes.Buffer
	err := format.Node(&buf, fset, file)
	if err != nil {
		return "", err
	}
	return strings.TrimRight(buf.String(), "\n"), nil
}
