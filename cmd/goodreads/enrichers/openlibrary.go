package enrichers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/lepinkainen/hermes/internal/cache"
	"github.com/lepinkainen/hermes/internal/enrichment/book"
	"github.com/lepinkainen/hermes/internal/ratelimit"
)

const (
	openLibraryBaseURL  = "https://openlibrary.org"
	openLibraryPriority = 1
)

// OpenLibraryEnricher implements the book.Enricher interface for OpenLibrary.
type OpenLibraryEnricher struct {
	httpClient  *http.Client
	rateLimiter *ratelimit.Limiter
	clientOnce  sync.Once
	limiterOnce sync.Once
}

// Compile-time check that OpenLibraryEnricher implements book.Enricher.
var _ book.Enricher = (*OpenLibraryEnricher)(nil)

// NewOpenLibraryEnricher creates a new OpenLibrary enricher.
func NewOpenLibraryEnricher() *OpenLibraryEnricher {
	return &OpenLibraryEnricher{}
}

// Name returns the human-readable name of this enricher.
func (e *OpenLibraryEnricher) Name() string {
	return "OpenLibrary"
}

// Priority returns the priority for merging data (lower = higher precedence).
func (e *OpenLibraryEnricher) Priority() int {
	return openLibraryPriority
}

// Ping tests the connection to OpenLibrary.
func (e *OpenLibraryEnricher) Ping(ctx context.Context) error {
	client := e.getHTTPClient()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, openLibraryBaseURL, nil)
	if err != nil {
		return fmt.Errorf("creating ping request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("OpenLibrary ping failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("OpenLibrary returned status %d", resp.StatusCode)
	}

	return nil
}

// Enrich fetches book data from OpenLibrary by ISBN.
func (e *OpenLibraryEnricher) Enrich(ctx context.Context, isbn string) (*book.EnrichmentData, error) {
	if isbn == "" {
		return nil, book.ErrInvalidISBN
	}

	// Use cached fetch
	cached, _, err := cache.GetOrFetchWithTTL("openlibrary_cache", isbn, func() (*cachedOpenLibraryResult, error) {
		return e.fetchFromAPI(ctx, isbn)
	}, cache.SelectNegativeCacheTTL(func(r *cachedOpenLibraryResult) bool {
		return r.NotFound
	}))

	if err != nil {
		return nil, err
	}

	if cached.NotFound {
		return nil, nil // Not found is not an error, allows other enrichers to try
	}

	return cached.Data, nil
}

// cachedOpenLibraryResult wraps EnrichmentData with metadata for caching.
type cachedOpenLibraryResult struct {
	Data     *book.EnrichmentData `json:"data"`
	NotFound bool                 `json:"not_found"`
}

// openLibraryBookResponse matches the API response structure.
type openLibraryBookResponse struct {
	Title       string `json:"title"`
	Subtitle    string `json:"subtitle"`
	Description any    `json:"description"`
	Publishers  []struct {
		Name string `json:"name"`
	} `json:"publishers"`
	Authors []struct {
		Name string `json:"name"`
	} `json:"authors"`
	Cover struct {
		Large string `json:"large"`
	} `json:"cover"`
	Subjects      []any  `json:"subjects"`
	SubjectPeople []any  `json:"subject_people"`
	NumberOfPages int    `json:"number_of_pages"`
	PublishDate   string `json:"publish_date"`
}

// openLibraryEditionResponse matches the edition API response.
type openLibraryEditionResponse struct {
	NumberOfPages int      `json:"number_of_pages"`
	Publishers    []string `json:"publishers"`
	Languages     []struct {
		Key string `json:"key"`
	} `json:"languages"`
	Subjects []string `json:"subjects"`
}

