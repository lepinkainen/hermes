package tmdb

import "testing"

func TestSanitizeGenreName(t *testing.T) {
	tests := map[string]string{
		"Sci-Fi & Fantasy":   "Sci-Fi-and-Fantasy",
		"Action & Adventure": "Action-and-Adventure",
		"Comedy/Drama":       "Comedy-Drama",
		"Horror#Thriller":    "HorrorThriller",
		"  Documentary  ":    "Documentary",
		"Science Fiction":    "Science-Fiction",
		"War & Politics":     "War-and-Politics",
	}
	for input, want := range tests {
		if got := sanitizeGenreName(input); got != want {
			t.Fatalf("sanitizeGenreName(%q) = %q, want %q", input, got, want)
		}
	}
}
