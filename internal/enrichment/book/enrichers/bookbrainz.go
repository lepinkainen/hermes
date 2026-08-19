package enrichers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/lepinkainen/hermes/internal/cache"
	"github.com/lepinkainen/hermes/internal/enrichment/book"
	"github.com/lepinkainen/hermes/internal/parseutil"
	"github.com/lepinkainen/hermes/internal/ratelimit"
)

const (
	bookBrainzBaseURL    = "https://bookbrainz.org"
	bookBrainzAPIBaseURL = "https://api.bookbrainz.org/1"
	bookBrainzPriority   = 3
)

// BookBrainzEnricher implements the book.Enricher interface for BookBrainz.
type BookBrainzEnricher struct {
	getHTTPClient  func() *http.Client
	getRateLimiter func() *ratelimit.Limiter
}

// Compile-time check that BookBrainzEnricher implements book.Enricher.
var _ book.Enricher = (*BookBrainzEnricher)(nil)

// NewBookBrainzEnricher creates a new BookBrainz enricher.
func NewBookBrainzEnricher() *BookBrainzEnricher {
	return &BookBrainzEnricher{
		getHTTPClient: sync.OnceValue(func() *http.Client {
			return &http.Client{Timeout: 15 * time.Second}
		}),
		getRateLimiter: sync.OnceValue(func() *ratelimit.Limiter {
			return ratelimit.New("BookBrainz", 1)
		}),
	}
}

// Name returns the human-readable name of this enricher.
func (e *BookBrainzEnricher) Name() string {
	return "BookBrainz"
}

// Priority returns the priority for merging data (lower = higher precedence).
func (e *BookBrainzEnricher) Priority() int {
	return bookBrainzPriority
}

// Ping tests the connection to BookBrainz.
func (e *BookBrainzEnricher) Ping(ctx context.Context) error {
	client := e.getHTTPClient()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, bookBrainzBaseURL, http.NoBody)
	if err != nil {
		return fmt.Errorf("creating ping request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("BookBrainz ping failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("BookBrainz returned status %d", resp.StatusCode)
	}

	return nil
}

// Enrich fetches book data from BookBrainz by ISBN.
func (e *BookBrainzEnricher) Enrich(ctx context.Context, isbn string) (*book.EnrichmentData, error) {
	if isbn == "" {
		return nil, book.ErrInvalidISBN
	}

	normalizedISBN := parseutil.NormalizeISBN(isbn)
	cached, _, err := cache.GetOrFetchWithTTL("bookbrainz_cache", normalizedISBN, func() (*cachedBookBrainzResult, error) {
		return e.fetchFromAPI(ctx, normalizedISBN)
	}, cache.SelectNegativeCacheTTL(func(r *cachedBookBrainzResult) bool {
		return r.NotFound
	}))
	if err != nil {
		return nil, err
	}

	if cached.NotFound {
		return nil, nil
	}

	return cached.Data, nil
}

// cachedBookBrainzResult wraps EnrichmentData with metadata for caching.
type cachedBookBrainzResult struct {
	Data     *book.EnrichmentData `json:"data"`
	NotFound bool                 `json:"not_found"`
}

// bbEditionSearchResponse matches the BookBrainz search API response.
type bbEditionSearchResponse struct {
	Results []bbEditionSearchResult `json:"results"`
	Total   int                     `json:"total"`
}

// bbEditionSearchResult represents a single edition result from the search API.
type bbEditionSearchResult struct {
	BBID          string           `json:"bbid"`
	Name          string           `json:"name"`
	Type          string           `json:"type"`
	Pages         int              `json:"pages"`
	DefaultAlias  *bbDefaultAlias  `json:"defaultAlias"`
	IdentifierSet *bbIdentifierSet `json:"identifierSet"`
}

type bbIdentifierSet struct {
	Identifiers []bbIdentifier `json:"identifiers"`
}

type bbIdentifier struct {
	TypeID int    `json:"typeId"`
	Value  string `json:"value"`
}

type bbDefaultAlias struct {
	Language string `json:"language"`
	Name     string `json:"name"`
	SortName string `json:"sortName"`
}

// bbEditionResponse matches the BookBrainz entity API edition response.
type bbEditionResponse struct {
	BBID             string           `json:"bbid"`
	DefaultAlias     *bbDefaultAlias  `json:"defaultAlias"`
	Pages            int              `json:"pages"`
	Languages        []string         `json:"languages"`
	Publishers       []bbPublisher    `json:"publishers"`
	ReleaseEventDate string           `json:"releaseEventDate"`
	AuthorCredits    *bbAuthorCredits `json:"authorCredits"`
}

type bbPublisher struct {
	BBID     string `json:"bbid"`
	Name     string `json:"name"`
	SortName string `json:"sortName"`
}

type bbAuthorCredits struct {
	Names []bbAuthorCreditName `json:"names"`
}

type bbAuthorCreditName struct {
	Name string `json:"name"`
}

