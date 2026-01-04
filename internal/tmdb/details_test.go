package tmdb

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetMovieDetails_Success(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify URL and API key
		require.Contains(t, r.URL.Path, "/movie/550")
		require.NotEmpty(t, r.URL.Query().Get("api_key"))

		// Return mock movie details
		response := map[string]any{
			"id":       550,
			"title":    "Fight Club",
			"runtime":  139,
			"overview": "A ticking-time-bomb insomniac...",
			"genres": []map[string]any{
				{"id": 18, "name": "Drama"},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		require.NoError(t, json.NewEncoder(w).Encode(response))
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))

	details, err := client.GetMovieDetails(context.Background(), 550)
	require.NoError(t, err)
	require.NotNil(t, details)

	require.Equal(t, float64(550), details["id"])
	require.Equal(t, "Fight Club", details["title"])
	require.Equal(t, float64(139), details["runtime"])
}

func TestGetMovieDetails_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"status_code": 34, "status_message": "The resource you requested could not be found."}`))
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))

	details, err := client.GetMovieDetails(context.Background(), 999999)
	require.Error(t, err)
	require.Nil(t, details)
}

func TestGetTVDetails_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Contains(t, r.URL.Path, "/tv/1396")
		require.NotEmpty(t, r.URL.Query().Get("api_key"))

		response := map[string]any{
			"id":                 1396,
			"name":               "Breaking Bad",
			"number_of_episodes": 62,
			"episode_run_time":   []any{45.0, 47.0},
			"status":             "Ended",
			"genres": []map[string]any{
				{"id": 18, "name": "Drama"},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		require.NoError(t, json.NewEncoder(w).Encode(response))
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))

	details, err := client.GetTVDetails(context.Background(), 1396, "")
	require.NoError(t, err)
	require.NotNil(t, details)

	require.Equal(t, float64(1396), details["id"])
	require.Equal(t, "Breaking Bad", details["name"])
	require.Equal(t, float64(62), details["number_of_episodes"])
}

func TestGetTVDetails_WithAppendToResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Contains(t, r.URL.Path, "/tv/1396")
		require.Equal(t, "external_ids,keywords", r.URL.Query().Get("append_to_response"))

		response := map[string]any{
			"id":   1396,
			"name": "Breaking Bad",
			"external_ids": map[string]any{
				"imdb_id": "tt0903747",
			},
		}
		w.Header().Set("Content-Type", "application/json")
		require.NoError(t, json.NewEncoder(w).Encode(response))
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))

	details, err := client.GetTVDetails(context.Background(), 1396, "external_ids,keywords")
	require.NoError(t, err)
	require.NotNil(t, details)

	externalIDs, ok := details["external_ids"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "tt0903747", externalIDs["imdb_id"])
}

func TestGetFullMovieDetails_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Contains(t, r.URL.Path, "/movie/550")
		require.Equal(t, "external_ids,keywords,credits", r.URL.Query().Get("append_to_response"))

		response := map[string]any{
			"id":    550,
			"title": "Fight Club",
			"external_ids": map[string]any{
				"imdb_id": "tt0137523",
			},
			"keywords": map[string]any{
				"keywords": []map[string]any{
					{"id": 825, "name": "support group"},
				},
			},
			"credits": map[string]any{
				"cast": []map[string]any{
					{"name": "Brad Pitt", "character": "Tyler Durden"},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		require.NoError(t, json.NewEncoder(w).Encode(response))
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))

	details, err := client.GetFullMovieDetails(context.Background(), 550)
	require.NoError(t, err)
	require.NotNil(t, details)

	require.Equal(t, float64(550), details["id"])
	require.Contains(t, details, "external_ids")
	require.Contains(t, details, "keywords")
	require.Contains(t, details, "credits")
}

func TestGetFullTVDetails_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Contains(t, r.URL.Path, "/tv/1396")
		require.Equal(t, "external_ids,keywords,content_ratings", r.URL.Query().Get("append_to_response"))

		response := map[string]any{
			"id":   1396,
			"name": "Breaking Bad",
			"external_ids": map[string]any{
				"imdb_id": "tt0903747",
			},
			"content_ratings": map[string]any{
				"results": []map[string]any{
					{"iso_3166_1": "US", "rating": "TV-MA"},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		require.NoError(t, json.NewEncoder(w).Encode(response))
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))

	details, err := client.GetFullTVDetails(context.Background(), 1396)
	require.NoError(t, err)
	require.NotNil(t, details)

	require.Equal(t, float64(1396), details["id"])
	require.Contains(t, details, "external_ids")
	require.Contains(t, details, "content_ratings")
}

func TestGetMetadataByID_Movie(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := map[string]any{
			"id":      550,
			"title":   "Fight Club",
			"runtime": 139,
			"genres": []map[string]any{
				{"id": 18, "name": "Drama"},
				{"id": 53, "name": "Thriller"},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		require.NoError(t, json.NewEncoder(w).Encode(response))
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))

	metadata, err := client.GetMetadataByID(context.Background(), 550, "movie")
	require.NoError(t, err)
	require.NotNil(t, metadata)

	require.Equal(t, 550, metadata.TMDBID)
	require.Equal(t, "movie", metadata.TMDBType)
	require.NotNil(t, metadata.Runtime)
	require.Equal(t, 139, *metadata.Runtime)
	require.Len(t, metadata.GenreTags, 2)
}

func TestGetMetadataByID_TV(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := map[string]any{
			"id":                 1396,
			"name":               "Breaking Bad",
			"number_of_episodes": 62,
			"episode_run_time":   []any{45.0, 47.0},
			"status":             "Ended",
			"genres": []map[string]any{
				{"id": 18, "name": "Drama"},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		require.NoError(t, json.NewEncoder(w).Encode(response))
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))

	metadata, err := client.GetMetadataByID(context.Background(), 1396, "tv")
	require.NoError(t, err)
	require.NotNil(t, metadata)

	require.Equal(t, 1396, metadata.TMDBID)
	require.Equal(t, "tv", metadata.TMDBType)
	require.NotNil(t, metadata.Runtime)
	require.Equal(t, 45, *metadata.Runtime) // Takes first value from episode_run_time array
	require.NotNil(t, metadata.TotalEpisodes)
	require.Equal(t, 62, *metadata.TotalEpisodes)
	require.Equal(t, "Ended", metadata.Status)
}

func TestGetMetadataByID_InvalidMediaType(t *testing.T) {
	client := NewClient("test-api-key")

	metadata, err := client.GetMetadataByID(context.Background(), 123, "invalid")
	require.Error(t, err)
	require.Equal(t, ErrInvalidMediaType, err)
	require.Nil(t, metadata)
}

func TestGetMetadataByResult_Movie(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := map[string]any{
			"id":      550,
			"title":   "Fight Club",
			"runtime": 139,
			"genres": []map[string]any{
				{"id": 18, "name": "Drama"},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		require.NoError(t, json.NewEncoder(w).Encode(response))
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))

	result := SearchResult{
		ID:        550,
		MediaType: "movie",
		Title:     "Fight Club",
	}

	metadata, err := client.GetMetadataByResult(context.Background(), result)
	require.NoError(t, err)
	require.NotNil(t, metadata)
	require.Equal(t, 550, metadata.TMDBID)
	require.Equal(t, "movie", metadata.TMDBType)
}

func TestGetMetadataByResult_TV(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := map[string]any{
			"id":                 1396,
			"name":               "Breaking Bad",
			"number_of_episodes": 62,
			"episode_run_time":   []any{45.0},
			"genres": []map[string]any{
				{"id": 18, "name": "Drama"},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		require.NoError(t, json.NewEncoder(w).Encode(response))
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))

	result := SearchResult{
		ID:        1396,
		MediaType: "tv",
		Name:      "Breaking Bad",
	}

	metadata, err := client.GetMetadataByResult(context.Background(), result)
	require.NoError(t, err)
	require.NotNil(t, metadata)
	require.Equal(t, 1396, metadata.TMDBID)
	require.Equal(t, "tv", metadata.TMDBType)
}

func TestGetMetadataByResult_InvalidMediaType(t *testing.T) {
	client := NewClient("test-api-key")

	result := SearchResult{
		ID:        123,
		MediaType: "person",
		Name:      "Test Person",
	}

	metadata, err := client.GetMetadataByResult(context.Background(), result)
	require.Error(t, err)
	require.Equal(t, ErrInvalidMediaType, err)
	require.Nil(t, metadata)
}

func TestGetMetadataByID_NoRuntime(t *testing.T) {
	setupTMDBCache(t)

	// Test that missing runtime is handled gracefully
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := map[string]any{
			"id":    550,
			"title": "Fight Club",
			// No runtime field
			"genres": []map[string]any{},
		}
		w.Header().Set("Content-Type", "application/json")
		require.NoError(t, json.NewEncoder(w).Encode(response))
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))

	metadata, err := client.GetMetadataByID(context.Background(), 550, "movie")
	require.NoError(t, err)
	require.NotNil(t, metadata)
	require.Nil(t, metadata.Runtime, "runtime should be nil when not provided")
}

func TestGetMetadataByID_NoGenres(t *testing.T) {
	setupTMDBCache(t)

	// Test that missing genres is handled gracefully
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := map[string]any{
			"id":      1396,
			"name":    "Breaking Bad",
			"runtime": 45,
			// No genres field
		}
		w.Header().Set("Content-Type", "application/json")
		require.NoError(t, json.NewEncoder(w).Encode(response))
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))

	metadata, err := client.GetMetadataByID(context.Background(), 1396, "tv")
	require.NoError(t, err)
	require.NotNil(t, metadata)
	require.Empty(t, metadata.GenreTags, "genre tags should be empty when genres not provided")
}
