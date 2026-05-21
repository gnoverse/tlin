package migration

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestInterrealmReadonlyBankerDotImportIsSafeEdit(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "token.gno")
	input := `package token

import . "chain/banker"

func Render() string {
	b := NewBanker(BankerTypeReadonly)
	return b.String()
}
`
	require.NoError(t, os.WriteFile(file, []byte(input), 0o644))

	report, err := Run([]string{file}, Options{}, InterrealmMigrators())
	require.NoError(t, err)
	require.Len(t, report.Files, 1)
	require.Len(t, report.Files[0].Edits, 1)
	require.Empty(t, report.Files[0].Findings)
	require.Contains(t, report.Files[0].Diff, "b := NewReadonlyBanker()")
}

func TestInterrealmOriginSendWithAliasedBankerImport(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "pay.gno")
	input := `package pay

import b "chain/banker"

func Pay() {
	amt := b.OriginSend()
	_ = amt
}
`
	require.NoError(t, os.WriteFile(file, []byte(input), 0o644))

	report, err := Run([]string{file}, Options{Apply: true}, InterrealmMigrators())
	require.NoError(t, err)
	require.Len(t, report.Files, 1)
	require.Len(t, report.Files[0].Edits, 1)

	got, err := os.ReadFile(file)
	require.NoError(t, err)
	require.Contains(t, string(got), `"chain/runtime/unsafe"`)
	require.NotContains(t, string(got), `"chain/banker"`)
	require.Contains(t, string(got), "amt := unsafe.OriginSend()")
	require.Contains(t, string(got), "import \"chain/runtime/unsafe\"\n\nfunc")
	require.NotContains(t, string(got), "import \"chain/runtime/unsafe\"\n\n\nfunc")
}

func TestInterrealmDefaultRunIsDryRun(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "readonly.gno")
	input := `package readonly

import "chain/banker"

func Render() string {
	b := banker.NewBanker(banker.BankerTypeReadonly)
	return b.String()
}
`
	require.NoError(t, os.WriteFile(file, []byte(input), 0o644))

	report, err := Run([]string{file}, Options{}, InterrealmMigrators())
	require.NoError(t, err)
	require.Len(t, report.Files, 1)
	require.True(t, report.Files[0].Changed)
	require.Contains(t, report.Files[0].Diff, "banker.NewReadonlyBanker()")

	got, err := os.ReadFile(file)
	require.NoError(t, err)
	require.Equal(t, input, string(got))
}

func TestRunSkipsSymlinkedGnoFiles(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "target.gno")
	link := filepath.Join(dir, "link.gno")
	input := `package demo

func Render() string { return "" }
`
	require.NoError(t, os.WriteFile(target, []byte(input), 0o644))
	require.NoError(t, os.Symlink(target, link))

	report, err := Run([]string{dir}, Options{}, InterrealmMigrators())
	require.NoError(t, err)
	require.Equal(t, 1, report.FilesScanned)
	require.Len(t, report.Files, 1)
	require.Equal(t, target, report.Files[0].Path)
}

func TestInterrealmPHelperFindingOnlyForPPackagePath(t *testing.T) {
	dir := t.TempDir()
	rDir := filepath.Join(dir, "gno.land", "r", "demo")
	pDir := filepath.Join(dir, "gno.land", "p", "demo")
	require.NoError(t, os.MkdirAll(rDir, 0o755))
	require.NoError(t, os.MkdirAll(pDir, 0o755))
	src := `package demo

func Transfer(cur realm) {}
`
	rFile := filepath.Join(rDir, "realm.gno")
	pFile := filepath.Join(pDir, "helper.gno")
	require.NoError(t, os.WriteFile(rFile, []byte(src), 0o644))
	require.NoError(t, os.WriteFile(pFile, []byte(src), 0o644))

	report, err := Run([]string{dir}, Options{}, InterrealmMigrators())
	require.NoError(t, err)

	var rFindings, pFindings []Finding
	for _, file := range report.Files {
		if strings.Contains(file.Path, string(filepath.Separator)+"r"+string(filepath.Separator)) {
			rFindings = file.Findings
		}
		if strings.Contains(file.Path, string(filepath.Separator)+"p"+string(filepath.Separator)) {
			pFindings = file.Findings
		}
	}
	require.Empty(t, rFindings)
	require.Len(t, pFindings, 1)
	require.Equal(t, "interrealm-3.5-p-helper", pFindings[0].Category)
}

