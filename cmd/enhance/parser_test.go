package enhance

import (
	"strings"
	"testing"

	"github.com/lepinkainen/hermes/internal/content"
	"github.com/lepinkainen/hermes/internal/enrichment"
	"github.com/lepinkainen/hermes/internal/importer/mediaids"
	"github.com/lepinkainen/hermes/internal/obsidian"
	"github.com/stretchr/testify/require"
)

func TestParseNote(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    *Note
		wantErr bool
	}{
		{
			name: "basic movie note",
			content: `---
title: "Test Movie"
tmdb_type: movie
year: 2021
imdb_id: "tt1234567"
---

This is the content of the note.`,
			want: &Note{
				Title:  "Test Movie",
				Type:   "movie",
				Year:   2021,
				IMDBID: "tt1234567",
			},
			wantErr: false,
		},
		{
			name: "note with tmdb_id",
			content: `---
title: "Test Movie"
tmdb_type: movie
year: 2021
tmdb_id: 12345
---

Content here.`,
			want: &Note{
				Title:  "Test Movie",
				Type:   "movie",
				Year:   2021,
				TMDBID: 12345,
			},
			wantErr: false,
		},
		{
			name:    "missing frontmatter",
			content: "Just some content without frontmatter",
			want: &Note{
				Title: "",
				Type:  "",
				Year:  0,
			},
			wantErr: false, // obsidian.ParseMarkdown accepts content without frontmatter
		},
		{
			name: "invalid yaml",
			content: `---
this is not: valid: yaml
---

Content`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseNote(tt.content)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseNote() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}

			if got.Title != tt.want.Title {
				t.Errorf("Title = %v, want %v", got.Title, tt.want.Title)
			}
			if got.Type != tt.want.Type {
				t.Errorf("Type = %v, want %v", got.Type, tt.want.Type)
			}
			if got.Year != tt.want.Year {
				t.Errorf("Year = %v, want %v", got.Year, tt.want.Year)
			}
			if got.IMDBID != tt.want.IMDBID {
				t.Errorf("IMDBID = %v, want %v", got.IMDBID, tt.want.IMDBID)
			}
			if got.TMDBID != tt.want.TMDBID {
				t.Errorf("TMDBID = %v, want %v", got.TMDBID, tt.want.TMDBID)
			}
			// Body is checked via parsed content, not comparing directly
		})
	}
}

