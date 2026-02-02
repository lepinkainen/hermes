package goodreads

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/lepinkainen/hermes/internal/ratelimit"
)

// Global HTTP client and rate limiter for reuse
var (
	httpClient      *http.Client
	clientOnce      sync.Once
	olRateLimiter   *ratelimit.Limiter
	rateLimiterOnce sync.Once
	httpClientNew   = func() *http.Client {
		return &http.Client{
			Timeout: 10 * time.Second,
		}
	}
)

var openLibraryBaseURL = "https://openlibrary.org"

// getHTTPClient returns a singleton HTTP client
func getHTTPClient() *http.Client {
	clientOnce.Do(func() {
		httpClient = httpClientNew()
	})
	return httpClient
}

// getOLRateLimiter returns a singleton rate limiter for OpenLibrary (1 req/sec)
func getOLRateLimiter() *ratelimit.Limiter {
	rateLimiterOnce.Do(func() {
		olRateLimiter = ratelimit.New("OpenLibrary", 1)
	})
	return olRateLimiter
}

// fetchBookData retrieves book data from OpenLibrary API using ISBN
func fetchBookData(isbn string) (*Book, *OpenLibraryBook, error) {
	return fetchBookDataWithContext(context.Background(), isbn)
}

// fetchBookDataWithContext retrieves book data from OpenLibrary API using ISBN with context support
func fetchBookDataWithContext(ctx context.Context, isbn string) (*Book, *OpenLibraryBook, error) {
	client := getHTTPClient()
	limiter := getOLRateLimiter()

	// Wait for rate limiter
	if err := limiter.Wait(ctx); err != nil {
		return nil, nil, fmt.Errorf("rate limit wait failed: %w", err)
	}

	// Use jscmd=data for more comprehensive data
	url := fmt.Sprintf("%s/api/books?bibkeys=ISBN:%s&format=json&jscmd=data", openLibraryBaseURL, isbn)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("OpenLibrary API request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	var result map[string]OpenLibraryBook
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, nil, fmt.Errorf("failed to decode OpenLibrary response: %w", err)
	}

	if len(result) == 0 {
		return nil, nil, fmt.Errorf("no data found in OpenLibrary for ISBN: %s", isbn)
	}

	olBook := result["ISBN:"+isbn]

	// Create a more enriched Book object
	book := &Book{
		Title:    olBook.Title,
		ISBN:     isbn,
		Subtitle: olBook.Subtitle,
	}

	// Extract additional data
	if len(olBook.Publishers) > 0 {
		book.Publisher = olBook.Publishers[0].Name
	}

	// Try to get additional edition data
	editionData, err := fetchEditionDataWithContext(ctx, isbn)
	if err == nil && editionData != nil {
		// Enrich with edition data
		if editionData.Number_of_pages > 0 {
			book.NumberOfPages = editionData.Number_of_pages
		}

		if len(editionData.Publishers) > 0 && book.Publisher == "" {
			book.Publisher = editionData.Publishers[0]
		}
	}

	return book, &olBook, nil
}

// fetchEditionData retrieves additional edition data from OpenLibrary
func fetchEditionData(isbn string) (*OpenLibraryEdition, error) {
	return fetchEditionDataWithContext(context.Background(), isbn)
}

// fetchEditionDataWithContext retrieves additional edition data from OpenLibrary with context support
func fetchEditionDataWithContext(ctx context.Context, isbn string) (*OpenLibraryEdition, error) {
	client := getHTTPClient()
	limiter := getOLRateLimiter()

	// Wait for rate limiter
	if err := limiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limit wait failed: %w", err)
	}

	// Use the books endpoint for edition-specific data
	url := fmt.Sprintf("%s/isbn/%s.json", openLibraryBaseURL, isbn)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("edition data request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Check if we got a successful response
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("edition data request returned status: %s", resp.Status)
	}

	var edition OpenLibraryEdition
	if err := json.NewDecoder(resp.Body).Decode(&edition); err != nil {
		return nil, fmt.Errorf("failed to decode edition data: %w", err)
	}

	return &edition, nil
}

// fetchCoverImage constructs cover image URLs from cover ID
func fetchCoverImage(coverID int) (string, error) {
	if coverID <= 0 {
		return "", fmt.Errorf("invalid cover ID: %d", coverID)
	}

	// OpenLibrary provides cover images in different sizes
	// We'll return the large size URL
	return fmt.Sprintf("https://covers.openlibrary.org/b/id/%d-L.jpg", coverID), nil
}
