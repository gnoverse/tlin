package analyzer

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

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
		err := os.WriteFile(filepath.Join(tmpDir, fileName), []byte(content), 0o644)
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

// XXX: avoid to use time.Sleep in tests
func TestSymbolTableConcurrency(t *testing.T) {
	st := NewSymbolTable()
	var wg sync.WaitGroup
	wg.Add(4)

	go func() {
		defer wg.Done()
		st.add("symbol1", SymbolInfo{Package: "pkg1", Type: "func", Exported: true})
	}()
	go func() {
		defer wg.Done()
		st.add("symbol2", SymbolInfo{Package: "pkg2", Type: "var", Exported: false})
	}()

	// search for symbols concurrently
	go func() {
		defer wg.Done()
		for !st.IsDefined("symbol1") {
			// wait until symbol1 is added
			time.Sleep(time.Millisecond)
		}
		assert.True(t, st.IsDefined("symbol1"))
	}()
	go func() {
		defer wg.Done()
		for !st.IsDefined("symbol2") {
			// wait until symbol2 is added
			time.Sleep(time.Millisecond)
		}
		assert.True(t, st.IsDefined("symbol2"))
	}()

	wg.Wait()

	allSymbols := st.GetAllSymbols()
	assert.Equal(t, 2, len(allSymbols))
}
