package tmdb

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSearchMovies_FiltersAndLimits(t *testing.T) {
	var capturedQuery url.Values
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedQuery = r.URL.Query()
		response := map[string]any{
			"results": []map[string]any{
				{
					"id":           1,
					"title":        "Unrated",
					"vote_average": 0.0,
				},
				{
					"id":           2,
					"title":        "The Matrix",
					"vote_average": 8.2,
				},
				{
					"id":           3,
					"title":        "The Matrix Reloaded",
					"vote_average": 7.1,
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		require.NoError(t, json.NewEncoder(w).Encode(response))
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))

	results, err := client.SearchMovies(context.Background(), "matrix", 1999, 0)
	require.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, 2, results[0].ID)
	assert.Equal(t, "The Matrix", results[0].Title)

	assert.Equal(t, "matrix", capturedQuery.Get("query"))
	assert.Equal(t, "1999", capturedQuery.Get("year"))
	assert.Equal(t, "false", capturedQuery.Get("include_adult"))
}
