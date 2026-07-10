package lints

import (
	"go/ast"
	"go/token"

	"github.com/gnolang/tlin/internal/rule"
	tt "github.com/gnolang/tlin/internal/types"
)

func init() {
	rule.Register(uselessBreakRule{})
}

type uselessBreakRule struct{}

func (uselessBreakRule) Name() string                 { return "useless-break" }
func (uselessBreakRule) DefaultSeverity() tt.Severity { return tt.SeverityError }

func (uselessBreakRule) Check(ctx *rule.AnalysisContext) ([]tt.Issue, error) {
	return DetectUselessBreak(ctx)
}

// DetectUselessBreak detects useless break statements in switch or
// select statements. Issues are constructed via ctx.NewIssue so the
// emitted Filename matches the user-supplied path even when the
// engine ran the parser against a temp .go.
func DetectUselessBreak(ctx *rule.AnalysisContext) ([]tt.Issue, error) {
	var issues []tt.Issue
	ast.Inspect(ctx.File, func(n ast.Node) bool {
		switch v := n.(type) {
		case *ast.SwitchStmt:
			for _, stmt := range v.Body.List {
				if caseClause, ok := stmt.(*ast.CaseClause); ok {
					checkUselessBreak(ctx, caseClause.Body, &issues)
				}
			}
		case *ast.SelectStmt:
			for _, stmt := range v.Body.List {
				if commClause, ok := stmt.(*ast.CommClause); ok {
					checkUselessBreak(ctx, commClause.Body, &issues)
				}
			}
		}
		return true
	})

	return issues, nil
}

func checkUselessBreak(ctx *rule.AnalysisContext, stmts []ast.Stmt, issues *[]tt.Issue) {
	if len(stmts) == 0 {
		return
	}

	lastStmt := stmts[len(stmts)-1]
	breakStmt, ok := lastStmt.(*ast.BranchStmt)
	if !ok || breakStmt.Tok != token.BREAK || breakStmt.Label != nil {
		return
	}
	issue := ctx.NewIssue("useless-break", breakStmt.Pos(), breakStmt.End())
	issue.Message = "useless break statement at the end of case clause"
	*issues = append(*issues, issue)
}
