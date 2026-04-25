package internal

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/gnolang/tlin/internal/rule"
	"github.com/gnolang/tlin/internal/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

func TestNewEngine(t *testing.T) {
	t.Parallel()

	engine, err := NewEngine(nil)
	assert.NoError(t, err)
	assert.NotNil(t, engine)
	assert.NotEmpty(t, engine.rules)
}

func TestNewEngineConfig(t *testing.T) {
	t.Parallel()

	config := map[string]types.ConfigRule{
		"useless-break": {
			Severity: types.SeverityOff,
		},
		"simplify-slice-range": {
			Severity: types.SeverityWarning,
		},
		"test-rule": {
			Severity: types.SeverityError,
		},
	}
	engine, err := NewEngine(config)
	assert.NoError(t, err)
	assert.NotNil(t, engine)

	assert.NotEmpty(t, engine.rules)

	// "test-rule" is not in allRules, so it must not show up in engine.rules.
	_, hasTestRule := engine.rules["test-rule"]
	assert.False(t, hasTestRule, "test-rule should not be in the rules")

	// "simplify-slice-range" defaults to Error; config overrides it to Warning.
	assert.Equal(t, types.SeverityWarning, engine.severityOverrides["simplify-slice-range"],
		"config should override the default Error severity")

	assert.True(t, engine.ignoredRules["useless-break"])
}

func TestNewEngineContent(t *testing.T) {
	t.Parallel()

	engine, err := NewEngine(nil)
	assert.NoError(t, err)
	assert.NotNil(t, engine)
	assert.NotEmpty(t, engine.rules)
}

func TestEngine_IgnoreRule(t *testing.T) {
	t.Parallel()
	engine := &Engine{}
	engine.IgnoreRule("test_rule")

	assert.True(t, engine.ignoredRules["test_rule"])
}

func TestEngine_IgnorePath(t *testing.T) {
	t.Parallel()
	engine := &Engine{}
	engine.IgnorePath("test_path")

	// EPR-7: IgnorePath normalizes patterns to absolute form so
	// relative and absolute callers both match against absolute
	// internal paths. Stored value is the normalized pattern, not
	// the raw input.
	require.Len(t, engine.ignoredPaths, 1)
	assert.True(t, filepath.IsAbs(engine.ignoredPaths[0]),
		"IgnorePath must persist patterns in absolute form, got %q",
		engine.ignoredPaths[0])
	assert.True(t, strings.HasSuffix(engine.ignoredPaths[0], "test_path"),
		"normalized pattern must end with the original input")
}

func TestEngine_PrepareFile(t *testing.T) {
	t.Parallel()
	engine := &Engine{}

	t.Run("Go file", func(t *testing.T) {
		t.Parallel()
		goFile := "test.go"
		result, err := engine.prepareFile(goFile)
		assert.NoError(t, err)
		assert.Equal(t, goFile, result)
	})

	t.Run("Gno file", func(t *testing.T) {
		t.Parallel()
		tempDir := createTempDir(t, "gno_test")

		gnoFile := filepath.Join(tempDir, "test.gno")
		err := os.WriteFile(gnoFile, []byte("package main"), 0o644)
		require.NoError(t, err)

		result, err := engine.prepareFile(gnoFile)
		assert.NoError(t, err)
		assert.NotEqual(t, gnoFile, result)
		assert.True(t, filepath.Ext(result) == ".go")
	})
}

func TestEngine_CleanupTemp(t *testing.T) {
	t.Parallel()
	engine := &Engine{}

	tempDir := createTempDir(t, "cleanup_test")

	tempFile := filepath.Join(tempDir, "temp_test.go")
	_, err := os.Create(tempFile)
	require.NoError(t, err)

	engine.cleanupTemp(tempFile)
	_, err = os.Stat(tempFile)
	assert.True(t, os.IsNotExist(err))
}

func TestReadSourceCode(t *testing.T) {
	t.Parallel()
	tempDir := createTempDir(t, "source_code_test")

	testFile := filepath.Join(tempDir, "test.go")
	content := "package main\n\nfunc main() {\n\tprintln(\"Hello, World!\")\n}"
	err := os.WriteFile(testFile, []byte(content), 0o644)
	require.NoError(t, err)

	sourceCode, err := ReadSourceCode(testFile)
	assert.NoError(t, err)
	assert.NotNil(t, sourceCode)
	assert.Len(t, sourceCode.Lines, 5)
	assert.Equal(t, "package main", sourceCode.Lines[0])
}

