package lints

import (
	"context"
	"encoding/json"
	"fmt"
	"go/token"
	"os/exec"

	"github.com/gnolang/tlin/internal/rule"
	tt "github.com/gnolang/tlin/internal/types"
)

func init() {
	rule.Register(golangciLintRule{})
}

type golangciLintRule struct{}

func (golangciLintRule) Name() string                 { return "golangci-lint" }
func (golangciLintRule) DefaultSeverity() tt.Severity { return tt.SeverityWarning }

// Check is the single-file fallback for non-engine callers; engine
// dispatch goes straight to CheckPackage with the live ctx.
func (r golangciLintRule) Check(ctx *rule.AnalysisContext) ([]tt.Issue, error) {
	return r.CheckPackage(context.Background(), ctx.SinglePackage())
}

type golangciOutput struct {
	Issues []struct {
		FromLinter string `json:"FromLinter"`
		Text       string `json:"Text"`
		Pos        struct {
			Filename string `json:"Filename"`
			Line     int    `json:"Line"`
			Column   int    `json:"Column"`
		} `json:"Pos"`
	} `json:"Issues"`
}

// CheckPackage runs golangci-lint once against the package directory
// and filters its output to files in pctx scope. ctx is wired into
// exec.CommandContext so a cancelled run kills the child too.
func (golangciLintRule) CheckPackage(ctx context.Context, pctx *rule.PackageContext) ([]tt.Issue, error) {
	target := pctx.Dir
	if target == "" && len(pctx.WorkingPaths) > 0 {
		target = pctx.WorkingPaths[0]
	}

	cmd := exec.CommandContext(ctx, "golangci-lint", "run", "--config=./.golangci.yml", "--out-format=json", target)
	output, _ := cmd.CombinedOutput()

	var golangciResult golangciOutput

	// golangci-lint occasionally prints non-JSON output for files that
	// contain gno package imports (p/demo, r/demo, std). When the
	// output fails to decode we surface the error to the engine so it
	// logs Warn via WithLogger; partial Issues are not returned because
	// any successful decode would mean no error.
	if err := json.Unmarshal(output, &golangciResult); err != nil {
		return nil, fmt.Errorf("decode golangci-lint output for %s: %w", target, err)
	}

	issues := make([]tt.Issue, 0, len(golangciResult.Issues))
	for _, gi := range golangciResult.Issues {
		if !pctx.InScope(gi.Pos.Filename) {
			continue
		}
		filename := pctx.RemapFilename(gi.Pos.Filename)
		issues = append(issues, tt.Issue{
			Rule:     gi.FromLinter,
			Filename: filename,
			Start:    token.Position{Filename: filename, Line: gi.Pos.Line, Column: gi.Pos.Column},
			End:      token.Position{Filename: filename, Line: gi.Pos.Line, Column: gi.Pos.Column + 1},
			Message:  gi.Text,
			Severity: pctx.Severity,
		})
	}

	return issues, nil
}
