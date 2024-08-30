package lints

import (
	"go/ast"
	"go/token"

	tt "github.com/gnoswap-labs/tlin/internal/types"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/packages"
)

var RepeatedRegexCompilationAnalyzer = &analysis.Analyzer{
	Name: "repeatedregexcompilation",
	Doc:  "Checks for repeated compilation of the same regex pattern",
	Run:  runRepeatedRegexCompilation,
}

func DetectRepeatedRegexCompilation(filename string) ([]tt.Issue, error) {
	issues, err := runAnalyzer(filename, RepeatedRegexCompilationAnalyzer)
	if err != nil {
		return nil, err
	}
	return issues, nil
}

func runAnalyzer(filename string, a *analysis.Analyzer) ([]tt.Issue, error) {
	cfg := &packages.Config{
		Mode:  packages.NeedFiles | packages.NeedSyntax | packages.NeedTypesInfo | packages.NeedTypes,
		Tests: false,
	}

	pkgs, err := packages.Load(cfg, filename)
	if err != nil {
		return nil, err
	}

	var diagnostics []analysis.Diagnostic
	pass := &analysis.Pass{
		Analyzer:  a,
		Fset:      pkgs[0].Fset,
		Files:     pkgs[0].Syntax,
		Pkg:       pkgs[0].Types,
		TypesInfo: pkgs[0].TypesInfo,
		ResultOf:  make(map[*analysis.Analyzer]interface{}),
		Report: func(d analysis.Diagnostic) {
			diagnostics = append(diagnostics, d)
		},
	}

	_, err = a.Run(pass)
	if err != nil {
		return nil, err
	}

	var issues []tt.Issue
	for _, diag := range diagnostics {
		issues = append(issues, tt.Issue{
			Rule:     a.Name,
			Filename: filename,
			Start:    pass.Fset.Position(diag.Pos),
			End:      pass.Fset.Position(diag.End),
			Message:  diag.Message,
		})
	}

	return issues, nil
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
