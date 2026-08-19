package enrichment

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lepinkainen/hermes/internal/cache"
	"github.com/lepinkainen/hermes/internal/ratelimit"
	"github.com/lepinkainen/hermes/internal/tui"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

// withTestCacheDB configures a temporary, isolated cache database for the
// duration of the test so that caching behavior (negative caching, exact
// match caching, etc.) can be exercised without touching the real cache.db.
func withTestCacheDB(t *testing.T) {
	t.Helper()

	tmpDB := t.TempDir() + "/test_cache.db"
	viper.Set("cache.dbfile", tmpDB)

	t.Cleanup(func() {
		_ = cache.ResetGlobalCache()
		viper.Set("cache.dbfile", "./cache.db")
	})
}

func withOpenLibrarySearchServer(t *testing.T, handler http.HandlerFunc) {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	prevURL := openLibrarySearchURL
	prevClient := openLibrarySearchHTTPClient
	prevLimiter := openLibrarySearchLimiter
	openLibrarySearchURL = server.URL
	openLibrarySearchHTTPClient = server.Client()
	// Use a fast, high-burst limiter so tests don't serialize behind the
	// production 1 req/s rate limit.
	fastLimiter := ratelimit.NewWithBurst("OpenLibrary test", 1000, 1000)
	openLibrarySearchLimiter = func() *ratelimit.Limiter { return fastLimiter }
	t.Cleanup(func() {
		openLibrarySearchURL = prevURL
		openLibrarySearchHTTPClient = prevClient
		openLibrarySearchLimiter = prevLimiter
	})
}

func TestFetchOpenLibrarySearch_ParsesResults(t *testing.T) {
	withOpenLibrarySearchServer(t, func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "Dune", r.URL.Query().Get("title"))
		require.Equal(t, "10", r.URL.Query().Get("limit"))
		resp := openLibrarySearchResponse{
			Docs: []openLibrarySearchDoc{
				{
					Title:            "Dune",
					AuthorName:       []string{"Frank Herbert"},
					FirstPublishYear: 1965,
					ISBN:             []string{"9780441172719", "0441172717"},
					CoverI:           12345,
					EditionCount:     100,
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		require.NoError(t, json.NewEncoder(w).Encode(resp))
	})

	results, err := fetchOpenLibrarySearch(t.Context(), "Dune", nil)
	require.NoError(t, err)
	require.Len(t, results, 1)
	require.Equal(t, "Dune", results[0].Title)
	require.Equal(t, []string{"Frank Herbert"}, results[0].Authors)
	require.Equal(t, 1965, results[0].FirstPublishYear)
	require.Equal(t, 100, results[0].EditionCount)
	require.Equal(t, 12345, results[0].CoverID)
	require.Equal(t, "9780441172719", results[0].ISBN13)
	require.Equal(t, "0441172717", results[0].ISBN)
}

func TestFetchOpenLibrarySearch_AuthorParamOnlyWhenProvided(t *testing.T) {
	withOpenLibrarySearchServer(t, func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "Frank Herbert", r.URL.Query().Get("author"))
		_ = json.NewEncoder(w).Encode(openLibrarySearchResponse{})
	})

	_, err := fetchOpenLibrarySearch(t.Context(), "Dune", []string{"Frank Herbert", "Someone Else"})
	require.NoError(t, err)
}

func TestFetchOpenLibrarySearch_NoAuthorParamWhenEmpty(t *testing.T) {
	withOpenLibrarySearchServer(t, func(w http.ResponseWriter, r *http.Request) {
		require.Empty(t, r.URL.Query().Get("author"))
		_ = json.NewEncoder(w).Encode(openLibrarySearchResponse{})
	})

	_, err := fetchOpenLibrarySearch(t.Context(), "Dune", nil)
	require.NoError(t, err)
}

func TestFetchOpenLibrarySearch_HTTPError(t *testing.T) {
	withOpenLibrarySearchServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("oops"))
	})

	_, err := fetchOpenLibrarySearch(t.Context(), "Dune", nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "status 500")
}

func TestFetchOpenLibrarySearch_InvalidJSON(t *testing.T) {
	withOpenLibrarySearchServer(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("not json"))
	})

	_, err := fetchOpenLibrarySearch(t.Context(), "Dune", nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "parse OpenLibrary search response")
}

func TestPickBookISBNs(t *testing.T) {
	tests := []struct {
		name       string
		isbns      []string
		wantISBN10 string
		wantISBN13 string
	}{
		{
			name:       "prefers 978 ISBN-13",
			isbns:      []string{"0441172717", "9780441172719"},
			wantISBN10: "0441172717",
			wantISBN13: "9780441172719",
		},
		{
			name:       "prefers 979 ISBN-13",
			isbns:      []string{"9791234567896"},
			wantISBN10: "",
			wantISBN13: "9791234567896",
		},
		{
			name:       "falls back to ISBN-10 only",
			isbns:      []string{"0441172717"},
			wantISBN10: "0441172717",
			wantISBN13: "",
		},
		{
			name:       "ignores malformed entries",
			isbns:      []string{"invalid", "123"},
			wantISBN10: "",
			wantISBN13: "",
		},
		{
			name:       "empty input",
			isbns:      nil,
			wantISBN10: "",
			wantISBN13: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isbn10, isbn13 := pickBookISBNs(tt.isbns)
			require.Equal(t, tt.wantISBN10, isbn10)
			require.Equal(t, tt.wantISBN13, isbn13)
		})
	}
}

