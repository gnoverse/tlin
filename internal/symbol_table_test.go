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

	assert.True(t, st.IsDefined("test.TestStruct"))
	assert.True(t, st.IsDefined("test.TestFunc"))
	assert.True(t, st.IsDefined("test.TestVar"))
	assert.True(t, st.IsDefined("test.AnotherStruct"))
	assert.True(t, st.IsDefined("test.AnotherFunc"))
	assert.False(t, st.IsDefined("test.NonExistentSymbol"))

	// validate symbol file paths
	path, exists := st.GetSymbolPath("test.TestStruct")
	assert.True(t, exists)
	assert.Equal(t, file1Path, path)

	path, exists = st.GetSymbolPath("test.AnotherFunc")
	assert.True(t, exists)
	assert.Equal(t, file2Path, path)

	_, exists = st.GetSymbolPath("test.NonExistentSymbol")
	assert.False(t, exists)
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
