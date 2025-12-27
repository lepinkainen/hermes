package tmdb

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCachedSearchMovies_DoesNotCacheMisses(t *testing.T) {
	setupTMDBCache(t)

	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.Header().Set("Content-Type", "application/json")
		if requestCount == 1 {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"results": []map[string]any{},
			})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"results": []map[string]any{
				{
					"id":           101,
					"title":        "Fresh Result",
					"poster_path":  "/poster.jpg",
					"overview":     "desc",
					"release_date": "2024-01-01",
					"vote_average": 7.5,
				},
			},
		})
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))
	ctx := context.Background()

	results, fromCache, err := client.CachedSearchMovies(ctx, "query", 0, 5)
	if err != nil {
		t.Fatalf("CachedSearchMovies first call error = %v", err)
	}
	if fromCache {
		t.Fatalf("Expected first call not from cache")
	}
	if len(results) != 0 {
		t.Fatalf("Expected no results on first call, got %d", len(results))
	}

	results, fromCache, err = client.CachedSearchMovies(ctx, "query", 0, 5)
	if err != nil {
		t.Fatalf("CachedSearchMovies second call error = %v", err)
	}
	if fromCache {
		t.Fatalf("Expected second call to bypass cached miss")
	}
	if len(results) != 1 {
		t.Fatalf("Expected fresh result after miss, got %d", len(results))
	}
	if results[0].ID != 101 {
		t.Fatalf("Expected result ID 101, got %d", results[0].ID)
	}
	if requestCount != 2 {
		t.Fatalf("Expected two HTTP requests, got %d", requestCount)
	}

	// Third call should use the cached hit
	_, fromCache, err = client.CachedSearchMovies(ctx, "query", 0, 5)
	if err != nil {
		t.Fatalf("CachedSearchMovies third call error = %v", err)
	}
	if !fromCache {
		t.Fatalf("Expected third call to hit cache")
	}
	if requestCount != 2 {
		t.Fatalf("Expected no additional HTTP requests after cache fill, got %d", requestCount)
	}
}

func TestCachedFindByIMDBID_DoesNotCacheMisses(t *testing.T) {
	setupTMDBCache(t)

	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.Header().Set("Content-Type", "application/json")
		if requestCount == 1 {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"movie_results": []map[string]any{},
				"tv_results":    []map[string]any{},
			})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"movie_results": []map[string]any{
				{"id": 202},
			},
			"tv_results": []map[string]any{},
		})
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))
	ctx := context.Background()

	tmdbID, mediaType, fromCache, err := client.CachedFindByIMDBID(ctx, "tt0123456")
	if err != nil {
		t.Fatalf("CachedFindByIMDBID first call error = %v", err)
	}
	if tmdbID != 0 || mediaType != "" {
		t.Fatalf("Expected no result on first call, got id=%d type=%s", tmdbID, mediaType)
	}
	if fromCache {
		t.Fatalf("Expected first call not from cache")
	}

	tmdbID, mediaType, fromCache, err = client.CachedFindByIMDBID(ctx, "tt0123456")
	if err != nil {
		t.Fatalf("CachedFindByIMDBID second call error = %v", err)
	}
	if fromCache {
		t.Fatalf("Expected second call to bypass cached miss")
	}
	if tmdbID != 202 || mediaType != "movie" {
		t.Fatalf("Expected movie result id=202, type=movie; got id=%d type=%s", tmdbID, mediaType)
	}
	if requestCount != 2 {
		t.Fatalf("Expected two HTTP requests, got %d", requestCount)
	}

	_, _, fromCache, err = client.CachedFindByIMDBID(ctx, "tt0123456")
	if err != nil {
		t.Fatalf("CachedFindByIMDBID third call error = %v", err)
	}
	if !fromCache {
		t.Fatalf("Expected third call to hit cache")
	}
	if requestCount != 2 {
		t.Fatalf("Expected no additional HTTP requests after cache fill, got %d", requestCount)
	}
}
