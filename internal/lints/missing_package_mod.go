package lints

import (
	"bufio"
	"fmt"
	"go/ast"
	"go/token"
	"os"
	"path/filepath"
	"strings"

	tt "github.com/gnoswap-labs/tlin/internal/types"
)

func DetectMissingModPackage(filename string, node *ast.File, fset *token.FileSet) ([]tt.Issue, error) {
	if !strings.HasSuffix(filename, ".gno") {
		return nil, nil
	}

	dir := filepath.Dir(filename)
	modFile := filepath.Join(dir, "gno.mod")

	requiredPackages, err := extractImports(node)
	if err != nil {
		return nil, fmt.Errorf("failed to extract imports: %w", err)
	}

	declaredPackages, err := extractDeclaredPackages(modFile)
	if err != nil {
		return nil, fmt.Errorf("failed to extract declared packages: %w", err)
	}

	var issues []tt.Issue
	for pkg := range requiredPackages {
		if !declaredPackages[pkg] {
			issue := tt.Issue{
				Rule:     "missing-package",
				Filename: modFile,
				Start:    token.Position{Filename: modFile},
				End:      token.Position{Filename: modFile},
				Message:  fmt.Sprintf("Package %s is imported but not declared in gno.mod file.\nRun `gno mod tidy`", pkg),
			}
			issues = append(issues, issue)
		}
	}

	return issues, nil
}

func extractImports(node *ast.File) (map[string]bool, error) {
	imports := make(map[string]bool)
	for _, imp := range node.Imports {
		if imp.Path != nil {
			path := strings.Trim(imp.Path.Value, "\"")
			if strings.HasPrefix(path, "gno.land/p/") || strings.HasPrefix(path, "gno.land/r/") {
				imports[path] = true
			}
		}
	}
	return imports, nil
}

func extractDeclaredPackages(modFile string) (map[string]bool, error) {
	file, err := os.Open(modFile)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	packages := make(map[string]bool)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "gno.land/p/") || strings.HasPrefix(line, "gno.land/r/") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				packages[parts[0]] = true
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return packages, nil
}
