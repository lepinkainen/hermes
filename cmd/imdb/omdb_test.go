package imdb

import (
	"net/http"
	"net/http/httptest"
	"testing"

	internalerrors "github.com/lepinkainen/hermes/internal/errors"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseFloat(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected float64
	}{
		{
			name:     "valid float",
			input:    "7.5",
			expected: 7.5,
		},
		{
			name:     "integer as string",
			input:    "8",
			expected: 8.0,
		},
		{
			name:     "empty string",
			input:    "",
			expected: 0.0,
		},
		{
			name:     "non-numeric string",
			input:    "N/A",
			expected: 0.0,
		},
		{
			name:     "string with whitespace",
			input:    " 9.2 ",
			expected: 0.0, // parseFloat doesn't trim whitespace
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := parseFloat(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestParseRuntime(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected int
	}{
		{
			name:     "standard format",
			input:    "120 min",
			expected: 120,
		},
		{
			name:     "single digit",
			input:    "9 min",
			expected: 9,
		},
		{
			name:     "empty string",
			input:    "",
			expected: 0,
		},
		{
			name:     "no min suffix",
			input:    "135",
			expected: 135, // parseRuntime will just convert the string to an integer
		},
		{
			name:     "non-numeric prefix",
			input:    "about 90 min",
			expected: 0, // parseRuntime can't handle non-numeric prefixes
		},
		{
			name:     "different suffix",
			input:    "120 minutes",
			expected: 0, // After trimming " min", "120 utes" is not a valid integer
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := parseRuntime(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestGetOMDBAPIKey(t *testing.T) {
	testCases := []struct {
		name          string
		imdbKey       string
		globalKey     string
		wantKey       string
		wantErrSubstr string
	}{
		{
			name:      "prefers_imdb_scoped_key",
			imdbKey:   "imdb-secret",
			globalKey: "global-secret",
			wantKey:   "imdb-secret",
		},
		{
			name:      "falls_back_to_global_key",
			globalKey: "global-secret",
			wantKey:   "global-secret",
		},
		{
			name:          "errors_when_missing",
			wantErrSubstr: "omdb.api_key or imdb.omdb_api_key",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			viper.Reset()
			if tc.imdbKey != "" {
				viper.Set("imdb.omdb_api_key", tc.imdbKey)
			}
			if tc.globalKey != "" {
				viper.Set("omdb.api_key", tc.globalKey)
			}

			got, err := getOMDBAPIKey()
			if tc.wantErrSubstr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.wantErrSubstr)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tc.wantKey, got)
		})
	}
}

func TestFetchMovieDataSuccess(t *testing.T) {
	viper.Reset()
	viper.Set("omdb.api_key", "test")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
	}))
	defer server.Close()

	origBase := omdbBaseURL
	origGet := omdbHTTPGet
	defer func() {
		omdbBaseURL = origBase
		omdbHTTPGet = origGet
	}()
	omdbBaseURL = server.URL
	omdbHTTPGet = server.Client().Get

	movie, err := fetchMovieData("tt0113277")
	require.NoError(t, err)
	require.Equal(t, "Heat", movie.Title)
	require.Equal(t, 170, movie.RuntimeMins)
	require.Equal(t, "Michael Mann", movie.Directors[0])
}

func TestFetchMovieDataRateLimit(t *testing.T) {
	viper.Reset()
	viper.Set("omdb.api_key", "test")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"Response":"False","Error":"Request limit reached!"}`))
	}))
	defer server.Close()

	origBase := omdbBaseURL
	origGet := omdbHTTPGet
	defer func() {
		omdbBaseURL = origBase
		omdbHTTPGet = origGet
	}()
	omdbBaseURL = server.URL
	omdbHTTPGet = server.Client().Get

	_, err := fetchMovieData("tt0113277")
	require.Error(t, err)
	require.True(t, internalerrors.IsRateLimitError(err), "expected rate limit error")
}
