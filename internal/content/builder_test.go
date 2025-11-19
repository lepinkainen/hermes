package content

import (
	"strings"
	"testing"
)

func TestWrapWithMarkers(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{
			name:    "basic content",
			content: "Some TMDB content",
			want:    "<!-- TMDB_DATA_START -->\nSome TMDB content\n<!-- TMDB_DATA_END -->",
		},
		{
			name:    "multiline content",
			content: "Line 1\nLine 2\nLine 3",
			want:    "<!-- TMDB_DATA_START -->\nLine 1\nLine 2\nLine 3\n<!-- TMDB_DATA_END -->",
		},
		{
			name:    "content with leading/trailing whitespace",
			content: "  \n  Content  \n  ",
			want:    "<!-- TMDB_DATA_START -->\nContent\n<!-- TMDB_DATA_END -->",
		},
		{
			name:    "empty content",
			content: "",
			want:    "",
		},
		{
			name:    "content with markdown",
			content: "## Overview\n\nThis is a **great** movie.",
			want:    "<!-- TMDB_DATA_START -->\n## Overview\n\nThis is a **great** movie.\n<!-- TMDB_DATA_END -->",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := WrapWithMarkers(tt.content)
			if got != tt.want {
				t.Errorf("WrapWithMarkers() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestHasTMDBContentMarkers(t *testing.T) {
	tests := []struct {
		name string
		body string
		want bool
	}{
		{
			name: "has both markers",
			body: "Some content\n<!-- TMDB_DATA_START -->\nTMDB content\n<!-- TMDB_DATA_END -->\nMore content",
			want: true,
		},
		{
			name: "only start marker",
			body: "Some content\n<!-- TMDB_DATA_START -->\nTMDB content",
			want: false,
		},
		{
			name: "only end marker",
			body: "Some content\nTMDB content\n<!-- TMDB_DATA_END -->",
			want: false,
		},
		{
			name: "no markers",
			body: "Just regular content",
			want: false,
		},
		{
			name: "empty body",
			body: "",
			want: false,
		},
		{
			name: "markers in wrong order",
			body: "<!-- TMDB_DATA_END -->\nContent\n<!-- TMDB_DATA_START -->",
			want: true, // Function just checks for presence, not order
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := HasTMDBContentMarkers(tt.body)
			if got != tt.want {
				t.Errorf("HasTMDBContentMarkers() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetTMDBContent(t *testing.T) {
	tests := []struct {
		name        string
		body        string
		wantContent string
		wantFound   bool
	}{
		{
			name:        "basic content between markers",
			body:        "Before\n<!-- TMDB_DATA_START -->\nTMDB content here\n<!-- TMDB_DATA_END -->\nAfter",
			wantContent: "TMDB content here",
			wantFound:   true,
		},
		{
			name:        "multiline content between markers",
			body:        "<!-- TMDB_DATA_START -->\n## Overview\n\nThis is a movie.\n\n## Cast\n\n- Actor 1\n<!-- TMDB_DATA_END -->",
			wantContent: "## Overview\n\nThis is a movie.\n\n## Cast\n\n- Actor 1",
			wantFound:   true,
		},
		{
			name:        "no markers",
			body:        "Just regular content",
			wantContent: "",
			wantFound:   false,
		},
		{
			name:        "only start marker",
			body:        "<!-- TMDB_DATA_START -->\nSome content",
			wantContent: "",
			wantFound:   false,
		},
		{
			name:        "empty content between markers",
			body:        "<!-- TMDB_DATA_START --><!-- TMDB_DATA_END -->",
			wantContent: "",
			wantFound:   true,
		},
		{
			name:        "whitespace between markers",
			body:        "<!-- TMDB_DATA_START -->\n   \n   \n<!-- TMDB_DATA_END -->",
			wantContent: "",
			wantFound:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotContent, gotFound := GetTMDBContent(tt.body)
			if gotContent != tt.wantContent {
				t.Errorf("GetTMDBContent() content = %q, want %q", gotContent, tt.wantContent)
			}
			if gotFound != tt.wantFound {
				t.Errorf("GetTMDBContent() found = %v, want %v", gotFound, tt.wantFound)
			}
		})
	}
}

func TestReplaceTMDBContent(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		newContent string
		want       string
	}{
		{
			name:       "replace existing content",
			body:       "Before\n<!-- TMDB_DATA_START -->\nOld content\n<!-- TMDB_DATA_END -->\nAfter",
			newContent: "New content",
			want:       "Before\n\n<!-- TMDB_DATA_START -->\nNew content\n<!-- TMDB_DATA_END -->\nAfter",
		},
		{
			name:       "replace with multiline content",
			body:       "<!-- TMDB_DATA_START -->\nOld\n<!-- TMDB_DATA_END -->",
			newContent: "Line 1\nLine 2\nLine 3",
			want:       "<!-- TMDB_DATA_START -->\nLine 1\nLine 2\nLine 3\n<!-- TMDB_DATA_END -->",
		},
		{
			name:       "no markers - return unchanged",
			body:       "Just regular content",
			newContent: "New content",
			want:       "Just regular content",
		},
		{
			name:       "replace with empty content",
			body:       "Before\n<!-- TMDB_DATA_START -->\nOld content\n<!-- TMDB_DATA_END -->\nAfter",
			newContent: "",
			want:       "Before\n\n<!-- TMDB_DATA_START -->\n\n<!-- TMDB_DATA_END -->\nAfter",
		},
		{
			name:       "content before markers",
			body:       "Original intro\n\n<!-- TMDB_DATA_START -->\nOld\n<!-- TMDB_DATA_END -->",
			newContent: "New TMDB data",
			want:       "Original intro\n\n<!-- TMDB_DATA_START -->\nNew TMDB data\n<!-- TMDB_DATA_END -->",
		},
		{
			name:       "content after markers",
			body:       "<!-- TMDB_DATA_START -->\nOld\n<!-- TMDB_DATA_END -->\n\nOriginal outro",
			newContent: "New TMDB data",
			want:       "<!-- TMDB_DATA_START -->\nNew TMDB data\n<!-- TMDB_DATA_END -->\nOriginal outro",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ReplaceTMDBContent(tt.body, tt.newContent)
			if got != tt.want {
				t.Errorf("ReplaceTMDBContent() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestBuildCoverImageEmbed(t *testing.T) {
	tests := []struct {
		name          string
		coverFilename string
		want          string
	}{
		{
			name:          "basic filename",
			coverFilename: "movie-poster.jpg",
			want:          "![[movie-poster.jpg|250]]",
		},
		{
			name:          "filename with path",
			coverFilename: "_attachments/cover.jpg",
			want:          "![[_attachments/cover.jpg|250]]",
		},
		{
			name:          "empty filename",
			coverFilename: "",
			want:          "",
		},
		{
			name:          "filename with spaces",
			coverFilename: "My Movie Poster.png",
			want:          "![[My Movie Poster.png|250]]",
		},
		{
			name:          "filename with special characters",
			coverFilename: "poster (2023) [4K].webp",
			want:          "![[poster (2023) [4K].webp|250]]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildCoverImageEmbed(tt.coverFilename)
			if got != tt.want {
				t.Errorf("BuildCoverImageEmbed() = %q, want %q", got, tt.want)
			}
		})
	}
}

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

	content := BuildTMDBContent(details, "movie", nil)

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

	content := BuildTMDBContent(details, "tv", nil)

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

func TestHelperExtractors(t *testing.T) {
	t.Run("intVal parses formats", func(t *testing.T) {
		if got, ok := intVal(map[string]any{"value": "42"}, "value"); !ok || got != 42 {
			t.Fatalf("intVal string parse = %d,%v want 42,true", got, ok)
		}
		if got, ok := intVal(map[string]any{"value": float64(10)}, "value"); !ok || got != 10 {
			t.Fatalf("intVal float parse = %d,%v want 10,true", got, ok)
		}
	})

	t.Run("floatVal handles ints and floats", func(t *testing.T) {
		if got, ok := floatVal(map[string]any{"value": 3}, "value"); !ok || got != 3 {
			t.Fatalf("floatVal int parse = %v,%v want 3,true", got, ok)
		}
		if got, ok := floatVal(map[string]any{"value": float32(1.5)}, "value"); !ok || got != 1.5 {
			t.Fatalf("floatVal float32 parse = %v,%v want 1.5,true", got, ok)
		}
	})

	t.Run("boolVal covers strings and numbers", func(t *testing.T) {
		if !boolVal(map[string]any{"value": "true"}, "value") {
			t.Fatalf("boolVal should treat \"true\" as true")
		}
		if boolVal(map[string]any{"value": 0}, "value") {
			t.Fatalf("boolVal should treat 0 as false")
		}
	})

	t.Run("string helpers", func(t *testing.T) {
		if got := stringVal(map[string]any{"value": 123}, "value"); got != "" {
			t.Fatalf("stringVal non-string = %q, want empty", got)
		}

		if got := nestedString(map[string]any{"outer": map[string]any{"inner": "yes"}}, "outer", "inner"); got != "yes" {
			t.Fatalf("nestedString = %q, want yes", got)
		}

		if got := firstStringFromArray(map[string]any{"arr": []any{
			map[string]any{"name": ""},
			map[string]any{"name": "first"},
			map[string]any{"name": "second"},
		}}, "arr", "name"); got != "first" {
			t.Fatalf("firstStringFromArray = %q, want first", got)
		}

		slice := stringSlice(map[string]any{"arr": []any{"a", 2, "b"}}, "arr")
		if len(slice) != 2 || slice[0] != "a" || slice[1] != "b" {
			t.Fatalf("stringSlice mixed content = %v, want [a b]", slice)
		}
	})

	t.Run("usContentRating picks US entry", func(t *testing.T) {
		data := map[string]any{
			"content_ratings": map[string]any{
				"results": []any{
					map[string]any{"iso_3166_1": "GB", "rating": "15"},
					map[string]any{"iso_3166_1": "US", "rating": "TV-MA"},
				},
			},
		}
		if got := usContentRating(data); got != "TV-MA" {
			t.Fatalf("usContentRating = %q, want TV-MA", got)
		}
	})

	t.Run("format helpers", func(t *testing.T) {
		if got := countryFlag("fi"); got != "🇫🇮" {
			t.Fatalf("countryFlag lower case = %q, want 🇫🇮", got)
		}
		if got := countryFlag("xx"); got != "🌐" {
			t.Fatalf("countryFlag unknown = %q, want globe", got)
		}
		if got := formatNumber(1234567); got != "1,234,567" {
			t.Fatalf("formatNumber = %q, want 1,234,567", got)
		}
	})
}

func assertContains(t *testing.T, body, substr string) {
	t.Helper()
	if !strings.Contains(body, substr) {
		t.Fatalf("expected %q to contain %q", body, substr)
	}
}
