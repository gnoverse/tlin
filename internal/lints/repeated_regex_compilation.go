package lints

import (
	"context"
	"go/ast"
	"go/parser"
	"go/token"
	"strconv"

	"github.com/gnolang/tlin/internal/rule"
	tt "github.com/gnolang/tlin/internal/types"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/packages"
)

func init() {
	rule.Register(repeatedRegexCompilationRule{})
}

type repeatedRegexCompilationRule struct{}

func (repeatedRegexCompilationRule) Name() string                 { return "repeated-regex-compilation" }
func (repeatedRegexCompilationRule) DefaultSeverity() tt.Severity { return tt.SeverityWarning }

// Check is the single-file fallback for non-engine callers; engine
// dispatch goes straight to CheckPackage with the live ctx.
func (r repeatedRegexCompilationRule) Check(ctx *rule.AnalysisContext) ([]tt.Issue, error) {
	return r.CheckPackage(context.Background(), ctx.SinglePackage())
}

var RepeatedRegexCompilationAnalyzer = &analysis.Analyzer{
	Name: "repeated-regex-compilation",
	Doc:  "Checks for repeated compilation of the same regex pattern",
	Run:  runRepeatedRegexCompilation,
}

// CheckPackage loads the package once and runs the analyzer across
// every file in it. Issues are filtered to files in pctx scope so
// loader-pulled siblings (e.g. _test.go) don't leak into output.
//
// The imports-only pre-scan short-circuits the common case where no
// file in the package imports regexp; packages.Load is much more
// expensive than parser.ParseFile(ImportsOnly).
func (repeatedRegexCompilationRule) CheckPackage(_ context.Context, pctx *rule.PackageContext) ([]tt.Issue, error) {
	if len(pctx.WorkingPaths) == 0 {
		return nil, nil
	}
	if !anyFileImportsRegexp(pctx.WorkingPaths) {
		return nil, nil
	}

	cfg := &packages.Config{
		Mode:  packages.NeedFiles | packages.NeedSyntax | packages.NeedTypesInfo | packages.NeedTypes,
		Tests: false,
	}

	// Load via the file paths (not "." against cfg.Dir) so orphan
	// files outside any go.mod still resolve — same calling pattern
	// the pre-EPR-6 per-file path used.
	pkgs, err := packages.Load(cfg, pctx.WorkingPaths...)
	if err != nil {
		return nil, err
	}
	if len(pkgs) == 0 {
		return nil, nil
	}
	pkg := pkgs[0]

	var diagnostics []analysis.Diagnostic
	pass := &analysis.Pass{
		Analyzer:  RepeatedRegexCompilationAnalyzer,
		Fset:      pkg.Fset,
		Files:     pkg.Syntax,
		Pkg:       pkg.Types,
		TypesInfo: pkg.TypesInfo,
		ResultOf:  make(map[*analysis.Analyzer]interface{}),
		Report: func(d analysis.Diagnostic) {
			diagnostics = append(diagnostics, d)
		},
	}

	if _, err = RepeatedRegexCompilationAnalyzer.Run(pass); err != nil {
		return nil, err
	}

	issues := make([]tt.Issue, 0, len(diagnostics))
	for _, diag := range diagnostics {
		start := pass.Fset.Position(diag.Pos)
		end := pass.Fset.Position(diag.End)
		if !pctx.InScope(start.Filename) {
			continue
		}
		start.Filename = pctx.RemapFilename(start.Filename)
		end.Filename = pctx.RemapFilename(end.Filename)
		issues = append(issues, tt.Issue{
			Rule:     RepeatedRegexCompilationAnalyzer.Name,
			Filename: start.Filename,
			Start:    start,
			End:      end,
			Message:  diag.Message,
			Severity: pctx.Severity,
		})
	}

	return issues, nil
}

// anyFileImportsRegexp parses each working path with parser.ImportsOnly
// (cheap — stops after the import block) and returns true on the first
// file that imports "regexp". Used as a fast gate in front of
// packages.Load, which is much more expensive than parsing imports.
func anyFileImportsRegexp(paths []string) bool {
	fset := token.NewFileSet()
	for _, p := range paths {
		f, err := parser.ParseFile(fset, p, nil, parser.ImportsOnly)
		if err != nil {
			continue
		}
		for _, imp := range f.Imports {
			path, err := strconv.Unquote(imp.Path.Value)
			if err == nil && path == "regexp" {
				return true
			}
		}
	}
	return false
}

func runRepeatedRegexCompilation(pass *analysis.Pass) (interface{}, error) {
	for _, file := range pass.Files {
		ast.Inspect(file, func(n ast.Node) bool {
			funcDecl, ok := n.(*ast.FuncDecl)
			if !ok {
				return true
			}

			regexPatterns := make(map[string]token.Pos)
			ast.Inspect(funcDecl.Body, func(node ast.Node) bool {
				callExpr, ok := node.(*ast.CallExpr)
				if !ok {
					return true
				}

				if isRegexpCompile(callExpr) {
					if pattern, ok := getRegexPattern(callExpr); ok {
						if firstPos, exists := regexPatterns[pattern]; exists {
							pass.Reportf(callExpr.Pos(), "regexp.Compile called with same pattern more than once. First occurrence at line %d", pass.Fset.Position(firstPos).Line)
						} else {
							regexPatterns[pattern] = callExpr.Pos()
						}
					}
				}

				return true
			})

			return true
		})
	}

	return nil, nil
}

func isRegexpCompile(callExpr *ast.CallExpr) bool {
	if selExpr, ok := callExpr.Fun.(*ast.SelectorExpr); ok {
		if ident, ok := selExpr.X.(*ast.Ident); ok {
			return ident.Name == "regexp" && (selExpr.Sel.Name == "Compile" || selExpr.Sel.Name == "MustCompile")
		}
	}
	return false
}

func getRegexPattern(callExpr *ast.CallExpr) (string, bool) {
	if len(callExpr.Args) > 0 {
		if lit, ok := callExpr.Args[0].(*ast.BasicLit); ok && lit.Kind == token.STRING {
			return lit.Value, true
		}
	}
	return "", false
}
