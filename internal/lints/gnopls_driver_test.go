package lints

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGnoDriverEnv(t *testing.T) {
	t.Parallel()

	missingBinary := func(string) (string, error) { return "", errors.New("not found") }
	binaryAt := func(name string) func(string) (string, error) {
		return func(string) (string, error) { return name, nil }
	}
	envFrom := func(m map[string]string) func(string) string {
		return func(k string) string { return m[k] }
	}
	stubShim := func(want string) func(string) (string, error) {
		return func(string) (string, error) { return want, nil }
	}
	failingShim := func(string) (string, error) { return "", errors.New("disk full") }

	t.Run("returns nil when gnopls is not on PATH", func(t *testing.T) {
		t.Parallel()
		env := gnoDriverEnv(driverDeps{
			lookPath:   missingBinary,
			getenv:     envFrom(map[string]string{"GNOROOT": "/gno"}),
			ensureShim: stubShim("/never/used"),
		})
		assert.Nil(t, env)
	})

	t.Run("returns nil when GNOROOT is unset", func(t *testing.T) {
		t.Parallel()
		env := gnoDriverEnv(driverDeps{
			lookPath:   binaryAt("/usr/bin/gnopls"),
			getenv:     envFrom(nil),
			ensureShim: stubShim("/never/used"),
		})
		assert.Nil(t, env)
	})

	t.Run("returns nil when shim creation fails", func(t *testing.T) {
		t.Parallel()
		env := gnoDriverEnv(driverDeps{
			lookPath:   binaryAt("/usr/bin/gnopls"),
			getenv:     envFrom(map[string]string{"GNOROOT": "/gno"}),
			ensureShim: failingShim,
		})
		assert.Nil(t, env)
	})

	t.Run("wires GOPACKAGESDRIVER and GNOROOT when both present", func(t *testing.T) {
		t.Parallel()
		env := gnoDriverEnv(driverDeps{
			lookPath:   binaryAt("/usr/bin/gnopls"),
			getenv:     envFrom(map[string]string{"GNOROOT": "/gno"}),
			ensureShim: stubShim("/tmp/shim"),
		})
		assert.Equal(t, []string{"GOPACKAGESDRIVER=/tmp/shim", "GNOROOT=/gno"}, env)
	})

	t.Run("propagates GNOBUILTIN when set", func(t *testing.T) {
		t.Parallel()
		env := gnoDriverEnv(driverDeps{
			lookPath:   binaryAt("/usr/bin/gnopls"),
			getenv:     envFrom(map[string]string{"GNOROOT": "/gno", "GNOBUILTIN": "/gno/builtin"}),
			ensureShim: stubShim("/tmp/shim"),
		})
		assert.Contains(t, env, "GNOBUILTIN=/gno/builtin")
	})
}

func TestWriteGnoplsShim_Unix(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "windows" {
		t.Skip("unix shim format only")
	}
	dir := t.TempDir()
	path, err := writeGnoplsShim(dir, "linux", 9001, "/usr/local/bin/gnopls")
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(dir, "tlin-gnopls-driver-9001"), path)

	body, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(string(body), "#!/bin/sh\n"), "shim must start with sh shebang")
	assert.Contains(t, string(body), `"/usr/local/bin/gnopls" resolve "$@"`)

	info, err := os.Stat(path)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o700), info.Mode().Perm())
}

func TestWriteGnoplsShim_Windows(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path, err := writeGnoplsShim(dir, "windows", 9002, `C:\bin\gnopls.exe`)
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(dir, "tlin-gnopls-driver-9002.cmd"), path)

	body, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Contains(t, string(body), "@echo off")
	assert.Contains(t, string(body), `"C:\bin\gnopls.exe" resolve %*`)
}

func TestWriteGnoplsShim_FailsOnUnwritableDir(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "windows" {
		t.Skip("permission semantics differ on windows")
	}
	dir := filepath.Join(t.TempDir(), "missing", "deeper")
	_, err := writeGnoplsShim(dir, "linux", 9003, "/bin/gnopls")
	require.Error(t, err)
}
