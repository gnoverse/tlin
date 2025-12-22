package fixer

import (
	"path/filepath"
	"strings"

	"golang.org/x/tools/imports"
)

// ProcessImports uses goimports-style processing to:
//  1. Add missing imports (for standard library packages)
//  2. Remove unused imports
//  3. Format the code
func ProcessImports(filename string, src []byte) ([]byte, error) {
	// For .gno files, use .go extension so imports package recognizes it
	processName := filename
	if strings.HasSuffix(filename, ".gno") {
		processName = strings.TrimSuffix(filepath.Base(filename), ".gno") + ".go"
	}

	opts := &imports.Options{
		Comments:   true,
		TabIndent:  true,
		TabWidth:   8,
		FormatOnly: false, // Process imports, not just format
	}

	result, err := imports.Process(processName, src, opts)
	if err != nil {
		return src, err
	}

	return result, nil
}
