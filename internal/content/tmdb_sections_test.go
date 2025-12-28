package content

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExtractLetterboxdDisplayText(t *testing.T) {
	tests := []struct {
		name     string
		uri      string
		expected string
	}{
		{
			name:     "short URL",
			uri:      "https://boxd.it/2bg8",
			expected: "boxd.it/2bg8",
		},
		{
			name:     "full film URL",
			uri:      "https://letterboxd.com/film/the-godfather/",
			expected: "film/the-godfather",
		},
		{
			name:     "full film URL without trailing slash",
			uri:      "https://letterboxd.com/film/the-dark-knight",
			expected: "film/the-dark-knight",
		},
		{
			name:     "search URL",
			uri:      "https://letterboxd.com/search/wildcat/",
			expected: "Search: wildcat",
		},
		{
			name:     "search URL without trailing slash",
			uri:      "https://letterboxd.com/search/inception",
			expected: "Search: inception",
		},
		{
			name:     "http protocol",
			uri:      "http://letterboxd.com/film/heat-1995/",
			expected: "film/heat-1995",
		},
		{
			name:     "short URL http",
			uri:      "http://boxd.it/abc",
			expected: "boxd.it/abc",
		},
		{
			name:     "search with spaces in URL",
			uri:      "https://letterboxd.com/search/the%20matrix/",
			expected: "Search: the%20matrix",
		},
		{
			name:     "unknown format returns as-is without protocol",
			uri:      "https://letterboxd.com/user/some-user/",
			expected: "letterboxd.com/user/some-user/",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractLetterboxdDisplayText(tt.uri)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestBuildTMDBContent_EmptySectionsDefaultsMovie(t *testing.T) {
	details := map[string]any{
		"overview": "A great movie",
	}

	result := BuildTMDBContent(details, "movie", []string{}, "")

	// Default sections for movies should be ["overview", "info"]
	assert.Contains(t, result, "## Overview")
	assert.Contains(t, result, "## Movie Info")
	assert.NotContains(t, result, "## Seasons")
}

func TestBuildTMDBContent_EmptySectionsDefaultsTV(t *testing.T) {
	details := map[string]any{
		"overview": "A great show",
		"seasons":  []any{},
	}

	result := BuildTMDBContent(details, "tv", []string{}, "")

	// Default sections for TV should be ["overview", "info", "seasons"]
	assert.Contains(t, result, "## Overview")
	assert.Contains(t, result, "## Series Info")
}

func TestBuildTMDBContent_OverviewSection(t *testing.T) {
	details := map[string]any{
		"overview": "A gripping tale of crime and justice.",
		"tagline":  "In a world of corruption, one man stands alone",
	}

	result := BuildTMDBContent(details, "movie", []string{"overview"}, "")

	assert.Contains(t, result, "## Overview")
	assert.Contains(t, result, "A gripping tale of crime and justice.")
	assert.Contains(t, result, "> _\"In a world of corruption, one man stands alone\"_")
}

func TestBuildTMDBContent_OverviewNoTagline(t *testing.T) {
	details := map[string]any{
		"overview": "Just an overview",
	}

	result := BuildTMDBContent(details, "movie", []string{"overview"}, "")

	assert.Contains(t, result, "## Overview")
	assert.Contains(t, result, "Just an overview")
	assert.NotContains(t, result, "> _\"")
}

func TestBuildTMDBContent_OverviewEmpty(t *testing.T) {
	details := map[string]any{
		"overview": "",
	}

	result := BuildTMDBContent(details, "movie", []string{"overview"}, "")

	// Empty overview should not produce any output
	assert.NotContains(t, result, "## Overview")
}

func TestBuildTMDBContent_MovieInfo(t *testing.T) {
	details := map[string]any{
		"status":       "Released",
		"runtime":      142,
		"release_date": "1995-12-15",
		"vote_average": 8.5,
		"vote_count":   10000,
		"budget":       63000000,
		"revenue":      170000000,
		"origin_country": []any{
			"US",
		},
		"homepage": "https://www.movie-site.com",
		"external_ids": map[string]any{
			"imdb_id": "tt0113277",
		},
	}

	result := BuildTMDBContent(details, "movie", []string{"info"}, "")

	assert.Contains(t, result, "## Movie Info")
	assert.Contains(t, result, "| **Status** | Released |")
	assert.Contains(t, result, "| **Runtime** | 142 min |")
	assert.Contains(t, result, "| **Released** | 1995-12-15 |")
	assert.Contains(t, result, "| **Rating** | ⭐ 8.5/10 (10,000 votes) |")
	assert.Contains(t, result, "| **Budget** | $63,000,000 |")
	assert.Contains(t, result, "| **Revenue** | $170,000,000 |")
	assert.Contains(t, result, "| **Origin** | 🇺🇸 US |")
	assert.Contains(t, result, "| **IMDB** | [imdb.com/title/tt0113277](https://www.imdb.com/title/tt0113277/) |")
	assert.Contains(t, result, "| **Homepage** | [Official Website](https://www.movie-site.com) |")
}

func TestBuildTMDBContent_MovieInfoWithLetterboxd(t *testing.T) {
	details := map[string]any{
		"status": "Released",
	}

	result := BuildTMDBContent(details, "movie", []string{"info"}, "https://letterboxd.com/film/heat-1995/")

	assert.Contains(t, result, "| **Letterboxd** | [film/heat-1995](https://letterboxd.com/film/heat-1995/) |")
}

func TestBuildTMDBContent_MovieInfoWithDirectorsWritersCast(t *testing.T) {
	details := map[string]any{
		"status": "Released",
		"credits": map[string]any{
			"crew": []any{
				map[string]any{
					"job":  "Director",
					"name": "Christopher Nolan",
					"id":   525,
				},
				map[string]any{
					"department": "Writing",
					"job":        "Screenplay",
					"name":       "Jonathan Nolan",
					"id":         7467,
				},
			},
			"cast": []any{
				map[string]any{
					"name":      "Christian Bale",
					"character": "Bruce Wayne",
					"id":        3894,
					"order":     0,
				},
				map[string]any{
					"name":      "Heath Ledger",
					"character": "Joker",
					"id":        1810,
					"order":     1,
				},
			},
		},
	}

	result := BuildTMDBContent(details, "movie", []string{"info"}, "")

	assert.Contains(t, result, "| **Director** | [Christopher Nolan](https://www.themoviedb.org/person/525) |")
	assert.Contains(t, result, "| **Writer** | [Jonathan Nolan](https://www.themoviedb.org/person/7467) (Screenplay) |")
	assert.Contains(t, result, "| **Cast** | [Christian Bale](https://www.themoviedb.org/person/3894) as Bruce Wayne, [Heath Ledger](https://www.themoviedb.org/person/1810) as Joker |")
}

func TestBuildTMDBContent_TVInfo(t *testing.T) {
	details := map[string]any{
		"status":             "Returning Series",
		"in_production":      true,
		"number_of_seasons":  5,
		"number_of_episodes": 50,
		"first_air_date":     "2018-01-01",
		// Note: When in_production is true, we use "Present" instead of last_air_date
		"vote_average":   8.2,
		"vote_count":     5000,
		"origin_country": []any{"GB", "US"},
		"networks":       []any{map[string]any{"name": "HBO"}},
		"content_ratings": map[string]any{
			"results": []any{
				map[string]any{
					"iso_3166_1": "US",
					"rating":     "TV-MA",
				},
			},
		},
		"external_ids": map[string]any{
			"tvdb_id": "12345",
		},
	}

	result := BuildTMDBContent(details, "tv", []string{"info"}, "")

	assert.Contains(t, result, "## Series Info")
	assert.Contains(t, result, "| **Status** | Returning Series (In Production) |")
	assert.Contains(t, result, "| **Seasons** | 5 (50 episodes) |")
	assert.Contains(t, result, "| **Aired** | 2018-01-01 → Present |")
	assert.Contains(t, result, "| **Rating** | ⭐ 8.2/10 (5,000 votes) |")
	assert.Contains(t, result, "| **Origin** | 🇬🇧 GB 🇺🇸 US |")
	assert.Contains(t, result, "| **Network** | HBO |")
	assert.Contains(t, result, "| **Content Rating** | TV-MA |")
	assert.Contains(t, result, "| **TVDB** | [thetvdb.com/12345](https://thetvdb.com/series/12345) |")
}

func TestBuildTMDBContent_TVInfoNotInProduction(t *testing.T) {
	details := map[string]any{
		"status":             "Ended",
		"in_production":      false,
		"first_air_date":     "2000-01-01",
		"last_air_date":      "2005-12-31",
		"number_of_seasons":  6,
		"number_of_episodes": 100,
	}

	result := BuildTMDBContent(details, "tv", []string{"info"}, "")

	assert.Contains(t, result, "| **Status** | Ended |")
	assert.Contains(t, result, "| **Aired** | 2000-01-01 → 2005-12-31 |")
}

func TestBuildTMDBContent_TVInfoSameAirDates(t *testing.T) {
	details := map[string]any{
		"status":             "Ended",
		"in_production":      false,
		"first_air_date":     "2020-05-15",
		"last_air_date":      "2020-05-15",
		"number_of_seasons":  1,
		"number_of_episodes": 1,
	}

	result := BuildTMDBContent(details, "tv", []string{"info"}, "")

	// Should not show arrow when dates are the same
	assert.Contains(t, result, "| **Aired** | 2020-05-15 |")
	assert.NotContains(t, result, "→")
}

func TestBuildTMDBContent_SeasonsSection(t *testing.T) {
	details := map[string]any{
		"in_production": false,
		"seasons": []any{
			map[string]any{
				"name":          "Season 1",
				"season_number": 1,
				"air_date":      "2019-04-14",
				"vote_average":  8.7,
				"overview":      "The beginning of the story",
				"episode_count": 6,
				"poster_path":   "/path/to/poster.jpg",
			},
			map[string]any{
				"name":          "Season 2",
				"season_number": 2,
				"air_date":      "2022-07-15",
				"vote_average":  8.5,
				"overview":      "The continuation",
				"episode_count": 8,
			},
		},
	}

	result := BuildTMDBContent(details, "tv", []string{"seasons"}, "")

	assert.Contains(t, result, "## Seasons")
	assert.Contains(t, result, "### Season 1 (2019) • ⭐ 8.7/10")
	assert.Contains(t, result, "![Season 1](https://image.tmdb.org/t/p/w300/path/to/poster.jpg)")
	assert.Contains(t, result, "_The beginning of the story_")
	assert.Contains(t, result, "**Episodes:** 6 • **Status:** ✅ Complete")
	assert.Contains(t, result, "### Season 2 (2022) • ⭐ 8.5/10")
	assert.Contains(t, result, "_The continuation_")
	assert.Contains(t, result, "**Episodes:** 8 • **Status:** ✅ Complete")
}

func TestBuildTMDBContent_SeasonsInProduction(t *testing.T) {
	details := map[string]any{
		"in_production": true,
		"seasons": []any{
			map[string]any{
				"name":          "Season 1",
				"season_number": 1,
				"air_date":      "2023-01-01",
				"episode_count": 10,
			},
			map[string]any{
				"name":          "Season 2",
				"season_number": 2,
				"air_date":      "2024-01-01",
				"episode_count": 8,
			},
		},
	}

	result := BuildTMDBContent(details, "tv", []string{"seasons"}, "")

	// First season should be complete
	assert.Contains(t, result, "### Season 1 (2023)")
	assert.Contains(t, result, "**Episodes:** 10 • **Status:** ✅ Complete")

	// Last season should show as currently airing
	assert.Contains(t, result, "### Season 2 (2024)")
	assert.Contains(t, result, "**Episodes:** 8 • **Status:** Currently Airing")
}

func TestBuildTMDBContent_SeasonsEmptyArray(t *testing.T) {
	details := map[string]any{
		"seasons": []any{},
	}

	result := BuildTMDBContent(details, "tv", []string{"seasons"}, "")

	// Empty seasons array should not produce output
	assert.NotContains(t, result, "## Seasons")
}

func TestBuildTMDBContent_SeasonsNoAirDate(t *testing.T) {
	details := map[string]any{
		"in_production": false,
		"seasons": []any{
			map[string]any{
				"name":          "Season 1",
				"season_number": 1,
				"air_date":      "",
				"episode_count": 10,
			},
		},
	}

	result := BuildTMDBContent(details, "tv", []string{"seasons"}, "")

	assert.Contains(t, result, "### Season 1 (TBA)")
}

func TestBuildTMDBContent_MultipleSections(t *testing.T) {
	details := map[string]any{
		"overview":           "A thrilling series",
		"tagline":            "Every episode counts",
		"status":             "Ended",
		"in_production":      false,
		"number_of_seasons":  3,
		"number_of_episodes": 30,
		"seasons": []any{
			map[string]any{
				"name":          "Season 1",
				"season_number": 1,
				"air_date":      "2020-01-01",
				"episode_count": 10,
			},
		},
	}

	result := BuildTMDBContent(details, "tv", []string{"overview", "info", "seasons"}, "")

	// All three sections should be present and separated by double newlines
	assert.Contains(t, result, "## Overview")
	assert.Contains(t, result, "## Series Info")
	assert.Contains(t, result, "## Seasons")

	// Sections should be separated by \n\n
	parts := strings.Split(result, "\n\n")
	assert.True(t, len(parts) >= 3)
}

func TestBuildTMDBContent_MovieDoesNotIncludeSeasons(t *testing.T) {
	details := map[string]any{
		"overview": "A movie overview",
		"status":   "Released",
		"seasons": []any{
			map[string]any{"name": "Should not appear"},
		},
	}

	result := BuildTMDBContent(details, "movie", []string{"overview", "info", "seasons"}, "")

	// Movie should never show seasons section even if requested
	assert.Contains(t, result, "## Overview")
	assert.Contains(t, result, "## Movie Info")
	assert.NotContains(t, result, "## Seasons")
	assert.NotContains(t, result, "Should not appear")
}

func TestBuildTMDBContent_OriginCountryLimit(t *testing.T) {
	details := map[string]any{
		"status": "Released",
		"origin_country": []any{
			"US", "GB", "FR", "DE", "IT", "ES",
		},
	}

	result := BuildTMDBContent(details, "movie", []string{"info"}, "")

	// Should only show first 3 countries
	assert.Contains(t, result, "🇺🇸 US")
	assert.Contains(t, result, "🇬🇧 GB")
	assert.Contains(t, result, "🇫🇷 FR")
	assert.NotContains(t, result, "🇩🇪 DE")
}

func TestBuildTMDBContent_ZeroRuntimeNotShown(t *testing.T) {
	details := map[string]any{
		"status":  "Released",
		"runtime": 0,
	}

	result := BuildTMDBContent(details, "movie", []string{"info"}, "")

	assert.NotContains(t, result, "| **Runtime** |")
}

func TestBuildTMDBContent_ZeroBudgetNotShown(t *testing.T) {
	details := map[string]any{
		"status": "Released",
		"budget": 0,
	}

	result := BuildTMDBContent(details, "movie", []string{"info"}, "")

	assert.NotContains(t, result, "| **Budget** |")
}
