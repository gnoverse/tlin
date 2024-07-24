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

	testFiles := map[string]string{
		"file1.go": `
package pkg1

type TestStruct struct{}

func ExportedFunc() {}

var privateVar int
`,
		"file2.go": `
package pkg2

import "fmt"

func init() {
	fmt.Println("Initializing pkg2")
}

const ExportedConst = 42
`,
	}

	for fileName, content := range testFiles {
		err := os.WriteFile(filepath.Join(tmpDir, fileName), []byte(content), 0644)
		require.NoError(t, err)
	}

	st, err := BuildSymbolTable(tmpDir)
	require.NoError(t, err)

	t.Run("Symbol existence", func(t *testing.T) {
		assert.True(t, st.IsDefined("TestStruct"))
		assert.True(t, st.IsDefined("ExportedFunc"))
		assert.True(t, st.IsDefined("privateVar"))
		assert.True(t, st.IsDefined("init"))
		assert.True(t, st.IsDefined("ExportedConst"))
		assert.False(t, st.IsDefined("NonExistentSymbol"))
	})

	t.Run("Symbol info", func(t *testing.T) {
		testStructInfo, exists := st.GetSymbolInfo("TestStruct")
		assert.True(t, exists)
		assert.Equal(t, "pkg1", testStructInfo.Package)
		assert.Equal(t, "type", testStructInfo.Type)
		assert.True(t, testStructInfo.Exported)

		privateVarInfo, exists := st.GetSymbolInfo("privateVar")
		assert.True(t, exists)
		assert.Equal(t, "pkg1", privateVarInfo.Package)
		assert.Equal(t, "var", privateVarInfo.Type)
		assert.False(t, privateVarInfo.Exported)

		exportedConstInfo, exists := st.GetSymbolInfo("ExportedConst")
		assert.True(t, exists)
		assert.Equal(t, "pkg2", exportedConstInfo.Package)
		assert.Equal(t, "var", exportedConstInfo.Type) // Note: consts are treated as vars
		assert.True(t, exportedConstInfo.Exported)
	})

	t.Run("All symbols", func(t *testing.T) {
		allSymbols := st.GetAllSymbols()
		assert.Equal(t, 5, len(allSymbols)) // TestStruct, ExportedFunc, privateVar, init, ExportedConst

		expectedSymbols := []string{"TestStruct", "ExportedFunc", "privateVar", "init", "ExportedConst"}
		for _, symbol := range expectedSymbols {
			_, exists := allSymbols[symbol]
			assert.True(t, exists, "Symbol %s should exist in all symbols", symbol)
		}
	})
}
