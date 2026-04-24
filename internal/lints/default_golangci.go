package lints

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
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

func (golangciLintRule) Check(ctx *rule.AnalysisContext) ([]tt.Issue, error) {
	return RunGolangciLint(ctx.WorkingPath, ctx.File, ctx.Fset, ctx.Severity)
}

func ParseFile(filename string, content []byte) (*ast.File, *token.FileSet, error) {
	fset := token.NewFileSet()
	var node *ast.File
	var err error
	if content == nil {
		node, err = parser.ParseFile(fset, filename, nil, parser.ParseComments)
	} else {
		node, err = parser.ParseFile(fset, filename, content, parser.ParseComments)
	}
	if err != nil {
		return nil, nil, err
	}

	return node, fset, nil
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

func RunGolangciLint(filename string, _ *ast.File, _ *token.FileSet, severity tt.Severity) ([]tt.Issue, error) {
	cmd := exec.Command("golangci-lint", "run", "--config=./.golangci.yml", "--out-format=json", filename)
	output, _ := cmd.CombinedOutput()

	var golangciResult golangciOutput

	// golangci-lint occasionally prints non-JSON output for files that
	// contain gno package imports (p/demo, r/demo, std). When the
	// output fails to decode we surface the error to the engine so it
	// logs Warn via WithLogger; partial Issues are not returned because
	// any successful decode would mean no error.
	if err := json.Unmarshal(output, &golangciResult); err != nil {
		return nil, fmt.Errorf("decode golangci-lint output for %s: %w", filename, err)
	}

	issues := make([]tt.Issue, 0, len(golangciResult.Issues))
	for _, gi := range golangciResult.Issues {
		issues = append(issues, tt.Issue{
			Rule:     gi.FromLinter,
			Filename: gi.Pos.Filename, // Use the filename from golangci-lint output
			Start:    token.Position{Filename: gi.Pos.Filename, Line: gi.Pos.Line, Column: gi.Pos.Column},
			End:      token.Position{Filename: gi.Pos.Filename, Line: gi.Pos.Line, Column: gi.Pos.Column + 1},
			Message:  gi.Text,
			Severity: severity,
		})
	}

	return issues, nil
}
