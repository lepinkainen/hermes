package omdb

import (
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/lepinkainen/hermes/internal/cache"
	"github.com/lepinkainen/hermes/internal/testutil"
	"github.com/spf13/viper"
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

// setupOMDBCache initializes a test cache environment for OMDB tests
func setupOMDBCache(t *testing.T) *cache.CacheDB {
	t.Helper()

	// Reset any existing global cache to ensure isolation between tests
	if err := cache.ResetGlobalCache(); err != nil {
		t.Fatalf("Failed to reset global cache: %v", err)
	}

	viper.Reset()
	t.Cleanup(func() {
		_ = cache.ResetGlobalCache()
		viper.Reset()
	})

	env := testutil.NewTestEnv(t)
	tmpDir := env.RootDir()

	viper.Set("cache.dbfile", filepath.Join(tmpDir, "omdb-cache.db"))
	viper.Set("cache.ttl", "24h")

	cacheDB, err := cache.GetGlobalCache()
	if err != nil {
		t.Fatalf("Failed to init cache: %v", err)
	}
	if err := cacheDB.ClearAll("omdb_cache"); err != nil {
		t.Fatalf("Failed to reset omdb_cache table: %v", err)
	}

	return cacheDB
}

func TestCheckCacheStatus(t *testing.T) {
	tests := []struct {
		name           string
		imdbID         string
		cacheData      any // can be *OMDBResponse (old format) or *CachedOMDBResponse (new format)
		expectedStatus CacheStatus
	}{
		{
			name:           "no cache entry",
			imdbID:         "tt0000001",
			cacheData:      nil,
			expectedStatus: CacheStatusNotCached,
		},
		{
			name:   "new format - cached with ratings",
			imdbID: "tt0000002",
			cacheData: &CachedOMDBResponse{
				Response: &OMDBResponse{
					Title:      "Test Movie",
					ImdbRating: "7.5",
					Ratings: []Rating{
						{Source: "Internet Movie Database", Value: "7.5/10"},
					},
				},
				NotFound: false,
			},
			expectedStatus: CacheStatusHasRatings,
		},
		{
			name:   "new format - cached as not found",
			imdbID: "tt0000003",
			cacheData: &CachedOMDBResponse{
				Response: nil,
				NotFound: true,
			},
			expectedStatus: CacheStatusNotFound,
		},
		{
			name:   "new format - cached with no ratings",
			imdbID: "tt0000004",
			cacheData: &CachedOMDBResponse{
				Response: &OMDBResponse{
					Title:      "New Release",
					ImdbRating: "N/A",
					Ratings:    []Rating{},
				},
				NotFound: false,
			},
			expectedStatus: CacheStatusNoRatings,
		},
		{
			name:   "old format - cached with ratings",
			imdbID: "tt0000005",
			cacheData: &OMDBResponse{
				Title:      "Old Format Movie",
				ImdbRating: "8.0",
				Ratings:    nil,
			},
			expectedStatus: CacheStatusHasRatings,
		},
		{
			name:   "old format - cached with N/A rating",
			imdbID: "tt0000006",
			cacheData: &OMDBResponse{
				Title:      "Old Format New Release",
				ImdbRating: "N/A",
				Ratings:    []Rating{},
			},
			expectedStatus: CacheStatusNoRatings,
		},
		{
			name:   "new format - has RT rating only",
			imdbID: "tt0000007",
			cacheData: &CachedOMDBResponse{
				Response: &OMDBResponse{
					Title:      "Movie with RT only",
					ImdbRating: "",
					Ratings: []Rating{
						{Source: "Rotten Tomatoes", Value: "85%"},
					},
				},
				NotFound: false,
			},
			expectedStatus: CacheStatusHasRatings,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cacheDB := setupOMDBCache(t)

			// Pre-populate cache if test case provides data
			if tt.cacheData != nil {
				jsonData, err := json.Marshal(tt.cacheData)
				if err != nil {
					t.Fatalf("Failed to marshal cache data: %v", err)
				}
				// Store with no TTL wrapper for simplicity
				if err := cacheDB.Set("omdb_cache", tt.imdbID, string(jsonData), 0); err != nil {
					t.Fatalf("Failed to set cache: %v", err)
				}
			}

			status := CheckCacheStatus(tt.imdbID)
			assert.Equal(t, tt.expectedStatus, status, "cache status mismatch")
		})
	}
}

func TestHasCachedRatings(t *testing.T) {
	// Test that HasCachedRatings correctly delegates to CheckCacheStatus
	tests := []struct {
		name               string
		imdbID             string
		cacheData          any
		expectedHasCached  bool
		expectedHasRatings bool
	}{
		{
			name:               "no cache entry",
			imdbID:             "tt0000001",
			cacheData:          nil,
			expectedHasCached:  false,
			expectedHasRatings: false,
		},
		{
			name:   "cached with ratings",
			imdbID: "tt0000002",
			cacheData: &CachedOMDBResponse{
				Response: &OMDBResponse{
					Title:      "Test Movie",
					ImdbRating: "7.5",
				},
				NotFound: false,
			},
			expectedHasCached:  true,
			expectedHasRatings: true,
		},
		{
			name:   "cached as not found",
			imdbID: "tt0000003",
			cacheData: &CachedOMDBResponse{
				Response: nil,
				NotFound: true,
			},
			expectedHasCached:  true,
			expectedHasRatings: false,
		},
		{
			name:   "cached with no ratings",
			imdbID: "tt0000004",
			cacheData: &CachedOMDBResponse{
				Response: &OMDBResponse{
					Title:      "New Release",
					ImdbRating: "N/A",
					Ratings:    []Rating{},
				},
				NotFound: false,
			},
			expectedHasCached:  true,
			expectedHasRatings: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cacheDB := setupOMDBCache(t)

			// Pre-populate cache if test case provides data
			if tt.cacheData != nil {
				jsonData, err := json.Marshal(tt.cacheData)
				if err != nil {
					t.Fatalf("Failed to marshal cache data: %v", err)
				}
				if err := cacheDB.Set("omdb_cache", tt.imdbID, string(jsonData), 0); err != nil {
					t.Fatalf("Failed to set cache: %v", err)
				}
			}

			hasCached, hasRatings := HasCachedRatings(tt.imdbID)
			assert.Equal(t, tt.expectedHasCached, hasCached, "hasCached mismatch")
			assert.Equal(t, tt.expectedHasRatings, hasRatings, "hasRatings mismatch")
		})
	}
}
