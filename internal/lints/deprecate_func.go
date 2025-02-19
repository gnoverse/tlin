package lints

import (
	"fmt"
	"go/ast"
	"go/token"
	"log"
	"os"

	"github.com/gnolang/tlin/internal/checker"
	tt "github.com/gnolang/tlin/internal/types"

	"gopkg.in/yaml.v3"
)

func register() *checker.DeprecatedFuncChecker {
	chck, err := loadDeprecatedFunctionsFromYAML("internal/lints/data/deprecated_functions.yml")
	if err != nil {
		log.Printf("Warning: Failed to load deprecated functions: %v", err)
		return checker.NewDeprecatedFuncChecker()
	}
	return chck
}

func DetectDeprecatedFunctions(
	filename string,
	node *ast.File,
	fset *token.FileSet,
	severity tt.Severity,
) ([]tt.Issue, error) {
	deprecated := register()

	imports := extractDeprecatedImports(node)
	if len(imports) == 0 {
		return nil, nil
	}

	hasDeprecatedPackage := false
	for imp := range imports {
		if deprecatedPackages[imp] {
			hasDeprecatedPackage = true
			break
		}
	}

	if !hasDeprecatedPackage {
		return nil, nil
	}

	dfuncs, err := deprecated.Check(filename, node, fset)
	if err != nil {
		return nil, err
	}

	issues := make([]tt.Issue, 0, len(dfuncs))
	for _, df := range dfuncs {
		issues = append(issues, tt.Issue{
			Rule:       "deprecated",
			Filename:   filename,
			Start:      df.Start,
			End:        df.End,
			Message:    createDeprecationMessage(df),
			Suggestion: df.Alternative,
			Severity:   severity,
		})
	}

	return issues, nil
}

func createDeprecationMessage(df checker.DeprecatedFunc) string {
	msg := "Use of deprecated function"
	if df.Alternative != "" {
		msg = fmt.Sprintf("%s. please use %s instead.", msg, df.Alternative)
		return msg
	}
	msg = fmt.Sprintf("%s. please remove it.", msg)
	return msg
}

type pkgContainsDeprecatedMap map[string]bool

var deprecatedPackages = pkgContainsDeprecatedMap{
	"std": true,
}

func extractDeprecatedImports(node *ast.File) pkgContainsDeprecatedMap {
	return extractImports(node, func(path string) bool {
		return true
	})
}

func extractImports[T any](node *ast.File, valueFunc func(string) T) map[string]T {
	imports := make(map[string]T)

	for _, imp := range node.Imports {
		path := imp.Path.Value[1 : len(imp.Path.Value)-1]
		imports[path] = valueFunc(path)
	}

	return imports
}

func loadDeprecatedFunctionsFromYAML(filepath string) (*checker.DeprecatedFuncChecker, error) {
	checker := checker.NewDeprecatedFuncChecker()

	data, err := os.ReadFile(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to read YAML file: %w", err)
	}

	var config struct {
		Functions map[string]map[string]string            `yaml:"functions"`
		Methods   map[string]map[string]map[string]string `yaml:"methods"`
	}

	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	// Register functions
	for pkg, funcs := range config.Functions {
		for funcName, alternative := range funcs {
			checker.Register(pkg, funcName, alternative)
		}
	}

	// Register methods
	for pkg, types := range config.Methods {
		for typeName, methods := range types {
			for methodName, alternative := range methods {
				checker.RegisterMethod(pkg, typeName, methodName, alternative)
			}
		}
	}

	return checker, nil
}
