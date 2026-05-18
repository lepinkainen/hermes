package omdb

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	internalerrors "github.com/lepinkainen/hermes/internal/errors"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func withOMDBServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	prevBase := omdbBaseURL
	prevDo := omdbHTTPDo
	prevWait := omdbRateWait
	omdbBaseURL = server.URL
	omdbHTTPDo = server.Client().Do
	omdbRateWait = func(context.Context) error { return nil }
	t.Cleanup(func() {
		omdbBaseURL = prevBase
		omdbHTTPDo = prevDo
		omdbRateWait = prevWait
	})

	viper.Reset()
	viper.Set("omdb.api_key", "test")
	t.Cleanup(viper.Reset)
	return server
}

func TestGetAPIKey(t *testing.T) {
	tests := []struct {
		name    string
		set     map[string]string
		want    string
		wantErr bool
	}{
		{"omdb scope", map[string]string{"omdb.api_key": "a"}, "a", false},
		{"imdb fallback", map[string]string{"imdb.omdb_api_key": "b"}, "b", false},
		{"letterboxd fallback", map[string]string{"letterboxd.omdb_api_key": "c"}, "c", false},
		{"prefers omdb over imdb", map[string]string{"omdb.api_key": "a", "imdb.omdb_api_key": "b"}, "a", false},
		{"missing", map[string]string{}, "", true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			viper.Reset()
			defer viper.Reset()
			for k, v := range tc.set {
				viper.Set(k, v)
			}
			got, err := GetAPIKey()
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestFetchByIMDBIDSuccess(t *testing.T) {
	withOMDBServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "tt0113277", r.URL.Query().Get("i"))
		_, _ = w.Write([]byte(`{"Response":"True","imdbID":"tt0113277","Title":"Heat","Runtime":"170 min"}`))
	})

	resp, err := FetchByIMDBID(context.Background(), "tt0113277")
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, "Heat", resp.Title)
	assert.Equal(t, "tt0113277", resp.ImdbID)
}

func TestFetchByIMDBIDNotFound(t *testing.T) {
	withOMDBServer(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"Response":"False","Error":"Incorrect IMDb ID, not found!"}`))
	})

	resp, err := FetchByIMDBID(context.Background(), "tt0000000")
	require.NoError(t, err)
	assert.Nil(t, resp)
}

func TestFetchByIMDBIDRateLimit(t *testing.T) {
	withOMDBServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"Response":"False","Error":"Request limit reached!"}`))
	})

	_, err := FetchByIMDBID(context.Background(), "tt0113277")
	require.Error(t, err)
	assert.True(t, internalerrors.IsRateLimitError(err))
}

func TestFetchByTitleYearSuccess(t *testing.T) {
	withOMDBServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "Heat", r.URL.Query().Get("t"))
		assert.Equal(t, "1995", r.URL.Query().Get("y"))
		_, _ = w.Write([]byte(`{"Response":"True","Title":"Heat","imdbID":"tt0113277"}`))
	})

	resp, err := FetchByTitleYear(context.Background(), "Heat", 1995)
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, "Heat", resp.Title)
}

func TestFetchByTitleYearNotFound(t *testing.T) {
	withOMDBServer(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"Response":"False","Error":"Movie not found!"}`))
	})

	resp, err := FetchByTitleYear(context.Background(), "Nope", 1900)
	require.NoError(t, err)
	assert.Nil(t, resp)
}

func TestFetchByTitleYearRateLimit(t *testing.T) {
	withOMDBServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"Response":"False","Error":"Request limit reached!"}`))
	})

	_, err := FetchByTitleYear(context.Background(), "X", 2000)
	require.Error(t, err)
	assert.True(t, internalerrors.IsRateLimitError(err))
}
