package internal

import (
	"go/ast"
	"go/token"

	"github.com/gnolang/tlin/internal/lints"
	tt "github.com/gnolang/tlin/internal/types"
)

/*
* Implement each lint rule as a a new LintRule struct.
 */

// LintRule defines the struct for all lint rules.
type LintRule struct {
	severity      tt.Severity
	check 	   	  func(filename string, node *ast.File, fset *token.FileSet, severity tt.Severity) ([]tt.Issue, error)
	name		  string
}

func (r LintRule) SetSeverity(severity tt.Severity) {
	r.severity = severity
}

func (r LintRule) Severity() tt.Severity {
	return r.severity
}

func (r LintRule) SetName(name string) {
	r.name = name
}

func (r LintRule) Name() string {
	return r.name
}

func (r LintRule) Check(filename string, node *ast.File, fset *token.FileSet) ([]tt.Issue, error) {
	return r.check(filename, node, fset, r.severity)
}

var (
	GolangciLintRule = LintRule{severity: tt.SeverityWarning, check: lints.RunGolangciLint}
	SimplifySliceExprRule = LintRule{severity: tt.SeverityError, check: lints.DetectUnnecessarySliceLength}
	UnnecessaryConversionRule = LintRule{severity: tt.SeverityWarning, check: lints.DetectUnnecessaryConversions}
	DetectCycleRule = LintRule{severity: tt.SeverityError, check: lints.DetectCycle}
	EmitFormatRule = LintRule{severity: tt.SeverityInfo, check: lints.DetectEmitFormat}
	UselessBreakRule = LintRule{severity: tt.SeverityError, check: lints.DetectUselessBreak}
	EarlyReturnOpportunityRule = LintRule{severity: tt.SeverityInfo, check: lints.DetectEarlyReturnOpportunities}
	DeferRule = LintRule{severity: tt.SeverityWarning, check: lints.DetectDeferIssues}
	ConstErrorDeclarationRule = LintRule{severity: tt.SeverityError, check: lints.DetectConstErrorDeclaration}
	RepeatedRegexCompilationRule = LintRule{severity: tt.SeverityWarning, check: lints.DetectRepeatedRegexCompilation}
	GnoSpecificRule = LintRule{severity: tt.SeverityWarning, check: lints.DetectGnoPackageImports}
)

// Define the ruleMap type
type ruleMap map[string]LintRule

// Create a map to hold the mappings of rule names to LintRule structs
var allRules = ruleMap{
	"golangci-lint":               GolangciLintRule,
	"simplify-slice-range":        SimplifySliceExprRule,
	"unnecessary-type-conversion": UnnecessaryConversionRule,
	"cycle-detection":             DetectCycleRule,
	"emit-format":                 EmitFormatRule,
	"useless-break":               UselessBreakRule,
	"early-return-opportunity":    EarlyReturnOpportunityRule,
	"defer-issues":                DeferRule,
	"const-error-declaration":     ConstErrorDeclarationRule,
	"repeated-regex-compilation":  RepeatedRegexCompilationRule,
	"unused-package":              GnoSpecificRule,
}
