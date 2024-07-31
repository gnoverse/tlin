package internal

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRemoveUnnecessaryElse(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name: "don't need to modify",
			input: `if x {
	println("x")
} else {
	println("hello")
}`,
			expected: `if x {
	println("x")
} else {
	println("hello")
}`,
		},
		{
			name: "remove unnecessary else",
			input: `if x {
	return 1
} else {
	return 2
}`,
			expected: `if x {
	return 1
}
return 2`,
		},
		{
			name: "nested if else",
			input: `if x {
	return 1
}
if z {
	println("x")
} else {
	if y {
		return 2
	} else {
		return 3
	}
}
`,
			expected: `if x {
	return 1
}
if z {
	println("x")
} else {
	if y {
		return 2
	}
	return 3

}`,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			improved, err := RemoveUnnecessaryElse(tt.input)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, improved, "Improved code does not match expected output")
		})
	}
}
