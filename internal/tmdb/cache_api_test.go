package tmdb

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

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
