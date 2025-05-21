package lints

import (
	"fmt"
	"go/ast"
	"go/token"

	"github.com/gnolang/tlin/internal/checker"
	"github.com/gnolang/tlin/internal/lints/utils"
	tt "github.com/gnolang/tlin/internal/types"
)

func register() *checker.DeprecatedFuncChecker {
	deprecated := checker.NewDeprecatedFuncChecker()

	// functions
	deprecated.Register("std", "GetCallerAt", "std.CallerAt")
	deprecated.Register("std", "GetOrigSend", "std.OriginSend")
	deprecated.Register("std", "GetOrigCaller", "std.OriginCaller")
	deprecated.Register("std", "TestSetOrigCaller", "std.TestSetOriginCaller")
	deprecated.Register("std", "TestSetOrigSend", "std.TestSetOriginSend")
	deprecated.Register("std", "TestSetOrigPkgAddr", "std.TestSetOriginPkgAddress")
	deprecated.Register("std", "PrevRealm", "std.PreviousRealm")
	deprecated.Register("std", "GetChainID", "std.ChainID")
	deprecated.Register("std", "GetBanker", "std.NewBanker")
	deprecated.Register("std", "GetChainDomain", "std.ChainDomain")
	deprecated.Register("std", "GetHeight", "std.ChainHeight")

	// chaining methods
	deprecated.RegisterMethod("std", "Address", "Addr", "Address")
	deprecated.RegisterMethod("std", "Realm", "Addr", "Address")

	return deprecated
}

func DetectDeprecatedFunctions(
	filename string,
	node *ast.File,
	fset *token.FileSet,
	severity tt.Severity,
) ([]tt.Issue, error) {
	deprecated := register()

	imports := extractDeprecatedImports(node)
	if len(imports) == 0 {
		return nil, nil
	}

	hasDeprecatedPackage := false
	for imp := range imports {
		if deprecatedPackages[imp] {
			hasDeprecatedPackage = true
			break
		}
	}

	if !hasDeprecatedPackage {
		return nil, nil
	}

	dfuncs, err := deprecated.Check(filename, node, fset)
	if err != nil {
		return nil, err
	}

	issues := make([]tt.Issue, 0, len(dfuncs))
	for _, df := range dfuncs {
		issues = append(issues, tt.Issue{
			Rule:       "deprecated",
			Filename:   filename,
			Start:      df.Start,
			End:        df.End,
			Message:    createDeprecationMessage(df),
			Suggestion: df.Alternative,
			Severity:   severity,
		})
	}

	return issues, nil
}

func createDeprecationMessage(df checker.DeprecatedFunc) string {
	msg := "Use of deprecated function"
	if df.Alternative != "" {
		msg = fmt.Sprintf("%s. please use %s instead.", msg, df.Alternative)
		return msg
	}
	msg = fmt.Sprintf("%s. please remove it.", msg)
	return msg
}

type pkgContainsDeprecatedMap map[string]bool

var deprecatedPackages = pkgContainsDeprecatedMap{
	"std": true,
}

func extractDeprecatedImports(node *ast.File) pkgContainsDeprecatedMap {
	return utils.ExtractImports(node, func(path string) bool {
		return true
	})
}
