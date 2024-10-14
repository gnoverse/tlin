package internal

import (
	"go/ast"
	"go/token"

	"github.com/gnolang/tlin/internal/lints"
	tt "github.com/gnolang/tlin/internal/types"
)

/*
* Implement each lint rule as a separate struct
 */

// LintRule defines the interface for all lint rules.
type LintRule interface {
	// Check runs the lint rule on the given file and returns a slice of Issues.
	Check(filename string, node *ast.File, fset *token.FileSet) ([]tt.Issue, error)

	// Name returns the name of the lint rule.
	Name() string

	// Severity returns the severity of the lint rule.
	Severity() tt.Severity

	// SetSeverity sets the severity of the lint rule.
	SetSeverity(tt.Severity)
}

type GolangciLintRule struct {
	severity tt.Severity
}

func NewGolangciLintRule() LintRule {
	return &GolangciLintRule{
		severity: tt.SeverityError,
	}
}

func (r *GolangciLintRule) Check(filename string, _ *ast.File, _ *token.FileSet) ([]tt.Issue, error) {
	return lints.RunGolangciLint(filename, r.severity)
}

func (r *GolangciLintRule) Name() string {
	return "golangci-lint"
}

func (r *GolangciLintRule) Severity() tt.Severity {
	return r.severity
}

func (r *GolangciLintRule) SetSeverity(severity tt.Severity) {
	r.severity = severity
}

type DeprecateFuncRule struct {
	severity tt.Severity
}

func NewDeprecateFuncRule() LintRule {
	return &DeprecateFuncRule{
		severity: tt.SeverityError,
	}
}

func (r *DeprecateFuncRule) Check(filename string, node *ast.File, fset *token.FileSet) ([]tt.Issue, error) {
	return lints.DetectDeprecatedFunctions(filename, node, fset, r.severity)
}

func (r *DeprecateFuncRule) Name() string {
	return "deprecated-function"
}

func (r *DeprecateFuncRule) Severity() tt.Severity {
	return r.severity
}

func (r *DeprecateFuncRule) SetSeverity(severity tt.Severity) {
	r.severity = severity
}

type SimplifySliceExprRule struct {
	severity tt.Severity
}

func NewSimplifySliceExprRule() LintRule {
	return &SimplifySliceExprRule{
		severity: tt.SeverityError,
	}
}

func (r *SimplifySliceExprRule) Check(filename string, node *ast.File, fset *token.FileSet) ([]tt.Issue, error) {
	return lints.DetectUnnecessarySliceLength(filename, node, fset, r.severity)
}

func (r *SimplifySliceExprRule) Name() string {
	return "simplify-slice-range"
}

func (r *SimplifySliceExprRule) Severity() tt.Severity {
	return r.severity
}

func (r *SimplifySliceExprRule) SetSeverity(severity tt.Severity) {
	r.severity = severity
}

type UnnecessaryConversionRule struct {
	severity tt.Severity
}

func NewUnnecessaryConversionRule() LintRule {
	return &UnnecessaryConversionRule{
		severity: tt.SeverityError,
	}
}

func (r *UnnecessaryConversionRule) Check(filename string, node *ast.File, fset *token.FileSet) ([]tt.Issue, error) {
	return lints.DetectUnnecessaryConversions(filename, node, fset, r.severity)
}

func (r *UnnecessaryConversionRule) Name() string {
	return "unnecessary-type-conversion"
}

func (r *UnnecessaryConversionRule) Severity() tt.Severity {
	return r.severity
}

func (r *UnnecessaryConversionRule) SetSeverity(severity tt.Severity) {
	r.severity = severity
}

type LoopAllocationRule struct {
	severity tt.Severity
}

func NewLoopAllocationRule() LintRule {
	return &LoopAllocationRule{
		severity: tt.SeverityError,
	}
}

func (r *LoopAllocationRule) Check(filename string, node *ast.File, fset *token.FileSet) ([]tt.Issue, error) {
	return lints.DetectLoopAllocation(filename, node, fset, r.severity)
}

func (r *LoopAllocationRule) Name() string {
	return "loop-allocation"
}

