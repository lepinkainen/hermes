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
type: movie
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
