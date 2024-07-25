package internal

import (
	"github.com/gnoswap-labs/lint/internal/lints"
	tt "github.com/gnoswap-labs/lint/internal/types"
)

/*
* Implement each lint rule as a separate struct
 */

// LintRule defines the interface for all lint rules.
type LintRule interface {
	// Check runs the lint rule on the given file and returns a slice of Issues.
	Check(filename string) ([]tt.Issue, error)

	// Name returns the name of the lint rule.
	Name() string
}

type GolangciLintRule struct{}

func (r *GolangciLintRule) Check(filename string) ([]tt.Issue, error) {
	return lints.RunGolangciLint(filename)
}

func (r *GolangciLintRule) Name() string {
	return "golangci-lint"
}

type UnnecessaryElseRule struct{}

func (r *UnnecessaryElseRule) Check(filename string) ([]tt.Issue, error) {
	return lints.DetectUnnecessaryElse(filename)
}

func (r *UnnecessaryElseRule) Name() string {
	return "unnecessary-else"
}

type UnusedFunctionRule struct{}

func (r *UnusedFunctionRule) Check(filename string) ([]tt.Issue, error) {
	return lints.DetectUnusedFunctions(filename)
}

func (r *UnusedFunctionRule) Name() string {
	return "unused-function"
}

type SimplifySliceExprRule struct{}

func (r *SimplifySliceExprRule) Check(filename string) ([]tt.Issue, error) {
	return lints.DetectUnnecessarySliceLength(filename)
}

func (r *SimplifySliceExprRule) Name() string {
	return "simplify-slice-range"
}

type UnnecessaryConversionRule struct{}

func (r *UnnecessaryConversionRule) Check(filename string) ([]tt.Issue, error) {
	return lints.DetectUnnecessaryConversions(filename)
}

func (r *UnnecessaryConversionRule) Name() string {
	return "unnecessary-type-conversion"
}

type LoopAllocationRule struct{}

func (r *LoopAllocationRule) Check(filename string) ([]tt.Issue, error) {
	return lints.DetectLoopAllocation(filename)
}

func (r *LoopAllocationRule) Name() string {
	return "loop-allocation"
}

type DetectCycleRule struct{}

func (r *DetectCycleRule) Check(filename string) ([]tt.Issue, error) {
	return lints.DetectCycle(filename)
}

func (r *DetectCycleRule) Name() string {
	return "cycle-detection"
}

// -----------------------------------------------------------------------------

type CyclomaticComplexityRule struct {
	Threshold int
}

func (r *CyclomaticComplexityRule) Check(filename string) ([]tt.Issue, error) {
	return lints.DetectHighCyclomaticComplexity(filename, r.Threshold)
}

func (r *CyclomaticComplexityRule) Name() string {
	return "high-cyclomatic-complexity"
}

// -----------------------------------------------------------------------------

// GnoSpecificRule checks for gno-specific package imports. (p, r and std)
type GnoSpecificRule struct{}

func (r *GnoSpecificRule) Check(filename string) ([]tt.Issue, error) {
	return lints.DetectGnoPackageImports(filename)
}

func (r *GnoSpecificRule) Name() string {
	return "unused-package"
}
