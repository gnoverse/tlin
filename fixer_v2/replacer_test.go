package fixerv2

import (
	"testing"
)

func TestReplacer(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name           string
		patternStr     string
		replacementStr string
		subjectStr     string
		expected       string
	}{
		{
			name:           "simple replacement",
			patternStr:     ":[name]",
			replacementStr: "Hello, :[name]!",
			subjectStr:     "John",
			expected:       "Hello, John!",
		},
		{
			name:           "arithmetic expression replacement",
			patternStr:     ":[lhs] + :[rhs]",
			replacementStr: ":[rhs] - :[lhs]",
			subjectStr:     "12 + 34",
			expected:       "34 - 12",
		},
		{
			name:           "multiple replacements",
			patternStr:     ":[lhs] + :[rhs]",
			replacementStr: ":[rhs] - :[lhs]",
			subjectStr:     "12 + 34",
			expected:       "34 - 12",
		},
		{
			name:           "multiple replacements 2",
			patternStr:     ":[lhs] + :[rhs]",
			replacementStr: ":[rhs] - :[lhs]",
			subjectStr:     "12 + 34, 56 + 78",
			expected:       "34 - 12, 78 - 56",
		},
		{
			name:           "function return replacement",
			patternStr:     "return :[expr]",
			replacementStr: "return (:[expr])",
			subjectStr:     "func test() {\n    return x + 1\n}",
			expected:       "func test() {\n    return (x + 1)\n}",
		},
		{
			name:           "error handling improvement",
			patternStr:     "if err != nil { return err }",
			replacementStr: "if err != nil { return fmt.Errorf(\"failed to process: %w\", err) }",
			subjectStr:     "func process() error {\n    if err != nil { return err }\n}",
			expected:       "func process() error {\n    if err != nil { return fmt.Errorf(\"failed to process: %w\", err) }\n}",
		},
		{
			name:           "context with cancel",
			patternStr:     "ctx, _ := context.WithTimeout(:[parent], :[duration])",
			replacementStr: "ctx, cancel := context.WithTimeout(:[parent], :[duration])\ndefer cancel()",
			subjectStr:     "func process() {\n    ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)\n}",
			expected:       "func process() {\n    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)\n    defer cancel()\n}",
		},
		{
			name:           "mutex lock with defer unlock",
			patternStr:     "mu.Lock()\n:[code]",
			replacementStr: "mu.Lock()\ndefer mu.Unlock()\n:[code]",
			subjectStr:     "func process() {\n    mu.Lock()\n    data.Write()\n}",
			expected:       "func process() {\n    mu.Lock()\n    defer mu.Unlock()\n    data.Write()\n}",
		},
		{
			name:           "error variable naming",
			patternStr:     ":[type], err := :[func]()",
			replacementStr: ":[type], :[type]Err := :[func]()",
			subjectStr:     "user, err := getUser()",
			expected:       "user, userErr := getUser()",
		},
		{
			name:           "http error handling",
			patternStr:     "http.Error(:[w], err.Error(), :[code])",
			replacementStr: "http.Error(:[w], fmt.Sprintf(\"internal error: %v\", err), :[code])",
			subjectStr:     "http.Error(w, err.Error(), http.StatusInternalServerError)",
			expected:       "http.Error(w, fmt.Sprintf(\"internal error: %v\", err), http.StatusInternalServerError)",
		},
		{
			name:           "nested if statements",
			patternStr:     "if :[cond1] {\n    if :[cond2] {\n        :[code]\n    }\n}",
			replacementStr: "if :[cond1] && :[cond2] {\n    :[code]\n}",
			subjectStr:     "if x > 0 {\n    if y < 10 {\n        process()\n    }\n}",
			expected:       "if x > 0 && y < 10 {\n    process()\n}",
		},
		{
			name:           "unnecessary else block",
			patternStr:     "if :[cond] {\n    :[code1]\n} else {\n    :[code2]\n}",
			replacementStr: "if :[cond] {\n    :[code1]\n}",
			subjectStr:     "if x > 0 {\n    process()\n} else {\n    log.Println(\"error\")\n}",
			expected:       "if x > 0 {\n    process()\n}",
		},
		{
			name:           "channel close with defer",
			patternStr:     "ch := make(chan :[type])",
			replacementStr: "ch := make(chan :[type])\ndefer close(ch)",
			subjectStr:     "func process() {\n    ch := make(chan int)\n}",
			expected:       "func process() {\n    ch := make(chan int)\n    defer close(ch)\n}",
		},
		{
			name:           "multiple errors in one line",
			patternStr:     "err1, err2 := :[func1](), :[func2]()",
			replacementStr: "err1Res, err2Res := :[func1](), :[func2]()",
			subjectStr:     "err1, err2 := readFile(), writeFile()",
			expected:       "err1Res, err2Res := readFile(), writeFile()",
		},
		{
			name:           "whitespace handling",
			patternStr:     "if   :[cond]   {",
			replacementStr: "if :[cond] {",
			subjectStr:     "func test() {\n    if   x > 0   {\n}",
			expected:       "func test() {\n    if x > 0 {\n}",
		},
		{
			name:           "complex nested replacement",
			patternStr:     "for :[i] := range :[slice] {\n    if :[cond] {\n        :[code]\n    }\n}",
			replacementStr: "for :[i] := range :[slice] {\n    switch {\n    case :[cond]:\n        :[code]\n    }\n}",
			subjectStr:     "for i := range items {\n    if items[i].Valid {\n        process(items[i])\n    }\n}",
			expected:       "for i := range items {\n    switch {\n    case items[i].Valid:\n        process(items[i])\n    }\n}",
		},
		// TODO (@notJoon): If capacity is not provided, it will be replaced with an empty string (ex: make([]int, 0, )).
		// To solve this, we need add an arbitary default value like I did here, or analyze the context
		// to find an appropriate value from other lines of code.
		// The context must be located in the same or higher scope and should be bounded by the lines
		// proceeding the current line.
		{
			name:           "slice capacity",
			patternStr:     "make([]:[type], 0)",
			replacementStr: "make([]:[type], 0, 10)",
			subjectStr:     "func test() {\n    data := make([]int, 0)\n}",
			expected:       "func test() {\n    data := make([]int, 0, 10)\n}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			patternTokens, err := Lex(tt.patternStr)
			if err != nil {
				t.Fatalf("pattern lex error: %v", err)
			}
			patternNodes, err := Parse(patternTokens)
			if err != nil {
				t.Fatalf("pattern parse error: %v", err)
			}

			replacementTokens, err := Lex(tt.replacementStr)
			if err != nil {
				t.Fatalf("replacement lex error: %v", err)
			}
			replacementNodes, err := Parse(replacementTokens)
			if err != nil {
				t.Fatalf("replacement parse error: %v", err)
			}

			repl := NewReplacer(patternNodes, replacementNodes)
			// result := ReplaceAll(patternNodes, replacementNodes, tt.subjectStr)
			result := repl.ReplaceAll(tt.subjectStr)

			if result != tt.expected {
				t.Errorf("replaceAll() = %q, want %q", result, tt.expected)
			}
		})
	}
}
