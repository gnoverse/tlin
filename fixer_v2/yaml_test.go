package fixerv2

import (
	"os"
	"testing"
)

func TestLoadFixRules(t *testing.T) {
	tests := []struct {
		name        string
		yamlContent string
		wantRules   []FixRule
		wantErr     bool
	}{
		{
			name: "valid rules",
			yamlContent: `
rules:
  - name: simple replacement
    pattern: ":[name]"
    replacement: "Hello, :[name]!"
  - name: arithmetic replacement
    pattern: ":[lhs] + :[rhs]"
    replacement: ":[rhs] - :[lhs]"
`,
			wantRules: []FixRule{
				{
					Name:        "simple replacement",
					Pattern:     ":[name]",
					Replacement: "Hello, :[name]!",
				},
				{
					Name:        "arithmetic replacement",
					Pattern:     ":[lhs] + :[rhs]",
					Replacement: ":[rhs] - :[lhs]",
				},
			},
			wantErr: false,
		},
		{
			name: "invalid yaml",
			yamlContent: `
rules:
  - name: missing colon
    pattern ":[abc]"
    replacement: "Should fail"
`,
			wantRules: nil,
			wantErr:   true,
		},
		{
			name: "golang lint rules",
			yamlContent: `
rules:
  - name: if err handling
    pattern: "if err != nil { return err }"
    replacement: "if err != nil { return fmt.Errorf(\"failed to process: %w\", err) }"
  - name: context timeout
    pattern: "context.WithTimeout(:[ctx], :[duration])"
    replacement: "context.WithTimeout(:[ctx], :[duration])\ndefer cancel()"
  - name: slice capacity
    pattern: "make([]:[type], 0)"
    replacement: "make([]:[type], 0, :[capacity])"
`,
			wantRules: []FixRule{
				{
					Name:        "if err handling",
					Pattern:     "if err != nil { return err }",
					Replacement: "if err != nil { return fmt.Errorf(\"failed to process: %w\", err) }",
				},
				{
					Name:        "context timeout",
					Pattern:     "context.WithTimeout(:[ctx], :[duration])",
					Replacement: "context.WithTimeout(:[ctx], :[duration])\ndefer cancel()",
				},
				{
					Name:        "slice capacity",
					Pattern:     "make([]:[type], 0)",
					Replacement: "make([]:[type], 0, :[capacity])",
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpfile, err := os.CreateTemp("", "rules_*.yaml")
			if err != nil {
				t.Fatalf("TempFile error: %v", err)
			}
			defer os.Remove(tmpfile.Name())

			if _, err := tmpfile.Write([]byte(tt.yamlContent)); err != nil {
				t.Fatalf("Write error: %v", err)
			}
			if err := tmpfile.Close(); err != nil {
				t.Fatalf("Close error: %v", err)
			}

			rules, err := Load(tmpfile.Name())
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error but got none")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(rules) != len(tt.wantRules) {
				t.Fatalf("expected %d rule(s), got %d", len(tt.wantRules), len(rules))
			}

			for i, want := range tt.wantRules {
				got := rules[i]
				if got.Name != want.Name {
					t.Errorf("rule[%d] Name: got %q, want %q", i, got.Name, want.Name)
				}
				if got.Pattern != want.Pattern {
					t.Errorf("rule[%d] Pattern: got %q, want %q", i, got.Pattern, want.Pattern)
				}
				if got.Replacement != want.Replacement {
					t.Errorf("rule[%d] Replacement: got %q, want %q", i, got.Replacement, want.Replacement)
				}
			}
		})
	}
}
