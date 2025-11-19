package enhance

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/lepinkainen/hermes/internal/enrichment"
	"github.com/stretchr/testify/require"
)

func TestFindMarkdownFiles(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "Movie.md"), []byte("ok"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "Readme.txt"), []byte("ignore"), 0644))

	sub := filepath.Join(dir, "sub")
	require.NoError(t, os.Mkdir(sub, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(sub, "Show.md"), []byte("ok"), 0644))

	files, err := findMarkdownFiles(dir, false)
	require.NoError(t, err)
	require.Equal(t, []string{filepath.Join(dir, "Movie.md")}, files)

	files, err = findMarkdownFiles(dir, true)
	require.NoError(t, err)
	require.ElementsMatch(t, []string{
		filepath.Join(dir, "Movie.md"),
		filepath.Join(sub, "Show.md"),
	}, files)
}

func TestUpdateNoteWithTMDBData(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "Heat.md")
	content := `---
title: Heat
tmdb_type: movie
year: 1995
---

Body`
	require.NoError(t, os.WriteFile(path, []byte(content), 0644))

	note, err := parseNoteFile(path)
	require.NoError(t, err)

	tmdbData := &enrichment.TMDBEnrichment{
		TMDBID:          949,
		TMDBType:        "movie",
		RuntimeMins:     170,
		TotalEpisodes:   0,
		GenreTags:       []string{"Action", "Crime"},
		CoverPath:       "attachments/Heat - cover.jpg",
		ContentMarkdown: "## Overview\n\nDetailed plot.",
	}

	err = updateNoteWithTMDBData(path, note, tmdbData, true)
	require.NoError(t, err)

	updated, err := os.ReadFile(path)
	require.NoError(t, err)
	body := string(updated)
	require.Contains(t, body, "tmdb_id: 949")
	require.Contains(t, body, "runtime: 170")
	require.Contains(t, body, "tags:")
	require.Contains(t, body, "<!-- TMDB_DATA_START -->")
	require.Contains(t, body, "## Overview")
}

// TestUpdateNoteWithTMDBData_NoTypeField is a regression test to ensure notes
// without a type field can still be processed. The type is detected from TMDB
// search results, so filtering based on missing type would break anime and
// other content that doesn't have a pre-set type.
func TestUpdateNoteWithTMDBData_NoTypeField(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "Cowboy Bebop.md")
	// Note: intentionally NO type field - TMDB will detect it as TV
	content := `---
title: Cowboy Bebop
year: 1998
---

An anime series.`
	require.NoError(t, os.WriteFile(path, []byte(content), 0644))

	note, err := parseNoteFile(path)
	require.NoError(t, err)
	require.Empty(t, note.Type, "note should have no type field initially")

	// Simulate TMDB returning TV show data
	tmdbData := &enrichment.TMDBEnrichment{
		TMDBID:          30991,
		TMDBType:        "tv",
		TotalEpisodes:   26,
		GenreTags:       []string{"Animation", "Action", "Sci-Fi"},
		ContentMarkdown: "## Overview\n\nSpace bounty hunters.",
	}

	// This must succeed - notes without type should still be processable
	err = updateNoteWithTMDBData(path, note, tmdbData, true)
	require.NoError(t, err)

	updated, err := os.ReadFile(path)
	require.NoError(t, err)
	body := string(updated)
	require.Contains(t, body, "tmdb_id: 30991")
	require.Contains(t, body, "tmdb_type: tv")
	require.Contains(t, body, "total_episodes: 26")
}
