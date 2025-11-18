package enrich

import (
	"testing"

	"github.com/lepinkainen/hermes/internal/enrichment"
	hermeserrors "github.com/lepinkainen/hermes/internal/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type movie struct {
	tmdbID int
}

func TestEnrich_ContinuesOnOMDBRateLimitAndAppliesTMDB(t *testing.T) {
	m := movie{}
	rateLimited := false
	tmdbApplied := false

	_, err := Enrich(&m, Options[movie, *movie]{
		FetchOMDB: func() (*movie, error) {
			return nil, hermeserrors.NewRateLimitError("limit")
		},
		ApplyOMDB: func(*movie, *movie) {
			t.Fatal("OMDB apply should not be called on rate limit")
		},
		OnOMDBRateLimit: func(error) { rateLimited = true },
		TMDBEnabled:     true,
		FetchTMDB: func() (*enrichment.TMDBEnrichment, error) {
			return &enrichment.TMDBEnrichment{TMDBID: 42}, nil
		},
		ApplyTMDB: func(target *movie, tmdbData *enrichment.TMDBEnrichment) {
			tmdbApplied = true
			target.tmdbID = tmdbData.TMDBID
		},
	})

	require.NoError(t, err)
	assert.True(t, rateLimited, "rate limit handler should be invoked")
	assert.True(t, tmdbApplied, "TMDB data should be applied even after OMDB rate limit")
	assert.Equal(t, 42, m.tmdbID)
}

func TestEnrich_ReturnsCombinedErrorWhenBothFail(t *testing.T) {
	m := movie{}
	omdbLogged := false
	tmdbLogged := false

	_, err := Enrich(&m, Options[movie, *movie]{
		FetchOMDB: func() (*movie, error) {
			return nil, assert.AnError
		},
		OnOMDBError: func(e error) {
			omdbLogged = true
			assert.Equal(t, assert.AnError, e)
		},
		TMDBEnabled: true,
		FetchTMDB: func() (*enrichment.TMDBEnrichment, error) {
			return nil, assert.AnError
		},
		OnTMDBError: func(e error) {
			tmdbLogged = true
			assert.Equal(t, assert.AnError, e)
		},
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "omdb")
	assert.True(t, omdbLogged)
	assert.True(t, tmdbLogged)
}

func TestEnrich_PropagatesStopProcessing(t *testing.T) {
	m := movie{}
	stopErr := hermeserrors.NewStopProcessingError("stop")

	_, err := Enrich(&m, Options[movie, *movie]{
		TMDBEnabled: true,
		FetchTMDB: func() (*enrichment.TMDBEnrichment, error) {
			return nil, stopErr
		},
	})

	require.ErrorIs(t, err, stopErr)
}
