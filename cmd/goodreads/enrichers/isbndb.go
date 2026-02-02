package enrichers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/lepinkainen/hermes/internal/cache"
	"github.com/lepinkainen/hermes/internal/enrichment/book"
	"github.com/lepinkainen/hermes/internal/ratelimit"
	"github.com/spf13/viper"
)

const (
	isbndbBaseURL  = "https://api2.isbndb.com"
	isbndbPriority = 0 // Highest priority - most comprehensive data
)

// ISBNdbEnricher implements the book.Enricher interface for ISBNdb API.
type ISBNdbEnricher struct {
	httpClient  *http.Client
	rateLimiter *ratelimit.Limiter
	clientOnce  sync.Once
	limiterOnce sync.Once
}

// Compile-time check that ISBNdbEnricher implements book.Enricher.
var _ book.Enricher = (*ISBNdbEnricher)(nil)

// NewISBNdbEnricher creates a new ISBNdb enricher.
func NewISBNdbEnricher() *ISBNdbEnricher {
	return &ISBNdbEnricher{}
}

// Name returns the human-readable name of this enricher.
func (e *ISBNdbEnricher) Name() string {
	return "ISBNdb"
}

// Priority returns the priority for merging data (lower = higher precedence).
func (e *ISBNdbEnricher) Priority() int {
	return isbndbPriority
}

// Ping tests the connection to ISBNdb API.
func (e *ISBNdbEnricher) Ping(ctx context.Context) error {
	apiKey := e.getAPIKey()
	if apiKey == "" {
		return fmt.Errorf("ISBNdb API key not configured")
	}

	// Test with a well-known ISBN
	url := fmt.Sprintf("%s/book/9780140447934", isbndbBaseURL)

	client := e.getHTTPClient()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("creating ping request: %w", err)
	}
	req.Header.Set("Authorization", apiKey)

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("ISBNdb ping failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusUnauthorized {
		return fmt.Errorf("ISBNdb API key invalid")
	}
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNotFound {
		return fmt.Errorf("ISBNdb returned status %d", resp.StatusCode)
	}

	return nil
}

// Enrich fetches book data from ISBNdb API by ISBN.
func (e *ISBNdbEnricher) Enrich(ctx context.Context, isbn string) (*book.EnrichmentData, error) {
	if isbn == "" {
		return nil, book.ErrInvalidISBN
	}

	apiKey := e.getAPIKey()
	if apiKey == "" {
		// No API key - skip this enricher silently
		return nil, nil
	}

	// Normalize ISBN
	normalizedISBN := normalizeISBN(isbn)

	// Use cached fetch
	cached, _, err := cache.GetOrFetchWithTTL("isbndb_cache", normalizedISBN, func() (*cachedISBNdbResult, error) {
		return e.fetchFromAPI(ctx, normalizedISBN, apiKey)
	}, cache.SelectNegativeCacheTTL(func(r *cachedISBNdbResult) bool {
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

// cachedISBNdbResult wraps EnrichmentData with metadata for caching.
type cachedISBNdbResult struct {
	Data     *book.EnrichmentData `json:"data"`
	NotFound bool                 `json:"not_found"`
}

// isbndbBookResponse matches the ISBNdb API response structure.
type isbndbBookResponse struct {
	Book struct {
		Title         string   `json:"title"`
		ISBN          string   `json:"isbn"`
		ISBN13        string   `json:"isbn13"`
		Publisher     string   `json:"publisher"`
		Language      string   `json:"language"`
		DatePublished string   `json:"date_published"`
		Binding       string   `json:"binding"`
		Pages         *int     `json:"pages"`
		Overview      string   `json:"overview"`
		Synopsis      string   `json:"synopsis"`
		Excerpt       string   `json:"excerpt"`
		ImageOriginal string   `json:"image_original"`
		Authors       []string `json:"authors"`
		Subjects      []string `json:"subjects"`
	} `json:"book"`
}

func (e *ISBNdbEnricher) fetchFromAPI(ctx context.Context, isbn, apiKey string) (*cachedISBNdbResult, error) {
	limiter := e.getRateLimiter()
	if err := limiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limit wait: %w", err)
	}

	client := e.getHTTPClient()
	url := fmt.Sprintf("%s/book/%s", isbndbBaseURL, isbn)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Authorization", apiKey)

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("API request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusNotFound {
		// Book not found
		return &cachedISBNdbResult{NotFound: true}, nil
	}

	if resp.StatusCode == http.StatusUnauthorized {
		return nil, fmt.Errorf("ISBNdb API key invalid or expired")
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var result isbndbBookResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	// Check if book is empty
	if result.Book.Title == "" && result.Book.ISBN == "" && result.Book.ISBN13 == "" {
		return &cachedISBNdbResult{NotFound: true}, nil
	}

	// Build enrichment data
	data := &book.EnrichmentData{}

	if result.Book.Title != "" {
		data.Title = &result.Book.Title
	}
	if result.Book.Publisher != "" {
		data.Publisher = &result.Book.Publisher
	}
	if result.Book.Language != "" {
		data.Language = &result.Book.Language
	}
	if result.Book.DatePublished != "" {
		data.PublishDate = &result.Book.DatePublished
	}
	if result.Book.Pages != nil && *result.Book.Pages > 0 {
		data.NumberOfPages = result.Book.Pages
	}
	if result.Book.ImageOriginal != "" {
		data.CoverURL = &result.Book.ImageOriginal
	}

	// Use synopsis for description if available, otherwise use overview
	if result.Book.Synopsis != "" {
		data.Description = &result.Book.Synopsis
	} else if result.Book.Overview != "" {
		data.Description = &result.Book.Overview
	}

	if len(result.Book.Authors) > 0 {
		data.Authors = result.Book.Authors
	}

	// Filter out generic "Subjects" entry
	if len(result.Book.Subjects) > 0 {
		subjects := make([]string, 0, len(result.Book.Subjects))
		for _, s := range result.Book.Subjects {
			if s != "" && s != "Subjects" {
				subjects = append(subjects, s)
			}
		}
		if len(subjects) > 0 {
			data.Subjects = subjects
		}
	}

	return &cachedISBNdbResult{Data: data}, nil
}

func (e *ISBNdbEnricher) getHTTPClient() *http.Client {
	e.clientOnce.Do(func() {
		e.httpClient = &http.Client{Timeout: 10 * time.Second}
	})
	return e.httpClient
}

func (e *ISBNdbEnricher) getRateLimiter() *ratelimit.Limiter {
	e.limiterOnce.Do(func() {
		// Free tier: 1 request per second
		e.rateLimiter = ratelimit.New("ISBNdb", 1)
	})
	return e.rateLimiter
}

func (e *ISBNdbEnricher) getAPIKey() string {
	// Try multiple config keys
	apiKey := viper.GetString("isbndb.api_key")
	if apiKey == "" {
		apiKey = viper.GetString("goodreads.isbndb_api_key")
	}
	return apiKey
}