func TestBookSearchCacheKey(t *testing.T) {
	require.Equal(t, "title:dune|author:frank herbert", bookSearchCacheKey("Dune", []string{"Frank Herbert"}))
	require.Equal(t, "title:dune|author:", bookSearchCacheKey("  Dune  ", nil))
	require.Equal(t, "title:dune|author:frank herbert", bookSearchCacheKey("  Dune  ", []string{"  Frank Herbert  ", "Ignored"}))
}

func TestSelectBookResult_SingleResult(t *testing.T) {
	results := []tui.BookSearchResult{
		{Title: "Dune", ISBN13: "9780441172719"},
	}

	selected, err := selectBookResult(results, "Dune", nil, false)
	require.NoError(t, err)
	require.NotNil(t, selected)
	require.Equal(t, "9780441172719", selected.ISBN13)
}

func TestSelectBookResult_EmptyResults(t *testing.T) {
	selected, err := selectBookResult(nil, "Dune", nil, false)
	require.NoError(t, err)
	require.Nil(t, selected)
}

func TestSelectBookResult_ExactMatchNonInteractive(t *testing.T) {
	results := []tui.BookSearchResult{
		{Title: "Dune", ISBN13: "111"},
		{Title: "Dune Messiah", ISBN13: "222"},
	}

	selected, err := selectBookResult(results, "Dune", nil, false)
	require.NoError(t, err)
	require.NotNil(t, selected)
	require.Equal(t, "111", selected.ISBN13)
}

func TestSelectBookResult_NoExactMatchNonInteractive(t *testing.T) {
	results := []tui.BookSearchResult{
		{Title: "Dune", ISBN13: "111"},
		{Title: "Dune Messiah", ISBN13: "222"},
	}

	selected, err := selectBookResult(results, "Children of Dune", nil, false)
	require.NoError(t, err)
	require.NotNil(t, selected)
	require.Equal(t, "111", selected.ISBN13, "should select first result when no exact match")
}

func TestFindExactBookMatch(t *testing.T) {
	results := []tui.BookSearchResult{
		{Title: "Dune", Authors: []string{"Frank Herbert"}, ISBN13: "111"},
		{Title: "Dune", Authors: []string{"Someone Else"}, ISBN13: "222"},
	}

	// Ambiguous by title alone.
	require.Nil(t, findExactBookMatch(results, "Dune", nil))

	// Disambiguated by author.
	match := findExactBookMatch(results, "Dune", []string{"Frank Herbert"})
	require.NotNil(t, match)
	require.Equal(t, "111", match.ISBN13)

	// No match.
	require.Nil(t, findExactBookMatch(results, "Nope", nil))
}

func TestFindExactBookMatch_CaseInsensitive(t *testing.T) {
	results := []tui.BookSearchResult{
		{Title: "Dune", Authors: []string{"Frank Herbert"}, ISBN13: "111"},
	}

	match := findExactBookMatch(results, "DUNE", []string{"frank herbert"})
	require.NotNil(t, match)
	require.Equal(t, "111", match.ISBN13)
}

func TestResolveBookISBN_NoResults(t *testing.T) {
	withTestCacheDB(t)
	withOpenLibrarySearchServer(t, func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(openLibrarySearchResponse{})
	})

	isbn10, isbn13, err := resolveBookISBN(t.Context(), "Nonexistent Book Title XYZ", BookEnrichmentOptions{})
	require.NoError(t, err)
	require.Empty(t, isbn10)
	require.Empty(t, isbn13)
}

func TestResolveBookISBN_NegativeCache(t *testing.T) {
	withTestCacheDB(t)

	calls := 0
	withOpenLibrarySearchServer(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		_ = json.NewEncoder(w).Encode(openLibrarySearchResponse{})
	})

	title := "Negative Cache Test Book Unique Title"
	_, _, err := resolveBookISBN(t.Context(), title, BookEnrichmentOptions{})
	require.NoError(t, err)

	_, _, err = resolveBookISBN(t.Context(), title, BookEnrichmentOptions{})
	require.NoError(t, err)

	require.Equal(t, 1, calls, "second lookup should be served from the negative cache")
}

func TestResolveBookISBN_AutoSelectsExactMatch(t *testing.T) {
	withTestCacheDB(t)

	withOpenLibrarySearchServer(t, func(w http.ResponseWriter, r *http.Request) {
		resp := openLibrarySearchResponse{
			Docs: []openLibrarySearchDoc{
				{Title: "Some Unique Book Title", AuthorName: []string{"Jane Doe"}, ISBN: []string{"9780441172719"}},
				{Title: "Some Unique Book Title 2", AuthorName: []string{"Jane Doe"}, ISBN: []string{"9780000000000"}},
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	})

	isbn10, isbn13, err := resolveBookISBN(t.Context(), "Some Unique Book Title", BookEnrichmentOptions{})
	require.NoError(t, err)
	require.Empty(t, isbn10)
	require.Equal(t, "9780441172719", isbn13)
}
