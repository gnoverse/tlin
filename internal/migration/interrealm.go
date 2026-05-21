package migration

import (
	"go/ast"
	"go/token"
	"strings"
)

func InterrealmMigrators() []Migrator {
	return []Migrator{
		interrealmMigrator{},
	}
}

type interrealmMigrator struct{}

func (interrealmMigrator) Name() string { return "interrealm" }

func (interrealmMigrator) Run(ctx *FileContext) ([]Edit, []Finding) {
	aliases := importAliases(ctx.File)
	bankerAliases := aliasSet(aliases["chain/banker"])
	runtimeAliases := aliasSet(aliases["chain/runtime"])
	grc20Aliases := aliasSet(aliases["gno.land/p/demo/grc/grc20"])
	importedAliases := importedAliasSet(aliases)
	parents := parentMap(ctx.File)
	var edits []Edit
	var findings []Finding
	ast.Inspect(ctx.File, func(n ast.Node) bool {
		if n == nil || ignoredNode(ctx, n) {
			return true
		}
		switch x := n.(type) {
		case *ast.CallExpr:
			if sel, ok := x.Fun.(*ast.SelectorExpr); ok {
				pkg, ok := sel.X.(*ast.Ident)
				if ok {
					switch {
					case bankerAliases[pkg.Name] && sel.Sel.Name == "NewBanker":
						if len(x.Args) == 1 && isReadonlyBankerArg(x.Args[0], bankerAliases) {
							pos := ctx.FileSet.Position(x.Pos())
							edits = append(edits, Edit{
								Start:      x.Pos(),
								End:        x.End(),
								NewText:    pkg.Name + ".NewReadonlyBanker()",
								Category:   "interrealm-3.3-readonly-banker",
								Confidence: Safe,
								Rationale:  "NewBanker(BankerTypeReadonly) has a direct replacement.",
								Position:   NewPosition(pos),
							})
						} else if len(x.Args) == 1 {
							if edit, ok := reviewArgAppendEdit(ctx, parents, x, "interrealm-3.2-new-banker", "banker.NewBanker now requires a realm capability argument."); ok {
								edits = append(edits, edit)
							} else {
								findings = append(findings, finding(ctx, x.Pos(), "interrealm-3.2-new-banker", Review, "banker.NewBanker now requires a realm capability argument.", "Thread cur realm/rlm realm to this scope and call banker.NewBanker(bt, cur)."))
							}
						}
					case bankerAliases[pkg.Name] && sel.Sel.Name == "OriginSend":
						pos := ctx.FileSet.Position(sel.Pos())
						edits = append(edits, Edit{
							Start:      sel.Pos(),
							End:        sel.End(),
							NewText:    "unsafe.OriginSend",
							Category:   "interrealm-3.4-origin-send",
							Confidence: Safe,
							Rationale:  "banker.OriginSend moved to chain/runtime/unsafe.",
							Position:   NewPosition(pos),
						})
					case runtimeAliases[pkg.Name] && removedRuntimeAPI(sel.Sel.Name):
						if edit, ok := reviewRuntimeAPIEdit(ctx, parents, x, sel.Sel.Name); ok {
							edits = append(edits, edit)
						} else {
							findings = append(findings, runtimeAPIFinding(ctx, parents, x, sel.Pos(), sel.Sel.Name))
						}
					case grc20Aliases[pkg.Name] && sel.Sel.Name == "NewToken" && len(x.Args) == 3:
						if edit, ok := reviewArgPrependEdit(ctx, parents, x, "interrealm-3.6a-grc20-new-token", "grc20.NewToken now requires origin and realm capability arguments."); ok {
							edits = append(edits, edit)
						} else {
							findings = append(findings, finding(ctx, x.Pos(), "interrealm-3.6a-grc20-new-token", Review, "grc20.NewToken now requires origin and realm capability arguments.", "Thread cur realm/rlm realm to this scope and call grc20.NewToken(0, cur, ...)."))
						}
					case sel.Sel.Name == "Origin":
						findings = append(findings, finding(ctx, sel.Pos(), "interrealm-3.8-realm-origin", Manual, "realm.Origin() was removed from the uverse realm interface.", "Remove this call or replace the design with explicit origin data."))
					}
				}
				if shouldReportTellerCall(x, sel, importedAliases, parents) {
					if edit, ok := reviewArgPrependEdit(ctx, parents, x, "interrealm-3.6b-teller-method", "Teller/RealmTeller APIs changed for realm-capability-aware calls."); ok {
						edits = append(edits, edit)
					} else {
						findings = append(findings, finding(ctx, sel.Pos(), "interrealm-3.6b-teller-method", Manual, "Teller/RealmTeller APIs changed for realm-capability-aware calls.", "Check whether this receiver is a Teller and migrate the call manually with the appropriate cur/rlm value."))
					}
				}
			}
			if id, ok := x.Fun.(*ast.Ident); ok {
				switch {
				case runtimeAliases["."] && removedRuntimeAPI(id.Name):
					if edit, ok := reviewRuntimeAPIEdit(ctx, parents, x, id.Name); ok {
						edits = append(edits, edit)
					} else {
						findings = append(findings, runtimeAPIFinding(ctx, parents, x, id.Pos(), id.Name))
					}
				case bankerAliases["."] && id.Name == "NewBanker" && len(x.Args) == 1:
					if isReadonlyBankerArg(x.Args[0], bankerAliases) {
						pos := ctx.FileSet.Position(x.Pos())
						edits = append(edits, Edit{
							Start:      x.Pos(),
							End:        x.End(),
							NewText:    "NewReadonlyBanker()",
							Category:   "interrealm-3.3-readonly-banker",
							Confidence: Safe,
							Rationale:  "NewBanker(BankerTypeReadonly) has a direct replacement.",
							Position:   NewPosition(pos),
						})
					} else {
						if edit, ok := reviewArgAppendEdit(ctx, parents, x, "interrealm-3.2-new-banker", "banker.NewBanker now requires a realm capability argument."); ok {
							edits = append(edits, edit)
						} else {
							findings = append(findings, finding(ctx, x.Pos(), "interrealm-3.2-new-banker", Review, "banker.NewBanker now requires a realm capability argument.", "Thread cur realm/rlm realm to this scope and call NewBanker(bt, cur)."))
						}
					}
				case grc20Aliases["."] && id.Name == "NewToken" && len(x.Args) == 3:
					if edit, ok := reviewArgPrependEdit(ctx, parents, x, "interrealm-3.6a-grc20-new-token", "grc20.NewToken now requires origin and realm capability arguments."); ok {
						edits = append(edits, edit)
					} else {
						findings = append(findings, finding(ctx, x.Pos(), "interrealm-3.6a-grc20-new-token", Review, "grc20.NewToken now requires origin and realm capability arguments.", "Thread cur realm/rlm realm to this scope and call NewToken(0, cur, ...)."))
					}
				}
			}
		case *ast.FuncDecl:
			if isPPackagePath(ctx.Path) && firstParamIsRealm(x.Type) {
				findings = append(findings, finding(ctx, x.Pos(), "interrealm-3.5-p-helper", Manual, "/p/ helpers cannot use a first-argument cur realm crossing signature.", "For /p/ helpers, use (_ int, rlm realm, ...) and validate rlm.IsCurrent() before authorization."))
			}
		}
		return true
	})
	return edits, findings
}

