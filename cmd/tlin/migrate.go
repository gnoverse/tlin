package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/gnolang/tlin/internal/migration"
)

func runMigrate(args []string) {
	if len(args) == 0 || args[0] != "interrealm" {
		fmt.Fprintln(os.Stderr, "usage: tlin migrate interrealm [flags] <path...>")
		os.Exit(2)
	}

	fs := flag.NewFlagSet("tlin migrate interrealm", flag.ExitOnError)
	apply := fs.Bool("w", false, "apply safe Tier 1 edits")
	applyAlias := fs.Bool("apply", false, "apply safe Tier 1 edits")
	force := fs.Bool("force", false, "allow applying with a dirty git worktree")
	reportPath := fs.String("report", "", "write JSON migration report")
	showDiff := fs.Bool("diff", true, "print unified diff")
	includeReview := fs.Bool("include-review", false, "include opt-in Review confidence migration edits")
	ignorePaths := fs.String("ignore-paths", "", "comma-separated path patterns to ignore")
	if err := fs.Parse(args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	paths := fs.Args()
	if len(paths) == 0 {
		fmt.Fprintln(os.Stderr, "error: Please provide file or directory paths")
		os.Exit(1)
	}

	shouldApply := *apply || *applyAlias
	if shouldApply && !*force && gitDirty() {
		fmt.Fprintln(os.Stderr, "error: git worktree is dirty; rerun with -force to apply migration edits")
		os.Exit(1)
	}

	report, err := migration.Run(paths, migration.Options{
		Apply:         shouldApply,
		Force:         *force,
		Diff:          *showDiff,
		IncludeReview: *includeReview,
		ReportPath:    *reportPath,
		IgnorePaths:   splitAndTrim(*ignorePaths),
	}, migration.InterrealmMigrators())
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	migration.PrintSummary(report, *showDiff)
}

func gitDirty() bool {
	cmd := exec.Command("git", "status", "--porcelain")
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return false
	}
	return strings.TrimSpace(out.String()) != ""
}
