package lints

import (
	"bytes"
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
	if env := cachedDriverEnv(); env != nil {
		cmd.Env = env
	}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	stdout, _ := cmd.Output()

	result, err := decodeGolangciOutput(stdout)
	if err != nil {
		return nil, fmt.Errorf("decode golangci-lint output for %s: %w (stderr: %s)", target, err, snippet(stderr.Bytes(), 200))
	}

	issues := make([]tt.Issue, 0, len(result.Issues))
	for _, gi := range result.Issues {
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

// decodeGolangciOutput parses golangci-lint stdout, treating empty
// input as "no issues". golangci-lint exits without writing JSON
// when go/packages fails to load the target — common for .gno-only
// directories whose temp .go file imports gno.land/... paths the
// stock Go loader cannot resolve — and erroring there would log a
// warn per file across the entire package.
func decodeGolangciOutput(stdout []byte) (golangciOutput, error) {
	stdout = bytes.TrimSpace(stdout)
	var result golangciOutput
	if len(stdout) == 0 {
		return result, nil
	}
	if err := json.Unmarshal(stdout, &result); err != nil {
		return result, err
	}
	return result, nil
}

func snippet(b []byte, n int) string {
	b = bytes.TrimSpace(b)
	if len(b) == 0 {
		return ""
	}
	if len(b) > n {
		b = b[:n]
	}
	return string(b)
}
