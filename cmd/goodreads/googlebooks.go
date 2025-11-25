package goodreads

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

// Package-level variables for Google Books API client
// These can be overridden in tests for dependency injection
var (
	googleBooksHTTPClient    *http.Client
	googleBooksClientOnce    sync.Once
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

// normalizeISBN strips hyphens and spaces from ISBN
func normalizeISBN(isbn string) string {
	normalized := strings.ReplaceAll(isbn, "-", "")
	normalized = strings.ReplaceAll(normalized, " ", "")
	return normalized
}

// fetchBookDataFromGoogleBooks fetches book data from Google Books API by ISBN
func fetchBookDataFromGoogleBooks(isbn string) (*GoogleBooksBook, error) {
	if isbn == "" {
		return nil, fmt.Errorf("ISBN is required")
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
	resp, err := client.Get(url)
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
