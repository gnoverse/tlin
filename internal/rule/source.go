package rule

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"

	"github.com/gnolang/tlin/internal/nolint"
)

// Source bundles a single input file's load state: paths, raw bytes,
// AST, FileSet, and nolint manager. LoadSource owns the .gno → temp
// .go conversion; Close removes the temp file if one was created.
// Close on plain .go inputs and source-only Sources is a no-op, so
// callers can always defer it.
type Source struct {
	// OriginalPath is the path the user supplied (may end in .gno).
	// Empty for source-only runs (LoadSourceFromBytes).
	OriginalPath string
	// WorkingPath is the path the parser read. Equals OriginalPath
	// for plain .go inputs; differs only when the engine converted
	// .gno → temp .go. Empty for source-only runs.
	WorkingPath string
	// Bytes is the raw file content. Always set.
	Bytes []byte
	// File is the parsed AST. Always set.
	File *ast.File
	// Fset is the FileSet that produced File. Always set.
	Fset *token.FileSet
	// NolintMgr resolves nolint comments. Always set.
	NolintMgr *nolint.Manager

	// isTemp records whether Close should remove WorkingPath.
	isTemp bool
}

// LoadSource opens path, converting .gno → temp .go transparently,
// reads the bytes, parses the AST, and registers nolint comments.
// On any failure after the temp file has been created the temp file
// is removed before the error propagates — callers do not have to
// Close partial Sources, though Close on a nil/empty Source is also
// safe.
func LoadSource(path string) (*Source, error) {
	s := &Source{OriginalPath: path}

	// prepareFile returns the bytes when it had to read them anyway
	// (the .gno → temp .go copy path) so we don't re-read from disk.
	workingPath, bytes, isTemp, err := prepareFile(path)
	if err != nil {
		return nil, err
	}
	s.WorkingPath = workingPath
	s.isTemp = isTemp

	if bytes == nil {
		bytes, err = os.ReadFile(workingPath)
		if err != nil {
			_ = s.Close()
			return nil, fmt.Errorf("error reading file: %w", err)
		}
	}
	s.Bytes = bytes

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, workingPath, bytes, parser.ParseComments)
	if err != nil {
		_ = s.Close()
		return nil, fmt.Errorf("error parsing file: %w", err)
	}
	s.File = file
	s.Fset = fset
	s.NolintMgr = nolint.ParseComments(file, fset)

	return s, nil
}

// LoadSourceFromBytes parses content directly without touching the
// filesystem — the engine.RunSource path.
func LoadSourceFromBytes(content []byte) (*Source, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "", content, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("error parsing content: %w", err)
	}
	return &Source{
		Bytes:     content,
		File:      file,
		Fset:      fset,
		NolintMgr: nolint.ParseComments(file, fset),
	}, nil
}

// Close removes the temp .go file LoadSource may have created. Safe
// to call repeatedly and on Sources that don't own a temp file.
func (s *Source) Close() error {
	if s == nil || !s.isTemp || s.WorkingPath == "" {
		return nil
	}
	err := os.Remove(s.WorkingPath)
	// Flip even on Remove failure: a failed unlink won't recover on
	// retry, and idempotent callers shouldn't see the same error
	// twice from defer src.Close() patterns.
	s.isTemp = false
	return err
}

// IsTemp reports whether Close will remove WorkingPath.
func (s *Source) IsTemp() bool { return s != nil && s.isTemp }

// prepareFile returns the path the parser should read, plus the file
// content when it had to be loaded anyway. For plain .go inputs the
// content is nil and the caller does the read. For .gno inputs the
// content is the bytes that were just copied into the temp .go, so
// the caller can skip a second os.ReadFile.
func prepareFile(path string) (workingPath string, content []byte, isTemp bool, err error) {
	if !strings.HasSuffix(path, ".gno") {
		return path, nil, false, nil
	}
	temp, content, err := createTempGoFile(path)
	if err != nil {
		return "", nil, false, err
	}
	return temp, content, true, nil
}

// createTempGoFile copies a .gno file to a temp .go in the same
// directory and returns both the temp path and the bytes it copied
// (so callers don't have to re-read the file). gno's syntax is a Go
// superset for the constructs lint rules inspect, so a bytewise copy
// is sufficient — the .go suffix just lets the standard parser and
// tools (golangci-lint, packages.Load) accept the file.
func createTempGoFile(gnoFile string) (string, []byte, error) {
	content, err := os.ReadFile(gnoFile)
	if err != nil {
		return "", nil, fmt.Errorf("error reading .gno file: %w", err)
	}

	dir := filepath.Dir(gnoFile)
	tempFile, err := os.CreateTemp(dir, "temp_*.go")
	if err != nil {
		return "", nil, fmt.Errorf("error creating temp file: %w", err)
	}

	if _, err := tempFile.Write(content); err != nil {
		_ = tempFile.Close()
		_ = os.Remove(tempFile.Name())
		return "", nil, fmt.Errorf("error writing to temp file: %w", err)
	}

	if err := tempFile.Close(); err != nil {
		_ = os.Remove(tempFile.Name())
		return "", nil, fmt.Errorf("error closing temp file: %w", err)
	}

	return tempFile.Name(), content, nil
}
