package letterboxd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoadExistingTMDBID(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "Heat (1995).md")

	content := `---
title: "Heat"
tmdb_id: 949
tmdb_type: movie
imdb_id: tt0113277
tags:
  - letterboxd/movie
  - genre/Action
  - genre/Crime
---

Movie body
`
	err := os.WriteFile(filePath, []byte(content), 0644)
	require.NoError(t, err)

	m := Movie{
		Name: "Heat",
		Year: 1995,
	}

	loadExistingTMDBID(&m, tempDir)

	require.NotNil(t, m.TMDBEnrichment, "tmdb enrichment should be initialized from frontmatter")
	require.Equal(t, 949, m.TMDBEnrichment.TMDBID)
	require.Equal(t, "movie", m.TMDBEnrichment.TMDBType)
	require.Equal(t, "tt0113277", m.ImdbID, "existing imdb id should be reused")
}

func TestWriteMoviesToJSON_UsesExistingTMDBIDWhenSkippingEnrich(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "Heat (1995).md")

	content := `---
title: "Heat"
tmdb_id: 949
tmdb_type: movie
imdb_id: tt0113277
---

Movie body
`
	err := os.WriteFile(filePath, []byte(content), 0644)
	require.NoError(t, err)

	prevOutputDir := outputDir
	prevSkipEnrich := skipEnrich
	prevOverwrite := overwrite
	outputDir = tempDir
	skipEnrich = true
	overwrite = true
	defer func() {
		outputDir = prevOutputDir
		skipEnrich = prevSkipEnrich
		overwrite = prevOverwrite
	}()

	movies := []Movie{
		{Name: "Heat", Year: 1995},
	}

	jsonPath := filepath.Join(tempDir, "movies.json")

	err = writeMoviesToJSON(movies, jsonPath)
	require.NoError(t, err)

	data, err := os.ReadFile(jsonPath)
	require.NoError(t, err)

	var saved []Movie
	err = json.Unmarshal(data, &saved)
	require.NoError(t, err)
	require.Len(t, saved, 1)
	require.NotNil(t, saved[0].TMDBEnrichment, "tmdb enrichment should be propagated from existing note")
	require.Equal(t, 949, saved[0].TMDBEnrichment.TMDBID)
	require.Equal(t, "movie", saved[0].TMDBEnrichment.TMDBType)
	require.Equal(t, "tt0113277", saved[0].ImdbID)
}