func runtimeAPIFinding(ctx *FileContext, parents map[ast.Node]ast.Node, node ast.Node, pos token.Pos, name string) Finding {
	confidence := Manual
	if name != "OriginCaller" {
		if _, ok := ResolveCapability(node, parents); ok {
			confidence = Review
		}
	}
	return finding(ctx, pos, "interrealm-3.1-runtime-api", confidence, "chain/runtime caller APIs were removed.", "Prefer cur realm threading and cur.Previous(); use chain/runtime/unsafe only for intentional tx-origin behavior.")
}

func reviewRuntimeAPIEdit(ctx *FileContext, parents map[ast.Node]ast.Node, call *ast.CallExpr, name string) (Edit, bool) {
	if !ctx.IncludeReview || name == "OriginCaller" {
		return Edit{}, false
	}
	cap, ok := ResolveCapability(call, parents)
	if !ok {
		return Edit{}, false
	}
	var replacement string
	switch name {
	case "CurrentRealm":
		replacement = cap.Name
	case "PreviousRealm":
		replacement = cap.Name + ".Previous()"
	default:
		return Edit{}, false
	}
	return Edit{
		Start:      call.Pos(),
		End:        call.End(),
		NewText:    replacement,
		Category:   "interrealm-3.1-runtime-api",
		Confidence: Review,
		Rationale:  "runtime caller APIs can be replaced by the in-scope realm capability.",
		Position:   NewPosition(ctx.FileSet.Position(call.Pos())),
	}, true
}

