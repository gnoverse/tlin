package internal

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

type SymbolInfo struct {
	FilePath string
	Package string
	Type string // "func", "type", "var", "const"
	Exported bool
}

type SymbolTable struct {
	symbols map[string]SymbolInfo
	mutex sync.RWMutex
}

func New() *SymbolTable {
	return &SymbolTable{
		symbols: make(map[string]SymbolInfo),
	}
}

func BuildSymbolTable(rootDir string) (*SymbolTable, error) {
	st := New()
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

	packageName := node.Name.Name

	ast.Inspect(node, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.TypeSpec:
			st.add(x.Name.Name, SymbolInfo{
				FilePath: filepath,
				Package: packageName,
				Type: "type",
				Exported: ast.IsExported(x.Name.Name),
			})
		case *ast.FuncDecl:
			st.add(x.Name.Name, SymbolInfo{
				FilePath: filepath,
				Package: packageName,
				Type: "func",
				Exported: ast.IsExported(x.Name.Name),
			})
		case *ast.ValueSpec:
			for _, ident := range x.Names {
				st.add(ident.Name, SymbolInfo{
					FilePath: filepath,
					Package: packageName,
					Type: "var",
					Exported: ast.IsExported(ident.Name),
				})
			}
		}
		return true
	})

	return nil
}

func (st *SymbolTable) add(name string, info SymbolInfo) {
	st.mutex.Lock()
	defer st.mutex.Unlock()
	st.symbols[name] = info
}

func (st *SymbolTable) IsDefined(symbol string) bool {
	st.mutex.RLock()
	defer st.mutex.RUnlock()

	_, exists := st.symbols[symbol]
	return exists
}

func (st *SymbolTable) GetSymbolInfo(sym string) (SymbolInfo, bool) {
	st.mutex.RLock()
	defer st.mutex.RUnlock()

	info, exists := st.symbols[sym]
	return info, exists
}

func (st *SymbolTable) GetAllSymbols() map[string]SymbolInfo {
	st.mutex.RLock()
	defer st.mutex.RUnlock()

	copy := make(map[string]SymbolInfo)
	for k, v := range st.symbols {
		copy[k] = v
	}
	return copy
}