func TestHasTMDBData(t *testing.T) {
	tests := []struct {
		name string
		note *Note
		want bool
	}{
		{
			name: "has tmdb_id and content markers",
			note: &Note{
				TMDBID: 12345,
				Body:   "Some content\n\n<!-- TMDB_DATA_START -->\nTMDB content here\n<!-- TMDB_DATA_END -->",
			},
			want: true,
		},
		{
			name: "has tmdb_id but no content markers",
			note: &Note{
				TMDBID: 12345,
				Body:   "Some content without markers",
			},
			want: false,
		},
		{
			name: "no tmdb_id",
			note: &Note{
				TMDBID: 0,
				Body:   "Some content",
			},
			want: false,
		},
		{
			name: "has content markers but no tmdb_id",
			note: &Note{
				TMDBID: 0,
				Body:   "Some content\n\n<!-- TMDB_DATA_START -->\nTMDB content here\n<!-- TMDB_DATA_END -->",
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.note.HasTMDBData(); got != tt.want {
				t.Errorf("HasTMDBData() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExtractTitleFromPath_WithParentheses(t *testing.T) {
	title := extractTitleFromPath("/notes/Red Sonja (2025).md")
	require.Equal(t, "Red Sonja (2025)", title)
}

func TestAddTMDBData(t *testing.T) {
	note := &Note{
		Frontmatter: obsidian.NewFrontmatter(),
	}
	note.Frontmatter.Set("title", "Test Movie")
	note.Frontmatter.Set("type", "movie")
	note.Frontmatter.Set("year", 2021)

	tmdbData := &enrichment.TMDBEnrichment{
		TMDBID:      12345,
		TMDBType:    "movie",
		RuntimeMins: 120,
		GenreTags:   []string{"Action", "Adventure"},
		CoverPath:   "_attachments/cover.jpg",
	}

	note.AddTMDBData(tmdbData)

	if note.Frontmatter.GetInt("tmdb_id") != 12345 {
		t.Errorf("tmdb_id not set correctly")
	}
	if note.Frontmatter.GetString("tmdb_type") != "movie" {
		t.Errorf("tmdb_type not set correctly")
	}
	if note.Frontmatter.GetInt("runtime") != 120 {
		t.Errorf("runtime not set correctly")
	}
	if note.Frontmatter.GetString("cover") != "_attachments/cover.jpg" {
		t.Errorf("cover not set correctly")
	}

	tags := note.Frontmatter.GetStringArray("tags")
	if len(tags) != 2 {
		t.Errorf("expected 2 tags, got %d", len(tags))
	}
}

func TestBuildMarkdown(t *testing.T) {
	note := &Note{
		Frontmatter: obsidian.NewFrontmatter(),
		Body:        "Original content here.",
	}
	note.Frontmatter.Set("title", "Test Movie")
	note.Frontmatter.Set("type", "movie")
	note.Frontmatter.Set("year", 2021)

	originalContent := `---
title: Test Movie
tmdb_type: movie
year: 2021
---

Original content here.`

	tmdbData := &enrichment.TMDBEnrichment{
		ContentMarkdown: "## TMDB Content\nSome TMDB data here.",
	}

	result := note.BuildMarkdown(originalContent, tmdbData, false)

	// Check that frontmatter is present
	if result[:3] != "---" {
		t.Errorf("Markdown should start with frontmatter delimiter")
	}

	// Check that original body is preserved
	if !containsSubstring(result, "Original content here.") {
		t.Errorf("Original content should be preserved")
	}

	// Check that TMDB content is appended
	if !containsSubstring(result, "## TMDB Content") {
		t.Errorf("TMDB content should be appended")
	}
}

func TestBuildMarkdownRegenerateDataBehavior(t *testing.T) {
	t.Run("overwriteTrueUpdatesOnlyTMDBFields", func(t *testing.T) {
		originalContent := `---
title: "Existing Show"
tmdb_id: 100
tmdb_type: "tv"
runtime: 30
total_episodes: 10
cover: "_attachments/old-cover.jpg"
tags:
  - watchlist
  - genre/animated
created: "2024-01-01"
modified: "2024-02-01"
service: "plex"
status: "watching"
episodes: 5
finished: false
custom_field: "keep-me"
---

Personal intro before TMDB block.

<!-- TMDB_DATA_START -->
## Overview
Outdated TMDB overview.
<!-- TMDB_DATA_END -->

Personal wrap up after TMDB block.
`
		note, err := parseNote(originalContent)
		require.NoError(t, err)

		// TMDB says show has ended, should overwrite the previous finished: false
		finished := true
		tmdbData := &enrichment.TMDBEnrichment{
			TMDBID:          2024,
			TMDBType:        "tv",
			RuntimeMins:     55,
			TotalEpisodes:   12,
			GenreTags:       []string{"Action", "Drama"},
			CoverPath:       "_attachments/new-cover.jpg",
			ContentMarkdown: "## Fresh Overview\nNew TMDB summary.",
			Finished:        &finished,
		}

		note.AddTMDBData(tmdbData)
		result := note.BuildMarkdown(originalContent, tmdbData, true)

		updated, err := parseNote(result)
		require.NoError(t, err)

		require.Equal(t, tmdbData.TMDBID, updated.Frontmatter.GetInt("tmdb_id"))
		require.Equal(t, tmdbData.TMDBType, updated.Frontmatter.GetString("tmdb_type"))
		require.Equal(t, tmdbData.RuntimeMins, updated.Frontmatter.GetInt("runtime"))
		require.Equal(t, tmdbData.TotalEpisodes, updated.Frontmatter.GetInt("total_episodes"))
		require.Equal(t, tmdbData.CoverPath, updated.Frontmatter.GetString("cover"))

		tags := updated.Frontmatter.GetStringArray("tags")
		require.ElementsMatch(t, []string{"Action", "Drama", "genre/animated", "watchlist"}, tags)

		require.Equal(t, "2024-01-01", updated.Frontmatter.GetString("created"))
		require.Equal(t, "2024-02-01", updated.Frontmatter.GetString("modified"))
		require.Equal(t, "plex", updated.Frontmatter.GetString("service"))
		require.Equal(t, "watching", updated.Frontmatter.GetString("status"))
		require.Equal(t, 5, updated.Frontmatter.GetInt("episodes"))
		// finished should be overwritten from false to true based on TMDB data
		require.Equal(t, true, updated.Frontmatter.GetBool("finished"))
		require.Equal(t, "keep-me", updated.Frontmatter.GetString("custom_field"))

		body := updated.Body
		require.Contains(t, body, "Personal intro before TMDB block.")
		require.Contains(t, body, "Personal wrap up after TMDB block.")
		require.Contains(t, body, tmdbData.ContentMarkdown)
		require.NotContains(t, body, "Outdated TMDB overview.")
		require.Equal(t, 1, strings.Count(body, content.TMDBDataStart))
		require.Equal(t, 1, strings.Count(body, content.TMDBDataEnd))
	})

	t.Run("appendWhenMarkersMissingAndOverwriteFalse", func(t *testing.T) {
		originalContent := `---
title: "Markerless Movie"
created: "2023-05-05"
service: "letterboxd"
---

Just me writing notes.
`
		note, err := parseNote(originalContent)
		require.NoError(t, err)

		tmdbData := &enrichment.TMDBEnrichment{
			TMDBID:          303,
			TMDBType:        "movie",
			RuntimeMins:     120,
			GenreTags:       []string{"Thriller"},
			CoverPath:       "_attachments/markerless-cover.jpg",
			ContentMarkdown: "## Appended Overview\nReplacement content should not remove notes.",
		}

		note.AddTMDBData(tmdbData)
		result := note.BuildMarkdown(originalContent, tmdbData, false)

		updated, err := parseNote(result)
		require.NoError(t, err)

		body := updated.Body
		require.Contains(t, body, "Just me writing notes.")
		require.Contains(t, body, content.TMDBDataStart)
		require.Contains(t, body, content.TMDBDataEnd)
		require.Contains(t, body, tmdbData.ContentMarkdown)

		originalIdx := strings.Index(body, "Just me writing notes.")
		tmdbIdx := strings.Index(body, tmdbData.ContentMarkdown)
		require.NotEqual(t, -1, originalIdx)
		require.NotEqual(t, -1, tmdbIdx)
		require.Less(t, originalIdx, tmdbIdx, "TMDB block should be appended after original content")
	})
}

// Helper function for substring checking
func containsSubstring(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestNeedsCover(t *testing.T) {
	tests := []struct {
		name string
		note *Note
		want bool
	}{
		{
			name: "no cover field",
			note: func() *Note {
				fm := obsidian.NewFrontmatter()
				fm.Set("title", "Test Movie")
				return &Note{Frontmatter: fm}
			}(),
			want: true,
		},
		{
			name: "empty cover field",
			note: func() *Note {
				fm := obsidian.NewFrontmatter()
				fm.Set("title", "Test Movie")
				fm.Set("cover", "")
				return &Note{Frontmatter: fm}
			}(),
			want: true,
		},
		{
			name: "cover field with value",
			note: func() *Note {
				fm := obsidian.NewFrontmatter()
				fm.Set("title", "Test Movie")
				fm.Set("cover", "_attachments/cover.jpg")
				return &Note{Frontmatter: fm}
			}(),
			want: false,
		},
		{
			name: "cover field is not a string",
			note: func() *Note {
				fm := obsidian.NewFrontmatter()
				fm.Set("title", "Test Movie")
				fm.Set("cover", 12345)
				return &Note{Frontmatter: fm}
			}(),
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.note.NeedsCover(); got != tt.want {
				t.Errorf("NeedsCover() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNeedsContent(t *testing.T) {
	tests := []struct {
		name string
		note *Note
		want bool
	}{
		{
			name: "no content markers",
			note: &Note{
				Body: "Some content without markers",
			},
			want: true,
		},
		{
			name: "has content markers",
			note: &Note{
				Body: "Some content\n\n<!-- TMDB_DATA_START -->\nTMDB content here\n<!-- TMDB_DATA_END -->",
			},
			want: false,
		},
		{
			name: "empty body",
			note: &Note{
				Body: "",
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.note.NeedsContent(); got != tt.want {
				t.Errorf("NeedsContent() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetMediaIDs(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    mediaids.MediaIDs
	}{
		{
			name: "TMDB ID only",
			content: `---
title: "Test Movie"
tmdb_type: movie
year: 2021
tmdb_id: 949
---

Content here.`,
			want: mediaids.MediaIDs{TMDBID: 949},
		},
		{
			name: "IMDB ID only",
			content: `---
title: "Test Movie"
tmdb_type: movie
year: 2021
imdb_id: "tt0113277"
---

Content here.`,
			want: mediaids.MediaIDs{IMDBID: "tt0113277"},
		},
		{
			name: "Letterboxd ID only",
			content: `---
title: "Test Movie"
tmdb_type: movie
year: 2021
letterboxd_id: "2bg8"
---

Content here.`,
			want: mediaids.MediaIDs{LetterboxdID: "2bg8"},
		},
		{
			name: "All IDs present",
			content: `---
title: "Test Movie"
tmdb_type: movie
year: 2021
tmdb_id: 949
imdb_id: "tt0113277"
letterboxd_id: "2bg8"
---

Content here.`,
			want: mediaids.MediaIDs{
				TMDBID:       949,
				IMDBID:       "tt0113277",
				LetterboxdID: "2bg8",
			},
		},
		{
			name: "No IDs",
			content: `---
title: "Test Movie"
tmdb_type: movie
year: 2021
---

Content here.`,
			want: mediaids.MediaIDs{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			note, err := parseNote(tt.content)
			if err != nil {
				t.Fatalf("Failed to parse note: %v", err)
			}

			ids := note.GetMediaIDs()
			if ids.TMDBID != tt.want.TMDBID {
				t.Errorf("TMDBID = %v, want %v", ids.TMDBID, tt.want.TMDBID)
			}
			if ids.IMDBID != tt.want.IMDBID {
				t.Errorf("IMDBID = %v, want %v", ids.IMDBID, tt.want.IMDBID)
			}
			if ids.LetterboxdID != tt.want.LetterboxdID {
				t.Errorf("LetterboxdID = %v, want %v", ids.LetterboxdID, tt.want.LetterboxdID)
			}
		})
	}
}

func TestHasAnyID(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    bool
	}{
		{
			name: "TMDB ID only",
			content: `---
title: "Test Movie"
tmdb_type: movie
year: 2021
tmdb_id: 949
---

Content here.`,
			want: true,
		},
		{
			name: "IMDB ID only",
			content: `---
title: "Test Movie"
tmdb_type: movie
year: 2021
imdb_id: "tt0113277"
---

Content here.`,
			want: true,
		},
		{
			name: "Letterboxd ID only",
			content: `---
title: "Test Movie"
tmdb_type: movie
year: 2021
letterboxd_id: "2bg8"
---

Content here.`,
			want: true,
		},
		{
			name: "All IDs present",
			content: `---
title: "Test Movie"
tmdb_type: movie
year: 2021
tmdb_id: 949
imdb_id: "tt0113277"
letterboxd_id: "2bg8"
---

Content here.`,
			want: true,
		},
		{
			name: "No IDs",
			content: `---
title: "Test Movie"
tmdb_type: movie
year: 2021
---

Content here.`,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			note, err := parseNote(tt.content)
			if err != nil {
				t.Fatalf("Failed to parse note: %v", err)
			}

			hasAny := note.HasAnyID()
			if hasAny != tt.want {
				t.Errorf("HasAnyID() = %v, want %v", hasAny, tt.want)
			}
		})
	}
}

func TestGetIDSummary(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{
			name: "All IDs present",
			content: `---
title: "Test Movie"
tmdb_type: movie
year: 2021
tmdb_id: 949
imdb_id: "tt0113277"
letterboxd_id: "2bg8"
---

Content here.`,
			want: "tmdb:949, imdb:tt0113277, letterboxd:2bg8",
		},
		{
			name: "TMDB and IMDB only",
			content: `---
title: "Test Movie"
tmdb_type: movie
year: 2021
tmdb_id: 949
imdb_id: "tt0113277"
---

Content here.`,
			want: "tmdb:949, imdb:tt0113277",
		},
		{
			name: "IMDB only",
			content: `---
title: "Test Movie"
tmdb_type: movie
year: 2021
imdb_id: "tt0113277"
---

Content here.`,
			want: "imdb:tt0113277",
		},
		{
			name: "No IDs",
			content: `---
title: "Test Movie"
tmdb_type: movie
year: 2021
---

Content here.`,
			want: "no IDs",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			note, err := parseNote(tt.content)
			if err != nil {
				t.Fatalf("Failed to parse note: %v", err)
			}

			summary := note.GetIDSummary()
			if summary != tt.want {
				t.Errorf("GetIDSummary() = %v, want %v", summary, tt.want)
			}
		})
	}
}

func TestParseNoteWithHeatFile(t *testing.T) {
	// Test with actual Heat (1995) file content
	content := `---
title: "Heat"
year: 1995
date_watched: "2019-04-12"
letterboxd_rating: 8.3
runtime: 170
duration: 2h 50m
directors:
  - "Michael Mann"
tags:
  - letterboxd/movie
  - rating/8
  - year/1990s
  - genre/Action
  - genre/Crime
  - genre/Drama
letterboxd_uri: "https://boxd.it/2bg8"
letterboxd_id: "2bg8"
cover: "attachments/Heat - cover.jpg"
tmdb_id: 949
tmdb_type: "movie"
---

![[Heat - cover.jpg|250]]

>[!summary]- Plot
> A group of high-end professional thieves start to feel the heat from the LAPD when they unknowingly leave a verbal clue at their latest heist.

>[!cast]- Cast
> - Al Pacino
> - Robert De Niro
> - Val Kilmer

>[!info]- Letterboxd
> [View on Letterboxd](https://boxd.it/2bg8)
`

	note, err := parseNote(content)
	if err != nil {
		t.Fatalf("Failed to parse Heat note: %v", err)
	}

	// Verify parsing
	if note.Title != "Heat" {
		t.Errorf("Title = %v, want Heat", note.Title)
	}
	if note.Type != "movie" {
		t.Errorf("Type = %v, want movie", note.Type)
	}
	if note.Year != 1995 {
		t.Errorf("Year = %v, want 1995", note.Year)
	}
	if note.TMDBID != 949 {
		t.Errorf("TMDBID = %v, want 949", note.TMDBID)
	}
	if note.LetterboxdID != "2bg8" {
		t.Errorf("LetterboxdID = %v, want 2bg8", note.LetterboxdID)
	}

	// Test ID extraction
	ids := note.GetMediaIDs()
	if ids.TMDBID != 949 {
		t.Errorf("GetMediaIDs().TMDBID = %v, want 949", ids.TMDBID)
	}
	if ids.LetterboxdID != "2bg8" {
		t.Errorf("GetMediaIDs().LetterboxdID = %v, want 2bg8", ids.LetterboxdID)
	}

	// Test utility functions
	if !note.HasAnyID() {
		t.Errorf("HasAnyID() = false, want true")
	}

	summary := note.GetIDSummary()
	expectedSummary := "tmdb:949, letterboxd:2bg8"
	if summary != expectedSummary {
		t.Errorf("GetIDSummary() = %v, want %v", summary, expectedSummary)
	}

	// Verify the original issue: HasTMDBData should return false because no TMDB content markers
	if note.HasTMDBData() {
		t.Errorf("HasTMDBData() = true, want false (Heat file has TMDB ID but no TMDB content markers)")
	}

	// Test why this file would trigger TMDB search: needs content but has TMDB ID
	if note.NeedsContent() {
		t.Logf("✓ Heat file needs TMDB content (no markers found)")
	} else {
		t.Errorf("Heat file should need TMDB content (no markers found)")
	}
}

func TestHasSeenField(t *testing.T) {
	tests := []struct {
		name     string
		note     *Note
		expected bool
	}{
		{
			name: "has seen field",
			note: func() *Note {
				fm := obsidian.NewFrontmatter()
				fm.Set("title", "Test Movie")
				fm.Set("seen", true)
				return &Note{Seen: true, Frontmatter: fm}
			}(),
			expected: true,
		},
		{
			name: "does not have seen field",
			note: func() *Note {
				fm := obsidian.NewFrontmatter()
				fm.Set("title", "Test Movie")
				return &Note{Seen: false, Frontmatter: fm}
			}(),
			expected: false,
		},
		{
			name: "has seen field set to false",
			note: func() *Note {
				fm := obsidian.NewFrontmatter()
				fm.Set("title", "Test Movie")
				fm.Set("seen", false)
				return &Note{Seen: false, Frontmatter: fm}
			}(),
			expected: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := tc.note.hasSeenField()
			if result != tc.expected {
				t.Errorf("hasSeenField() = %v, want %v", result, tc.expected)
			}
		})
	}
}

func TestHasAnyRating(t *testing.T) {
	tests := []struct {
		name     string
		note     *Note
		expected bool
	}{
		{
			name: "has imdb_rating",
			note: func() *Note {
				fm := obsidian.NewFrontmatter()
				fm.Set("title", "Test Movie")
				fm.Set("imdb_rating", 8.5)
				return &Note{Frontmatter: fm}
			}(),
			expected: true,
		},
		{
			name: "has my_rating",
			note: func() *Note {
				fm := obsidian.NewFrontmatter()
				fm.Set("title", "Test Movie")
				fm.Set("my_rating", 9)
				return &Note{Frontmatter: fm}
			}(),
			expected: true,
		},
		{
			name: "has letterboxd_rating",
			note: func() *Note {
				fm := obsidian.NewFrontmatter()
				fm.Set("title", "Test Movie")
				fm.Set("letterboxd_rating", 4.5)
				return &Note{Frontmatter: fm}
			}(),
			expected: true,
		},
		{
			name: "has zero ratings",
			note: func() *Note {
				fm := obsidian.NewFrontmatter()
				fm.Set("title", "Test Movie")
				fm.Set("imdb_rating", 0.0)
				fm.Set("my_rating", 0)
				fm.Set("letterboxd_rating", 0.0)
				return &Note{Frontmatter: fm}
			}(),
			expected: false,
		},
		{
			name: "has no ratings",
			note: func() *Note {
				fm := obsidian.NewFrontmatter()
				fm.Set("title", "Test Movie")
				fm.Set("year", 2020)
				return &Note{Frontmatter: fm}
			}(),
			expected: false,
		},
		{
			name: "has mixed rating types (int and float)",
			note: func() *Note {
				fm := obsidian.NewFrontmatter()
				fm.Set("title", "Test Movie")
				fm.Set("my_rating", 8)     // int
				fm.Set("imdb_rating", 7.5) // float64
				return &Note{Frontmatter: fm}
			}(),
			expected: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := tc.note.hasAnyRating()
			if result != tc.expected {
				t.Errorf("hasAnyRating() = %v, want %v", result, tc.expected)
			}
		})
	}
}

func TestAddTMDBDataFinishedField(t *testing.T) {
	t.Run("setsFinishedTrueForEndedTVShow", func(t *testing.T) {
		fm := obsidian.NewFrontmatter()
		fm.Set("title", "Ended Show")
		fm.Set("tmdb_type", "tv")
		note := &Note{
			Title:       "Ended Show",
			Type:        "tv",
			Frontmatter: fm,
		}

		finished := true
		tmdbData := &enrichment.TMDBEnrichment{
			TMDBID:        12345,
			TMDBType:      "tv",
			RuntimeMins:   45,
			TotalEpisodes: 100,
			Finished:      &finished,
		}

		note.AddTMDBData(tmdbData)

		require.Equal(t, true, note.Frontmatter.GetBool("finished"))
	})

	t.Run("setsFinishedFalseForOngoingTVShow", func(t *testing.T) {
		fm := obsidian.NewFrontmatter()
		fm.Set("title", "Ongoing Show")
		fm.Set("tmdb_type", "tv")
		note := &Note{
			Title:       "Ongoing Show",
			Type:        "tv",
			Frontmatter: fm,
		}

		finished := false
		tmdbData := &enrichment.TMDBEnrichment{
			TMDBID:        12346,
			TMDBType:      "tv",
			RuntimeMins:   45,
			TotalEpisodes: 50,
			Finished:      &finished,
		}

		note.AddTMDBData(tmdbData)

		require.Equal(t, false, note.Frontmatter.GetBool("finished"))
	})

	t.Run("doesNotSetFinishedForMovie", func(t *testing.T) {
		fm := obsidian.NewFrontmatter()
		fm.Set("title", "Test Movie")
		fm.Set("tmdb_type", "movie")
		note := &Note{
			Title:       "Test Movie",
			Type:        "movie",
			Frontmatter: fm,
		}

		tmdbData := &enrichment.TMDBEnrichment{
			TMDBID:      12347,
			TMDBType:    "movie",
			RuntimeMins: 120,
			Finished:    nil, // Movies don't have this field
		}

		note.AddTMDBData(tmdbData)

		_, exists := note.Frontmatter.Get("finished")
		require.False(t, exists, "finished field should not be set for movies")
	})

	t.Run("doesNotSetFinishedWhenStatusNotAvailable", func(t *testing.T) {
		fm := obsidian.NewFrontmatter()
		fm.Set("title", "Show Without Status")
		fm.Set("tmdb_type", "tv")
		note := &Note{
			Title:       "Show Without Status",
			Type:        "tv",
			Frontmatter: fm,
		}

		tmdbData := &enrichment.TMDBEnrichment{
			TMDBID:      12348,
			TMDBType:    "tv",
			RuntimeMins: 30,
			Finished:    nil, // Status not available from TMDB
		}

		note.AddTMDBData(tmdbData)

		// Should not set finished field when TMDB status is not available
		_, exists := note.Frontmatter.Get("finished")
		require.False(t, exists, "finished field should not be set when TMDB status is unavailable")
	})
}