func reviewArgAppendEdit(ctx *FileContext, parents map[ast.Node]ast.Node, call *ast.CallExpr, category, rationale string) (Edit, bool) {
	if !ctx.IncludeReview || len(call.Args) == 0 {
		return Edit{}, false
	}
	cap, ok := ResolveCapability(call, parents)
	if !ok {
		return Edit{}, false
	}
	pos := call.Args[len(call.Args)-1].End()
	return Edit{
		Start:      pos,
		End:        pos,
		NewText:    ", " + cap.Name,
		Category:   category,
		Confidence: Review,
		Rationale:  rationale,
		Position:   NewPosition(ctx.FileSet.Position(call.Pos())),
	}, true
}

func reviewArgPrependEdit(ctx *FileContext, parents map[ast.Node]ast.Node, call *ast.CallExpr, category, rationale string) (Edit, bool) {
	if !ctx.IncludeReview {
		return Edit{}, false
	}
	cap, ok := ResolveCapability(call, parents)
	if !ok {
		return Edit{}, false
	}
	pos := call.Lparen + 1
	return Edit{
		Start:      pos,
		End:        pos,
		NewText:    "0, " + cap.Name + ", ",
		Category:   category,
		Confidence: Review,
		Rationale:  rationale,
		Position:   NewPosition(ctx.FileSet.Position(call.Pos())),
	}, true
}

func finding(ctx *FileContext, pos token.Pos, category string, confidence Confidence, msg, suggestion string) Finding {
	return Finding{
		Category:   category,
		Confidence: confidence,
		Position:   NewPosition(ctx.FileSet.Position(pos)),
		Message:    msg,
		Suggestion: suggestion,
	}
}

func aliasSet(names []string) map[string]bool {
	set := make(map[string]bool, len(names))
	for _, name := range names {
		set[name] = true
	}
	return set
}

func importedAliasSet(aliases map[string][]string) map[string]bool {
	set := map[string]bool{}
	for _, names := range aliases {
		for _, name := range names {
			if name != "." && name != "_" {
				set[name] = true
			}
		}
	}
	return set
}

func shouldReportTellerCall(call *ast.CallExpr, sel *ast.SelectorExpr, importedAliases map[string]bool, parents map[ast.Node]ast.Node) bool {
	if !tellerMethod(sel.Sel.Name) || firstArgIsCrossingMarker(call) {
		return false
	}
	if id, ok := sel.X.(*ast.Ident); ok && importedAliases[id.Name] {
		return false
	}
	fn := enclosingFunc(call, parents)
	vars := knownTellerVars(fn)
	return knownTellerExpr(sel.X, vars)
}

func removedRuntimeAPI(name string) bool {
	return name == "CurrentRealm" || name == "PreviousRealm" || name == "OriginCaller"
}

func tellerMethod(name string) bool {
	return name == "Transfer" || name == "Approve" || name == "TransferFrom"
}

func isReadonlyBankerArg(expr ast.Expr, bankerAliases map[string]bool) bool {
	if sel, ok := expr.(*ast.SelectorExpr); ok {
		id, ok := sel.X.(*ast.Ident)
		return ok && bankerAliases[id.Name] && sel.Sel.Name == "BankerTypeReadonly"
	}
	if id, ok := expr.(*ast.Ident); ok {
		return bankerAliases["."] && id.Name == "BankerTypeReadonly"
	}
	return false
}

func firstParamIsRealm(t *ast.FuncType) bool {
	if t == nil || t.Params == nil || len(t.Params.List) == 0 {
		return false
	}
	first := t.Params.List[0]
	id, ok := first.Type.(*ast.Ident)
	return ok && id.Name == "realm"
}

func ignoredNode(ctx *FileContext, n ast.Node) bool {
	start := ctx.FileSet.Position(n.Pos()).Line
	end := ctx.FileSet.Position(n.End()).Line
	for _, group := range ctx.File.Comments {
		for _, comment := range group.List {
			line := ctx.FileSet.Position(comment.Pos()).Line
			if line >= start-1 && line <= end && containsMigrateIgnore(comment.Text) {
				return true
			}
		}
	}
	return false
}

func containsMigrateIgnore(text string) bool {
	return text == "//tlin:migrate-ignore" || text == "// tlin:migrate-ignore"
}

func isPPackagePath(path string) bool {
	for _, sep := range []string{"/", "\\"} {
		if strings.Contains(path, sep+"p"+sep) {
			return true
		}
	}
	return false
}