func (e *BookBrainzEnricher) fetchFromAPI(ctx context.Context, isbn string) (*cachedBookBrainzResult, error) {
	limiter := e.getRateLimiter()
	if err := limiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limit wait: %w", err)
	}

	client := e.getHTTPClient()
	searchURL := fmt.Sprintf("%s/search/search?q=%s&type=edition", bookBrainzBaseURL, url.QueryEscape(isbn))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, searchURL, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("creating search request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("search API request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("search API returned status %d", resp.StatusCode)
	}

	var searchResp bbEditionSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&searchResp); err != nil {
		return nil, fmt.Errorf("decoding search response: %w", err)
	}

	if searchResp.Total == 0 || len(searchResp.Results) == 0 {
		return &cachedBookBrainzResult{NotFound: true}, nil
	}

	edition := findBestEditionMatch(searchResp.Results, isbn)
	if edition == nil {
		return &cachedBookBrainzResult{NotFound: true}, nil
	}

	details, err := e.fetchEditionDetails(ctx, client, edition.BBID)
	if err != nil {
		return nil, err
	}

	data := extractEditionEnrichmentData(edition, details)
	return &cachedBookBrainzResult{Data: data}, nil
}

func (e *BookBrainzEnricher) fetchEditionDetails(ctx context.Context, client *http.Client, bbid string) (*bbEditionResponse, error) {
	limiter := e.getRateLimiter()
	if err := limiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limit wait: %w", err)
	}

	detailsURL := fmt.Sprintf("%s/edition/%s", bookBrainzAPIBaseURL, url.PathEscape(bbid))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, detailsURL, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("creating edition request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("edition API request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("edition API returned status %d", resp.StatusCode)
	}

	var details bbEditionResponse
	if err := json.NewDecoder(resp.Body).Decode(&details); err != nil {
		return nil, fmt.Errorf("decoding edition response: %w", err)
	}
	return &details, nil
}

// findBestEditionMatch selects the edition whose identifiers match the queried ISBN.
func findBestEditionMatch(results []bbEditionSearchResult, isbn string) *bbEditionSearchResult {
	normalizedISBN := parseutil.NormalizeISBN(isbn)
	for i := range results {
		r := &results[i]
		if r.Type != "Edition" {
			continue
		}
		if editionMatchesISBN(r, normalizedISBN) {
			return r
		}
	}
	return nil
}

func editionMatchesISBN(edition *bbEditionSearchResult, normalizedISBN string) bool {
	if edition.IdentifierSet == nil {
		return false
	}
	for _, ident := range edition.IdentifierSet.Identifiers {
		if parseutil.NormalizeISBN(ident.Value) == normalizedISBN {
			return true
		}
	}
	return false
}

// extractEditionEnrichmentData builds EnrichmentData from a BookBrainz edition.
func extractEditionEnrichmentData(searchResult *bbEditionSearchResult, details *bbEditionResponse) *book.EnrichmentData {
	data := &book.EnrichmentData{}

	if title := editionTitle(searchResult, details); title != "" {
		data.Title = &title
	}

	if details != nil {
		if details.Pages > 0 {
			data.NumberOfPages = &details.Pages
		}
		if len(details.Publishers) > 0 && details.Publishers[0].Name != "" {
			data.Publisher = &details.Publishers[0].Name
		}
		if details.ReleaseEventDate != "" {
			publishDate := normalizeBookBrainzDate(details.ReleaseEventDate)
			data.PublishDate = &publishDate
		}
		if len(details.Languages) > 0 && details.Languages[0] != "" {
			data.Language = &details.Languages[0]
		}
		if details.AuthorCredits != nil && len(details.AuthorCredits.Names) > 0 {
			authors := make([]string, 0, len(details.AuthorCredits.Names))
			for _, author := range details.AuthorCredits.Names {
				if author.Name != "" {
					authors = append(authors, author.Name)
				}
			}
			if len(authors) > 0 {
				data.Authors = authors
			}
		}
	}

	if data.NumberOfPages == nil && searchResult != nil && searchResult.Pages > 0 {
		data.NumberOfPages = &searchResult.Pages
	}

	return data
}

func editionTitle(searchResult *bbEditionSearchResult, details *bbEditionResponse) string {
	if details != nil && details.DefaultAlias != nil && details.DefaultAlias.Name != "" {
		return details.DefaultAlias.Name
	}
	if searchResult != nil {
		if searchResult.DefaultAlias != nil && searchResult.DefaultAlias.Name != "" {
			return searchResult.DefaultAlias.Name
		}
		return searchResult.Name
	}
	return ""
}

func normalizeBookBrainzDate(date string) string {
	date = strings.TrimPrefix(date, "+")
	parts := strings.SplitN(date, "-", 2)
	if len(parts) == 0 {
		return date
	}

	year := strings.TrimLeft(parts[0], "0")
	if year == "" {
		year = "0"
	}
	if len(year) < 4 {
		year = strings.Repeat("0", 4-len(year)) + year
	}
	if len(parts) == 1 {
		return year
	}
	return year + "-" + parts[1]
}
