package goodreads

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// Global HTTP client for reuse
var (
	httpClient *http.Client
	clientOnce sync.Once
)

// getHTTPClient returns a singleton HTTP client
func getHTTPClient() *http.Client {
	clientOnce.Do(func() {
		httpClient = &http.Client{
			Timeout: 10 * time.Second,
		}
	})
	return httpClient
}

// fetchBookData retrieves book data from OpenLibrary API using ISBN
func fetchBookData(isbn string) (*Book, *OpenLibraryBook, error) {
	client := getHTTPClient()

	// Use jscmd=data for more comprehensive data
	url := fmt.Sprintf("https://openlibrary.org/api/books?bibkeys=ISBN:%s&format=json&jscmd=data", isbn)
	resp, err := client.Get(url)
	if err != nil {
		return nil, nil, fmt.Errorf("OpenLibrary API request failed: %w", err)
	}
	defer resp.Body.Close()

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
	editionData, err := fetchEditionData(isbn)
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
	client := getHTTPClient()

	// Use the books endpoint for edition-specific data
	url := fmt.Sprintf("https://openlibrary.org/isbn/%s.json", isbn)
	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("edition data request failed: %w", err)
	}
	defer resp.Body.Close()

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

