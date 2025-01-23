package query

import (
	"testing"
)

func TestHoleExtraction(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty hole name",
			input:    ":[[]]",
			expected: "",
		},
		{
			name:     "simple hole",
			input:    ":[name]",
			expected: "name",
		},
		{
			name:     "double bracket hole",
			input:    ":[[variable]]",
			expected: "variable",
		},
		{
			name:     "hole with numbers",
			input:    ":[test123]",
			expected: "test123",
		},
		{
			name:     "hole with underscores",
			input:    ":[test_var_123]",
			expected: "test_var_123",
		},
		{
			name:     "hole with special characters",
			input:    ":[[test-var]]",
			expected: "test-var",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractHoleName(tt.input)
			if result != tt.expected {
				t.Errorf("extractHoleName() got = %v, want %v", result, tt.expected)
			}
		})
	}
}
