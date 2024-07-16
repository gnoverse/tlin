package internal

import (
	"encoding/gob"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

func buildSymbolTable(rootDir string) (*symbolTable, error) {
	st := &symbolTable{symbols: make(map[string]string)}
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

type symbolTable struct {
	symbols     map[string]string // symbol name -> file path
	lastUpdated map[string]time.Time
	mutex       sync.RWMutex
	cacheFile   string
}

func newSymbolTable(cacheFile string) *symbolTable {
	return &symbolTable{
		symbols:     make(map[string]string),
		lastUpdated: make(map[string]time.Time),
		cacheFile:   cacheFile,
	}
}

func (st *symbolTable) loadCache() error {
	st.mutex.Lock()
	defer st.mutex.Unlock()

	file, err := os.Open(st.cacheFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // ignore if no cache file exists
		}
		return err
	}
	defer file.Close()

	decoder := gob.NewDecoder(file)
	return decoder.Decode(&st.symbols)
}

func (st *symbolTable) saveCache() error {
	st.mutex.RLock()
	defer st.mutex.RUnlock()

	file, err := os.Create(st.cacheFile)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := gob.NewEncoder(file)
	return encoder.Encode(st.symbols)
}

func (st *symbolTable) updateSymbols(rootDir string) error {
	err := filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && (filepath.Ext(path) == ".go" || filepath.Ext(path) == ".gno") {
			lastMod := info.ModTime()
			if lastUpdated, ok := st.lastUpdated[path]; !ok || lastMod.After(lastUpdated) {
				if err := st.parseFile(path); err != nil {
					return err
				}
				st.lastUpdated[path] = lastMod
			}
		}
		return nil
	})

	if err == nil {
		err = st.saveCache()
	}

	return err
}

func (st *symbolTable) parseFile(filepath string) error {
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

func (st *symbolTable) isDefined(symbol string) bool {
	_, exists := st.symbols[symbol]
	return exists
}

func (st *symbolTable) getSymbolPath(symbol string) (string, bool) {
	path, exists := st.symbols[symbol]
	return path, exists
}
