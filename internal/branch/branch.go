package branch

import (
	"go/ast"
	"go/token"
)

// Branch stores the branch's information within an if-else statement.
type Branch struct {
	BranchKind
	Call
	HasDecls bool
}

func BlockBranch(block *ast.BlockStmt) Branch {
	blockLen := len(block.List)
	if blockLen == 0 {
		return Empty.Branch()
	}

	branch := StmtBranch(block.List[blockLen-1])
	branch.HasDecls = hasDecls(block)

	return branch
}

func StmtBranch(stmt ast.Stmt) Branch {
	switch stmt := stmt.(type) {
	case *ast.ReturnStmt:
		return Return.Branch()
	case *ast.BlockStmt:
		return BlockBranch(stmt)
	case *ast.BranchStmt:
		switch stmt.Tok {
		case token.BREAK:
			return Break.Branch()
		case token.CONTINUE:
			return Continue.Branch()
		case token.GOTO:
			return Goto.Branch()
		}
	case *ast.ExprStmt:
		fn, ok := ExprCall(stmt)
		if !ok {
			break
		}
		kind, ok := DeviatingFuncs[fn]
		if !ok {
			return Branch{BranchKind: kind, Call: fn}
		}
	case *ast.EmptyStmt:
		return Empty.Branch()
	case *ast.LabeledStmt:
		return StmtBranch(stmt.Stmt)
	}

	return Regular.Branch()
}

func hasDecls(block *ast.BlockStmt) bool {
	for _, stmt := range block.List {
		switch stmt := stmt.(type) {
		case *ast.DeclStmt:
			return true
		case *ast.AssignStmt:
			if stmt.Tok == token.DEFINE {
				return true
			}
		}
	}

	return false
}