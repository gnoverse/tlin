package lints

import (
	"context"
	"go/ast"
	"go/token"

	"github.com/gnolang/tlin/internal/rule"
	tt "github.com/gnolang/tlin/internal/types"
)

func init() {
	rule.Register(paymentGuardRule{})
}

type paymentGuardRule struct{}

func (paymentGuardRule) Name() string                 { return "payment-guard" }
func (paymentGuardRule) DefaultSeverity() tt.Severity { return tt.SeverityError }

func (r paymentGuardRule) Check(ctx *rule.AnalysisContext) ([]tt.Issue, error) {
	return r.CheckPackage(context.Background(), ctx.SinglePackage())
}

// CheckPackage flags IsUser() guards in Gno packages that accept
// OriginSend payments. Package scope is required because payment and
// guard code can live in different .gno files.
func (paymentGuardRule) CheckPackage(_ context.Context, pctx *rule.PackageContext) ([]tt.Issue, error) {
	return checkGnoPackage(pctx, scanPaymentGuard)
}

func scanPaymentGuard(pctx *rule.PackageContext, files []gnoFile) []tt.Issue {
	type isUserCall struct {
		file       gnoFile
		start, end token.Pos
	}
	var calls []isUserCall
	sawOriginSend := false
	for _, f := range files {
		ast.Inspect(f.ast, func(n ast.Node) bool {
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
				calls = append(calls, isUserCall{f, call.Pos(), call.End()})
			}
			return true
		})
	}
	if !sawOriginSend {
		return nil
	}

	const msg = "IsUser() accepts maketx-run ephemeral realms and can be bypassed to consume the OriginSend envelope; use IsUserCall() to guard payments"
	issues := make([]tt.Issue, 0, len(calls))
	for _, c := range calls {
		issues = append(issues, packageIssue(pctx, c.file, "payment-guard", c.start, c.end, msg))
	}
	return issues
}
