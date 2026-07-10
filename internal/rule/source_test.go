package rule

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Plain .go inputs share OriginalPath and WorkingPath; no temp file.
func TestLoadSourceGoFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "demo.go")
	require.NoError(t, os.WriteFile(path, []byte("package demo\n\nfunc f() {}\n"), 0o644))

	src, err := LoadSource(path)
	require.NoError(t, err)
	defer src.Close()

	assert.Equal(t, path, src.OriginalPath)
	assert.Equal(t, path, src.WorkingPath)
	assert.False(t, src.IsTemp(), "plain .go input must not be marked as temp")
	assert.NotNil(t, src.File)
	assert.NotNil(t, src.Fset)
	assert.NotNil(t, src.NolintMgr)
	assert.Equal(t, "package demo\n\nfunc f() {}\n", string(src.Bytes))
}

// .gno inputs get a temp .go in the same directory that Close
// removes; the original .gno file stays put.
func TestLoadSourceGnoFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	gnoPath := filepath.Join(dir, "demo.gno")
	require.NoError(t, os.WriteFile(gnoPath, []byte("package demo\n\nfunc f() {}\n"), 0o644))

	src, err := LoadSource(gnoPath)
	require.NoError(t, err)

	assert.Equal(t, gnoPath, src.OriginalPath)
	assert.NotEqual(t, gnoPath, src.WorkingPath, "WorkingPath must be the temp .go copy")
	assert.True(t, strings.HasSuffix(src.WorkingPath, ".go"))
	assert.True(t, src.IsTemp())

	_, err = os.Stat(src.WorkingPath)
	require.NoError(t, err, "temp .go must exist while Source is open")

	require.NoError(t, src.Close())

	_, err = os.Stat(src.WorkingPath)
	assert.True(t, os.IsNotExist(err), "Close must remove the temp .go file")

	_, err = os.Stat(gnoPath)
	require.NoError(t, err, "original .gno must survive Source.Close")
}

// Calling Close more than once must not error and must not try to
// remove the temp file twice (which would error on the second call).
func TestSourceCloseIsIdempotent(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	gnoPath := filepath.Join(dir, "demo.gno")
	require.NoError(t, os.WriteFile(gnoPath, []byte("package demo\n"), 0o644))

	src, err := LoadSource(gnoPath)
	require.NoError(t, err)

	require.NoError(t, src.Close())
	require.NoError(t, src.Close(), "second Close must be a no-op")
}

// LoadSource of a .gno file whose contents fail to parse must remove
// the temp file before returning the error — the canonical leak case.
func TestSourceCleanupOnError(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	gnoPath := filepath.Join(dir, "broken.gno")
	require.NoError(t, os.WriteFile(gnoPath, []byte("this is not valid go syntax!"), 0o644))

	_, err := LoadSource(gnoPath)
	require.Error(t, err, "broken .gno must fail LoadSource")

	// No leftover temp_*.go in the dir — Close on the partial Source
	// fired during the error path.
	matches, globErr := filepath.Glob(filepath.Join(dir, "temp_*.go"))
	require.NoError(t, globErr)
	assert.Empty(t, matches, "parse failure must not leak a temp .go: found %v", matches)
}

// LoadSourceFromBytes never touches the filesystem and Close is a
// no-op. The Source still exposes File/Fset/NolintMgr for rules.
func TestLoadSourceFromBytes(t *testing.T) {
	t.Parallel()

	src, err := LoadSourceFromBytes([]byte("package demo\n"))
	require.NoError(t, err)

	assert.Equal(t, "", src.OriginalPath)
	assert.Equal(t, "", src.WorkingPath)
	assert.False(t, src.IsTemp())
	assert.NotNil(t, src.File)
	assert.NotNil(t, src.Fset)
	assert.NotNil(t, src.NolintMgr)

	require.NoError(t, src.Close(), "Close on a bytes-source must be a no-op")
}

// Close on a nil Source must not panic — defer src.Close() patterns
// in callers depend on this when LoadSource fails before assigning.
func TestSourceCloseNilSafe(t *testing.T) {
	t.Parallel()

	var src *Source
	assert.NoError(t, src.Close())
}

func BenchmarkLoadSourceGno(b *testing.B) {
	dir := b.TempDir()
	gnoPath := filepath.Join(dir, "main.gno")
	content := []byte(strings.Repeat("// hello world\n", 5000) + "package main\n")
	require.NoError(b, os.WriteFile(gnoPath, content, 0o644))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		src, err := LoadSource(gnoPath)
		if err != nil {
			b.Fatalf("LoadSource: %v", err)
		}
		_ = src.Close()
	}
}