// create dummy source code for benchmark
var testSrc = strings.Repeat("hello world", 5000)

func BenchmarkCreateTempGoFile(b *testing.B) {
	tempDir := createTempDir(b, "benchmark")

	gnoContent := []byte(testSrc)
	gnoFile := filepath.Join(tempDir, "main.gno")
	if err := os.WriteFile(gnoFile, gnoContent, 0o644); err != nil {
		b.Fatalf("failed to write temp gno file: %v", err)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		f, err := createTempGoFile(gnoFile)
		if err != nil {
			b.Fatalf("failed to create temp go file: %v", err)
		}
		os.Remove(f)
	}
}

// TestIsIgnoredPathPatterns covers the doublestar matcher introduced
// in EPR-7. Validates that ** crosses directory separators, single *
// stays segment-bounded, and that absolute/relative input both match
// against absolute internal paths.
func TestIsIgnoredPathPatterns(t *testing.T) {
	t.Parallel()
	_, currentFile, _, ok := runtime.Caller(0)
	require.True(t, ok)
	testDataDir := filepath.Join(filepath.Dir(currentFile), "../testdata")

	tests := []struct {
		name    string
		pattern string
		path    string
		want    bool
	}{
		{
			name:    "** matches across directories",
			pattern: filepath.Join(testDataDir, "**"),
			path:    filepath.Join(testDataDir, "regex/regex0.gno"),
			want:    true,
		},
		{
			name:    "** matches arbitrarily deep",
			pattern: filepath.Join(testDataDir, "**/*.gno"),
			path:    filepath.Join(testDataDir, "early_return/a3.gno"),
			want:    true,
		},
		{
			name:    "single * stays segment-bounded (no recursive descent)",
			pattern: filepath.Join(testDataDir, "regex/*"),
			path:    filepath.Join(testDataDir, "early_return/a3.gno"),
			want:    false,
		},
		{
			name:    "single * matches sibling files",
			pattern: filepath.Join(testDataDir, "regex/*"),
			path:    filepath.Join(testDataDir, "regex/regex0.gno"),
			want:    true,
		},
		{
			name:    "*.pb.go single-wildcard suffix",
			pattern: filepath.Join(testDataDir, "*.pb.go"),
			path:    filepath.Join(testDataDir, "thing.pb.go"),
			want:    true,
		},
		{
			name:    "exact file pattern matches",
			pattern: filepath.Join(testDataDir, "slice0.gno"),
			path:    filepath.Join(testDataDir, "slice0.gno"),
			want:    true,
		},
		{
			name:    "non-matching pattern",
			pattern: filepath.Join(testDataDir, "no-such-dir/*"),
			path:    filepath.Join(testDataDir, "regex/regex0.gno"),
			want:    false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			engine, err := NewEngine(nil)
			require.NoError(t, err)
			engine.IgnorePath(tt.pattern)
			assert.Equal(t, tt.want, engine.isIgnoredPath(tt.path))
		})
	}
}

// TestIsIgnoredPathRelativeAbsoluteMix verifies that the engine's
// internal absolute-path representation matches whether the user
// supplies the pattern (or the path under check) as relative or
// absolute. The engine normalizes both at insert and query time.
func TestIsIgnoredPathRelativeAbsoluteMix(t *testing.T) {
	t.Parallel()
	_, currentFile, _, ok := runtime.Caller(0)
	require.True(t, ok)
	testDataDir := filepath.Join(filepath.Dir(currentFile), "../testdata")
	absPath := filepath.Join(testDataDir, "slice0.gno")

	wd, err := os.Getwd()
	require.NoError(t, err)
	relPath, err := filepath.Rel(wd, absPath)
	require.NoError(t, err)

	cases := []struct {
		name      string
		pattern   string
		queryPath string
	}{
		{"abs pattern, abs query", absPath, absPath},
		{"rel pattern, abs query", relPath, absPath},
		{"abs pattern, rel query", absPath, relPath},
		{"rel pattern, rel query", relPath, relPath},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			engine, err := NewEngine(nil)
			require.NoError(t, err)
			engine.IgnorePath(tc.pattern)
			assert.True(t, engine.isIgnoredPath(tc.queryPath),
				"pattern=%q query=%q should match after normalization",
				tc.pattern, tc.queryPath)
		})
	}
}

