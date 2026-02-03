package tmdb

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetCoverAndMetadataByResult(t *testing.T) {
	setupTMDBCache(t)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/movie/123":
			response := map[string]any{
				"id":          123,
				"poster_path": "/poster.jpg",
				"runtime":     120,
				"genres": []map[string]any{
					{"id": 18, "name": "Drama"},
				},
				"external_ids": map[string]any{"imdb_id": "tt123"},
			}
			w.Header().Set("Content-Type", "application/json")
			require.NoError(t, json.NewEncoder(w).Encode(response))
		case "/genre/movie/list":
			response := map[string]any{
				"genres": []map[string]any{{"id": 18, "name": "Drama"}},
			}
			w.Header().Set("Content-Type", "application/json")
			require.NoError(t, json.NewEncoder(w).Encode(response))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := NewClient("key", WithBaseURL(server.URL), WithHTTPClient(server.Client()), WithImageBaseURL("https://images.example"), WithRateLimiter(nil))

	cover, metadata, err := client.GetCoverAndMetadataByResult(context.Background(), SearchResult{ID: 123, MediaType: "movie", PosterPath: "/poster.jpg"})
	require.NoError(t, err)
	require.Equal(t, "https://images.example/poster.jpg", cover)
	require.NotNil(t, metadata)
	require.NotNil(t, metadata.Runtime)
	require.Equal(t, 120, *metadata.Runtime)
	require.Equal(t, []string{"movie/Drama"}, metadata.GenreTags)
	require.Equal(t, "tt123", metadata.IMDBID)
}
