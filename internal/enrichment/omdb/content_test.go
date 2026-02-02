package omdb

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildRatingsTable(t *testing.T) {
	tests := []struct {
		name     string
		ratings  *RatingsEnrichment
		expected []string // Expected strings that should be in the output
		notIn    []string // Strings that should NOT be in the output
	}{
		{
			name: "all ratings present",
			ratings: &RatingsEnrichment{
				IMDbRating:     8.8,
				RottenTomatoes: "94%",
				RTTomatometer:  94,
				Metacritic:     85,
			},
			expected: []string{
				"## Ratings",
				"| Source | Score |",
				"| IMDb | ⭐ 8.8/10 |",
				"| Rotten Tomatoes | 🍅 94% |",
				"| Metacritic | 📊 85/100 |",
			},
			notIn: []string{},
		},
		{
			name: "only imdb rating",
			ratings: &RatingsEnrichment{
				IMDbRating:     7.5,
				RottenTomatoes: "",
				RTTomatometer:  0,
				Metacritic:     0,
			},
			expected: []string{
				"## Ratings",
				"| IMDb | ⭐ 7.5/10 |",
			},
			notIn: []string{
				"Rotten Tomatoes",
				"Metacritic",
			},
		},
		{
			name: "only rt and metacritic",
			ratings: &RatingsEnrichment{
				IMDbRating:     0,
				RottenTomatoes: "78%",
				RTTomatometer:  78,
				Metacritic:     62,
			},
			expected: []string{
				"## Ratings",
				"| Rotten Tomatoes | 🍅 78% |",
				"| Metacritic | 📊 62/100 |",
			},
			notIn: []string{
				"IMDb",
			},
		},
		{
			name:     "nil ratings",
			ratings:  nil,
			expected: []string{},
			notIn:    []string{"## Ratings"},
		},
		{
			name: "empty ratings",
			ratings: &RatingsEnrichment{
				IMDbRating:     0,
				RottenTomatoes: "",
				RTTomatometer:  0,
				Metacritic:     0,
			},
			expected: []string{},
			notIn:    []string{"## Ratings"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := BuildRatingsTable(tt.ratings)

			// Check expected strings are present
			for _, exp := range tt.expected {
				assert.Contains(t, result, exp, "Expected string not found: %s", exp)
			}

			// Check unwanted strings are not present
			for _, notExp := range tt.notIn {
				assert.NotContains(t, result, notExp, "Unexpected string found: %s", notExp)
			}

			// If no ratings, result should be empty
			if tt.ratings == nil || (tt.ratings.IMDbRating == 0 && tt.ratings.RottenTomatoes == "" && tt.ratings.Metacritic == 0) {
				assert.Empty(t, result, "Expected empty result for no ratings")
			}
		})
	}
}

func TestBuildRatingsTable_Format(t *testing.T) {
	ratings := &RatingsEnrichment{
		IMDbRating:     9.3,
		RottenTomatoes: "91%",
		RTTomatometer:  91,
		Metacritic:     81,
	}

	result := BuildRatingsTable(ratings)

	// Check that it's valid markdown table
	lines := strings.Split(result, "\n")
	assert.Greater(t, len(lines), 3, "Should have header, separator, and data rows")

	// Check header formatting
	assert.Contains(t, lines[0], "## Ratings", "Should have Ratings header")
	assert.Contains(t, lines[2], "| Source | Score |", "Should have table headers")
	assert.Contains(t, lines[3], "|--------|-------|", "Should have table separator")

	// Check that emojis are present
	assert.Contains(t, result, "⭐", "Should contain star emoji for IMDb")
	assert.Contains(t, result, "🍅", "Should contain tomato emoji for RT")
	assert.Contains(t, result, "📊", "Should contain chart emoji for Metacritic")
}
