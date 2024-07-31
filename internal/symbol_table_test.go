package internal

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
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
func (ts TestStruct) TestMethod() {}
`
	file1Path := filepath.Join(tmpDir, "file1.go")
	err = os.WriteFile(file1Path, []byte(file1Content), 0o644)
	require.NoError(t, err)

	file2Content := `package test
type AnotherStruct struct {}
func AnotherFunc() {}
`
	file2Path := filepath.Join(tmpDir, "file2.go")
	err = os.WriteFile(file2Path, []byte(file2Content), 0o644)
	require.NoError(t, err)

	st, err := BuildSymbolTable(tmpDir)
	require.NoError(t, err)

	testCases := []struct {
		symbol   string
		expected bool
		symType  SymbolType
		filePath string
	}{
		{"test.TestStruct", true, Type, file1Path},
		{"test.TestFunc", true, Function, file1Path},
		{"test.TestVar", true, Variable, file1Path},
		{"test.TestMethod", true, Method, file1Path},
		{"test.AnotherStruct", true, Type, file2Path},
		{"test.AnotherFunc", true, Function, file2Path},
		{"test.NonExistentSymbol", false, Function, ""},
	}

	for _, tc := range testCases {
		t.Run(tc.symbol, func(t *testing.T) {
			assert.Equal(t, tc.expected, st.IsDefined(tc.symbol))

			if tc.expected {
				info, exists := st.GetSymbolInfo(tc.symbol)
				assert.True(t, exists)
				assert.Equal(t, tc.symType, info.Type)
				assert.Equal(t, tc.filePath, info.FilePath)
				assert.Equal(t, "test", info.Package)
			} else {
				_, exists := st.GetSymbolInfo(tc.symbol)
				assert.False(t, exists)
			}
		})
	}

	// Test AddInterfaceImplementation
	st.AddInterfaceImplementation("test.TestStruct", "SomeInterface")
	info, exists := st.GetSymbolInfo("test.TestStruct")
	assert.True(t, exists)
	assert.Contains(t, info.Interfaces, "SomeInterface")
}

func TestConcurrentSymbolTable(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "concurrent-symboltable-test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	for i := 0; i < 100; i++ {
		content := fmt.Sprintf(`package test%d
func Func%d() {}
var Var%d int
`, i, i, i)
		err = os.WriteFile(filepath.Join(tmpDir, fmt.Sprintf("file%d.go", i)), []byte(content), 0o644)
		require.NoError(t, err)
	}

	st, err := BuildSymbolTable(tmpDir)
	require.NoError(t, err)

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			assert.True(t, st.IsDefined(fmt.Sprintf("test%d.Func%d", i, i)))
			assert.True(t, st.IsDefined(fmt.Sprintf("test%d.Var%d", i, i)))
		}(i)
	}
	wg.Wait()
}

func BenchmarkBuildSymbolTable(b *testing.B) {
	tmpDir, err := os.MkdirTemp("", "benchmark-symboltable")
	require.NoError(b, err)
	defer os.RemoveAll(tmpDir)

	numFiles := 1000
	for i := 0; i < numFiles; i++ {
		content := fmt.Sprintf(`package test%d
func Func%d() {}
var Var%d int
type Struct%d struct{}
`, i, i, i, i)
		err = os.WriteFile(filepath.Join(tmpDir, fmt.Sprintf("file%d.go", i)), []byte(content), 0o644)
		require.NoError(b, err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := BuildSymbolTable(tmpDir)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkSymbolTableIsDefined(b *testing.B) {
	tmpDir, err := os.MkdirTemp("", "benchmark-symboltable-isdefined")
	require.NoError(b, err)
	defer os.RemoveAll(tmpDir)

	numFiles := 1000
	for i := 0; i < numFiles; i++ {
		content := fmt.Sprintf(`package test%d
func Func%d() {}
var Var%d int
type Struct%d struct{}
`, i, i, i, i)
		err = os.WriteFile(filepath.Join(tmpDir, fmt.Sprintf("file%d.go", i)), []byte(content), 0o644)
		require.NoError(b, err)
	}

	st, err := BuildSymbolTable(tmpDir)
	require.NoError(b, err)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		randomIndex := i % numFiles
		st.IsDefined(fmt.Sprintf("test%d.Func%d", randomIndex, randomIndex))
		st.IsDefined(fmt.Sprintf("test%d.Var%d", randomIndex, randomIndex))
		st.IsDefined(fmt.Sprintf("test%d.Struct%d", randomIndex, randomIndex))
	}
}
