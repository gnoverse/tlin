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
