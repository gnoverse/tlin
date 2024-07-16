package internal

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSymbolTable(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "symboltable-test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// generate test files
	file1Content := `package test
type TestStruct struct {}
func TestFunc() {}
var TestVar int
`
	err = os.WriteFile(filepath.Join(tmpDir, "file1.go"), []byte(file1Content), 0o644)
	require.NoError(t, err)

	file2Content := `package test
type AnotherStruct struct {}
func AnotherFunc() {}
`
	err = os.WriteFile(filepath.Join(tmpDir, "file2.go"), []byte(file2Content), 0o644)
	require.NoError(t, err)

	// create symbol table
	st, err := BuildSymbolTable(tmpDir)
	require.NoError(t, err)

	assert.True(t, st.IsDefined("TestStruct"))
	assert.True(t, st.IsDefined("TestFunc"))
	assert.True(t, st.IsDefined("TestVar"))
	assert.True(t, st.IsDefined("AnotherStruct"))
	assert.True(t, st.IsDefined("AnotherFunc"))
	assert.False(t, st.IsDefined("NonExistentSymbol"))

	// validate symbol file paths
	path, exists := st.GetSymbolPath("TestStruct")
	assert.True(t, exists)
	assert.Equal(t, filepath.Join(tmpDir, "file1.go"), path)

	path, exists = st.GetSymbolPath("AnotherFunc")
	assert.True(t, exists)
	assert.Equal(t, filepath.Join(tmpDir, "file2.go"), path)

	_, exists = st.GetSymbolPath("NonExistentSymbol")
	assert.False(t, exists)
}
