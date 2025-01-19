package query

import (
	"reflect"
	"testing"
)

func TestSpecialPatterns(t *testing.T) {
	patterns := []struct {
		input    string
		desc     string
		expected []States
	}{
		{
			input: ":[[var:identifier]]*",
			desc:  "Identifier with zero-or-more",
			expected: []States{
				CL, OB, DB,
				NM, NM, NM,
				ID, ID, ID,
				ID, ID, ID,
				ID, ID, ID,
				ID, ID, CB,
				QB, QT,
			},
		},
		{
			input: ":[var] :[next]",
			desc:  "Multiple holes",
			expected: []States{
				CL, OB, NM,
				NM, NM, CB,
				WS, CL, OB,
				NM, NM, NM,
				NM, CB,
			},
		},
	}

	for _, p := range patterns {
		t.Run(p.desc, func(t *testing.T) {
			sm := NewStateMachine(p.input)
			transitions := sm.recordTransitions()

			states := make([]States, len(transitions))
			for i, tr := range transitions {
				states[i] = tr.toState
			}

			t.Logf("\n Input: %s", p.input)
			t.Logf("\nTransitions:\n%s", visualizeTransitions(transitions))

			if !reflect.DeepEqual(states, p.expected) {
				t.Errorf("\nGot:  %v\nWant: %v", states, p.expected)
				t.Logf("\nTransitions:\n%s", visualizeTransitions(transitions))
			}
		})
	}
}

func TestStrictFinalState(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		shouldError bool
	}{
		{
			name:        "Properly closed double brackets",
			input:       ":[[var]]",
			shouldError: false,
		},
		{
			name:        "Missing closing brackets",
			input:       ":[[var",
			shouldError: true,
		},
		{
			name:        "Multiple properly closed single brackets",
			input:       ":[var] :[next]",
			shouldError: false,
		},
		{
			name:        "Multiple properly closed double brackets",
			input:       ":[[var:foo]] :[[next:bar]]",
			shouldError: false,
		},
		{
			name:        "Properly closed with quantifier (*)",
			input:       ":[[var:identifier]]*",
			shouldError: false,
		},
		{
			name:        "Properly closed with quantifier (?)",
			input:       ":[[var:identifier]]?",
			shouldError: false,
		},
		{
			name:        "Properly closed with quantifier (+)",
			input:       ":[[var:identifier]]+",
			shouldError: false,
		},
		{
			name:        "Missing closing bracket for double brackets",
			input:       ":[[var:identifier]",
			shouldError: true,
		},
		{
			name:        "Single bracket with quantifier",
			input:       ":[var]*",
			shouldError: false,
		},
		{
			name:        "Single bracket with quantifier and text",
			input:       ":[var:*]",
			shouldError: true,
		},
		{
			name:        "Properly closed with quantifier and some text",
			input:       ":[[var:identifier]]? some text",
			shouldError: false,
		},
		{
			name:        "Closing bracket inside text",
			input:       ":[[var:identifier]]? incomplete ]",
			shouldError: true,
		},
		{
			name:        "Empty input",
			input:       "",
			shouldError: true,
		},
		{
			name:        "Input with only text",
			input:       "some regular text without brackets",
			shouldError: false,
		},
		{
			name:        "Input starting with quantifier",
			input:       "*:[var]",
			shouldError: true,
		},
		{
			name:        "Input with unexpected special characters",
			input:       ":[[var@]]",
			shouldError: true,
		},
		{
			name:        "Nested brackets",
			input:       ":[[var:[[nested]]]]",
			shouldError: true,
		},
		{
			name:        "Multiple quantifiers in sequence",
			input:       ":[[var:identifier]]**",
			shouldError: true,
		},
		{
			name:        "Identifier with spaces",
			input:       ":[var name]",
			shouldError: true,
		},
		{
			name:        "Input with multiple errors",
			input:       ":[var]*]",
			shouldError: true,
		},
		{
			name:        "Quantifier without identifier",
			input:       ":[*]",
			shouldError: true,
		},
		{
			name:        "Colon without identifier",
			input:       ":[]",
			shouldError: true,
		},
		{
			name:        "Unmatched closing brace",
			input:       ":{:[var]}",
			shouldError: true,
		},
		// {
		// 	name:        "Unmatched opening brace",
		// 	input:       "{:[var]",
		// 	shouldError: true,
		// },
		// {
		// 	name:        "Properly closed with nested braces",
		// 	input:       "{:[[var:foo]]}",
		// 	shouldError: false,
		// },
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sm := NewStateMachine(tt.input)
			transitions, err := sm.recordTransitionsStrict()

			t.Logf("Input: %q\n%s", tt.input, visualizeTransitions(transitions))

			if tt.shouldError {
				if err == nil {
					t.Errorf("Expected error, got nil; final state = %v", sm.state)
				} else {
					t.Logf("Got expected error: %v", err)
				}
			} else {
				if err != nil {
					t.Errorf("Did not expect error, got %v; final state = %v", err, sm.state)
				} else {
					t.Logf("Completed parse successfully in state %v", sm.state)
				}
			}
		})
	}
}
