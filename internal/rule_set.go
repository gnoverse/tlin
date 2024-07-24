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
}

type GolangciLintRule struct{}

func (r *GolangciLintRule) Check(filename string) ([]tt.Issue, error) {
	return lints.RunGolangciLint(filename)
}

type UnnecessaryElseRule struct{}

func (r *UnnecessaryElseRule) Check(filename string) ([]tt.Issue, error) {
	return lints.DetectUnnecessaryElse(filename)
}

type UnusedFunctionRule struct{}

func (r *UnusedFunctionRule) Check(filename string) ([]tt.Issue, error) {
	return lints.DetectUnusedFunctions(filename)
}

type SimplifySliceExprRule struct{}

func (r *SimplifySliceExprRule) Check(filename string) ([]tt.Issue, error) {
	return lints.DetectUnnecessarySliceLength(filename)
}

type UnnecessaryConversionRule struct{}

func (r *UnnecessaryConversionRule) Check(filename string) ([]tt.Issue, error) {
	return lints.DetectUnnecessaryConversions(filename)
}

type LoopAllocationRule struct{}

func (r *LoopAllocationRule) Check(filename string) ([]tt.Issue, error) {
	return lints.DetectLoopAllocation(filename)
}

type DetectCycleRule struct{}

func (r *DetectCycleRule) Check(filename string) ([]tt.Issue, error) {
	return lints.DetectCycle(filename)
}

// -----------------------------------------------------------------------------

type CyclomaticComplexityRule struct {
	Threshold int
}

func (r *CyclomaticComplexityRule) Check(filename string) ([]tt.Issue, error) {
	return lints.DetectHighCyclomaticComplexity(filename, r.Threshold)
}