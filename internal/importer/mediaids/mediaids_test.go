package mediaids

import (
	"testing"

	"github.com/lepinkainen/hermes/internal/testutil"
	"github.com/stretchr/testify/require"
)

func TestFromFrontmatter(t *testing.T) {
	got := FromFrontmatter(map[string]any{
		"tmdb_id":       949.0,
		"tmdb_type":     "movie",
		"imdb_id":       "tt0113277",
		"letterboxd_id": "2bg8",
	})

	require.Equal(t, 949, got.TMDBID)
	require.Equal(t, "movie", got.TMDBType)
	require.Equal(t, "tt0113277", got.IMDBID)
	require.Equal(t, "2bg8", got.LetterboxdID)

	zero := FromFrontmatter(nil)
	require.Zero(t, zero)
}

func TestFromFile(t *testing.T) {
	env := testutil.NewTestEnv(t)

	content := `---
title: "Heat"
tmdb_id: 949
tmdb_type: movie
imdb_id: tt0113277
letterboxd_id: 2bg8
---

Body
`
	env.WriteFileString("Heat.md", content)
	path := env.Path("Heat.md")

	got, err := FromFile(path)
	require.NoError(t, err)
	require.Equal(t, 949, got.TMDBID)
	require.Equal(t, "movie", got.TMDBType)
	require.Equal(t, "tt0113277", got.IMDBID)
	require.Equal(t, "2bg8", got.LetterboxdID)

	_, err = FromFile(env.Path("missing.md"))
	require.Error(t, err)
}

func TestHasAnyAndSummary(t *testing.T) {
	ids := MediaIDs{TMDBID: 949, IMDBID: "tt0113277"}
	require.True(t, ids.HasAny())
	require.Equal(t, "tmdb:949, imdb:tt0113277", ids.Summary())

	zero := MediaIDs{}
	require.False(t, zero.HasAny())
	require.Equal(t, "no IDs", zero.Summary())
}
