package lints

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"regexp"
	"strings"

	"github.com/gnolang/tlin/internal/rule"
	tt "github.com/gnolang/tlin/internal/types"
)

// versionSuffix matches a trailing path segment of the form v0, v1, …
// gno import paths use this convention (see gnolang/gno#5220), so the
// real package name is the segment preceding it, not "vN".
var versionSuffix = regexp.MustCompile(`^v\d+$`)

func init() {
	rule.Register(unusedPackageRule{})
}

type unusedPackageRule struct{}

func (unusedPackageRule) Name() string                 { return "unused-package" }
func (unusedPackageRule) DefaultSeverity() tt.Severity { return tt.SeverityWarning }

func (unusedPackageRule) Check(ctx *rule.AnalysisContext) ([]tt.Issue, error) {
	return DetectGnoPackageImports(ctx)
}

const (
	GNO_PKG_PREFIX  = "gno.land/"
	GNO_STD_PACKAGE = "std"
)

type Dependency struct {
	ImportPath string
	IsGno      bool
	IsUsed     bool
	IsIgnored  bool // aliased as `_`
	Line       token.Pos
	Column     token.Pos
}

type Dependencies map[string]*Dependency

func DetectGnoPackageImports(ctx *rule.AnalysisContext) ([]tt.Issue, error) {
	deps := extractDependencies(ctx.File)
	return runGnoPackageLinter(ctx, deps), nil
}

// analyzeFile reads + parses the file from disk and returns the AST
// alongside its computed dependencies. Kept for tests; production
// callers go through DetectGnoPackageImports which uses ctx.File and
// avoids the second os.ReadFile + parse.
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

	return file, extractDependencies(file), nil
}

func extractDependencies(file *ast.File) Dependencies {
	deps := make(Dependencies)
	for _, imp := range file.Imports {
		impPath := strings.Trim(imp.Path.Value, `"`)
		deps[impPath] = &Dependency{
			ImportPath: impPath,
			IsGno:      isGnoPackage(impPath),
			IsUsed:     false,
			IsIgnored:  imp.Name != nil && imp.Name.Name == blankIdentifier,
			Line:       imp.Pos(),
			Column:     imp.End(),
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
					} else if pkgName := effectivePackageName(strings.Trim(imp.Path.Value, `"`)); pkgName == ident.Name {
						deps[strings.Trim(imp.Path.Value, `"`)].IsUsed = true
					}
				}
			}
		}
		return true
	})

	return deps
}

func runGnoPackageLinter(ctx *rule.AnalysisContext, deps Dependencies) []tt.Issue {
	issues := make([]tt.Issue, 0, len(deps))
	for imp, dep := range deps {
		if dep.IsUsed || dep.IsIgnored {
			continue
		}
		issue := ctx.NewIssue("unused-package", dep.Line, dep.Column)
		issue.Message = fmt.Sprintf("unused import: %s", imp)
		issues = append(issues, issue)
	}
	return issues
}

func isGnoPackage(importPath string) bool {
	return strings.HasPrefix(importPath, GNO_PKG_PREFIX) || importPath == GNO_STD_PACKAGE
}

func effectivePackageName(path string) string {
	parts := strings.Split(path, "/")
	if len(parts) >= 2 && versionSuffix.MatchString(parts[len(parts)-1]) {
		return parts[len(parts)-2]
	}
	return parts[len(parts)-1]
}
