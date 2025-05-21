package interrealm

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"testing"

	tt "github.com/gnolang/tlin/internal/types"
)

func TestDetectCrossingPosition(t *testing.T) {
	// Create temporary test directory
	tmpDir, err := os.MkdirTemp("", "gno-interrealm-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test case structure
	tests := []struct {
		name           string
		pkgType        PackageType
		code           string
		wantIssueCount int
		wantIssueRules []string // Rules that should be detected
		Severity       tt.Severity
	}{
		{
			name:    "p package with crossing",
			pkgType: TypePackage,
			code: `package pkgname

func ValidFunc() {
    crossing() // Not allowed in p package
    println("Hello")
}
`,
			wantIssueCount: 1,
			wantIssueRules: []string{"crossing-in-p-package"},
			Severity:       tt.SeverityWarning,
		},
		{
			name:    "r package with crossing not first",
			pkgType: TypeRealm,
			code: `package realmname

func InvalidFunc() {
    println("This should come after crossing()")
    crossing() // Not first statement
}
`,
			wantIssueCount: 1,
			wantIssueRules: []string{"crossing-position"},
			Severity:       tt.SeverityWarning,
		},
		{
			name:    "r package with correct crossing",
			pkgType: TypeRealm,
			code: `package realmname

func ValidFunc() {
    crossing() // Correct - first statement in r package
    println("Hello")
}
`,
			wantIssueCount: 0,
			Severity:       tt.SeverityWarning,
		},
		{
			name:    "crossing with arguments",
			pkgType: TypeRealm,
			code: `package realmname

func InvalidFunc() {
    crossing("invalid arg") // Should not have arguments
    println("Hello")
}
`,
			wantIssueCount: 1,
			wantIssueRules: []string{"crossing-with-args"},
			Severity:       tt.SeverityWarning,
		},
		{
			name:    "public function without crossing in r package",
			pkgType: TypeRealm,
			code: `package realmname

func PublicFunc() { // Missing crossing() in public function
    println("Public function without crossing")
}
`,
			wantIssueCount: 1,
			wantIssueRules: []string{"public-function-without-crossing"},
			Severity:       tt.SeverityWarning,
		},
		{
			name:    "private function without crossing in r package",
			pkgType: TypeRealm,
			code: `package realmname

func privateFunc() { // private is fine without crossing
    println("Private function")
}
`,
			wantIssueCount: 0,
			Severity:       tt.SeverityWarning,
		},
		{
			name:    "multiple crossing issues",
			pkgType: TypeRealm,
			code: `package realmname

func PublicFunc1() { // Missing crossing
    println("Public function without crossing")
}

func PublicFunc2() {
    println("First line")
    crossing() // Not first
}

func PublicFunc3() {
    crossing(123) // With invalid args
    println("After crossing")
}
`,
			wantIssueCount: 3,
			wantIssueRules: []string{
				"public-function-without-crossing",
				"crossing-position",
				"crossing-with-args",
			},
			Severity: tt.SeverityWarning,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testDir := filepath.Join(tmpDir, tt.name)
			if err := os.Mkdir(testDir, 0755); err != nil {
				t.Fatalf("Failed to create test directory: %v", err)
			}

			// generate mod file
			var moduleDecl string
			if tt.pkgType == TypePackage {
				moduleDecl = "module gno.land/p/test/pkgname"
			} else {
				moduleDecl = "module gno.land/r/test/realmname"
			}

			if err := os.WriteFile(filepath.Join(testDir, "gno.mod"), []byte(moduleDecl), 0644); err != nil {
				t.Fatalf("Failed to write gno.mod file: %v", err)
			}

			// generate test file
			filename := filepath.Join(testDir, "test.gno")
			if err := os.WriteFile(filename, []byte(tt.code), 0644); err != nil {
				t.Fatalf("Failed to write test file: %v", err)
			}

			// Parse the code
			fset := token.NewFileSet()
			node, err := parser.ParseFile(fset, filename, nil, parser.ParseComments)
			if err != nil {
				t.Fatalf("Failed to parse test file: %v", err)
			}

			// mock GetPackageContextForFile to return expected package type
			originalGetPkgCtx := GetPackageContextForFile
			defer func() { GetPackageContextForFile = originalGetPkgCtx }()

			GetPackageContextForFile = func(filePath string) (*PackageContext, error) {
				var name string
				if tt.pkgType == TypePackage {
					name = "pkgname"
				} else {
					name = "realmname"
				}

				var path string
				if tt.pkgType == TypePackage {
					path = "gno.land/p/test/pkgname"
				} else {
					path = "gno.land/r/test/realmname"
				}

				return &PackageContext{
					Type:       tt.pkgType,
					Name:       name,
					ModulePath: path,
				}, nil
			}

			issues, err := DetectCrossingPosition(filename, node, fset, tt.Severity)
			if err != nil {
				t.Fatalf("Unexpected error from DetectCrossingPosition: %v", err)
			}

			// verify issue count
			if len(issues) != tt.wantIssueCount {
				t.Errorf("Issue count = %d, want %d", len(issues), tt.wantIssueCount)
				for i, issue := range issues {
					t.Logf("Issue %d: %s - %s", i+1, issue.Rule, issue.Message)
				}
			}

			// verify issue rules
			if len(tt.wantIssueRules) > 0 {
				foundRules := make(map[string]bool)
				for _, issue := range issues {
					foundRules[issue.Rule] = true
				}

				for _, wantRule := range tt.wantIssueRules {
					if !foundRules[wantRule] {
						t.Errorf("Expected issue with rule %q but none found", wantRule)
					}
				}
			}
		})
	}
}

