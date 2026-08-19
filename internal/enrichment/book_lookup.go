package enrichment

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/lepinkainen/hermes/internal/cache"
	hermeserrors "github.com/lepinkainen/hermes/internal/errors"
	"github.com/lepinkainen/hermes/internal/ratelimit"
	"github.com/lepinkainen/hermes/internal/tui"
)

// HTTP seams — package vars so tests can redirect to an httptest.Server.
var (
	openLibrarySearchURL        = "https://openlibrary.org/search.json"
	openLibrarySearchHTTPClient = &http.Client{Timeout: 30 * time.Second}
)

// openLibrarySearchLimiter is a package-level singleton rate limiter for
// OpenLibrary search requests (1 req/s, per OpenLibrary's usage guidelines).
// It is separate from the per-enricher limiter used by
// internal/enrichment/book/enrichers.OpenLibraryEnricher, which is not
// exported for reuse.
var openLibrarySearchLimiter = sync.OnceValue(func() *ratelimit.Limiter {
	return ratelimit.New("OpenLibrary", 1)
})

// openLibrarySearchResponse matches the OpenLibrary search.json response.
type openLibrarySearchResponse struct {
	Docs []openLibrarySearchDoc `json:"docs"`
}

type openLibrarySearchDoc struct {
	Title            string   `json:"title"`
	AuthorName       []string `json:"author_name"`
	FirstPublishYear int      `json:"first_publish_year"`
	ISBN             []string `json:"isbn"`
	CoverI           int      `json:"cover_i"`
	EditionCount     int      `json:"edition_count"`
}

// cachedBookSearchResult wraps search results for caching, with a NotFound
// flag to enable negative caching of empty result sets.
type cachedBookSearchResult struct {
	Results  []tui.BookSearchResult `json:"results"`
	NotFound bool                   `json:"not_found"`
}

// resolveBookISBN searches OpenLibrary for the given title (and authors, if
// provided) and returns the ISBN-10 and ISBN-13 of the selected result. It
// returns ("", "", nil) when no results are found or the user skips
// selection in interactive mode.
func resolveBookISBN(ctx context.Context, title string, opts BookEnrichmentOptions) (isbn10, isbn13 string, err error) {
	results, err := searchOpenLibraryBooks(ctx, title, opts.Authors)
	if err != nil {
		return "", "", fmt.Errorf("openlibrary search failed: %w", err)
	}

	if len(results) == 0 {
		slog.Debug("No OpenLibrary results found", "title", title)
		return "", "", nil
	}

	selected, err := selectBookResult(results, title, opts.Authors, opts.Interactive)
	if err != nil {
		return "", "", err
	}
	if selected == nil {
		return "", "", nil
	}

	return selected.ISBN, selected.ISBN13, nil
}

// searchOpenLibraryBooks searches OpenLibrary by title (and optionally the
// first author) with caching. Empty result sets are negatively cached with
// a shorter TTL.
func searchOpenLibraryBooks(ctx context.Context, title string, authors []string) ([]tui.BookSearchResult, error) {
	cacheKey := bookSearchCacheKey(title, authors)

	cached, _, err := cache.GetOrFetchWithTTL(
		"openlibrary_search_cache",
		cacheKey,
		func() (*cachedBookSearchResult, error) {
			results, fetchErr := fetchOpenLibrarySearch(ctx, title, authors)
			if fetchErr != nil {
				return nil, fetchErr
			}
			return &cachedBookSearchResult{Results: results, NotFound: len(results) == 0}, nil
		},
		cache.SelectNegativeCacheTTL(func(r *cachedBookSearchResult) bool {
			return r.NotFound
		}),
	)
	if err != nil {
		return nil, err
	}

	return cached.Results, nil
}

// bookSearchCacheKey builds a deterministic cache key from title and the
// first author (if provided).
func bookSearchCacheKey(title string, authors []string) string {
	author := ""
	if len(authors) > 0 {
		author = strings.ToLower(strings.TrimSpace(authors[0]))
	}
	return fmt.Sprintf("title:%s|author:%s", strings.ToLower(strings.TrimSpace(title)), author)
}

