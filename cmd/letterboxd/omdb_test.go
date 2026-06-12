package letterboxd

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lepinkainen/hermes/internal/enrichment/omdb"
	internalerrors "github.com/lepinkainen/hermes/internal/errors"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func withOMDBServer(t *testing.T, handler http.HandlerFunc) {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	t.Cleanup(omdb.SetTestClient(server.URL, server.Client().Do))

	viper.Reset()
	viper.Set("omdb.api_key", "test")
	t.Cleanup(viper.Reset)
}

func TestFetchMovieDataSuccess(t *testing.T) {
	withOMDBServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "Heat", r.URL.Query().Get("t"))
		assert.Equal(t, "1995", r.URL.Query().Get("y"))
		_, _ = w.Write([]byte(`{
			"Title":"Heat",
			"Year":"1995",
			"imdbID":"tt0113277",
			"Runtime":"170 min",
			"Genre":"Crime, Drama",
			"Director":"Michael Mann",
			"Actors":"Al Pacino, Robert De Niro",
			"imdbRating":"8.2",
			"Poster":"poster.jpg",
			"Plot":"Bank heist."
		}`))
	})

	movie, err := fetchMovieData("Heat", 1995)
	require.NoError(t, err)
	require.NotNil(t, movie)
	assert.Equal(t, "Heat", movie.Name)
	assert.Equal(t, 1995, movie.Year)
	assert.Equal(t, 170, movie.Runtime)
	assert.Equal(t, []string{"Crime", "Drama"}, movie.Genres)
	assert.Equal(t, []string{"Al Pacino", "Robert De Niro"}, movie.Cast)
	assert.InDelta(t, 8.2, movie.Rating, 0.0001)
	assert.Equal(t, "Michael Mann", movie.Director)
	assert.Equal(t, "tt0113277", movie.ImdbID)
}

func TestFetchMovieDataRateLimit(t *testing.T) {
	withOMDBServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"Response":"False","Error":"Request limit reached!"}`))
	})

	_, err := fetchMovieData("Heat", 1995)
	require.Error(t, err)
	assert.True(t, internalerrors.IsRateLimitError(err))
}

func TestFetchMovieDataEmptyResponse(t *testing.T) {
	withOMDBServer(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{}`))
	})

	_, err := fetchMovieData("Nope", 1900)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid or empty response")
}

func TestFetchMovieDataNotFound(t *testing.T) {
	withOMDBServer(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"Response":"False","Error":"Movie not found!"}`))
	})

	_, err := fetchMovieData("Nope", 1900)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found in OMDB")
}

func TestFetchMovieDataMissingAPIKey(t *testing.T) {
	viper.Reset()
	defer viper.Reset()
	_, err := fetchMovieData("Heat", 1995)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "OMDB API key not found")
}

func TestFetchMovieDataFallbackToIMDBKey(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "fallback", r.URL.Query().Get("apikey"))
		_, _ = w.Write([]byte(`{"Title":"Heat","imdbID":"tt0113277"}`))
	}))
	defer server.Close()
	t.Cleanup(omdb.SetTestClient(server.URL, server.Client().Do))

	viper.Reset()
	defer viper.Reset()
	viper.Set("imdb.omdb_api_key", "fallback")

	movie, err := fetchMovieData("Heat", 1995)
	require.NoError(t, err)
	assert.Equal(t, "Heat", movie.Name)
}
