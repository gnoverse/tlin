package lints

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strings"

	tt "github.com/gnolang/tlin/internal/types"
)

const (
	GNO_PKG_PREFIX  = "gno.land/"
	GNO_STD_PACKAGE = "std"
)

type Dependency struct {
	ImportPath string
	IsGno      bool
	IsUsed     bool
	IsIgnored  bool // aliased as `_`
}

type Dependencies map[string]*Dependency

func DetectGnoPackageImports(filename string, severity tt.Severity) ([]tt.Issue, error) {
	file, deps, err := analyzeFile(filename)
	if err != nil {
		return nil, fmt.Errorf("error analyzing file: %w", err)
	}

	issues := runGnoPackageLinter(file, deps, severity)

	for i := range issues {
		issues[i].Filename = filename
	}

	return issues, nil
}

func analyzeFile(filename string) (*ast.File, Dependencies, error) {
	content, err := os.ReadFile(filename)
	if err != nil {
		return nil, nil, err
	}

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filename, content, parser.ParseComments)
	if err != nil {
		return nil, nil, err
	}

	deps := make(Dependencies)
	for _, imp := range file.Imports {
		impPath := strings.Trim(imp.Path.Value, `"`)
		deps[impPath] = &Dependency{
			ImportPath: impPath,
			IsGno:      isGnoPackage(impPath),
			IsUsed:     false,
			IsIgnored:  imp.Name != nil && imp.Name.Name == "_",
		}
	}

	// Determine which dependencies are used in this file
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

	return file, deps, nil
}

func runGnoPackageLinter(_ *ast.File, deps Dependencies, severity tt.Severity) []tt.Issue {
	var issues []tt.Issue

	for impPath, dep := range deps {
		if !dep.IsUsed && !dep.IsIgnored {
			issue := tt.Issue{
				Rule:     "unused-import",
				Message:  fmt.Sprintf("unused import: %s", impPath),
				Severity: severity,
			}
			issues = append(issues, issue)
		}
	}

	return issues
}

func isGnoPackage(importPath string) bool {
	return strings.HasPrefix(importPath, GNO_PKG_PREFIX) || importPath == GNO_STD_PACKAGE
}

func getLastPart(path string) string {
	parts := strings.Split(path, "/")
	return parts[len(parts)-1]
}
