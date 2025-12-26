package tmdb

import (
	"testing"

	"github.com/lepinkainen/hermes/internal/obsidian"
)

func TestNormalizeGenreName(t *testing.T) {
	tests := map[string]string{
		"Sci-Fi & Fantasy":   "Sci-Fi-and-Fantasy",
		"Action & Adventure": "Action-and-Adventure",
		"Comedy/Drama":       "Comedy/Drama", // obsidian.NormalizeTag preserves / for hierarchical tags
		"Horror#Thriller":    "HorrorThriller",
		"  Documentary  ":    "Documentary",
		"Science Fiction":    "Science-Fiction",
		"War & Politics":     "War-and-Politics",
	}
	for input, want := range tests {
		if got := obsidian.NormalizeTag(input); got != want {
			t.Fatalf("obsidian.NormalizeTag(%q) = %q, want %q", input, got, want)
		}
	}
}