func (r *LoopAllocationRule) Severity() tt.Severity {
	return r.severity
}

func (r *LoopAllocationRule) SetSeverity(severity tt.Severity) {
	r.severity = severity
}

type DetectCycleRule struct {
	severity tt.Severity
}

func NewDetectCycleRule() LintRule {
	return &DetectCycleRule{
		severity: tt.SeverityError,
	}
}

func (r *DetectCycleRule) Check(filename string, node *ast.File, fset *token.FileSet) ([]tt.Issue, error) {
	return lints.DetectCycle(filename, node, fset, r.severity)
}

func (r *DetectCycleRule) Name() string {
	return "cycle-detection"
}

func (r *DetectCycleRule) Severity() tt.Severity {
	return r.severity
}

func (r *DetectCycleRule) SetSeverity(severity tt.Severity) {
	r.severity = severity
}

type EmitFormatRule struct {
	severity tt.Severity
}

func NewEmitFormatRule() LintRule {
	return &EmitFormatRule{
		severity: tt.SeverityError,
	}
}

func (r *EmitFormatRule) Check(filename string, node *ast.File, fset *token.FileSet) ([]tt.Issue, error) {
	return lints.DetectEmitFormat(filename, node, fset, r.severity)
}

func (r *EmitFormatRule) Name() string {
	return "emit-format"
}

func (r *EmitFormatRule) Severity() tt.Severity {
	return r.severity
}

func (r *EmitFormatRule) SetSeverity(severity tt.Severity) {
	r.severity = severity
}

type SliceBoundCheckRule struct {
	severity tt.Severity
}

func NewSliceBoundCheckRule() LintRule {
	return &SliceBoundCheckRule{
		severity: tt.SeverityError,
	}
}

func (r *SliceBoundCheckRule) Check(filename string, node *ast.File, fset *token.FileSet) ([]tt.Issue, error) {
	return lints.DetectSliceBoundCheck(filename, node, fset, r.severity)
}

func (r *SliceBoundCheckRule) Name() string {
	return "slice-bounds-check"
}

func (r *SliceBoundCheckRule) Severity() tt.Severity {
	return r.severity
}

func (r *SliceBoundCheckRule) SetSeverity(severity tt.Severity) {
	r.severity = severity
}

type UselessBreakRule struct {
	severity tt.Severity
}

func NewUselessBreakRule() LintRule {
	return &UselessBreakRule{
		severity: tt.SeverityError,
	}
}

func (r *UselessBreakRule) Check(filename string, node *ast.File, fset *token.FileSet) ([]tt.Issue, error) {
	return lints.DetectUselessBreak(filename, node, fset, r.severity)
}

func (r *UselessBreakRule) Name() string {
	return "useless-break"
}

func (r *UselessBreakRule) Severity() tt.Severity {
	return r.severity
}

func (r *UselessBreakRule) SetSeverity(severity tt.Severity) {
	r.severity = severity
}

type EarlyReturnOpportunityRule struct {
	severity tt.Severity
}

func NewEarlyReturnOpportunityRule() LintRule {
	return &EarlyReturnOpportunityRule{
		severity: tt.SeverityError,
	}
}

func (r *EarlyReturnOpportunityRule) Check(filename string, node *ast.File, fset *token.FileSet) ([]tt.Issue, error) {
	return lints.DetectEarlyReturnOpportunities(filename, node, fset, r.severity)
}

func (r *EarlyReturnOpportunityRule) Name() string {
	return "early-return-opportunity"
}

func (r *EarlyReturnOpportunityRule) Severity() tt.Severity {
	return r.severity
}

func (r *EarlyReturnOpportunityRule) SetSeverity(severity tt.Severity) {
	r.severity = severity
}

type DeferRule struct {
	severity tt.Severity
}

func NewDeferRule() LintRule {
	return &DeferRule{
		severity: tt.SeverityError,
	}
}

func (r *DeferRule) Check(filename string, node *ast.File, fset *token.FileSet) ([]tt.Issue, error) {
	return lints.DetectDeferIssues(filename, node, fset, r.severity)
}

func (r *DeferRule) Name() string {
	return "defer-issues"
}

