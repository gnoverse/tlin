package lints

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDetectMissingPackageInMod(t *testing.T) {
	tests := []struct {
		name           string
		gnoContent     string
		modContent     string
		expectedIssues int
	}{
		{
			name: "No missing packages",
			gnoContent: `
				package foo
				import (
					"gno.land/p/demo/avl"
					"gno.land/r/demo/users"
				)
				func SomeFunc() {}
			`,
			modContent: `
				module foo
				require (
					gno.land/p/demo/avl v0.0.0-latest
					gno.land/r/demo/users v0.0.0-latest
				)
			`,
			expectedIssues: 0,
		},
		{
			name: "One missing package",
			gnoContent: `
				package foo
				import (
					"gno.land/p/demo/avl"
					"gno.land/r/demo/users"
				)
				func Foo() {}
			`,
			modContent: `
				module foo
				require (
					gno.land/p/demo/avl v0.0.0-latest
				)
			`,
			expectedIssues: 1,
		},
		{
			name: "Multiple missing packages",
			gnoContent: `
				package bar
				import (
					"gno.land/p/demo/avl"
					"gno.land/r/demo/users"
					"gno.land/p/demo/ufmt"
				)
				func main() {}
			`,
			modContent: `
				module bar
				require (
					gno.land/p/demo/avl v0.0.0-latest
				)
			`,
			expectedIssues: 2,
		},
		{
			name: "declared but not imported",
			gnoContent: `
				package bar

				func main() {}
			`,
			modContent: `
				module bar
				require (
					gno.land/p/demo/avl v0.0.0-latest
				)
			`,
			expectedIssues: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir, err := os.MkdirTemp("", "test")
			require.NoError(t, err)
			defer os.RemoveAll(tmpDir)

			gnoFile := filepath.Join(tmpDir, "main.gno")
			err = os.WriteFile(gnoFile, []byte(tt.gnoContent), 0644)
			require.NoError(t, err)

			modFile := filepath.Join(tmpDir, "gno.mod")
			err = os.WriteFile(modFile, []byte(tt.modContent), 0644)
			require.NoError(t, err)

			fset := token.NewFileSet()
			node, err := parser.ParseFile(fset, gnoFile, nil, parser.ParseComments)
			require.NoError(t, err)

			issues, err := DetectMissingModPackage(gnoFile, node, fset)

			require.NoError(t, err)
			assert.Len(t, issues, tt.expectedIssues)

			for _, issue := range issues {
				assert.Equal(t, "gno-mod-tidy", issue.Rule)
				assert.Equal(t, modFile, issue.Filename)
				println(issue.String())
			}
		})
	}
}