// fetchOpenLibrarySearch performs the actual OpenLibrary search.json API call.
func fetchOpenLibrarySearch(ctx context.Context, title string, authors []string) ([]tui.BookSearchResult, error) {
	limiter := openLibrarySearchLimiter()
	if err := limiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limit wait: %w", err)
	}

	params := url.Values{}
	params.Set("title", title)
	if len(authors) > 0 && authors[0] != "" {
		params.Set("author", authors[0])
	}
	params.Set("limit", "10")
	params.Set("fields", "title,author_name,first_publish_year,isbn,cover_i,edition_count")

	searchURL := openLibrarySearchURL + "?" + params.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, searchURL, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := openLibrarySearchHTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch OpenLibrary search: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("openlibrary search returned status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var searchResp openLibrarySearchResponse
	if err := json.Unmarshal(body, &searchResp); err != nil {
		return nil, fmt.Errorf("failed to parse OpenLibrary search response: %w", err)
	}

	results := make([]tui.BookSearchResult, len(searchResp.Docs))
	for i, doc := range searchResp.Docs {
		bookISBN, bookISBN13 := pickBookISBNs(doc.ISBN)
		results[i] = tui.BookSearchResult{
			Title:            doc.Title,
			Authors:          doc.AuthorName,
			FirstPublishYear: doc.FirstPublishYear,
			EditionCount:     doc.EditionCount,
			CoverID:          doc.CoverI,
			ISBN:             bookISBN,
			ISBN13:           bookISBN13,
		}
	}

	return results, nil
}

// pickBookISBNs picks a representative ISBN-10 and ISBN-13 from an
// OpenLibrary search doc's isbn array: the first 13-digit entry starting
// with 978/979 for ISBN-13, and the first 10-character entry for ISBN-10.
func pickBookISBNs(isbns []string) (isbn10, isbn13 string) {
	for _, v := range isbns {
		if isbn13 == "" && len(v) == 13 && (strings.HasPrefix(v, "978") || strings.HasPrefix(v, "979")) {
			isbn13 = v
		}
		if isbn10 == "" && len(v) == 10 {
			isbn10 = v
		}
		if isbn10 != "" && isbn13 != "" {
			break
		}
	}
	return isbn10, isbn13
}

// selectBookResult picks a result from search results: auto-selects when
// there's a single result or an unambiguous exact title/author match,
// otherwise defers to the TUI (interactive) or picks the first result
// (non-interactive).
func selectBookResult(results []tui.BookSearchResult, title string, authors []string, interactive bool) (*tui.BookSearchResult, error) {
	if len(results) == 0 {
		return nil, nil
	}

	if len(results) == 1 {
		slog.Debug("Auto-selected single OpenLibrary result", "title", title)
		return &results[0], nil
	}

	exact := findExactBookMatch(results, title, authors)

	if interactive {
		selection, err := selectBookInteractive(title, results)
		if err != nil {
			return nil, err
		}
		return selection, nil
	}

	if exact != nil {
		slog.Debug("Auto-selected exact OpenLibrary match", "title", title)
		return exact, nil
	}

	slog.Debug("Auto-selected first OpenLibrary result", "title", title)
	return &results[0], nil
}

// findExactBookMatch returns the unique result whose title matches
// (case-insensitively) and, when authors is non-empty, whose author list
// contains the first given author. Returns nil if there's no match or the
// match is ambiguous.
func findExactBookMatch(results []tui.BookSearchResult, title string, authors []string) *tui.BookSearchResult {
	normalizedTitle := strings.ToLower(strings.TrimSpace(title))

	wantAuthor := ""
	if len(authors) > 0 {
		wantAuthor = strings.ToLower(strings.TrimSpace(authors[0]))
	}

	var match *tui.BookSearchResult
	matchCount := 0

	for i := range results {
		result := &results[i]
		if strings.ToLower(strings.TrimSpace(result.Title)) != normalizedTitle {
			continue
		}
		if wantAuthor != "" && !bookAuthorMatches(result.Authors, wantAuthor) {
			continue
		}

		match = result
		matchCount++
		if matchCount > 1 {
			return nil // Ambiguous
		}
	}

	return match
}

func bookAuthorMatches(authors []string, want string) bool {
	for _, a := range authors {
		if strings.ToLower(strings.TrimSpace(a)) == want {
			return true
		}
	}
	return false
}

// selectBookInteractive presents a TUI for book search result selection.
func selectBookInteractive(title string, results []tui.BookSearchResult) (*tui.BookSearchResult, error) {
	selection, err := tui.SelectBook(title, results, nil)
	if err != nil {
		return nil, fmt.Errorf("TUI selection failed: %w", err)
	}

	switch selection.Action {
	case tui.ActionSelected:
		if selection.BookSelection != nil {
			return selection.BookSelection, nil
		}
		return nil, nil
	case tui.ActionStopped:
		return nil, hermeserrors.NewStopProcessingError("Book selection stopped by user")
	default:
		slog.Debug("User skipped Book selection")
		return nil, nil
	}
}
