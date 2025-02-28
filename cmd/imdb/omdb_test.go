package imdb

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseFloat(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected float64
	}{
		{
			name:     "valid float",
			input:    "7.5",
			expected: 7.5,
		},
		{
			name:     "integer as string",
			input:    "8",
			expected: 8.0,
		},
		{
			name:     "empty string",
			input:    "",
			expected: 0.0,
		},
		{
			name:     "non-numeric string",
			input:    "N/A",
			expected: 0.0,
		},
		{
			name:     "string with whitespace",
			input:    " 9.2 ",
			expected: 0.0, // parseFloat doesn't trim whitespace
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := parseFloat(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestParseRuntime(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected int
	}{
		{
			name:     "standard format",
			input:    "120 min",
			expected: 120,
		},
		{
			name:     "single digit",
			input:    "9 min",
			expected: 9,
		},
		{
			name:     "empty string",
			input:    "",
			expected: 0,
		},
		{
			name:     "no min suffix",
			input:    "135",
			expected: 135, // parseRuntime will just convert the string to an integer
		},
		{
			name:     "non-numeric prefix",
			input:    "about 90 min",
			expected: 0, // parseRuntime can't handle non-numeric prefixes
		},
		{
			name:     "different suffix",
			input:    "120 minutes",
			expected: 0, // After trimming " min", "120 utes" is not a valid integer
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := parseRuntime(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}
