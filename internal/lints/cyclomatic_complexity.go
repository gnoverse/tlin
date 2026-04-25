package lints

import (
	"fmt"
	"go/ast"
	"go/token"

	"github.com/fzipp/gocyclo"
	"github.com/gnolang/tlin/internal/rule"
	tt "github.com/gnolang/tlin/internal/types"
)

const defaultCyclomaticThreshold = 10

func init() {
	rule.Register(&cyclomaticComplexityRule{threshold: defaultCyclomaticThreshold})
}

// cyclomaticComplexityRule reports functions whose cyclomatic
// complexity exceeds threshold. It is registered with
// DefaultSeverity == SeverityOff so it stays silent unless explicitly
// enabled via config (severity override) or the -cyclo CLI shorthand.
//
// The threshold is mutated by ParseConfig at engine construction
// time. Because the rule lives as a singleton in the package-level
// registry, two engines built in the same process share its
// threshold; the last ParseConfig wins. Acceptable for the CLI
// (single engine per invocation); library users running multiple
// engines concurrently with different thresholds need an isolated
// registry (rule.NewRegistry).
type cyclomaticComplexityRule struct {
	threshold int
}

func (r *cyclomaticComplexityRule) Name() string                 { return "high-cyclomatic-complexity" }
func (r *cyclomaticComplexityRule) DefaultSeverity() tt.Severity { return tt.SeverityOff }

func (r *cyclomaticComplexityRule) Check(ctx *rule.AnalysisContext) ([]tt.Issue, error) {
	return analyzeCyclomatic(ctx.WorkingPath, ctx.File, ctx.Fset, r.threshold, ctx.Severity)
}

// ParseConfig accepts {"threshold": <int>}. A missing threshold key
// keeps the rule's existing value; any other shape returns an error
// the engine surfaces as a Warn.
func (r *cyclomaticComplexityRule) ParseConfig(raw any) error {
	m, ok := raw.(map[string]any)
	if !ok {
		return fmt.Errorf("expected map, got %T", raw)
	}
	v, present := m["threshold"]
	if !present {
		return nil
	}
	var threshold int
	switch t := v.(type) {
	case int:
		threshold = t
	case int64:
		threshold = int(t)
	case float64:
		threshold = int(t)
	default:
		return fmt.Errorf("threshold: expected number, got %T", v)
	}
	if threshold <= 0 {
		return fmt.Errorf("threshold must be positive, got %d", threshold)
	}
	r.threshold = threshold
	return nil
}

func analyzeCyclomatic(filename string, file *ast.File, fset *token.FileSet, threshold int, severity tt.Severity) ([]tt.Issue, error) {
	stats := gocyclo.AnalyzeASTFile(file, fset, nil)

	funcNodes := make(map[string]*ast.FuncDecl)
	ast.Inspect(file, func(n ast.Node) bool {
		if fn, ok := n.(*ast.FuncDecl); ok {
			funcNodes[fn.Name.Name] = fn
		}
		return true
	})

	var issues []tt.Issue
	for _, stat := range stats {
		if stat.Complexity <= threshold {
			continue
		}
		funcNode, ok := funcNodes[stat.FuncName]
		if !ok {
			continue
		}
		issues = append(issues, tt.Issue{
			Rule:       "high-cyclomatic-complexity",
			Filename:   filename,
			Start:      fset.Position(funcNode.Pos()),
			End:        fset.Position(funcNode.End()),
			Message:    fmt.Sprintf("function %s has a cyclomatic complexity of %d (threshold %d)", stat.FuncName, stat.Complexity, threshold),
			Suggestion: "consider refactoring this function to reduce its complexity. you can split it into smaller functions or simplify the logic.\n",
			Note:       "high cyclomatic complexity can make the code harder to understand, test, and maintain. aim for a complexity score of 10 or less for most functions.\n",
			Severity:   severity,
		})
	}
	return issues, nil
}
