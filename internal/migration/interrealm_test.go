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