func TestIgnorePaths(t *testing.T) {
	_, currentFile, _, ok := runtime.Caller(0)
	require.True(t, ok)
	testDataDir := filepath.Join(filepath.Dir(currentFile), "../testdata")

	engine, err := NewEngine(nil)
	if err != nil {
		t.Fatalf("failed to create engine: %v", err)
	}

	engine.IgnorePath(filepath.Join(testDataDir, "regex/*"))
	engine.IgnorePath(filepath.Join(testDataDir, "slice0.gno"))

	files, err := filepath.Glob(filepath.Join(testDataDir, "*/*.gno"))
	if err != nil {
		t.Fatalf("failed to list files: %v", err)
	}

	for _, file := range files {
		issues, err := engine.Run(file)
		if err != nil {
			t.Fatalf("failed to run engine: %v", err)
		}

		// Check if the ignored file is not in the issues
		for _, issue := range issues {
			assert.NotEqual(t, issue.Filename, filepath.Join(testDataDir, "slice0.gno"))
			assert.NotContains(t, issue.Filename, filepath.Join(testDataDir, "regex"))
		}
	}
}

func BenchmarkRun(b *testing.B) {
	_, currentFile, _, ok := runtime.Caller(0)
	require.True(b, ok)
	testDataDir := filepath.Join(filepath.Dir(currentFile), "../testdata")

	engine, err := NewEngine(nil)
	if err != nil {
		b.Fatalf("failed to create engine: %v", err)
	}

	files, err := filepath.Glob(filepath.Join(testDataDir, "*/*.gno"))
	if err != nil {
		b.Fatalf("failed to list files: %v", err)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		for _, file := range files {
			_, err := engine.Run(file)
			if err != nil {
				b.Fatalf("failed to run engine: %v", err)
			}
		}
	}
}

// TestRulesFireOnTestdata pins the engine's rule wiring: each default
// rule must fire on a known-positive testdata file with Issue.Rule
// equal to its registered name. Any divergence between registration
// and emission fails here.
//
// Excluded:
//   - golangci-lint: depends on the external binary.
//   - high-cyclomatic-complexity: not wired into the default Engine
//     (uses the lint.ProcessCyclomaticComplexity special path; covered
//     by TestDetectHighCyclomaticComplexity).
func TestRulesFireOnTestdata(t *testing.T) {
	t.Parallel()
	_, currentFile, _, ok := runtime.Caller(0)
	require.True(t, ok)
	testDataDir := filepath.Join(filepath.Dir(currentFile), "../testdata")

	cases := []struct {
		rule string
		file string // relative to testDataDir
	}{
		{"simplify-slice-range", "slice0.gno"},
		{"unnecessary-type-conversion", "coversion/conv0.gno"},
		{"cycle-detection", "cycle/types.gno"},
		{"emit-format", "emit/emit1.gno"},
		{"useless-break", "break/break1.gno"},
		{"early-return-opportunity", "early_return/a0.gno"},
		{"const-error-declaration", "const-error-decl/const_decl.gno"},
		{"repeated-regex-compilation", "regex/regex6.gno"},
		{"unused-package", "pkg/pkg0.gno"},
		{"simplify-for-range", "simple_for/for_len.gno"},
		{"format-without-verb", "format_verb/positive.gno"},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.rule, func(t *testing.T) {
			t.Parallel()
			engine, err := NewEngine(nil)
			require.NoError(t, err)

			path := filepath.Join(testDataDir, tc.file)
			issues, err := engine.Run(path)
			require.NoError(t, err)

			rules := make([]string, 0, len(issues))
			for _, issue := range issues {
				if issue.Rule == tc.rule {
					return
				}
				rules = append(rules, issue.Rule)
			}
			t.Fatalf("rule %q did not fire on %s; rules that fired: %v",
				tc.rule, tc.file, rules)
		})
	}
}

