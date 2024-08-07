package lints

import (
	"go/ast"
	"go/token"

	tt "github.com/gnoswap-labs/lint/internal/types"
	"github.com/gnoswap-labs/lint/internal/branch"
)

// DetectEarlyReturnOpportunities checks for opportunities to use early returns
func DetectEarlyReturnOpportunities(filename string, node *ast.File, fset *token.FileSet) ([]tt.Issue, error) {
    var issues []tt.Issue

	var inspectNode func(n ast.Node) bool
	inspectNode = func(n ast.Node) bool {
		ifStmt, ok := n.(*ast.IfStmt)
		if !ok {
			return true
		}

		chain := analyzeIfElseChain(ifStmt)
		if canUseEarlyReturn(chain) {
			issue := tt.Issue{
				Rule:     "early-return-opportunity",
                Filename: filename,
                Start:    fset.Position(ifStmt.Pos()),
                End:      fset.Position(ifStmt.End()),
                Message:  "This if-else chain can be simplified using early returns",
			}
			issues = append(issues, issue)
		}

		// recursively check the body of the if statement
		ast.Inspect(ifStmt.Body, inspectNode)

		if ifStmt.Else != nil {
			if elseIf, ok := ifStmt.Else.(*ast.IfStmt); ok {
				inspectNode(elseIf)
			} else {
				ast.Inspect(ifStmt.Else, inspectNode)
			}
		}

		return false
	}

	ast.Inspect(node, inspectNode)

    return issues, nil
}

func analyzeIfElseChain(ifStmt *ast.IfStmt) branch.Chain {
    chain := branch.Chain{
        If:   branch.BlockBranch(ifStmt.Body),
        Else: branch.Branch{BranchKind: branch.Empty},
    }

    if ifStmt.Else != nil {
        if elseIfStmt, ok := ifStmt.Else.(*ast.IfStmt); ok {
            chain.Else = analyzeIfElseChain(elseIfStmt).If
        } else if elseBlock, ok := ifStmt.Else.(*ast.BlockStmt); ok {
            chain.Else = branch.BlockBranch(elseBlock)
        }
    }

    return chain
}

func canUseEarlyReturn(chain branch.Chain) bool {
    // If the 'if' branch deviates (returns, breaks, etc.) and there's an else branch,
    // we might be able to use an early return
    return chain.If.BranchKind.Deviates() && !chain.Else.BranchKind.IsEmpty()
}
