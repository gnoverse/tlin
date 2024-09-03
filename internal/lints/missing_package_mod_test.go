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
				package main
				import (
					"gno.land/p/demo/avl"
					"gno.land/r/demo/users"
				)
				func main() {}
			`,
			modContent: `
				module test
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
				package main
				import (
					"gno.land/p/demo/avl"
					"gno.land/r/demo/users"
				)
				func main() {}
			`,
			modContent: `
				module test
				require (
					gno.land/p/demo/avl v0.0.0-latest
				)
			`,
			expectedIssues: 1,
		},
		{
			name: "Multiple missing packages",
			gnoContent: `
				package main
				import (
					"gno.land/p/demo/avl"
					"gno.land/r/demo/users"
					"gno.land/p/demo/ufmt"
				)
				func main() {}
			`,
			modContent: `
				module test
				require (
					gno.land/p/demo/avl v0.0.0-latest
				)
			`,
			expectedIssues: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a temporary directory
			tmpDir, err := os.MkdirTemp("", "test")
			require.NoError(t, err)
			defer os.RemoveAll(tmpDir)

			// Create .gno file
			gnoFile := filepath.Join(tmpDir, "main.gno")
			err = os.WriteFile(gnoFile, []byte(tt.gnoContent), 0644)
			require.NoError(t, err)

			// Create gno.mod file
			modFile := filepath.Join(tmpDir, "gno.mod")
			err = os.WriteFile(modFile, []byte(tt.modContent), 0644)
			require.NoError(t, err)

			// Parse the .gno file
			fset := token.NewFileSet()
			node, err := parser.ParseFile(fset, gnoFile, nil, parser.ParseComments)
			require.NoError(t, err)

			// Run the function
			issues, err := DetectMissingModPackage(gnoFile, node, fset)

			// Check the results
			require.NoError(t, err)
			assert.Len(t, issues, tt.expectedIssues)

			// Check that all issues are of the correct type
			for _, issue := range issues {
				assert.Equal(t, "gno-mod-tidy", issue.Rule)
				assert.Equal(t, modFile, issue.Filename)
				println(issue.String())
			}
		})
	}
}
