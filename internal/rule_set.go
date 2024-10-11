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
}

type GolangciLintRule struct {
	severity tt.Severity
}

func (r *GolangciLintRule) Check(filename string, _ *ast.File, _ *token.FileSet) ([]tt.Issue, error) {
	return lints.RunGolangciLint(filename)
}

func (r *GolangciLintRule) Name() string {
	return "golangci-lint"
}

func (r *GolangciLintRule) Severity() tt.Severity {
	return r.severity
}

type DeprecateFuncRule struct {
	severity tt.Severity
}

func (r *DeprecateFuncRule) Check(filename string, node *ast.File, fset *token.FileSet) ([]tt.Issue, error) {
	return lints.DetectDeprecatedFunctions(filename, node, fset)
}

func (r *DeprecateFuncRule) Name() string {
	return "deprecated-function"
}

func (r *DeprecateFuncRule) Severity() tt.Severity {
	return r.severity
}

type SimplifySliceExprRule struct {
	severity tt.Severity
}

func (r *SimplifySliceExprRule) Check(filename string, node *ast.File, fset *token.FileSet) ([]tt.Issue, error) {
	return lints.DetectUnnecessarySliceLength(filename, node, fset)
}

func (r *SimplifySliceExprRule) Name() string {
	return "simplify-slice-range"
}

func (r *SimplifySliceExprRule) Severity() tt.Severity {
	return r.severity
}

type UnnecessaryConversionRule struct {
	severity tt.Severity
}

func (r *UnnecessaryConversionRule) Check(filename string, node *ast.File, fset *token.FileSet) ([]tt.Issue, error) {
	return lints.DetectUnnecessaryConversions(filename, node, fset)
}

func (r *UnnecessaryConversionRule) Name() string {
	return "unnecessary-type-conversion"
}

func (r *UnnecessaryConversionRule) Severity() tt.Severity {
	return r.severity
}

type LoopAllocationRule struct {
	severity tt.Severity
}

func (r *LoopAllocationRule) Check(filename string, node *ast.File, fset *token.FileSet) ([]tt.Issue, error) {
	return lints.DetectLoopAllocation(filename, node, fset)
}

func (r *LoopAllocationRule) Name() string {
	return "loop-allocation"
}

func (r *LoopAllocationRule) Severity() tt.Severity {
	return r.severity
}

type DetectCycleRule struct {
	severity tt.Severity
}

func (r *DetectCycleRule) Check(filename string, node *ast.File, fset *token.FileSet) ([]tt.Issue, error) {
	return lints.DetectCycle(filename, node, fset)
}

func (r *DetectCycleRule) Name() string {
	return "cycle-detection"
}

func (r *DetectCycleRule) Severity() tt.Severity {
	return r.severity
}

type EmitFormatRule struct {
	severity tt.Severity
}

func (r *EmitFormatRule) Check(filename string, node *ast.File, fset *token.FileSet) ([]tt.Issue, error) {
	return lints.DetectEmitFormat(filename, node, fset)
}

func (r *EmitFormatRule) Name() string {
	return "emit-format"
}

func (r *EmitFormatRule) Severity() tt.Severity {
	return r.severity
}

type SliceBoundCheckRule struct {
	severity tt.Severity
}

func (r *SliceBoundCheckRule) Check(filename string, node *ast.File, fset *token.FileSet) ([]tt.Issue, error) {
	return lints.DetectSliceBoundCheck(filename, node, fset)
}

func (r *SliceBoundCheckRule) Name() string {
	return "slice-bounds-check"
}

func (r *SliceBoundCheckRule) Severity() tt.Severity {
	return r.severity
}

type UselessBreakRule struct {
	severity tt.Severity
}

func (r *UselessBreakRule) Check(filename string, node *ast.File, fset *token.FileSet) ([]tt.Issue, error) {
	return lints.DetectUselessBreak(filename, node, fset)
}

func (r *UselessBreakRule) Name() string {
	return "useless-break"
}

func (r *UselessBreakRule) Severity() tt.Severity {
	return r.severity
}

type EarlyReturnOpportunityRule struct {
	severity tt.Severity
}

func (r *EarlyReturnOpportunityRule) Check(filename string, node *ast.File, fset *token.FileSet) ([]tt.Issue, error) {
	return lints.DetectEarlyReturnOpportunities(filename, node, fset)
}

func (r *EarlyReturnOpportunityRule) Name() string {
	return "early-return-opportunity"
}

func (r *EarlyReturnOpportunityRule) Severity() tt.Severity {
	return r.severity
}

type DeferRule struct {
	severity tt.Severity
}

func (r *DeferRule) Check(filename string, node *ast.File, fset *token.FileSet) ([]tt.Issue, error) {
	return lints.DetectDeferIssues(filename, node, fset)
}

func (r *DeferRule) Name() string {
	return "defer-issues"
}

func (r *DeferRule) Severity() tt.Severity {
	return r.severity
}

type MissingModPackageRule struct {
	severity tt.Severity
}

func (r *MissingModPackageRule) Check(filename string, node *ast.File, fset *token.FileSet) ([]tt.Issue, error) {
	return lints.DetectMissingModPackage(filename, node, fset)
}

func (r *MissingModPackageRule) Name() string {
	return "gno-mod-tidy"
}

func (r *MissingModPackageRule) Severity() tt.Severity {
	return r.severity
}

// -----------------------------------------------------------------------------
// Regex related rules

type RepeatedRegexCompilationRule struct {
	severity tt.Severity
}

func (r *RepeatedRegexCompilationRule) Check(filename string, node *ast.File, fset *token.FileSet) ([]tt.Issue, error) {
	return lints.DetectRepeatedRegexCompilation(filename, node)
}

func (r *RepeatedRegexCompilationRule) Name() string {
	return "repeated-regex-compilation"
}

func (r *RepeatedRegexCompilationRule) Severity() tt.Severity {
	return r.severity
}

// -----------------------------------------------------------------------------

type CyclomaticComplexityRule struct {
	Threshold int
	severity  tt.Severity
}

func (r *CyclomaticComplexityRule) Check(filename string, node *ast.File) ([]tt.Issue, error) {
	return lints.DetectHighCyclomaticComplexity(filename, r.Threshold)
}

func (r *CyclomaticComplexityRule) Name() string {
	return "high-cyclomatic-complexity"
}

func (r *CyclomaticComplexityRule) Severity() tt.Severity {
	return r.severity
}

// -----------------------------------------------------------------------------

// GnoSpecificRule checks for gno-specific package imports. (p, r and std)
type GnoSpecificRule struct {
	severity tt.Severity
}

func (r *GnoSpecificRule) Check(filename string, _ *ast.File, _ *token.FileSet) ([]tt.Issue, error) {
	return lints.DetectGnoPackageImports(filename)
}

func (r *GnoSpecificRule) Name() string {
	return "unused-package"
}

func (r *GnoSpecificRule) Severity() tt.Severity {
	return r.severity
}
