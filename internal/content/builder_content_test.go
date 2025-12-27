package content

import (
	"strings"
	"testing"
)

func TestBuildTMDBContentMovie(t *testing.T) {
	details := map[string]any{
		"overview":     "A classic crime saga.",
		"tagline":      "A Los Angeles crime saga.",
		"status":       "Released",
		"runtime":      170,
		"release_date": "1995-12-15",
		"vote_average": 8.2,
		"vote_count":   12345,
		"budget":       60000000,
		"revenue":      187436818,
		"origin_country": []any{
			"US", "GB",
		},
		"external_ids": map[string]any{
			"imdb_id": "tt0113277",
			"tvdb_id": "123",
		},
		"homepage": "https://www.netflix.com/title/123",
	}

	content := BuildTMDBContent(details, "movie", nil, "")

	assertContains(t, content, "## Overview")
	assertContains(t, content, "A classic crime saga.")
	assertContains(t, content, "_\"A Los Angeles crime saga.\"_")

	assertContains(t, content, "## Movie Info")
	assertContains(t, content, "**Runtime** | 170 min")
	assertContains(t, content, "**Released** | 1995-12-15")
	assertContains(t, content, "⭐ 8.2/10 (12,345 votes)")
	assertContains(t, content, "**Budget** | $60,000,000")
	assertContains(t, content, "**Revenue** | $187,436,818")
	assertContains(t, content, "🇺🇸 US")
	assertContains(t, content, "🇬🇧 GB")
	assertContains(t, content, "IMDB")
	assertContains(t, content, "[Netflix]")
}

func TestBuildTMDBContentTVWithSeasons(t *testing.T) {
	details := map[string]any{
		"overview":           "A sprawling saga.",
		"in_production":      true,
		"status":             "Returning Series",
		"tagline":            "Never ends.",
		"number_of_seasons":  2,
		"number_of_episodes": 16,
		"first_air_date":     "2020-01-01",
		"last_air_date":      "2024-07-01",
		"vote_average":       7.4,
		"vote_count":         420,
		"origin_country":     []string{"US"},
		"content_ratings": map[string]any{
			"results": []any{
				map[string]any{"iso_3166_1": "US", "rating": "TV-MA"},
			},
		},
		"networks": []any{
			map[string]any{"name": "HBO"},
		},
		"seasons": []any{
			map[string]any{
				"name":          "Season 1",
				"air_date":      "2020-01-01",
				"vote_average":  7.1,
				"overview":      "The beginning.",
				"episode_count": 8,
				"poster_path":   "/season1.jpg",
			},
			map[string]any{
				"season_number": 2,
				"air_date":      "2024-01-01",
				"vote_average":  8.7,
				"overview":      "A strong follow-up.",
				"episode_count": 8,
			},
		},
	}

	content := BuildTMDBContent(details, "tv", nil, "")

	assertContains(t, content, "## Overview")
	assertContains(t, content, "## Series Info")
	assertContains(t, content, "**Seasons** | 2 (16 episodes)")
	assertContains(t, content, "2020-01-01 → 2024-07-01")
	assertContains(t, content, "⭐ 7.4/10 (420 votes)")
	assertContains(t, content, "**Network** | HBO")
	assertContains(t, content, "**Content Rating** | TV-MA")
	assertContains(t, content, "## Seasons")
	assertContains(t, content, "Season 1")
	assertContains(t, content, "⭐ 7.1/10")
	assertContains(t, content, "![Season 1](https://image.tmdb.org/t/p/w300/season1.jpg)")
	assertContains(t, content, "_The beginning._")
	assertContains(t, content, "Episodes:** 8 • **Status:** ✅ Complete")
	assertContains(t, content, "Season 2 (2024)")
	assertContains(t, content, "A strong follow-up.")
	assertContains(t, content, "Currently Airing")
}

func TestBuildTMDBContentMovie_WithCredits(t *testing.T) {
	details := map[string]any{
		"overview":     "A mind-bending thriller.",
		"status":       "Released",
		"runtime":      148,
		"release_date": "2010-07-16",
		"vote_average": 8.8,
		"vote_count":   25000,
		"credits": map[string]any{
			"crew": []any{
				map[string]any{"name": "Christopher Nolan", "id": 525, "job": "Director", "department": "Directing"},
				map[string]any{"name": "Christopher Nolan", "id": 525, "job": "Screenplay", "department": "Writing"},
				map[string]any{"name": "Emma Thomas", "id": 1233, "job": "Producer", "department": "Production"},
			},
			"cast": []any{
				map[string]any{"name": "Leonardo DiCaprio", "id": 6193, "character": "Dom Cobb", "order": 0},
				map[string]any{"name": "Joseph Gordon-Levitt", "id": 24045, "character": "Arthur", "order": 1},
				map[string]any{"name": "Ellen Page", "id": 27578, "character": "Ariadne", "order": 2},
			},
		},
	}

	content := BuildTMDBContent(details, "movie", nil, "")

	assertContains(t, content, "## Movie Info")

	// Verify Director with TMDB link
	assertContains(t, content, "**Director** | [Christopher Nolan](https://www.themoviedb.org/person/525)")

	// Verify Writer with TMDB link and job title
	assertContains(t, content, "**Writer** | [Christopher Nolan](https://www.themoviedb.org/person/525) (Screenplay)")

	// Verify Cast with TMDB links and character names
	assertContains(t, content, "**Cast** |")
	assertContains(t, content, "[Leonardo DiCaprio](https://www.themoviedb.org/person/6193) as Dom Cobb")
	assertContains(t, content, "[Joseph Gordon-Levitt](https://www.themoviedb.org/person/24045) as Arthur")
	assertContains(t, content, "[Ellen Page](https://www.themoviedb.org/person/27578) as Ariadne")

	// Verify it's in table row format
	assertContains(t, content, "| **Director** |")
	assertContains(t, content, "| **Writer** |")
	assertContains(t, content, "| **Cast** |")
}

func assertContains(t *testing.T, body, substr string) {
	t.Helper()
	if !strings.Contains(body, substr) {
		t.Fatalf("expected %q to contain %q", body, substr)
	}
}
