package lints

import (
	"fmt"
	"go/ast"
	"go/token"

	"github.com/gnolang/tlin/internal/checker"
	tt "github.com/gnolang/tlin/internal/types"
)

func DetectDeprecatedFunctions(
	filename string,
	node *ast.File,
	fset *token.FileSet,
) ([]tt.Issue, error) {
	deprecated := checker.NewDeprecatedFuncChecker()

	deprecated.Register("std", "SetOrigCaller", "std.PrevRealm")
	deprecated.Register("std", "GetOrigCaller", "std.PrevRealm")
	deprecated.Register("std", "TestSetOrigCaller", "")

	dfuncs, err := deprecated.Check(filename, node, fset)
	if err != nil {
		return nil, err
	}

	issues := make([]tt.Issue, 0, len(dfuncs))
	for _, df := range dfuncs {
		issues = append(issues, tt.Issue{
			Rule:       "deprecated",
			Filename:   filename,
			Start:      df.Position,
			End:        df.Position,
			Message:    createDeprecationMessage(df),
			Suggestion: df.Alternative,
		})
	}

	return issues, nil
}

func createDeprecationMessage(df checker.DeprecatedFunc) string {
	msg := "Use of deprecated function"
	if df.Alternative != "" {
		msg = fmt.Sprintf("%s. Please use %s instead.", msg, df.Alternative)
		return msg
	}
	msg = fmt.Sprintf("%s. Please remove it.", msg)
	return msg
}
