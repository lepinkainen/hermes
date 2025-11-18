package tmdb

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/lepinkainen/hermes/internal/cache"
	"github.com/spf13/viper"
)

func TestNormalizeQuery(t *testing.T) {
	tests := map[string]string{
		"The Matrix":           "the_matrix",
		"The Matrix 1999":      "the_matrix_1999",
		"  Spaces  Around  ":   "spaces__around",
		"Special!@#$%^&*Chars": "special________chars",
		"UPPERCASE":            "uppercase",
		"already_normalized":   "already_normalized",
		"dash-separated":       "dash-separated",
		"numbers123":           "numbers123",
		"":                     "",
	}

	for input, want := range tests {
		got := normalizeQuery(input)
		if got != want {
			t.Errorf("normalizeQuery(%q) = %q, want %q", input, got, want)
		}
	}
}

func setupTMDBCache(t *testing.T) {
	t.Helper()

	viper.Reset()
	t.Cleanup(viper.Reset)

	tmpDir := filepath.Join(os.TempDir(), "hermes-tmdb-cache-tests")
	if err := os.MkdirAll(tmpDir, 0o755); err != nil {
		t.Fatalf("Failed to create temp cache dir: %v", err)
	}

	viper.Set("cache.dbfile", filepath.Join(tmpDir, "tmdb-cache.db"))
	viper.Set("cache.ttl", "24h")

	cacheDB, err := cache.GetGlobalCache()
	if err != nil {
		t.Fatalf("Failed to init cache: %v", err)
	}
	if err := cacheDB.ClearAll("tmdb_cache"); err != nil {
		t.Fatalf("Failed to reset tmdb_cache table: %v", err)
	}
}

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

func TestFindByIMDBID(t *testing.T) {
	tests := []struct {
		name     string
		imdbID   string
		response map[string]any
		wantID   int
		wantType string
		wantErr  bool
	}{
		{
			name:   "finds movie by IMDB ID",
			imdbID: "tt0111161",
			response: map[string]any{
				"movie_results": []map[string]any{
					{"id": 278},
				},
				"tv_results": []map[string]any{},
			},
			wantID:   278,
			wantType: "movie",
			wantErr:  false,
		},
		{
			name:   "finds TV show by IMDB ID",
			imdbID: "tt0903747",
			response: map[string]any{
				"movie_results": []map[string]any{},
				"tv_results": []map[string]any{
					{"id": 1396},
				},
			},
			wantID:   1396,
			wantType: "tv",
			wantErr:  false,
		},
		{
			name:   "prefers movie over TV when both found",
			imdbID: "tt1234567",
			response: map[string]any{
				"movie_results": []map[string]any{
					{"id": 100},
				},
				"tv_results": []map[string]any{
					{"id": 200},
				},
			},
			wantID:   100,
			wantType: "movie",
			wantErr:  false,
		},
		{
			name:   "returns zero when not found",
			imdbID: "tt0000000",
			response: map[string]any{
				"movie_results": []map[string]any{},
				"tv_results":    []map[string]any{},
			},
			wantID:   0,
			wantType: "",
			wantErr:  false,
		},
		{
			name:     "empty IMDB ID returns zero",
			imdbID:   "",
			response: nil,
			wantID:   0,
			wantType: "",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var server *httptest.Server
			if tt.response != nil {
				server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "application/json")
					_ = json.NewEncoder(w).Encode(tt.response)
				}))
				defer server.Close()
			}

			opts := []Option{}
			if server != nil {
				opts = append(opts, WithBaseURL(server.URL))
			}

			client := NewClient("test-api-key", opts...)

			gotID, gotType, err := client.FindByIMDBID(context.Background(), tt.imdbID)
			if (err != nil) != tt.wantErr {
				t.Errorf("FindByIMDBID() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotID != tt.wantID {
				t.Errorf("FindByIMDBID() gotID = %v, want %v", gotID, tt.wantID)
			}
			if gotType != tt.wantType {
				t.Errorf("FindByIMDBID() gotType = %v, want %v", gotType, tt.wantType)
			}
		})
	}
}

func TestSearchMulti(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := map[string]any{
			"results": []map[string]any{
				{
					"id":           278,
					"media_type":   "movie",
					"title":        "The Shawshank Redemption",
					"poster_path":  "/poster.jpg",
					"overview":     "Test overview",
					"release_date": "1994-09-23",
					"vote_average": 8.7,
				},
				{
					"id":             1396,
					"media_type":     "tv",
					"name":           "Breaking Bad",
					"poster_path":    "/bb.jpg",
					"overview":       "TV overview",
					"first_air_date": "2008-01-20",
					"vote_average":   9.5,
				},
				{
					"id":         999,
					"media_type": "person", // Should be filtered out
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))

	results, err := client.SearchMulti(context.Background(), "test", 1994, 10)
	if err != nil {
		t.Fatalf("SearchMulti() error = %v", err)
	}

	// Should have 2 results (person filtered out)
	if len(results) != 2 {
		t.Errorf("SearchMulti() got %d results, want 2", len(results))
	}

	// Check first result (API order preserved)
	if results[0].ID != 278 {
		t.Errorf("SearchMulti() first result ID = %d, want 278", results[0].ID)
	}
	if results[0].MediaType != "movie" {
		t.Errorf("SearchMulti() first result MediaType = %s, want movie", results[0].MediaType)
	}
	if results[0].DisplayTitle() != "The Shawshank Redemption" {
		t.Errorf("SearchMulti() first result DisplayTitle = %s, want The Shawshank Redemption", results[0].DisplayTitle())
	}

	// Check second result
	if results[1].ID != 1396 {
		t.Errorf("SearchMulti() second result ID = %d, want 1396", results[1].ID)
	}
	if results[1].MediaType != "tv" {
		t.Errorf("SearchMulti() second result MediaType = %s, want tv", results[1].MediaType)
	}
}

func TestSearchResultYear(t *testing.T) {
	tests := []struct {
		name     string
		result   SearchResult
		wantYear string
	}{
		{
			name: "movie with release date",
			result: SearchResult{
				MediaType:   "movie",
				ReleaseDate: "1994-09-23",
			},
			wantYear: "1994",
		},
		{
			name: "tv show with first air date",
			result: SearchResult{
				MediaType:    "tv",
				FirstAirDate: "2008-01-20",
			},
			wantYear: "2008",
		},
		{
			name: "missing date",
			result: SearchResult{
				MediaType: "movie",
			},
			wantYear: "Unknown",
		},
		{
			name: "short date string",
			result: SearchResult{
				MediaType:   "movie",
				ReleaseDate: "94",
			},
			wantYear: "94",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.result.Year(); got != tt.wantYear {
				t.Errorf("Year() = %s, want %s", got, tt.wantYear)
			}
		})
	}
}
