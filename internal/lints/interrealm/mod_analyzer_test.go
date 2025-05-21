package interrealm

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDeterminePackageTypeFromDir(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "gno-lint-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	tests := []struct {
		name        string
		moduleDecl  string
		wantType    PackageType
		wantName    string
		wantPath    string
		shouldError bool
	}{
		{
			name:       "p package",
			moduleDecl: "module gno.land/p/demo/package",
			wantType:   TypePackage,
			wantName:   "package",
			wantPath:   "gno.land/p/demo/package",
		},
		{
			name:       "r package",
			moduleDecl: "module gno.land/r/demo/realm",
			wantType:   TypeRealm,
			wantName:   "realm",
			wantPath:   "gno.land/r/demo/realm",
		},
		{
			name:        "invalid format",
			moduleDecl:  "module invalid/format",
			shouldError: true,
		},
		{
			name:        "empty file",
			moduleDecl:  "",
			shouldError: true,
		},
		{
			name:       "with comments and whitespace",
			moduleDecl: "# This is a comment\n\nmodule gno.land/p/test/package  \n\n# Another comment",
			wantType:   TypePackage,
			wantName:   "package",
			wantPath:   "gno.land/p/test/package",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a subdirectory for this test
			testDir := filepath.Join(tmpDir, tt.name)
			if err := os.Mkdir(testDir, 0755); err != nil {
				t.Fatalf("Failed to create test directory: %v", err)
			}

			// Create gno.mod file
			gnoModPath := filepath.Join(testDir, "gno.mod")
			if err := os.WriteFile(gnoModPath, []byte(tt.moduleDecl), 0644); err != nil {
				t.Fatalf("Failed to write gno.mod file: %v", err)
			}

			// Test the function
			ctx, err := DeterminePackageTypeFromDir(testDir)

			if tt.shouldError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if ctx.Type != tt.wantType {
				t.Errorf("Package type = %v, want %v", ctx.Type, tt.wantType)
			}

			if ctx.Name != tt.wantName {
				t.Errorf("Package name = %v, want %v", ctx.Name, tt.wantName)
			}

			if ctx.ModulePath != tt.wantPath {
				t.Errorf("Module path = %v, want %v", ctx.ModulePath, tt.wantPath)
			}
		})
	}
}

func TestGetPackageContextForFile(t *testing.T) {
	// Create temporary test directory with nested structure
	tmpDir, err := os.MkdirTemp("", "gno-lint-nested-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a nested directory structure
	// tmpDir/
	//   ├── p-pkg/
	//   │    ├── gno.mod (p package)
	//   │    ├── file1.gno
	//   │    └── subdir/
	//   │         └── file2.gno
	//   └── r-pkg/
	//        ├── gno.mod (r package)
	//        └── file3.gno

	// Create p package
	pPkgDir := filepath.Join(tmpDir, "p-pkg")
	if err := os.Mkdir(pPkgDir, 0755); err != nil {
		t.Fatalf("Failed to create p-pkg directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pPkgDir, "gno.mod"), []byte("module gno.land/p/test/package"), 0644); err != nil {
		t.Fatalf("Failed to write p-pkg gno.mod: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pPkgDir, "file1.gno"), []byte("package package"), 0644); err != nil {
		t.Fatalf("Failed to write file1.gno: %v", err)
	}

	// Create subdirectory in p package
	pSubdir := filepath.Join(pPkgDir, "subdir")
	if err := os.Mkdir(pSubdir, 0755); err != nil {
		t.Fatalf("Failed to create subdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pSubdir, "file2.gno"), []byte("package package"), 0644); err != nil {
		t.Fatalf("Failed to write file2.gno: %v", err)
	}

	// Create r package
	rPkgDir := filepath.Join(tmpDir, "r-pkg")
	if err := os.Mkdir(rPkgDir, 0755); err != nil {
		t.Fatalf("Failed to create r-pkg directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(rPkgDir, "gno.mod"), []byte("module gno.land/r/test/realm"), 0644); err != nil {
		t.Fatalf("Failed to write r-pkg gno.mod: %v", err)
	}
	if err := os.WriteFile(filepath.Join(rPkgDir, "file3.gno"), []byte("package realm"), 0644); err != nil {
		t.Fatalf("Failed to write file3.gno: %v", err)
	}

	// Test files
	tests := []struct {
		name     string
		filePath string
		wantType PackageType
		wantName string
	}{
		{
			name:     "file in p package root",
			filePath: filepath.Join(pPkgDir, "file1.gno"),
			wantType: TypePackage,
			wantName: "package",
		},
		{
			name:     "file in p package subdirectory",
			filePath: filepath.Join(pSubdir, "file2.gno"),
			wantType: TypePackage,
			wantName: "package",
		},
		{
			name:     "file in r package",
			filePath: filepath.Join(rPkgDir, "file3.gno"),
			wantType: TypeRealm,
			wantName: "realm",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, err := GetPackageContextForFile(tt.filePath)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if ctx.Type != tt.wantType {
				t.Errorf("Package type = %v, want %v", ctx.Type, tt.wantType)
			}

			if ctx.Name != tt.wantName {
				t.Errorf("Package name = %v, want %v", ctx.Name, tt.wantName)
			}
		})
	}

	// Test non-existent file
	t.Run("non-existent file", func(t *testing.T) {
		nonExistentPath := filepath.Join(tmpDir, "non-existent", "file.gno")
		_, err := GetPackageContextForFile(nonExistentPath)
		if err == nil {
			t.Errorf("Expected error for non-existent file, got none")
		}
	})
}

func TestPackageTypeCache(t *testing.T) {
	// Create temporary test directory
	tmpDir, err := os.MkdirTemp("", "gno-lint-cache-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a package
	pkgDir := filepath.Join(tmpDir, "pkg")
	if err := os.Mkdir(pkgDir, 0755); err != nil {
		t.Fatalf("Failed to create pkg directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pkgDir, "gno.mod"), []byte("module gno.land/p/test/cached"), 0644); err != nil {
		t.Fatalf("Failed to write gno.mod: %v", err)
	}

	file1Path := filepath.Join(pkgDir, "file1.gno")
	file2Path := filepath.Join(pkgDir, "file2.gno")

	if err := os.WriteFile(file1Path, []byte("package cached"), 0644); err != nil {
		t.Fatalf("Failed to write file1.gno: %v", err)
	}
	if err := os.WriteFile(file2Path, []byte("package cached"), 0644); err != nil {
		t.Fatalf("Failed to write file2.gno: %v", err)
	}

	// Test cache behavior
	cache := NewPackageTypeCache()

	// First call should determine type from file
	ctx1, err := cache.GetPackageContext(file1Path)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if ctx1.Type != TypePackage {
		t.Errorf("Package type = %v, want %v", ctx1.Type, TypePackage)
	}

	// Second call should use cached value
	ctx2, err := cache.GetPackageContext(file2Path)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if ctx2.Type != TypePackage {
		t.Errorf("Package type = %v, want %v", ctx2.Type, TypePackage)
	}

	// Verify we got the same context object (by pointer) for both files
	// This confirms caching is working
	if ctx1 != ctx2 {
		t.Errorf("Expected same context object from cache")
	}
}