func (r *DeferRule) Severity() tt.Severity {
	return r.severity
}

func (r *DeferRule) SetSeverity(severity tt.Severity) {
	r.severity = severity
}

type MissingModPackageRule struct {
	severity tt.Severity
}

func NewMissingModPackageRule() LintRule {
	return &MissingModPackageRule{
		severity: tt.SeverityError,
	}
}

func (r *MissingModPackageRule) Check(filename string, node *ast.File, fset *token.FileSet) ([]tt.Issue, error) {
	return lints.DetectMissingModPackage(filename, node, fset, r.severity)
}

func (r *MissingModPackageRule) Name() string {
	return "gno-mod-tidy"
}

func (r *MissingModPackageRule) Severity() tt.Severity {
	return r.severity
}

func (r *MissingModPackageRule) SetSeverity(severity tt.Severity) {
	r.severity = severity
}

type ConstErrorDeclarationRule struct {
	severity tt.Severity
}

func NewConstErrorDeclarationRule() LintRule {
	return &ConstErrorDeclarationRule{
		severity: tt.SeverityError,
	}
}

func (r *ConstErrorDeclarationRule) Check(filename string, node *ast.File, fset *token.FileSet) ([]tt.Issue, error) {
	return lints.DetectConstErrorDeclaration(filename, node, fset, r.severity)
}

func (r *ConstErrorDeclarationRule) Name() string {
	return "const-error-declaration"
}

func (r *ConstErrorDeclarationRule) SetSeverity(severity tt.Severity) {
	r.severity = severity
}

func (r *ConstErrorDeclarationRule) Severity() tt.Severity {
	return r.severity
}

// -----------------------------------------------------------------------------
// Regex related rules

type RepeatedRegexCompilationRule struct {
	severity tt.Severity
}

func NewRepeatedRegexCompilationRule() LintRule {
	return &RepeatedRegexCompilationRule{
		severity: tt.SeverityError,
	}
}

func (r *RepeatedRegexCompilationRule) Check(filename string, node *ast.File, fset *token.FileSet) ([]tt.Issue, error) {
	return lints.DetectRepeatedRegexCompilation(filename, node, r.severity)
}

func (r *RepeatedRegexCompilationRule) Name() string {
	return "repeated-regex-compilation"
}

func (r *RepeatedRegexCompilationRule) Severity() tt.Severity {
	return r.severity
}

func (r *RepeatedRegexCompilationRule) SetSeverity(severity tt.Severity) {
	r.severity = severity
}

// -----------------------------------------------------------------------------

type CyclomaticComplexityRule struct {
	Threshold int
	severity  tt.Severity
}

func NewCyclomaticComplexityRule(threshold int) LintRule {
	return &CyclomaticComplexityRule{
		Threshold: threshold,
		severity:  tt.SeverityError,
	}
}

func (r *CyclomaticComplexityRule) Check(filename string, node *ast.File, fset *token.FileSet) ([]tt.Issue, error) {
	return lints.DetectHighCyclomaticComplexity(filename, r.Threshold, r.severity)
}

func (r *CyclomaticComplexityRule) Name() string {
	return "high-cyclomatic-complexity"
}

func (r *CyclomaticComplexityRule) Severity() tt.Severity {
	return r.severity
}

func (r *CyclomaticComplexityRule) SetSeverity(severity tt.Severity) {
	r.severity = severity
}

// -----------------------------------------------------------------------------

// GnoSpecificRule checks for gno-specific package imports. (p, r and std)
type GnoSpecificRule struct {
	severity tt.Severity
}

func NewGnoSpecificRule() LintRule {
	return &GnoSpecificRule{
		severity: tt.SeverityError,
	}
}

func (r *GnoSpecificRule) Check(filename string, _ *ast.File, _ *token.FileSet) ([]tt.Issue, error) {
	return lints.DetectGnoPackageImports(filename, r.severity)
}

func (r *GnoSpecificRule) Name() string {
	return "unused-package"
}

func (r *GnoSpecificRule) Severity() tt.Severity {
	return r.severity
}

func (r *GnoSpecificRule) SetSeverity(severity tt.Severity) {
	r.severity = severity
}