// TestConfigSeverityReachesIssues is the integration anchor for the
// severity-resolution chain:
//
//	Engine.severityOverrides
//	  → effectiveSeverity(r)
//	  → AnalysisContext.Severity
//	  → legacyRule forwards to inner check's severity arg
//	  → Issue.Severity
//
// PR-2's TestLegacyAdapterPropagatesContextSeverity and PR-3's
// TestNewEngineConfig pin individual links. This test pins the whole
// chain end-to-end, so a future change that quietly bypasses
// effectiveSeverity (e.g. simplifying it back to r.DefaultSeverity())
// fails here even when the unit tests stay green.
func TestConfigSeverityReachesIssues(t *testing.T) {
	t.Parallel()
	_, currentFile, _, ok := runtime.Caller(0)
	require.True(t, ok)
	testDataDir := filepath.Join(filepath.Dir(currentFile), "../testdata")

	// simplify-slice-range defaults to Error; override to Warning so a
	// passing test cannot be explained by the default already matching.
	config := map[string]types.ConfigRule{
		"simplify-slice-range": {Severity: types.SeverityWarning},
	}
	engine, err := NewEngine(config)
	require.NoError(t, err)

	issues, err := engine.Run(filepath.Join(testDataDir, "slice0.gno"))
	require.NoError(t, err)

	for _, issue := range issues {
		if issue.Rule == "simplify-slice-range" {
			assert.Equal(t, types.SeverityWarning, issue.Severity,
				"config override must propagate to Issue.Severity")
			return
		}
	}
	t.Fatal("simplify-slice-range did not fire on slice0.gno")
}

// --- EPR anchor tests (skipped until target PR lands) ---
//
// The four tests below are behavior gates for the execution-layer
// refactor (docs/architecture/09-execution-layer-todo.md). They are
// committed in t.Skip state so the test names show up in CI reports
// from EPR-0 onward; the corresponding EPR removes the skip and fills
// in the assertion. Doing it this way means the gate is auditable
// before the feature exists, and a future PR cannot quietly omit it.

// TestRuleErrorsAreLogged pins the EPR-1 contract: rule check failures
// surface as engine Warn entries instead of being silently dropped.
// Builds a zap observer logger, injects a fake rule that returns an
// error, and asserts the engine logged the failure with rule/file/err
// fields.
func TestRuleErrorsAreLogged(t *testing.T) {
	core, observed := observer.New(zap.WarnLevel)
	logger := zap.New(core)

	engine, err := NewEngine(nil,
		WithLogger(logger),
		WithRules(map[string]rule.Rule{"test-failing-rule": failingFakeRule{}}),
	)
	require.NoError(t, err)

	_, err = engine.RunSource([]byte("package main\n"))
	require.NoError(t, err, "engine itself must not return the rule's error")

	entries := observed.FilterMessage("rule check failed").All()
	require.Len(t, entries, 1, "engine must emit exactly one Warn for the failing rule")

	fields := entries[0].ContextMap()
	assert.Equal(t, "test-failing-rule", fields["rule"])
	errMsg, ok := fields["error"].(string)
	require.True(t, ok, "error field must be a string (zap.Error renders to string), got %T", fields["error"])
	assert.Contains(t, errMsg, "synthetic failure")
}

// TestEngineGoroutineFootprint pins the EPR-1 contract: the engine
// dispatches rules sequentially within a single Run call. Pre-EPR-1
// each Run spawned len(rules) ~ 12 goroutines; post-EPR-1 the count
// returns to baseline.
func TestEngineGoroutineFootprint(t *testing.T) {
	engine, err := NewEngine(nil)
	require.NoError(t, err)

	// Warm up internal lazy-inits (e.g. importer machinery) so the
	// pre-count isn't biased low.
	_, _ = engine.RunSource([]byte("package main\n"))

	runtime.GC()
	pre := runtime.NumGoroutine()

	_, err = engine.RunSource([]byte("package main\nfunc main() {}\n"))
	require.NoError(t, err)

	// Allow the runtime a beat to recycle any transient goroutines
	// that finished during Run (workers in the importer, etc.) so the
	// post count reflects the engine's own footprint.
	time.Sleep(20 * time.Millisecond)
	runtime.GC()
	post := runtime.NumGoroutine()

	assert.LessOrEqual(t, post, pre,
		"engine.RunSource must not leak per-rule goroutines (per-rule fan-out removed in EPR-1); pre=%d post=%d",
		pre, post)
}

