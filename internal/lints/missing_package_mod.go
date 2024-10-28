package lints

import (
	"bufio"
	"fmt"
	"go/ast"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	tt "github.com/gnolang/tlin/internal/types"
)

func DetectMissingModPackage(filename string, node *ast.File, fset *token.FileSet, severity tt.Severity) ([]tt.Issue, error) {
	dir := filepath.Dir(filename)
	modFile := filepath.Join(dir, "gno.mod")

	requiredPackages, err := extractGnoImports(node)
	if err != nil {
		return nil, fmt.Errorf("failed to extract gno imports: %w", err)
	}

	declaredPackages, err := extractDeclaredPackages(modFile)
	if err != nil {
		return nil, fmt.Errorf("failed to extract declared packages: %w", err)
	}

	issues := make([]tt.Issue, 0)

	var unusedPackages []string
	for pkg := range declaredPackages {
		if _, ok := requiredPackages[pkg]; !ok {
			unusedPackages = append(unusedPackages, pkg)
		}
	}

	if len(unusedPackages) > 0 {
		issue := tt.Issue{
			Rule:     "gno-mod-tidy",
			Filename: modFile,
			Start:    token.Position{Filename: modFile},
			End:      token.Position{Filename: modFile},
			Message:  fmt.Sprintf("packages %s are declared in gno.mod file but not imported.\nrun `gno mod tidy`", strings.Join(unusedPackages, ", ")),
			Severity: severity,
		}
		issues = append(issues, issue)
	}

	for pkg := range requiredPackages {
		if !declaredPackages[pkg] {
			issue := tt.Issue{
				Rule:     "gno-mod-tidy",
				Filename: modFile,
				Start:    token.Position{Filename: modFile},
				End:      token.Position{Filename: modFile},
				Message:  fmt.Sprintf("package %s is imported but not declared in gno.mod file. please consider to remove.\nrun `gno mod tidy`", pkg),
				Severity: severity,
			}
			issues = append(issues, issue)
		}
	}

	return issues, nil
}

func extractGnoImports(node *ast.File) (map[string]bool, error) {
	imports := make(map[string]bool)
	for _, imp := range node.Imports {
		if imp.Path != nil {
			path, err := strconv.Unquote(imp.Path.Value)
			if err != nil {
				return nil, fmt.Errorf("failed to unquote import path: %w", err)
			}
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
