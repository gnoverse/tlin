package lints

import (
	"go/ast"

	"github.com/gnolang/tlin/internal/rule"
	tt "github.com/gnolang/tlin/internal/types"
)

func init() {
	rule.Register(paymentGuardRule{})
}

type paymentGuardRule struct{}

func (paymentGuardRule) Name() string                 { return "payment-guard" }
func (paymentGuardRule) DefaultSeverity() tt.Severity { return tt.SeverityError }

func (paymentGuardRule) Check(ctx *rule.AnalysisContext) ([]tt.Issue, error) {
	return DetectPaymentGuard(ctx)
}

// DetectPaymentGuard flags IsUser() caller guards in files that accept
// payment via OriginSend. IsUser() accepts maketx-run ephemeral realms,
// which can consume the origin-send envelope before calling the
// function and bypass the payment check; the guard must be
// IsUserCall().
func DetectPaymentGuard(ctx *rule.AnalysisContext) ([]tt.Issue, error) {
	// matches by bare method names, which could collide in plain Go
	if !isGnoFile(ctx.OriginalPath) {
		return nil, nil
	}

	var issues []tt.Issue
	sawOriginSend := false
	ast.Inspect(ctx.File, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		switch sel.Sel.Name {
		case "OriginSend":
			sawOriginSend = true
		case "IsUser":
			issue := ctx.NewIssue("payment-guard", call.Pos(), call.End())
			issue.Message = "IsUser() accepts maketx-run ephemeral realms and can be bypassed to consume the OriginSend envelope; use IsUserCall() to guard payments"
			issues = append(issues, issue)
		}
		return true
	})

	if !sawOriginSend {
		return nil, nil
	}
	return issues, nil
}
