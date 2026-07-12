package lints

import (
	"go/ast"
	"go/parser"
	"go/token"

	"github.com/gnolang/tlin/internal/rule"
	tt "github.com/gnolang/tlin/internal/types"
)

// gnoFile is a parsed .gno file with its user-visible path.
type gnoFile struct {
	path string // user-visible (original) path
	ast  *ast.File
	fset *token.FileSet
}

// gnoPackageFiles returns pctx's .gno files grouped by package name.
func gnoPackageFiles(pctx *rule.PackageContext) map[string][]gnoFile {
	pkgs := map[string][]gnoFile{}
	for i, wp := range pctx.WorkingPaths {
		if wp == "" {
			continue
		}
		orig := pctx.RemapFilename(wp)
		if !isGnoFile(orig) {
			continue
		}
		var (
			f    *ast.File
			fset *token.FileSet
		)
		if i < len(pctx.Files) && pctx.Files[i] != nil {
			f, fset = pctx.Files[i], pctx.Fsets[i]
		} else {
			fset = token.NewFileSet()
			var err error
			f, err = parser.ParseFile(fset, wp, nil, parser.SkipObjectResolution)
			if err != nil {
				continue
			}
		}
		pkgs[f.Name.Name] = append(pkgs[f.Name.Name], gnoFile{path: orig, ast: f, fset: fset})
	}
	return pkgs
}

// checkGnoPackage runs scan once per Gno package in pctx.
func checkGnoPackage(pctx *rule.PackageContext, scan func(*rule.PackageContext, []gnoFile) []tt.Issue) ([]tt.Issue, error) {
	pkgs := gnoPackageFiles(pctx)
	issues := make([]tt.Issue, 0, len(pkgs))
	for _, files := range pkgs {
		issues = append(issues, scan(pctx, files)...)
	}
	return issues, nil
}

// packageIssue builds an issue for a package-scope rule.
func packageIssue(pctx *rule.PackageContext, f gnoFile, ruleName string, start, end token.Pos, msg string) tt.Issue {
	s := f.fset.Position(start)
	s.Filename = f.path
	e := f.fset.Position(end)
	e.Filename = f.path
	return tt.Issue{
		Rule:     ruleName,
		Filename: f.path,
		Start:    s,
		End:      e,
		Message:  msg,
		Severity: pctx.Severity,
	}
}
