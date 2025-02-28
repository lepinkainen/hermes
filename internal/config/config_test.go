package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSetOverwriteFiles(t *testing.T) {
	// Save the original value to restore after the test
	originalValue := OverwriteFiles

	testCases := []struct {
		name     string
		input    bool
		expected bool
	}{
		{
			name:     "set to true",
			input:    true,
			expected: true,
		},
		{
			name:     "set to false",
			input:    false,
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Set the value
			SetOverwriteFiles(tc.input)

			// Check that the global variable was updated
			assert.Equal(t, tc.expected, OverwriteFiles)
		})
	}

	// Restore the original value
	OverwriteFiles = originalValue
}
