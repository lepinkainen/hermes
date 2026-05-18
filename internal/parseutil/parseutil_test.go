package parseutil

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNormalizeISBN(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"ISBN with hyphens", "978-0-316-76948-8", "9780316769488"},
		{"ISBN with spaces", "978 0 316 76948 8", "9780316769488"},
		{"ISBN with hyphens and spaces", "978-0-316 76948-8", "9780316769488"},
		{"ISBN already clean", "9780316769488", "9780316769488"},
		{"empty", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, NormalizeISBN(tt.input))
		})
	}
}

func TestParseFloat(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected float64
	}{
		{"valid float", "7.5", 7.5},
		{"integer as string", "8", 8.0},
		{"empty string", "", 0.0},
		{"non-numeric string", "N/A", 0.0},
		{"string with whitespace", " 9.2 ", 0.0},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, ParseFloat(tc.input))
		})
	}
}

func TestParseRuntime(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int
	}{
		{"standard format", "120 min", 120},
		{"single digit", "9 min", 9},
		{"empty string", "", 0},
		{"no min suffix", "135", 135},
		{"non-numeric prefix", "about 90 min", 0},
		{"different suffix", "120 minutes", 0},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, ParseRuntime(tc.input))
		})
	}
}

func TestParseYear(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int
	}{
		{"single year", "2010", 2010},
		{"range with en-dash", "2010–2014", 2010},
		{"range with hyphen", "2010-2014", 2010},
		{"empty", "", 0},
		{"garbage", "N/A", 0},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, ParseYear(tc.input))
		})
	}
}

func TestParseCommaList(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{"empty", "", nil},
		{"N/A sentinel", "N/A", nil},
		{"two items", "Crime, Drama", []string{"Crime", "Drama"}},
		{"comma without space", "Crime,Drama", []string{"Crime", "Drama"}},
		{"trailing empties", "Crime, Drama, ,", []string{"Crime", "Drama"}},
		{"single item", "Crime", []string{"Crime"}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, ParseCommaList(tc.input))
		})
	}
}
