package goodreads

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/lepinkainen/hermes/internal/ratelimit"
)

// Package-level variables for Google Books API client
// These can be overridden in tests for dependency injection
var (
	googleBooksHTTPClient    *http.Client
	googleBooksClientOnce    sync.Once
	googleBooksRateLimiter   *ratelimit.Limiter
	googleBooksLimiterOnce   sync.Once
	googleBooksHTTPClientNew = func() *http.Client {
		return &http.Client{Timeout: 10 * time.Second}
	}
	googleBooksBaseURL = "https://www.googleapis.com/books/v1"
)

// getGoogleBooksHTTPClient returns a singleton HTTP client for Google Books API
func getGoogleBooksHTTPClient() *http.Client {
	googleBooksClientOnce.Do(func() {
		googleBooksHTTPClient = googleBooksHTTPClientNew()
	})
	return googleBooksHTTPClient
}

// getGoogleBooksRateLimiter returns a singleton rate limiter for Google Books (1 req/sec)
func getGoogleBooksRateLimiter() *ratelimit.Limiter {
	googleBooksLimiterOnce.Do(func() {
		googleBooksRateLimiter = ratelimit.New("GoogleBooks", 1)
	})
	return googleBooksRateLimiter
}

// normalizeISBN strips hyphens and spaces from ISBN
func normalizeISBN(isbn string) string {
	normalized := strings.ReplaceAll(isbn, "-", "")
	normalized = strings.ReplaceAll(normalized, " ", "")
	return normalized
}

// fetchBookDataFromGoogleBooks fetches book data from Google Books API by ISBN
func fetchBookDataFromGoogleBooks(isbn string) (*GoogleBooksBook, error) {
	return fetchBookDataFromGoogleBooksWithContext(context.Background(), isbn)
}

// fetchBookDataFromGoogleBooksWithContext fetches book data from Google Books API by ISBN with context support
func fetchBookDataFromGoogleBooksWithContext(ctx context.Context, isbn string) (*GoogleBooksBook, error) {
	if isbn == "" {
		return nil, fmt.Errorf("ISBN is required")
	}

	// Wait for rate limiter
	limiter := getGoogleBooksRateLimiter()
	if err := limiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limit wait failed: %w", err)
	}

	// Normalize ISBN
	normalizedISBN := normalizeISBN(isbn)

	// Build API URL
	url := fmt.Sprintf("%s/volumes?q=isbn:%s", googleBooksBaseURL, normalizedISBN)

	// Add API key if available
	apiKey := os.Getenv("GOOGLE_BOOKS_API_KEY")
	if apiKey != "" {
		url = fmt.Sprintf("%s&key=%s", url, apiKey)
	}

	slog.Debug("Fetching book data from Google Books", "isbn", isbn, "normalized_isbn", normalizedISBN)

	client := getGoogleBooksHTTPClient()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("google Books API request failed for ISBN %s: %w", isbn, err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Check HTTP status code
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("google Books API returned non-200 status code: %d for ISBN: %s", resp.StatusCode, isbn)
	}

	// Decode JSON response
	var result GoogleBooksResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode Google Books response for ISBN %s: %w", isbn, err)
	}

	// Validate response has items
	if result.TotalItems == 0 || len(result.Items) == 0 {
		return nil, fmt.Errorf("no data found in Google Books for ISBN: %s", isbn)
	}

	// Return first item (best match)
	slog.Debug("Successfully fetched book from Google Books",
		"isbn", isbn,
		"title", result.Items[0].VolumeInfo.Title,
	)

	return &result.Items[0], nil
}