// failingFakeRule satisfies rule.Rule by returning a synthetic error
// from Check, used by TestRuleErrorsAreLogged to drive the Warn path.
type failingFakeRule struct{}

func (failingFakeRule) Name() string                                        { return "test-failing-rule" }
func (failingFakeRule) DefaultSeverity() types.Severity                     { return types.SeverityError }
func (failingFakeRule) Check(*rule.AnalysisContext) ([]types.Issue, error) {
	return nil, errors.New("synthetic failure")
}

// TestConfigurableRuleReceivesData pins the contract that ConfigRule.Data
// from .tlin.yaml flows into a rule's ParseConfig at engine
// construction. Without this wiring the `data:` block in the YAML
// would be silently ignored.
func TestConfigurableRuleReceivesData(t *testing.T) {
	reg := rule.NewRegistry()
	cr := &fakeConfigurableRule{name: "configurable-rule", sev: types.SeverityInfo}
	reg.Register(cr)

	cfg := map[string]types.ConfigRule{
		"configurable-rule": {
			Severity: types.SeverityWarning,
			Data:     map[string]any{"threshold": 5},
		},
	}
	_, err := NewEngine(cfg, WithRegistry(reg))
	require.NoError(t, err)

	require.NotNil(t, cr.captured, "ParseConfig must be called when Data is non-nil")
	captured, ok := cr.captured.(map[string]any)
	require.True(t, ok, "captured raw should be the YAML-decoded value, got %T", cr.captured)
	assert.Equal(t, 5, captured["threshold"])
}

// TestConfigurableRuleParseErrorLogsWarn pins the fail-open contract:
// a rule whose ParseConfig returns an error must not stop NewEngine.
// The engine logs Warn and the rule keeps its default behavior.
func TestConfigurableRuleParseErrorLogsWarn(t *testing.T) {
	core, observed := observer.New(zap.WarnLevel)
	logger := zap.New(core)

	reg := rule.NewRegistry()
	reg.Register(&fakeConfigurableRule{
		name:     "broken-rule",
		sev:      types.SeverityInfo,
		parseErr: errors.New("invalid threshold"),
	})

	_, err := NewEngine(
		map[string]types.ConfigRule{
			"broken-rule": {Severity: types.SeverityInfo, Data: "anything"},
		},
		WithRegistry(reg),
		WithLogger(logger),
	)
	require.NoError(t, err, "engine init must not propagate ParseConfig errors")

	entries := observed.FilterMessage("rule config parse failed").All()
	require.Len(t, entries, 1, "engine must emit exactly one Warn for the failing ParseConfig")
	assert.Equal(t, "broken-rule", entries[0].ContextMap()["rule"])
}

// fakeConfigurableRule satisfies both rule.Rule and
// rule.ConfigurableRule for the tests above. captured holds the raw
// data ParseConfig saw; parseErr lets a test simulate config failure.
type fakeConfigurableRule struct {
	name     string
	sev      types.Severity
	captured any
	parseErr error
}

func (f *fakeConfigurableRule) Name() string                                        { return f.name }
func (f *fakeConfigurableRule) DefaultSeverity() types.Severity                     { return f.sev }
func (f *fakeConfigurableRule) Check(*rule.AnalysisContext) ([]types.Issue, error)  { return nil, nil }
func (f *fakeConfigurableRule) ParseConfig(raw any) error {
	f.captured = raw
	return f.parseErr
}

// TestIssueFilenameIsOriginalPath pins the EPR-3 contract that
// every issue's Filename is the user-supplied path, structurally
// — not via an after-the-fact remap loop in the engine. With every
// rule built around ctx.NewIssue, the loop is gone and this test is
// the guarantee.
func TestIssueFilenameIsOriginalPath(t *testing.T) {
	t.Parallel()

	tempDir := createTempDir(t, "issue_filename_test")

	gnoFile := filepath.Join(tempDir, "src.gno")
	content := `package main

func main() {
	x := 5
	y := int(x) // unnecessary type conversion
	_ = y

	for i := 0; i < 5; i++ {
		if i == 0 {
			break
		}
		break
	}
}
`
	require.NoError(t, os.WriteFile(gnoFile, []byte(content), 0o644))

	engine, err := NewEngine(nil)
	require.NoError(t, err)

	issues, err := engine.Run(gnoFile)
	require.NoError(t, err)
	require.NotEmpty(t, issues, "fixture should trigger at least one rule")

	for _, issue := range issues {
		assert.Equal(t, gnoFile, issue.Filename,
			"top-level Filename must be the original .gno path (rule %q)", issue.Rule)
		assert.Equal(t, gnoFile, issue.Start.Filename,
			"Start.Filename must be remapped (rule %q)", issue.Rule)
		assert.Equal(t, gnoFile, issue.End.Filename,
			"End.Filename must be remapped (rule %q)", issue.Rule)
		assert.True(t, strings.HasSuffix(issue.Filename, ".gno"),
			"Filename must keep the .gno suffix; rule %q produced %q", issue.Rule, issue.Filename)
	}
}