func (e *OpenLibraryEnricher) fetchFromAPI(ctx context.Context, isbn string) (*cachedOpenLibraryResult, error) {
	limiter := e.getRateLimiter()
	if err := limiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limit wait: %w", err)
	}

	client := e.getHTTPClient()

	// Fetch book data
	url := fmt.Sprintf("%s/api/books?bibkeys=ISBN:%s&format=json&jscmd=data", openLibraryBaseURL, isbn)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("API request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	var result map[string]openLibraryBookResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	if len(result) == 0 {
		// Book not found - cache this result with shorter TTL
		return &cachedOpenLibraryResult{NotFound: true}, nil
	}

	olBook := result["ISBN:"+isbn]

	// Build enrichment data
	data := &book.EnrichmentData{}

	if olBook.Title != "" {
		data.Title = &olBook.Title
	}
	if olBook.Subtitle != "" {
		data.Subtitle = &olBook.Subtitle
	}

	// Handle description (can be string or object with "value" key)
	if desc := extractDescription(olBook.Description); desc != "" {
		data.Description = &desc
	}

	if len(olBook.Publishers) > 0 {
		data.Publisher = &olBook.Publishers[0].Name
	}

	if olBook.NumberOfPages > 0 {
		data.NumberOfPages = &olBook.NumberOfPages
	}

	if olBook.Cover.Large != "" {
		data.CoverURL = &olBook.Cover.Large
	}

	if olBook.PublishDate != "" {
		data.PublishDate = &olBook.PublishDate
	}

	// Extract authors
	if len(olBook.Authors) > 0 {
		authors := make([]string, 0, len(olBook.Authors))
		for _, author := range olBook.Authors {
			if author.Name != "" {
				authors = append(authors, author.Name)
			}
		}
		if len(authors) > 0 {
			data.Authors = authors
		}
	}

	// Extract subjects
	data.Subjects = extractStringSlice(olBook.Subjects)
	data.SubjectPeople = extractStringSlice(olBook.SubjectPeople)

	// Try to get additional edition data
	editionData, err := e.fetchEditionData(ctx, isbn)
	if err == nil && editionData != nil {
		// Fill in missing data from edition
		if data.NumberOfPages == nil && editionData.NumberOfPages > 0 {
			data.NumberOfPages = &editionData.NumberOfPages
		}
		if data.Publisher == nil && len(editionData.Publishers) > 0 {
			data.Publisher = &editionData.Publishers[0]
		}
		if len(editionData.Languages) > 0 {
			// Extract language code from key like "/languages/eng"
			langKey := editionData.Languages[0].Key
			if parts := strings.Split(langKey, "/"); len(parts) > 0 {
				lang := parts[len(parts)-1]
				data.Language = &lang
			}
		}
		if len(data.Subjects) == 0 && len(editionData.Subjects) > 0 {
			data.Subjects = editionData.Subjects
		}
	}

	return &cachedOpenLibraryResult{Data: data}, nil
}

func (e *OpenLibraryEnricher) fetchEditionData(ctx context.Context, isbn string) (*openLibraryEditionResponse, error) {
	limiter := e.getRateLimiter()
	if err := limiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limit wait: %w", err)
	}

	client := e.getHTTPClient()
	url := fmt.Sprintf("%s/isbn/%s.json", openLibraryBaseURL, isbn)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("edition request returned status %d", resp.StatusCode)
	}

	var edition openLibraryEditionResponse
	if err := json.NewDecoder(resp.Body).Decode(&edition); err != nil {
		return nil, err
	}

	return &edition, nil
}

func (e *OpenLibraryEnricher) getHTTPClient() *http.Client {
	e.clientOnce.Do(func() {
		e.httpClient = &http.Client{Timeout: 10 * time.Second}
	})
	return e.httpClient
}

func (e *OpenLibraryEnricher) getRateLimiter() *ratelimit.Limiter {
	e.limiterOnce.Do(func() {
		e.rateLimiter = ratelimit.New("OpenLibrary", 1)
	})
	return e.rateLimiter
}

// extractDescription handles the various forms description can take.
func extractDescription(desc any) string {
	if desc == nil {
		return ""
	}
	switch v := desc.(type) {
	case string:
		return v
	case map[string]any:
		if val, ok := v["value"].(string); ok {
			return val
		}
	}
	return ""
}

// extractStringSlice converts []any to []string, handling various element types.
func extractStringSlice(items []any) []string {
	if len(items) == 0 {
		return nil
	}
	result := make([]string, 0, len(items))
	for _, item := range items {
		switch v := item.(type) {
		case string:
			result = append(result, v)
		case map[string]any:
			if name, ok := v["name"].(string); ok {
				result = append(result, name)
			}
		}
	}
	return result
}
