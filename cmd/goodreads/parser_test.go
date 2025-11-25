package goodreads

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSplitString(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "empty string",
			input:    "",
			expected: nil,
		},
		{
			name:     "single value",
			input:    "Fiction",
			expected: []string{"Fiction"},
		},
		{
			name:     "multiple values",
			input:    "Fiction,Fantasy,Adventure",
			expected: []string{"Fiction", "Fantasy", "Adventure"},
		},
		{
			name:     "values with spaces",
			input:    "Fiction, Fantasy, Adventure",
			expected: []string{"Fiction", "Fantasy", "Adventure"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := splitString(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestGetDescription(t *testing.T) {
	testCases := []struct {
		name     string
		input    any
		expected string
	}{
		{
			name:     "string input",
			input:    "This is a description",
			expected: "This is a description",
		},
		{
			name:     "map with value key",
			input:    map[string]any{"value": "Description from map"},
			expected: "Description from map",
		},
		{
			name:     "map without value key",
			input:    map[string]any{"other": "Not a description"},
			expected: "",
		},
		{
			name:     "map with non-string value",
			input:    map[string]any{"value": 123},
			expected: "",
		},
		{
			name:     "nil input",
			input:    nil,
			expected: "",
		},
		{
			name:     "other type input",
			input:    123,
			expected: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := getDescription(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestGetSubjects(t *testing.T) {
	testCases := []struct {
		name     string
		input    []any
		expected []string
	}{
		{
			name:     "empty input",
			input:    []any{},
			expected: []string{},
		},
		{
			name:     "string subjects",
			input:    []any{"Fiction", "Fantasy", "Adventure"},
			expected: []string{"Fiction", "Fantasy", "Adventure"},
		},
		{
			name: "map subjects with name key",
			input: []any{
				map[string]any{"name": "Fiction"},
				map[string]any{"name": "Fantasy"},
			},
			expected: []string{"Fiction", "Fantasy"},
		},
		{
			name: "mixed subjects",
			input: []any{
				"Fiction",
				map[string]any{"name": "Fantasy"},
				map[string]any{"other": "Not included"},
			},
			expected: []string{"Fiction", "Fantasy"},
		},
		{
			name: "map with non-string name",
			input: []any{
				map[string]any{"name": 123},
			},
			expected: []string{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := getSubjects(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}
