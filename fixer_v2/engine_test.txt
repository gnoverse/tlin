package fixerv2

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

type TestResult struct {
	vars    map[string]string
	rewrite string
}

type TestCase struct {
	name       string
	pattern    Pattern
	input      string
	wantMatch  bool
	wantResult TestResult
}

func TestPatternMatching(t *testing.T) {
	tests := []TestCase{
		{
			name: "basic if-else to return",
			pattern: Pattern{
				Match: `if :[[cond]] {
                    return true
                } else {
                    return false
                }`,
				Rewrite: "return :[[cond]]",
			},
			input: `
                func example() bool {
                    if x > 0 {
                        return true
                    } else {
                        return false
                    }
                }`,
			wantMatch: true,
			wantResult: TestResult{
				vars: map[string]string{
					"cond": "x > 0",
				},
				rewrite: "return x > 0",
			},
		},
		{
			name: "no match for different pattern",
			pattern: Pattern{
				Match: `if :[[cond]] {
                    return true
                } else {
                    return false
                }`,
				Rewrite: "return :[[cond]]",
			},
			input: `
                func example() bool {
                    if x > 0 {
                        return true
                    }
                    return false
                }`,
			wantMatch: false,
			wantResult: TestResult{
				vars:    nil,
				rewrite: "",
			},
		},
		{
			name: "match with nested conditions",
			pattern: Pattern{
				Match:   "if :[[outer]] { if :[[inner]] { :[[body]] } }",
				Rewrite: "if :[[outer]] && :[[inner]] { :[[body]] }",
			},
			input: `
                func example() {
                    if x > 0 { if y < 10 { doSomething() } }
                }`,
			wantMatch: true,
			wantResult: TestResult{
				vars: map[string]string{
					"outer": "x > 0",
					"inner": "y < 10",
					"body":  "doSomething()",
				},
				rewrite: "if x > 0 && y < 10 { doSomething() }",
			},
		},
		{
			name: "match with short syntax",
			pattern: Pattern{
				Match:   "func :[name]() :[ret] { :[body] }",
				Rewrite: "func :[name]() :[ret] {\n  // Added comment\n  :[body]\n}",
			},
			input: `
                func example() bool { return true }`,
			wantMatch: true,
			wantResult: TestResult{
				vars: map[string]string{
					"name": "example",
					"ret":  "bool",
					"body": "return true",
				},
				rewrite: "func example() bool {\n  // Added comment\n  return true\n}",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resultOpt := patternToRegex(tt.pattern.Match)
			assert.NoError(t, resultOpt.err, "patternToRegex should not return error")

			if resultOpt.err != nil {
				return
			}

			result := resultOpt.value
			normalizedInput := normalizePattern(tt.input)
			matches := result.regex.FindAllStringSubmatch(normalizedInput, -1)

			if tt.wantMatch {
				assert.NotEmpty(t, matches, "expected to find matches")
				if len(matches) > 0 {
					env := extractEnvironment(t, matches[0], result.captures)
					assert.Equal(t, tt.wantResult.vars, env, "captured variables should match")

					rewritten := rewrite(tt.pattern.Rewrite, env)
					assert.Equal(t, tt.wantResult.rewrite, rewritten, "rewritten code should match")
				}
			} else {
				assert.Empty(t, matches, "expected no matches")
			}
		})
	}
}

// extractEnvironment is a helper function to extract captured variables
func extractEnvironment(t *testing.T, match []string, captures map[string]int) map[string]string {
	t.Helper()
	env := make(map[string]string)
	for name, idx := range captures {
		if idx < len(match) {
			env[name] = strings.TrimSpace(match[idx])
		}
	}
	return env
}
