package internal

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	_ "os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
)

type SymbolTable struct {
	symbols map[string]string // symbol name -> file path
	mu      sync.RWMutex
}

func BuildSymbolTable(rootDir string) (*SymbolTable, error) {
	st := &SymbolTable{symbols: make(map[string]string)}
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
			funcName := x.Name.Name
			fullName := packageName + "." + funcName
			st.addSymbol(fullName, path)

		case *ast.TypeSpec:
			typeName := x.Name.Name
			fullName := packageName + "." + typeName
			st.addSymbol(fullName, path)
		case *ast.ValueSpec:
			for _, ident := range x.Names {
				varName := ident.Name
				fullName := packageName + "." + varName
				st.addSymbol(fullName, path)
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

func (st *SymbolTable) GetSymbolPath(symbol string) (string, bool) {
	st.mu.RLock()
	defer st.mu.RUnlock()

	path, exists := st.symbols[symbol]
	return path, exists
}

func (st *SymbolTable) addSymbol(key, value string) {
	st.mu.Lock()
	defer st.mu.Unlock()

	st.symbols[key] = value
}
