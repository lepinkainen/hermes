package tmdb

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCachedGetMetadataByID_ForceBypassesCache(t *testing.T) {
	setupTMDBCache(t)

	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.Header().Set("Content-Type", "application/json")
		runtime := 100
		if requestCount > 1 {
			runtime = 200
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":      101,
			"runtime": runtime,
			"genres":  []map[string]any{},
		})
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))
	ctx := context.Background()

	metadata, fromCache, err := client.CachedGetMetadataByID(ctx, 101, "movie", false)
	if err != nil {
		t.Fatalf("CachedGetMetadataByID initial call error = %v", err)
	}
	if fromCache {
		t.Fatalf("Expected initial fetch not from cache")
	}
	if metadata.Runtime == nil || *metadata.Runtime != 100 {
		t.Fatalf("Expected runtime 100 from initial fetch, got %+v", metadata.Runtime)
	}

	metadata, fromCache, err = client.CachedGetMetadataByID(ctx, 101, "movie", true)
	if err != nil {
		t.Fatalf("CachedGetMetadataByID force call error = %v", err)
	}
	if fromCache {
		t.Fatalf("Expected force call to bypass cache")
	}
	if metadata.Runtime == nil || *metadata.Runtime != 200 {
		t.Fatalf("Expected runtime 200 from forced fetch, got %+v", metadata.Runtime)
	}
	if requestCount != 2 {
		t.Fatalf("Expected two HTTP requests after force refresh, got %d", requestCount)
	}

	metadata, fromCache, err = client.CachedGetMetadataByID(ctx, 101, "movie", false)
	if err != nil {
		t.Fatalf("CachedGetMetadataByID cached call error = %v", err)
	}
	if !fromCache {
		t.Fatalf("Expected cached value after force refresh")
	}
	if metadata.Runtime == nil || *metadata.Runtime != 200 {
		t.Fatalf("Expected cached runtime 200 after force refresh, got %+v", metadata.Runtime)
	}
	if requestCount != 2 {
		t.Fatalf("Expected no additional HTTP requests after cached read, got %d", requestCount)
	}
}

func TestCachedGetFullMovieDetails_ForceBypassesCache(t *testing.T) {
	setupTMDBCache(t)

	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.Header().Set("Content-Type", "application/json")
		title := fmt.Sprintf("Title-%d", requestCount)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":    101,
			"title": title,
		})
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))
	ctx := context.Background()

	details, fromCache, err := client.CachedGetFullMovieDetails(ctx, 101, false)
	if err != nil {
		t.Fatalf("CachedGetFullMovieDetails initial call error = %v", err)
	}
	if fromCache {
		t.Fatalf("Expected initial full details fetch not from cache")
	}
	if details["title"] != "Title-1" {
		t.Fatalf("Expected initial title Title-1, got %v", details["title"])
	}

	details, fromCache, err = client.CachedGetFullMovieDetails(ctx, 101, true)
	if err != nil {
		t.Fatalf("CachedGetFullMovieDetails force call error = %v", err)
	}
	if fromCache {
		t.Fatalf("Expected force full details call to bypass cache")
	}
	if details["title"] != "Title-2" {
		t.Fatalf("Expected forced title Title-2, got %v", details["title"])
	}
	if requestCount != 2 {
		t.Fatalf("Expected two HTTP requests after force refresh, got %d", requestCount)
	}

	details, fromCache, err = client.CachedGetFullMovieDetails(ctx, 101, false)
	if err != nil {
		t.Fatalf("CachedGetFullMovieDetails cached call error = %v", err)
	}
	if !fromCache {
		t.Fatalf("Expected cached data after force refresh")
	}
	if details["title"] != "Title-2" {
		t.Fatalf("Expected cached title Title-2, got %v", details["title"])
	}
	if requestCount != 2 {
		t.Fatalf("Expected no additional HTTP requests after cached read, got %d", requestCount)
	}
}
