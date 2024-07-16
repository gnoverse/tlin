package internal

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

type SymbolTable struct {
	symbols map[string]string // symbol name -> file path
}

func BuildSymbolTable(rootDir string) (*SymbolTable, error) {
	st := &SymbolTable{symbols: make(map[string]string)}
	err := filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && (strings.HasSuffix(path, ".go") || strings.HasSuffix(path, ".gno")) {
			if err := st.parseFile(path); err != nil {
				return err
			}
		}
		return err
	})
	return st, err
}

func (st *SymbolTable) parseFile(filepath string) error {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filepath, nil, parser.AllErrors)
	if err != nil {
		return err
	}

	ast.Inspect(node, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.TypeSpec:
			st.symbols[x.Name.Name] = filepath
		case *ast.FuncDecl:
			st.symbols[x.Name.Name] = filepath
		case *ast.ValueSpec:
			for _, ident := range x.Names {
				st.symbols[ident.Name] = filepath
			}
		}
		return true
	})

	return nil
}

func (st *SymbolTable) IsDefined(symbol string) bool {
	_, exists := st.symbols[symbol]
	return exists
}

func (st *SymbolTable) GetSymbolPath(symbol string) (string, bool) {
	path, exists := st.symbols[symbol]
	return path, exists
}
