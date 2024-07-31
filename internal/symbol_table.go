package internal

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
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
	mu      sync.RWMutex
}

func BuildSymbolTable(rootDir string) (*SymbolTable, error) {
	st := &SymbolTable{symbols: make(map[string]SymbolInfo)}
	errChan := make(chan error, 1)
	fileChan := make(chan string, 100)

	var wg sync.WaitGroup
	for i := 0; i < runtime.NumCPU(); i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for path := range fileChan {
				if err := st.parseFile(path); err != nil {
					select {
					case errChan <- err:
					default:
					}
					return
				}
			}
		}()
	}

	go func() {
		err := filepath.WalkDir(rootDir, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if !d.IsDir() && (strings.HasSuffix(path, ".go") || strings.HasSuffix(path, ".gno")) {
				fileChan <- path
			}
			return nil
		})
		close(fileChan)
		if err != nil {
			errChan <- err
		}
	}()

	wg.Wait()

	select {
	case err := <-errChan:
		return nil, err
	default:
		return st, nil
	}
}

func (st *SymbolTable) parseFile(path string) error {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
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
	st.mu.RLock()
	defer st.mu.RUnlock()

	_, exists := st.symbols[symbol]
	return exists
}

func (st *SymbolTable) GetSymbolInfo(symbol string) (SymbolInfo, bool) {
	st.mu.RLock()
	defer st.mu.RUnlock()

	info, exists := st.symbols[symbol]
	return info, exists
}

func (st *SymbolTable) addSymbol(info SymbolInfo) {
	st.mu.Lock()
	defer st.mu.Unlock()

	key := info.Package + "." + info.Name
	st.symbols[key] = info
}

func (st *SymbolTable) AddInterfaceImplementation(typeName, interfaceName string) {
	st.mu.Lock()
	defer st.mu.Unlock()

	if info, exists := st.symbols[typeName]; exists {
		info.Interfaces = append(info.Interfaces, interfaceName)
		st.symbols[typeName] = info
	}
}
