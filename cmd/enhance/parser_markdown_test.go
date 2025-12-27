package enhance

import (
	"strings"
	"testing"

	"github.com/lepinkainen/hermes/internal/content"
	"github.com/lepinkainen/hermes/internal/enrichment"
	"github.com/lepinkainen/hermes/internal/obsidian"
	"github.com/stretchr/testify/require"
)

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