// TestRunHonoursContextCancel pins the EPR-4 contract that the
// engine respects context.Context cancellation. A pre-cancelled
// context returned to RunWithContext / RunSourceWithContext stops
// dispatch on the next rule boundary and surfaces ctx.Err().
func TestRunHonoursContextCancel(t *testing.T) {
	t.Parallel()

	tempDir := createTempDir(t, "ctx_cancel_test")
	gnoFile := filepath.Join(tempDir, "src.gno")
	require.NoError(t, os.WriteFile(gnoFile, []byte("package main\nfunc main() {}\n"), 0o644))

	engine, err := NewEngine(nil)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err = engine.RunWithContext(ctx, gnoFile)
	assert.ErrorIs(t, err, context.Canceled,
		"RunWithContext on a cancelled ctx must surface ctx.Err()")

	_, err = engine.RunSourceWithContext(ctx, []byte("package main\n"))
	assert.ErrorIs(t, err, context.Canceled,
		"RunSourceWithContext on a cancelled ctx must surface ctx.Err()")
}

// A cancelled engine run must leave no dangling goroutines — a
// guard against reintroducing the pre-EPR-1 goroutine-per-rule
// fan-out that the old runWithTimeout orphaned on os.Exit.
func TestTimeoutLeakFree(t *testing.T) {
	defer goleak.VerifyNone(t)

	engine, err := NewEngine(nil)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, _ = engine.RunSourceWithContext(ctx, []byte("package main\n"))
}

func createTempDir(tb testing.TB, prefix string) string {
	tb.Helper()
	tempDir, err := os.MkdirTemp("", prefix)
	require.NoError(tb, err)
	tb.Cleanup(func() { os.RemoveAll(tempDir) })
	return tempDir
}

// TestGnoFileMapping tests that issues from .gno files are correctly mapped back
func TestGnoFileMapping(t *testing.T) {
	t.Parallel()

	tempDir := createTempDir(t, "gno_mapping_test")

	// Create a .gno file with known issues
	gnoFile := filepath.Join(tempDir, "test.gno")
	content := `package main

func test() {
	// Unnecessary type conversion that will be detected
	x := 5
	y := int(x) // This will trigger unnecessary-type-conversion rule
	
	// Useless break that will be detected
	for i := 0; i < 10; i++ {
		if i == 5 {
			break
		}
		break // This will trigger useless-break rule
	}
	
	_ = y
}
`
	err := os.WriteFile(gnoFile, []byte(content), 0o644)
	require.NoError(t, err)

	engine, err := NewEngine(nil)
	require.NoError(t, err)

	issues, err := engine.Run(gnoFile)
	assert.NoError(t, err)

	// Assert that we found at least one issue
	assert.NotEmpty(t, issues, "Should have found at least one issue in the .gno file")

	// Check that all issues have the original .gno filename
	issueCount := 0
	for _, issue := range issues {
		assert.Equal(t, gnoFile, issue.Filename, "Issue filename should be mapped back to .gno file")
		assert.True(t, strings.HasSuffix(issue.Filename, ".gno"), "Issue filename should end with .gno")
		issueCount++
	}

	// Ensure we actually checked some issues
	assert.Greater(t, issueCount, 0, "Should have found and checked at least one issue")
}

