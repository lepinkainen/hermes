package tmdb

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetGenresCachesResponse(t *testing.T) {
	var calls int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		response := map[string]any{
			"genres": []map[string]any{{"id": 1, "name": "Action"}},
		}
		w.Header().Set("Content-Type", "application/json")
		require.NoError(t, json.NewEncoder(w).Encode(response))
	}))
	defer server.Close()

	client := NewClient("key", WithBaseURL(server.URL), WithHTTPClient(server.Client()), WithRateLimiter(nil))

	genres, err := client.getGenres(context.Background(), "movie")
	require.NoError(t, err)
	assert.Equal(t, "Action", genres[1])

	genres, err = client.getGenres(context.Background(), "movie")
	require.NoError(t, err)
	assert.Equal(t, "Action", genres[1])

	assert.Equal(t, int32(1), atomic.LoadInt32(&calls))
}

func TestBuildGenreTags(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := map[string]any{
			"genres": []map[string]any{{"id": 10, "name": "Science Fiction"}},
		}
		w.Header().Set("Content-Type", "application/json")
		require.NoError(t, json.NewEncoder(w).Encode(response))
	}))
	defer server.Close()

	client := NewClient("key", WithBaseURL(server.URL), WithHTTPClient(server.Client()), WithRateLimiter(nil))

	details := map[string]any{
		"genres": []any{
			map[string]any{"id": 10},
			map[string]any{"id": 999},
			"invalid",
		},
	}

	tags, err := client.buildGenreTags(context.Background(), "movie", details)
	require.NoError(t, err)
	assert.Equal(t, []string{"movie/Science-Fiction"}, tags)
}
