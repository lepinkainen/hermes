package tmdb

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCachedGetMovieDetailsCaches(t *testing.T) {
	setupTMDBCache(t)

	var calls int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		response := map[string]any{"id": 101, "title": "Cache Movie"}
		w.Header().Set("Content-Type", "application/json")
		require.NoError(t, json.NewEncoder(w).Encode(response))
	}))
	defer server.Close()

	client := NewClient("key", WithBaseURL(server.URL), WithHTTPClient(server.Client()))

	details, fromCache, err := client.CachedGetMovieDetails(context.Background(), 101)
	require.NoError(t, err)
	require.False(t, fromCache)
	require.Equal(t, float64(101), details["id"])

	details, fromCache, err = client.CachedGetMovieDetails(context.Background(), 101)
	require.NoError(t, err)
	require.True(t, fromCache)
	require.Equal(t, float64(101), details["id"])

	require.Equal(t, int32(1), atomic.LoadInt32(&calls))
}

func TestCachedGetTVDetailsCaches(t *testing.T) {
	setupTMDBCache(t)

	var calls int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		response := map[string]any{"id": 202, "name": "Cache Show"}
		w.Header().Set("Content-Type", "application/json")
		require.NoError(t, json.NewEncoder(w).Encode(response))
	}))
	defer server.Close()

	client := NewClient("key", WithBaseURL(server.URL), WithHTTPClient(server.Client()))

	details, fromCache, err := client.CachedGetTVDetails(context.Background(), 202, "external_ids")
	require.NoError(t, err)
	require.False(t, fromCache)
	require.Equal(t, float64(202), details["id"])

	details, fromCache, err = client.CachedGetTVDetails(context.Background(), 202, "external_ids")
	require.NoError(t, err)
	require.True(t, fromCache)
	require.Equal(t, float64(202), details["id"])

	require.Equal(t, int32(1), atomic.LoadInt32(&calls))
}
