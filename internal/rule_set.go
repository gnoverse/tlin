package internal

import (
	"go/ast"
	"go/token"

	"github.com/gnolang/tlin/internal/lints"
	tt "github.com/gnolang/tlin/internal/types"
)

// LintRule defines the struct for all lint rules.
type LintRule struct {
	severity tt.Severity
	check    func(filename string, node *ast.File, fset *token.FileSet, severity tt.Severity) ([]tt.Issue, error)
	name     string
}

func (r LintRule) Severity() tt.Severity {
	return r.severity
}

func (r LintRule) Name() string {
	return r.name
}

func (r LintRule) Check(filename string, node *ast.File, fset *token.FileSet) ([]tt.Issue, error) {
	return r.check(filename, node, fset, r.severity)
}

var (
	EarlyReturnOpportunityRule   = LintRule{severity: tt.SeverityInfo, check: lints.DetectEarlyReturnOpportunities}
	RepeatedRegexCompilationRule = LintRule{severity: tt.SeverityWarning, check: lints.DetectRepeatedRegexCompilation}
	GnoSpecificRule              = LintRule{severity: tt.SeverityWarning, check: lints.DetectGnoPackageImports}
	SimplifyForRangeRule         = LintRule{severity: tt.SeverityWarning, check: lints.DetectSimplifiableForLoops}
)

type ruleMap map[string]LintRule

var allRules = ruleMap{
	"early-return-opportunity":   EarlyReturnOpportunityRule,
	"repeated-regex-compilation": RepeatedRegexCompilationRule,
	"unused-package":             GnoSpecificRule,
	"simplify-for-range":         SimplifyForRangeRule,
}
