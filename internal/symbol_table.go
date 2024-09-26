package internal

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strings"
)

type SymbolType int

const (
	Function SymbolType = iota
	Type
	Variable
	Method
)

type SymbolInfo struct {
	Name       string
	Type       SymbolType
	Package    string
	FilePath   string
	Interfaces []string // list of interfaces the symbol implements
}

func newSymbolInfo(name string, typ SymbolType, pkg, filePath string) SymbolInfo {
	return SymbolInfo{
		Name:     name,
		Type:     typ,
		Package:  pkg,
		FilePath: filePath,
	}
}

type SymbolTable struct {
	symbols map[string]SymbolInfo
}

func BuildSymbolTable(rootDir string, source []byte) (*SymbolTable, error) {
	st := &SymbolTable{symbols: make(map[string]SymbolInfo)}

	if source != nil {
		st.parseFile("", source)
	} else {
		err := filepath.WalkDir(rootDir, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if !d.IsDir() && (strings.HasSuffix(path, ".go") || strings.HasSuffix(path, ".gno")) {
				if err := st.parseFile(path, nil); err != nil {
					return err
				}
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}

	return st, nil
}

func (st *SymbolTable) parseFile(path string, source []byte) error {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, path, source, parser.ParseComments)
	if err != nil {
		return err
	}

	packageName := node.Name.Name
	ast.Inspect(node, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.FuncDecl:
			symType := Function
			if x.Recv != nil {
				symType = Method
			}
			st.addSymbol(newSymbolInfo(x.Name.Name, symType, packageName, path))
		case *ast.TypeSpec:
			st.addSymbol(newSymbolInfo(x.Name.Name, Type, packageName, path))
		case *ast.ValueSpec:
			for _, ident := range x.Names {
				st.addSymbol(newSymbolInfo(ident.Name, Variable, packageName, path))
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

func (st *SymbolTable) GetSymbolInfo(symbol string) (SymbolInfo, bool) {
	info, exists := st.symbols[symbol]
	return info, exists
}

func (st *SymbolTable) addSymbol(info SymbolInfo) {
	key := info.Package + "." + info.Name
	st.symbols[key] = info
}

func (st *SymbolTable) AddInterfaceImplementation(typeName, interfaceName string) {
	if info, exists := st.symbols[typeName]; exists {
		info.Interfaces = append(info.Interfaces, interfaceName)
		st.symbols[typeName] = info
	}
}
