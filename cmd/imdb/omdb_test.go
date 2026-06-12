package imdb

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lepinkainen/hermes/internal/enrichment/omdb"
	internalerrors "github.com/lepinkainen/hermes/internal/errors"
	"github.com/spf13/viper"
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
		_, _ = w.Write([]byte(`{
			"Title":"Heat",
			"imdbID":"tt0113277",
			"Type":"movie",
			"imdbRating":"8.2",
			"Runtime":"170 min",
			"Genre":"Crime, Drama",
			"Director":"Michael Mann",
			"Plot":"Bank heist.",
			"Poster":"poster.jpg",
			"Rated":"R",
			"Awards":"Oscar"
		}`))
	})

	movie, err := fetchMovieData("tt0113277")
	require.NoError(t, err)
	require.Equal(t, "Heat", movie.Title)
	require.Equal(t, 170, movie.RuntimeMins)
	require.Equal(t, "Michael Mann", movie.Directors[0])
}

func TestFetchMovieDataRateLimit(t *testing.T) {
	withOMDBServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"Response":"False","Error":"Request limit reached!"}`))
	})

	_, err := fetchMovieData("tt0113277")
	require.Error(t, err)
	require.True(t, internalerrors.IsRateLimitError(err), "expected rate limit error")
}

func TestFetchMovieDataNotFound(t *testing.T) {
	withOMDBServer(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"Response":"False","Error":"Incorrect IMDb ID. Movie not found!"}`))
	})

	_, err := fetchMovieData("tt0000000")
	require.Error(t, err)
	require.Contains(t, err.Error(), "not found in OMDB")
}
