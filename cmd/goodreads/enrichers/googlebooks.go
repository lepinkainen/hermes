package enrichers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/lepinkainen/hermes/internal/cache"
	"github.com/lepinkainen/hermes/internal/enrichment/book"
	"github.com/lepinkainen/hermes/internal/ratelimit"
)

const (
	googleBooksBaseURL  = "https://www.googleapis.com/books/v1"
	googleBooksPriority = 2
)

// GoogleBooksEnricher implements the book.Enricher interface for Google Books API.
type GoogleBooksEnricher struct {
	httpClient  *http.Client
	rateLimiter *ratelimit.Limiter
	clientOnce  sync.Once
	limiterOnce sync.Once
}

// Compile-time check that GoogleBooksEnricher implements book.Enricher.
var _ book.Enricher = (*GoogleBooksEnricher)(nil)

// NewGoogleBooksEnricher creates a new Google Books enricher.
func NewGoogleBooksEnricher() *GoogleBooksEnricher {
	return &GoogleBooksEnricher{}
}

// Name returns the human-readable name of this enricher.
func (e *GoogleBooksEnricher) Name() string {
	return "Google Books"
}

// Priority returns the priority for merging data (lower = higher precedence).
func (e *GoogleBooksEnricher) Priority() int {
	return googleBooksPriority
}

// Ping tests the connection to Google Books API.
func (e *GoogleBooksEnricher) Ping(ctx context.Context) error {
	// Use a simple search that should always return results
	url := fmt.Sprintf("%s/volumes?q=isbn:0140447938&maxResults=1", googleBooksBaseURL)

	client := e.getHTTPClient()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("creating ping request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("google books ping failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("google books returned status %d", resp.StatusCode)
	}

	return nil
}

// Enrich fetches book data from Google Books API by ISBN.
func (e *GoogleBooksEnricher) Enrich(ctx context.Context, isbn string) (*book.EnrichmentData, error) {
	if isbn == "" {
		return nil, book.ErrInvalidISBN
	}

	// Normalize ISBN
	normalizedISBN := normalizeISBN(isbn)

	// Use cached fetch
	cached, _, err := cache.GetOrFetchWithTTL("googlebooks_cache", normalizedISBN, func() (*cachedGoogleBooksResult, error) {
		return e.fetchFromAPI(ctx, normalizedISBN)
	}, cache.SelectNegativeCacheTTL(func(r *cachedGoogleBooksResult) bool {
		return r.NotFound
	}))

	if err != nil {
		return nil, err
	}

	if cached.NotFound {
		return nil, nil // Not found allows other enrichers to try
	}

	return cached.Data, nil
}

// cachedGoogleBooksResult wraps EnrichmentData with metadata for caching.
type cachedGoogleBooksResult struct {
	Data     *book.EnrichmentData `json:"data"`
	NotFound bool                 `json:"not_found"`
}

// googleBooksResponse matches the Google Books API response structure.
type googleBooksResponse struct {
	TotalItems int `json:"totalItems"`
	Items      []struct {
		VolumeInfo struct {
			Title         string   `json:"title"`
			Subtitle      string   `json:"subtitle"`
			Authors       []string `json:"authors"`
			Publisher     string   `json:"publisher"`
			PublishedDate string   `json:"publishedDate"`
			Description   string   `json:"description"`
			PageCount     int      `json:"pageCount"`
			Categories    []string `json:"categories"`
			Language      string   `json:"language"`
			ImageLinks    struct {
				Thumbnail      string `json:"thumbnail"`
				SmallThumbnail string `json:"smallThumbnail"`
			} `json:"imageLinks"`
		} `json:"volumeInfo"`
	} `json:"items"`
}

func (e *GoogleBooksEnricher) fetchFromAPI(ctx context.Context, isbn string) (*cachedGoogleBooksResult, error) {
	limiter := e.getRateLimiter()
	if err := limiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limit wait: %w", err)
	}

	client := e.getHTTPClient()

	// Build URL with API key if available
	url := fmt.Sprintf("%s/volumes?q=isbn:%s", googleBooksBaseURL, isbn)
	if apiKey := os.Getenv("GOOGLE_BOOKS_API_KEY"); apiKey != "" {
		url = fmt.Sprintf("%s&key=%s", url, apiKey)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("API request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var result googleBooksResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	if result.TotalItems == 0 || len(result.Items) == 0 {
		// Book not found
		return &cachedGoogleBooksResult{NotFound: true}, nil
	}

	// Use first item (best match)
	vol := result.Items[0].VolumeInfo

	// Build enrichment data
	data := &book.EnrichmentData{}

	if vol.Title != "" {
		data.Title = &vol.Title
	}
	if vol.Subtitle != "" {
		data.Subtitle = &vol.Subtitle
	}
	if vol.Description != "" {
		data.Description = &vol.Description
	}
	if vol.Publisher != "" {
		data.Publisher = &vol.Publisher
	}
	if vol.PageCount > 0 {
		data.NumberOfPages = &vol.PageCount
	}
	if vol.PublishedDate != "" {
		data.PublishDate = &vol.PublishedDate
	}
	if vol.Language != "" {
		data.Language = &vol.Language
	}

	// Prefer larger thumbnail
	coverURL := vol.ImageLinks.Thumbnail
	if coverURL == "" {
		coverURL = vol.ImageLinks.SmallThumbnail
	}
	if coverURL != "" {
		// Remove zoom parameter for higher quality
		coverURL = strings.Replace(coverURL, "zoom=1", "zoom=0", 1)
		data.CoverURL = &coverURL
	}

	if len(vol.Authors) > 0 {
		data.Authors = vol.Authors
	}

	if len(vol.Categories) > 0 {
		data.Subjects = vol.Categories
	}

	return &cachedGoogleBooksResult{Data: data}, nil
}

func (e *GoogleBooksEnricher) getHTTPClient() *http.Client {
	e.clientOnce.Do(func() {
		e.httpClient = &http.Client{Timeout: 10 * time.Second}
	})
	return e.httpClient
}

func (e *GoogleBooksEnricher) getRateLimiter() *ratelimit.Limiter {
	e.limiterOnce.Do(func() {
		e.rateLimiter = ratelimit.New("GoogleBooks", 1)
	})
	return e.rateLimiter
}

// normalizeISBN strips hyphens and spaces from ISBN.
func normalizeISBN(isbn string) string {
	normalized := strings.ReplaceAll(isbn, "-", "")
	normalized = strings.ReplaceAll(normalized, " ", "")
	return normalized
}
