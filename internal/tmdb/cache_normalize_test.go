package tmdb

import (
	"testing"
)

func TestNormalizeQuery(t *testing.T) {
	tests := map[string]string{
		"The Matrix":           "the_matrix",
		"The Matrix 1999":      "the_matrix_1999",
		"  Spaces  Around  ":   "spaces__around",
		"Special!@#$%^&*Chars": "special________chars",
		"UPPERCASE":            "uppercase",
		"already_normalized":   "already_normalized",
		"dash-separated":       "dash-separated",
		"numbers123":           "numbers123",
		"":                     "",
	}

	for input, want := range tests {
		got := normalizeQuery(input)
		if got != want {
			t.Errorf("normalizeQuery(%q) = %q, want %q", input, got, want)
		}
	}
}