func TestInterrealmTellerDetectorIgnoresAlreadyCrossingPackageCalls(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "swap.gno")
	input := `package swap

import foo "gno.land/r/demo/foo"

func Swap(cur realm) {
	foo.Approve(cross, poolAddr, 100)
	foo.Transfer(cross(cur), to, 100)
	foo.TransferFrom(cross, from, to, 100)
}
`
	require.NoError(t, os.WriteFile(file, []byte(input), 0o644))

	report, err := Run([]string{file}, Options{}, InterrealmMigrators())
	require.NoError(t, err)
	require.Empty(t, categories(report, "interrealm-3.6b-teller-method"))
}

func TestInterrealmTellerDetectorFindsKnownLocalTellerReceivers(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "teller.gno")
	input := `package teller

import "gno.land/p/demo/grc/grc20"

func Pay(cur realm) {
	userTeller := grc20.CallerTeller()
	userTeller.Transfer(to, 100)
	GetTokenTeller(path).Approve(spender, 100)
}
`
	require.NoError(t, os.WriteFile(file, []byte(input), 0o644))

	report, err := Run([]string{file}, Options{}, InterrealmMigrators())
	require.NoError(t, err)
	require.Len(t, categories(report, "interrealm-3.6b-teller-method"), 2)
}

func TestInterrealmRecoverIsNotReportedAsCrossRealmPanic(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "recover.gno")
	input := `package recoverdemo

func TestOverflow() {
	defer func() {
		if r := recover(); r == nil {
			panic("expected panic")
		}
	}()
	panic("overflow")
}
`
	require.NoError(t, os.WriteFile(file, []byte(input), 0o644))

	report, err := Run([]string{file}, Options{}, InterrealmMigrators())
	require.NoError(t, err)
	require.Empty(t, categories(report, "interrealm-3.7-cross-panic"))
}

func TestInterrealmRuntimeAPIFindingConfidenceDependsOnCapabilityScope(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "runtime.gno")
	input := `package runtimedemo

import "chain/runtime"

func Crossing(cur realm) {
	_ = runtime.PreviousRealm()
}

func Helper() {
	_ = runtime.CurrentRealm()
}
`
	require.NoError(t, os.WriteFile(file, []byte(input), 0o644))

	report, err := Run([]string{file}, Options{}, InterrealmMigrators())
	require.NoError(t, err)
	findings := categories(report, "interrealm-3.1-runtime-api")
	require.Len(t, findings, 2)
	require.Equal(t, Review, findings[0].Confidence)
	require.Equal(t, Manual, findings[1].Confidence)
}

func TestInterrealmIncludeReviewRewritesRuntimeAPIInCapabilityScope(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "runtime_fix.gno")
	input := `package runtimefix

import "chain/runtime"

func Crossing(cur realm) {
	caller := runtime.PreviousRealm().Address()
	self := runtime.CurrentRealm()
	_, _ = caller, self
}
`
	require.NoError(t, os.WriteFile(file, []byte(input), 0o644))

	report, err := Run([]string{file}, Options{Apply: true, IncludeReview: true}, InterrealmMigrators())
	require.NoError(t, err)
	require.Len(t, report.Files[0].Edits, 2)

	got, err := os.ReadFile(file)
	require.NoError(t, err)
	require.NotContains(t, string(got), `"chain/runtime"`)
	require.Contains(t, string(got), "caller := cur.Previous().Address()")
	require.Contains(t, string(got), "self := cur")
}

func TestInterrealmIncludeReviewAddsCapabilityArguments(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "review_args.gno")
	input := `package reviewargs

import (
	"chain/banker"
	"gno.land/p/demo/grc/grc20"
)

func Crossing(cur realm) {
	b := banker.NewBanker(banker.BankerTypeRealmSend)
	token := grc20.NewToken("Foo", "FOO", 6)
	userTeller := grc20.CallerTeller()
	userTeller.Transfer(to, 100)
	_, _ = b, token
}
`
	require.NoError(t, os.WriteFile(file, []byte(input), 0o644))

	report, err := Run([]string{file}, Options{Apply: true, IncludeReview: true}, InterrealmMigrators())
	require.NoError(t, err)
	require.Len(t, report.Files[0].Edits, 3)

	got, err := os.ReadFile(file)
	require.NoError(t, err)
	require.Contains(t, string(got), "banker.NewBanker(banker.BankerTypeRealmSend, cur)")
	require.Contains(t, string(got), `grc20.NewToken(0, cur, "Foo", "FOO", 6)`)
	require.Contains(t, string(got), "userTeller.Transfer(0, cur, to, 100)")
}

func categories(report Report, category string) []Finding {
	var findings []Finding
	for _, file := range report.Files {
		for _, finding := range file.Findings {
			if finding.Category == category {
				findings = append(findings, finding)
			}
		}
	}
	return findings
}
