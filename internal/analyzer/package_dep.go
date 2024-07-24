package analyzer

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"
	"sync"
)

type (
	Strength int
	Matrix   map[string]map[string]Strength
)

type DependencyAnalyzer struct {
	SymbolTable *SymbolTable
	Imports     map[string][]string // key: package path, value: imported packages
	UsedSymbols Matrix              // key: package path, value: map of used symbols
	mutex       sync.RWMutex
}

func NewDependencyAnalyzer(symbolTable *SymbolTable) *DependencyAnalyzer {
	return &DependencyAnalyzer{
		SymbolTable: symbolTable,
		Imports:     make(map[string][]string),
		UsedSymbols: make(Matrix),
	}
}

func (da *DependencyAnalyzer) AnalyzeFiles(filePaths []string) error {
	var wg sync.WaitGroup
	errChan := make(chan error, len(filePaths))

	for _, filepath := range filePaths {
		wg.Add(1)
		go func(path string) {
			defer wg.Done()
			if err := da.AnalyzeFile(path); err != nil {
				errChan <- err
			}
		}(filepath)
	}

	wg.Wait()
	close(errChan)

	for err := range errChan {
		if err != nil {
			return err
		}
	}

	return nil
}

func (da *DependencyAnalyzer) AnalyzeFile(filePath string) error {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
	if err != nil {
		return err
	}

	packagePath := filepath.Dir(filePath)

	da.mutex.Lock()
	if _, exists := da.Imports[packagePath]; !exists {
		da.Imports[packagePath] = make([]string, 0)
	}
	if _, exists := da.UsedSymbols[packagePath]; !exists {
		da.UsedSymbols[packagePath] = make(map[string]Strength)
	}
	da.mutex.Unlock()

	// Analyze imports
	for _, imp := range node.Imports {
		importPath := strings.Trim(imp.Path.Value, "\"")
		da.mutex.Lock()
		da.Imports[packagePath] = append(da.Imports[packagePath], importPath)
		da.mutex.Unlock()
	}

	// Analyze used symbols
	ast.Inspect(node, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.Ident:
			if info, exists := da.SymbolTable.GetSymbolInfo(x.Name); exists && info.Package != packagePath {
				da.mutex.Lock()
				da.UsedSymbols[packagePath][info.Package]++
				da.mutex.Unlock()
			}
		}
		return true
	})

	return nil
}

func (da *DependencyAnalyzer) BuildDependencyMatrix() Matrix {
	da.mutex.RLock()
	defer da.mutex.RUnlock()

	matrix := make(Matrix)

	for pkg := range da.Imports {
		matrix[pkg] = make(map[string]Strength)
		for _, imp := range da.Imports[pkg] {
			matrix[pkg][imp] = 1
		}
		for usedPkg, strength := range da.UsedSymbols[pkg] {
			matrix[pkg][usedPkg] += strength
		}
	}

	return matrix
}

func (da *DependencyAnalyzer) DetectCyclicDependencies(matrix Matrix) [][]string {
    var cycles [][]string
    visited := make(map[string]bool)
    path := make([]string, 0)
    cycleSet := make(map[string]bool)

    var dfs func(pkg string)
    dfs = func(pkg string) {
        visited[pkg] = true
        path = append(path, pkg)

        for dep := range matrix[pkg] {
            if !visited[dep] {
                dfs(dep)
            } else if index := indexOf(path, dep); index != -1 {
                cycle := path[index:]
                normalizedCycle := normalizeCycle(cycle)
                cycleKey := strings.Join(normalizedCycle, ",")
                if !cycleSet[cycleKey] {
                    cycles = append(cycles, normalizedCycle)
                    cycleSet[cycleKey] = true
                }
            }
        }

        path = path[:len(path)-1]
        visited[pkg] = false
    }

    for pkg := range matrix {
        if !visited[pkg] {
            dfs(pkg)
        }
    }

    return cycles
}

func normalizeCycle(cycle []string) []string {
    if len(cycle) == 0 {
        return cycle
    }
    minIndex := 0
    for i, pkg := range cycle {
        if pkg < cycle[minIndex] {
            minIndex = i
        }
    }
    normalizedCycle := make([]string, len(cycle))
    for i := 0; i < len(cycle); i++ {
        normalizedCycle[i] = cycle[(minIndex+i)%len(cycle)]
    }
    return normalizedCycle
}

func (da *DependencyAnalyzer) GetDirectDependencies(pkg string) []string {
	da.mutex.RLock()
	defer da.mutex.RUnlock()

	deps := make([]string, 0)
	for dep := range da.UsedSymbols[pkg] {
		deps = append(deps, dep)
	}
	return deps
}

func (da *DependencyAnalyzer) GetAllDependencies(pkg string) map[string]bool {
	matrix := da.BuildDependencyMatrix()
	allDeps := make(map[string]bool)

	var dfs func(p string)
	dfs = func(p string) {
		for dep := range matrix[p] {
			if !allDeps[dep] {
				allDeps[dep] = true
				dfs(dep)
			}
		}
	}

	dfs(pkg)
	return allDeps
}

func (da *DependencyAnalyzer) GetDependencyStrength(from, to string) Strength {
	da.mutex.RLock()
	defer da.mutex.RUnlock()

	return da.UsedSymbols[from][to]
}

func indexOf(slice []string, item string) int {
	for i, v := range slice {
		if v == item {
			return i
		}
	}
	return -1
}
