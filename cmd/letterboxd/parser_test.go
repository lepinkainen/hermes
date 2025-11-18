package letterboxd

import (
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