// TestCrossingIntegration tests the complete flow from file analysis to issue detection
// This test creates a realistic directory structure with real Gno package layouts
func TestCrossingIntegration(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "gno-interrealm-integration-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// create a p package structure
	pPkgPath := filepath.Join(tmpDir, "p", "demo", "package")
	if err := os.MkdirAll(pPkgPath, 0755); err != nil {
		t.Fatalf("Failed to create p package directory: %v", err)
	}

	// create an r package structure
	rPkgPath := filepath.Join(tmpDir, "r", "demo", "realm")
	if err := os.MkdirAll(rPkgPath, 0755); err != nil {
		t.Fatalf("Failed to create r package directory: %v", err)
	}

	// write mod files
	if err := os.WriteFile(filepath.Join(pPkgPath, "gno.mod"), []byte("module gno.land/p/demo/package"), 0644); err != nil {
		t.Fatalf("Failed to write p package gno.mod: %v", err)
	}

	if err := os.WriteFile(filepath.Join(rPkgPath, "gno.mod"), []byte("module gno.land/r/demo/realm"), 0644); err != nil {
		t.Fatalf("Failed to write r package gno.mod: %v", err)
	}

	// write source files with various crossing issues
	pFile := filepath.Join(pPkgPath, "invalid.gno")
	pCode := `package demo

// This function incorrectly uses crossing in a p package
func InvalidFunc() {
    crossing() // This is not allowed in p packages
    println("Hello from p package")
}
`
	if err := os.WriteFile(pFile, []byte(pCode), 0644); err != nil {
		t.Fatalf("Failed to write p package file: %v", err)
	}

	rFile := filepath.Join(rPkgPath, "mixed.gno")
	rCode := `package realm

// This function correctly uses crossing
func ValidCrossingFunc() {
    crossing()
    println("Hello from realm")
}

// This function incorrectly uses crossing (not first statement)
func InvalidPositionFunc() {
    println("This should come after crossing")
    crossing()
}

// This function is public but doesn't use crossing
func PublicWithoutCrossing() {
    println("Public function without crossing")
}

// This private function doesn't need crossing
func privateFunc() {
    println("Private function")
}
`
	if err := os.WriteFile(rFile, []byte(rCode), 0644); err != nil {
		t.Fatalf("Failed to write r package file: %v", err)
	}

	t.Run("p package with crossing", func(t *testing.T) {
		fset := token.NewFileSet()
		node, err := parser.ParseFile(fset, pFile, nil, parser.ParseComments)
		if err != nil {
			t.Fatalf("Failed to parse p package file: %v", err)
		}

		issues, err := DetectCrossingPosition(pFile, node, fset, tt.SeverityError)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if len(issues) != 1 {
			t.Errorf("Expected 1 issue, got %d", len(issues))
		} else if issues[0].Rule != "crossing-in-p-package" {
			t.Errorf("Expected rule 'crossing-in-p-package', got %q", issues[0].Rule)
		}
	})

	t.Run("r package with mixed issues", func(t *testing.T) {
		fset := token.NewFileSet()
		node, err := parser.ParseFile(fset, rFile, nil, parser.ParseComments)
		if err != nil {
			t.Fatalf("Failed to parse r package file: %v", err)
		}

		issues, err := DetectCrossingPosition(rFile, node, fset, tt.SeverityError)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		expectedIssues := 2 // One for invalid position, one for public without crossing
		if len(issues) != expectedIssues {
			t.Errorf("Expected %d issues, got %d", expectedIssues, len(issues))
			for i, issue := range issues {
				t.Logf("Issue %d: %s - %s", i+1, issue.Rule, issue.Message)
			}
		}

		// Check that we found all expected issue types
		foundRules := make(map[string]bool)
		for _, issue := range issues {
			foundRules[issue.Rule] = true
		}

		expectedRules := []string{
			"crossing-position",
			"public-function-without-crossing",
		}

		for _, rule := range expectedRules {
			if !foundRules[rule] {
				t.Errorf("Expected to find issue with rule %q, but none found", rule)
			}
		}
	})
}
