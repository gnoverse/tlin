package internal

import (
    "os"
    "path/filepath"
    "testing"
    "time"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestSymbolTableCache(t *testing.T) {
    tmpDir, err := os.MkdirTemp("", "symboltable-cache-test")
    require.NoError(t, err)
    defer os.RemoveAll(tmpDir)

    cacheFile := filepath.Join(tmpDir, ".symbol_cache")

    file1Content := `package test
type TestStruct struct {}
func TestFunc() {}
var TestVar int
`
    file1Path := filepath.Join(tmpDir, "file1.go")
    err = os.WriteFile(file1Path, []byte(file1Content), 0644)
    require.NoError(t, err)

    st := newSymbolTable(cacheFile)
    err = st.updateSymbols(tmpDir)
    require.NoError(t, err)

    assert.True(t, st.isDefined("TestStruct"))
    assert.True(t, st.isDefined("TestFunc"))
    assert.True(t, st.isDefined("TestVar"))

    err = st.saveCache()
    require.NoError(t, err)

    st2 := newSymbolTable(cacheFile)
    err = st2.loadCache()
    require.NoError(t, err)

    assert.True(t, st2.isDefined("TestStruct"))
    assert.True(t, st2.isDefined("TestFunc"))
    assert.True(t, st2.isDefined("TestVar"))

	// update file
    time.Sleep(time.Second)
    file1UpdatedContent := `package test
type TestStruct struct {}
func TestFunc() {}
var TestVar int
func NewFunc() {}
`
    err = os.WriteFile(file1Path, []byte(file1UpdatedContent), 0644)
    require.NoError(t, err)

    err = st2.updateSymbols(tmpDir)
    require.NoError(t, err)

    assert.True(t, st2.isDefined("NewFunc"))

    file2Content := `package test
type AnotherStruct struct {}
`
    file2Path := filepath.Join(tmpDir, "file2.go")
    err = os.WriteFile(file2Path, []byte(file2Content), 0644)
    require.NoError(t, err)

    err = st2.updateSymbols(tmpDir)
    require.NoError(t, err)

    assert.True(t, st2.isDefined("AnotherStruct"))

    _, err = os.Stat(cacheFile)
    assert.NoError(t, err)
}
