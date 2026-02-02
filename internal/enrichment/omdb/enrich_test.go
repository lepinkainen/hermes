package omdb

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseRatings(t *testing.T) {
	tests := []struct {
		name        string
		ratings     []Rating
		imdbRating  string
		expected    *RatingsEnrichment
		description string
	}{
		{
			name: "all ratings present",
			ratings: []Rating{
				{Source: "Internet Movie Database", Value: "8.8/10"},
				{Source: "Rotten Tomatoes", Value: "94%"},
				{Source: "Metacritic", Value: "85/100"},
			},
			imdbRating: "8.8",
			expected: &RatingsEnrichment{
				IMDbRating:     8.8,
				RottenTomatoes: "94%",
				RTTomatometer:  94,
				Metacritic:     85,
			},
			description: "Should parse all available ratings",
		},
		{
			name: "only imdb rating",
			ratings: []Rating{
				{Source: "Internet Movie Database", Value: "7.5/10"},
			},
			imdbRating: "7.5",
			expected: &RatingsEnrichment{
				IMDbRating:     7.5,
				RottenTomatoes: "",
				RTTomatometer:  0,
				Metacritic:     0,
			},
			description: "Should handle only IMDb rating",
		},
		{
			name:       "no ratings array but imdbRating field",
			ratings:    []Rating{},
			imdbRating: "9.2",
			expected: &RatingsEnrichment{
				IMDbRating:     9.2,
				RottenTomatoes: "",
				RTTomatometer:  0,
				Metacritic:     0,
			},
			description: "Should use imdbRating field when Ratings array is empty",
		},
		{
			name: "rt and metacritic only",
			ratings: []Rating{
				{Source: "Rotten Tomatoes", Value: "78%"},
				{Source: "Metacritic", Value: "62/100"},
			},
			imdbRating: "",
			expected: &RatingsEnrichment{
				IMDbRating:     0,
				RottenTomatoes: "78%",
				RTTomatometer:  78,
				Metacritic:     62,
			},
			description: "Should handle missing IMDb rating",
		},
		{
			name:       "empty ratings",
			ratings:    []Rating{},
			imdbRating: "N/A",
			expected: &RatingsEnrichment{
				IMDbRating:     0,
				RottenTomatoes: "",
				RTTomatometer:  0,
				Metacritic:     0,
			},
			description: "Should handle empty ratings gracefully",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseRatings(tt.ratings, tt.imdbRating)
			assert.Equal(t, tt.expected, result, tt.description)
		})
	}
}

func TestParseIMDbRating(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		expected float64
		hasError bool
	}{
		{"valid rating with slash", "8.8/10", 8.8, false},
		{"valid rating without slash", "7.5", 7.5, false},
		{"empty value", "", 0, true},
		{"N/A value", "N/A", 0, true},
		{"low rating", "3.2/10", 3.2, false},
		{"perfect rating", "10.0/10", 10.0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseIMDbRating(tt.value)
			if tt.hasError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestParseRottenTomatoes(t *testing.T) {
	tests := []struct {
		name              string
		value             string
		expectedPercent   string
		expectedTomatoMtr int
		hasError          bool
	}{
		{"valid percentage", "94%", "94%", 94, false},
		{"low percentage", "23%", "23%", 23, false},
		{"perfect score", "100%", "100%", 100, false},
		{"empty value", "", "", 0, true},
		{"N/A value", "N/A", "", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resultPercent, resultTomato, err := parseRottenTomatoes(tt.value)
			if tt.hasError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedPercent, resultPercent)
				assert.Equal(t, tt.expectedTomatoMtr, resultTomato)
			}
		})
	}
}

func TestParseMetacritic(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		expected int
		hasError bool
	}{
		{"valid score", "85/100", 85, false},
		{"low score", "42/100", 42, false},
		{"perfect score", "100/100", 100, false},
		{"empty value", "", 0, true},
		{"N/A value", "N/A", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseMetacritic(tt.value)
			if tt.hasError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}
