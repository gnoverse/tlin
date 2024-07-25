package lints

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"

	tt "github.com/gnoswap-labs/lint/internal/types"
)

const (
	GNO_PKG_PREFIX  = "gno.land/"
	GNO_STD_PACKAGE = "std"
)

// Dependency represents an imported package and its usage status.
type Dependency struct {
	ImportPath string
	IsGno      bool
	IsUsed     bool
	IsIgnored  bool // alias with `_` should be ignored
}

type (
	// Dependencies is a map of import paths to their Dependency information.
	Dependencies map[string]*Dependency

	// FileMap is a map of filenames to their parsed AST representation.
	FileMap map[string]*ast.File
)

// Package represents a Go/Gno package with its name and files.
type Package struct {
	Name  string
	Files FileMap
}

// DetectGnoPackageImports analyzes the given file for Gno package imports and returns any issues found.
func DetectGnoPackageImports(filename string) ([]tt.Issue, error) {
	dir := filepath.Dir(filename)

	pkg, deps, err := analyzePackage(dir)
	if err != nil {
		return nil, fmt.Errorf("error analyzing package: %w", err)
	}

	issues := runGnoPackageLinter(pkg, deps)

	for i := range issues {
		issues[i].Filename = filename
	}

	return issues, nil
}

// parses all gno files and collect their imports and usage.
func analyzePackage(dir string) (*Package, Dependencies, error) {
	pkg := &Package{
		Files: make(FileMap),
	}
	deps := make(Dependencies)

	files, err := filepath.Glob(filepath.Join(dir, "*.gno"))
	if err != nil {
		return nil, nil, err
	}

	// 1. Parse all file contents and collect dependencies
	for _, file := range files {
		f, err := parseFile(file)
		if err != nil {
			return nil, nil, err
		}

		pkg.Files[file] = f
		if pkg.Name == "" {
			pkg.Name = f.Name.Name
		}

		for _, imp := range f.Imports {
			impPath := strings.Trim(imp.Path.Value, `"`)
			if _, exists := deps[impPath]; !exists {
				deps[impPath] = &Dependency{
					ImportPath: impPath,
					IsGno:      isGnoPackage(impPath),
					IsUsed:     false,
					IsIgnored:  imp.Name != nil && imp.Name.Name == "_",
				}
			}
		}
	}

	// 2. Determine which dependencies are used
	for _, file := range pkg.Files {
		ast.Inspect(file, func(n ast.Node) bool {
			switch x := n.(type) {
			case *ast.SelectorExpr:
				if ident, ok := x.X.(*ast.Ident); ok {
					for _, imp := range file.Imports {
						if imp.Name != nil && imp.Name.Name == ident.Name {
							deps[strings.Trim(imp.Path.Value, `"`)].IsUsed = true
						} else if lastPart := getLastPart(strings.Trim(imp.Path.Value, `"`)); lastPart == ident.Name {
							deps[strings.Trim(imp.Path.Value, `"`)].IsUsed = true
						}
					}
				}
			}
			return true
		})
	}

	return pkg, deps, nil
}

func runGnoPackageLinter(pkg *Package, deps Dependencies) []tt.Issue {
	var issues []tt.Issue

	for _, file := range pkg.Files {
		ast.Inspect(file, func(n ast.Node) bool {
			switch x := n.(type) {
			case *ast.SelectorExpr:
				// check unused imports
				if ident, ok := x.X.(*ast.Ident); ok {
					if dep, exists := deps[ident.Name]; exists {
						dep.IsUsed = true
					}
				}
			}
			return true
		})
	}

	for impPath, dep := range deps {
		if !dep.IsUsed && !dep.IsIgnored {
			issue := tt.Issue{
				Rule:    "unused-import",
				Message: fmt.Sprintf("unused import: %s", impPath),
			}
			issues = append(issues, issue)
		}
	}

	return issues
}

func isGnoPackage(importPath string) bool {
	return strings.HasPrefix(importPath, GNO_PKG_PREFIX) || importPath == GNO_STD_PACKAGE
}

func parseFile(filename string) (*ast.File, error) {
	content, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	fset := token.NewFileSet()
	return parser.ParseFile(fset, filename, content, parser.ParseComments)
}

func getLastPart(path string) string {
	parts := strings.Split(path, "/")
	return parts[len(parts)-1]
}
