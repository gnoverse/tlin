package migration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/pmezard/go-difflib/difflib"
)

type Options struct {
	Apply       bool
	Force       bool
	Diff        bool
	ReportPath  string
	IgnorePaths []string
}

type FileResult struct {
	Path       string    `json:"path"`
	Changed    bool      `json:"changed"`
	Skipped    bool      `json:"skipped"`
	Error      string    `json:"error,omitempty"`
	Edits      []Edit    `json:"edits,omitempty"`
	Findings   []Finding `json:"findings,omitempty"`
	Diff       string    `json:"-"`
	ParseError string    `json:"parse_error,omitempty"`
}

type Report struct {
	Files          []FileResult `json:"files"`
	FilesScanned   int          `json:"files_scanned"`
	FilesChanged   int          `json:"files_changed"`
	SafeEdits      int          `json:"safe_edits"`
	ManualFindings int          `json:"manual_findings"`
}

func Run(paths []string, opts Options, migrators []Migrator) (Report, error) {
	files, err := collectFiles(paths, opts.IgnorePaths)
	if err != nil {
		return Report{}, err
	}
	var report Report
	report.FilesScanned = len(files)
	for _, path := range files {
		res := runFile(path, opts, migrators)
		report.Files = append(report.Files, res)
		if res.Changed {
			report.FilesChanged++
		}
		report.SafeEdits += len(res.Edits)
		report.ManualFindings += len(res.Findings)
	}
	if opts.ReportPath != "" {
		if err := writeReport(opts.ReportPath, report); err != nil {
			return report, err
		}
	}
	return report, nil
}

func runFile(path string, opts Options, migrators []Migrator) FileResult {
	res := FileResult{Path: path}
	src, err := os.ReadFile(path)
	if err != nil {
		res.Skipped = true
		res.Error = err.Error()
		return res
	}
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, src, parser.ParseComments)
	if err != nil {
		res.Skipped = true
		res.ParseError = err.Error()
		return res
	}
	ctx := &FileContext{Path: path, Source: src, FileSet: fset, File: file}
	var edits []Edit
	var findings []Finding
	for _, migrator := range migrators {
		es, fs := migrator.Run(ctx)
		edits = append(edits, es...)
		findings = append(findings, fs...)
	}
	res.Findings = findings
	if len(edits) == 0 {
		return res
	}
	next, err := Apply(src, fset, edits)
	if err != nil {
		res.Skipped = true
		res.Error = err.Error()
		return res
	}
	imports := newImportManager()
	for _, edit := range edits {
		switch edit.Category {
		case "interrealm-3.4-origin-send":
			imports.Add("chain/runtime/unsafe")
			imports.RemoveIfAliasUnused("chain/banker", "banker")
		}
	}
	next, err = imports.Apply(path, next)
	if err != nil {
		res.Skipped = true
		res.Error = err.Error()
		return res
	}
	if _, err := parser.ParseFile(token.NewFileSet(), path, next, parser.ParseComments); err != nil {
		res.Skipped = true
		res.Error = "modified source did not parse: " + err.Error()
		return res
	}
	if bytes.Equal(src, next) {
		return res
	}
	res.Edits = edits
	res.Changed = true
	res.Diff = unifiedDiff(path, string(src), string(next))
	if opts.Apply {
		if err := os.WriteFile(path, next, 0o644); err != nil {
			res.Skipped = true
			res.Changed = false
			res.Error = err.Error()
		}
	}
	return res
}

func collectFiles(paths []string, ignore []string) ([]string, error) {
	var files []string
	for _, p := range paths {
		info, err := os.Stat(p)
		if err != nil {
			return nil, err
		}
		if !info.IsDir() {
			if info.Mode().IsRegular() && strings.HasSuffix(p, ".gno") && !ignored(p, ignore) {
				files = append(files, p)
			}
			continue
		}
		err = filepath.Walk(p, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				return nil
			}
			if !info.Mode().IsRegular() {
				return nil
			}
			if strings.HasSuffix(path, ".gno") && !ignored(path, ignore) {
				files = append(files, path)
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	sort.Strings(files)
	return files, nil
}

func ignored(path string, patterns []string) bool {
	for _, pattern := range patterns {
		pattern = strings.TrimSpace(pattern)
		if pattern == "" {
			continue
		}
		ok, _ := filepath.Match(pattern, path)
		if ok || strings.Contains(path, pattern) {
			return true
		}
	}
	return false
}

func unifiedDiff(path, oldText, newText string) string {
	diff, _ := difflib.GetUnifiedDiffString(difflib.UnifiedDiff{
		A:        difflib.SplitLines(oldText),
		B:        difflib.SplitLines(newText),
		FromFile: path,
		ToFile:   path,
		Context:  3,
	})
	return diff
}

func writeReport(path string, report Report) error {
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0o644)
}

func PrintSummary(report Report, showDiff bool) {
	if showDiff {
		for _, file := range report.Files {
			if file.Diff != "" {
				fmt.Print(file.Diff)
			}
		}
	}
	fmt.Printf("%d files scanned · Tier1 %d edits", report.FilesScanned, report.SafeEdits)
	if report.FilesChanged > 0 {
		fmt.Printf(" · %d files changed", report.FilesChanged)
	}
	if report.ManualFindings > 0 {
		fmt.Printf(" · manual migration needed %d findings", report.ManualFindings)
	}
	fmt.Println()
	if report.ManualFindings > 0 {
		fmt.Println("Note: this tool is a starting point. Review/manual findings must still be migrated by hand.")
	}
}