// TestNolintIsolation tests that nolint comments from one file don't affect another
func TestNolintIsolation(t *testing.T) {
	t.Parallel()

	tempDir := createTempDir(t, "nolint_isolation_test")

	// Create two files with a detectable issue
	file1 := filepath.Join(tempDir, "file1.go")
	content1 := `package main

//nolint:unnecessary-type-conversion
func test1() {
	x := int(5)
	y := int(x) // unnecessary type conversion
	_ = y
}
`
	err := os.WriteFile(file1, []byte(content1), 0o644)
	require.NoError(t, err)

	file2 := filepath.Join(tempDir, "file2.go")
	content2 := `package main

func test2() {
	x := int(5)
	y := int(x) // unnecessary type conversion
	_ = y
}
`
	err = os.WriteFile(file2, []byte(content2), 0o644)
	require.NoError(t, err)

	engine, err := NewEngine(nil)
	require.NoError(t, err)

	// Run both files
	issues1, err := engine.Run(file1)
	assert.NoError(t, err)

	issues2, err := engine.Run(file2)
	assert.NoError(t, err)

	// File1 should have fewer issues due to nolint comment
	// File2 should have issues
	assert.Less(t, len(issues1), len(issues2), "File1 with nolint should have fewer issues than file2")

	// Verify that file2 has the unnecessary-type-conversion issue
	hasTypeConversionIssue := false
	for _, issue := range issues2 {
		if issue.Rule == "unnecessary-type-conversion" {
			hasTypeConversionIssue = true
			break
		}
	}
	assert.True(t, hasTypeConversionIssue, "File2 should have unnecessary-type-conversion issue")

	// Verify that file1 doesn't have the unnecessary-type-conversion issue
	for _, issue := range issues1 {
		assert.NotEqual(t, "unnecessary-type-conversion", issue.Rule, "File1 should not have unnecessary-type-conversion issue due to nolint")
	}
}

// TestConcurrentRuns tests that concurrent runs don't interfere with each other
func TestConcurrentRuns(t *testing.T) {
	t.Parallel()

	tempDir := createTempDir(t, "concurrent_test")

	// Create multiple test files with deterministic issues
	numFiles := 10
	files := make([]string, numFiles)
	for i := 0; i < numFiles; i++ {
		filename := filepath.Join(tempDir, fmt.Sprintf("test%d.go", i))
		// Each file has a unique pattern of unnecessary type conversions
		// File i will have i+1 unnecessary conversions
		content := fmt.Sprintf(`package main

// File %d
func test%d() {
	x := 5
`, i, i)

		// Add i+1 unnecessary type conversions
		for j := 0; j <= i; j++ {
			content += fmt.Sprintf("\tv%d := int(x) // unnecessary conversion %d in file %d\n", j, j, i)
		}

		content += "\t_ = x\n"
		// Add usages to avoid unused variable warnings
		for j := 0; j <= i; j++ {
			content += fmt.Sprintf("\t_ = v%d\n", j)
		}
		content += "}\n"

		err := os.WriteFile(filename, []byte(content), 0o644)
		require.NoError(t, err)
		files[i] = filename
	}

	engine, err := NewEngine(nil)
	require.NoError(t, err)

	// Run all files concurrently
	type result struct {
		filename string
		issues   []types.Issue
		err      error
	}

	resultChan := make(chan result, numFiles)
	for _, file := range files {
		go func(f string) {
			issues, err := engine.Run(f)
			resultChan <- result{filename: f, issues: issues, err: err}
		}(file)
	}

	// Collect results
	results := make(map[string][]types.Issue)
	totalIssues := 0
	for i := 0; i < numFiles; i++ {
		r := <-resultChan
		assert.NoError(t, r.err)
		results[r.filename] = r.issues
		totalIssues += len(r.issues)
	}

	// Verify we found issues
	assert.Greater(t, totalIssues, 0, "Should have found at least some issues across all files")

	// Verify that each file has at least one issue (since each file has at least one unnecessary conversion)
	for filename, issues := range results {
		assert.NotEmpty(t, issues, "File %s should have at least one issue", filename)

		// Verify that all issues belong to the correct file
		for _, issue := range issues {
			assert.Equal(t, filename, issue.Filename, "Issue should belong to the correct file")
		}
	}

	// Verify that files have different numbers of issues (as designed)
	// This helps ensure isolation - if nolint or other state leaked between files,
	// we might see identical issue counts
	issueCounts := make(map[int]int)
	for _, issues := range results {
		issueCounts[len(issues)]++
	}
	assert.Greater(t, len(issueCounts), 1, "Files should have varying numbers of issues, indicating proper isolation")
}
