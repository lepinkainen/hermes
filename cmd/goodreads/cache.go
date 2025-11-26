package goodreads

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/lepinkainen/hermes/internal/cache"
)

const (
	// Negative cache TTL for books not found in APIs (7 days)
	negativeCacheTTL = 168 * time.Hour
	// Normal cache TTL for found books (30 days)
	normalCacheTTL = 720 * time.Hour
)

func getCachedBook(isbn string) (*Book, *OpenLibraryBook, bool, error) {
	// Use negative caching with TTL selector
	cached, fromCache, err := cache.GetOrFetchWithTTL("openlibrary_cache", isbn, func() (*CachedOpenLibraryBook, error) {
		_, olBookData, err := fetchBookData(isbn)
		if err != nil {
			// Check if this is a "not found" error
			if strings.Contains(err.Error(), "no data found in OpenLibrary") {
				// Cache the "not found" response
				return &CachedOpenLibraryBook{
					Book:     nil,
					NotFound: true,
				}, nil
			}
			// Other errors (network, etc.) should not be cached
			return nil, err
		}
		// Cache the successful response
		return &CachedOpenLibraryBook{
			Book:     olBookData,
			NotFound: false,
		}, nil
	}, func(result *CachedOpenLibraryBook) time.Duration {
		// Use shorter TTL for "not found" responses
		if result.NotFound {
			return negativeCacheTTL // 7 days
		}
		return normalCacheTTL // 30 days
	})

	if err != nil {
		return nil, nil, false, err
	}

	// If it's a "not found" cache entry, return the appropriate error
	if cached.NotFound {
		return nil, nil, fromCache, fmt.Errorf("no data found in OpenLibrary for ISBN: %s", isbn)
	}

	olBook := cached.Book

	// Create a more enriched Book object from cached data
	book := &Book{
		Title:    olBook.Title,
		ISBN:     isbn,
		Subtitle: olBook.Subtitle,
	}

	// Extract additional data from cached OpenLibraryBook
	if olBook.NumberOfPages > 0 {
		book.NumberOfPages = olBook.NumberOfPages
	}

	if len(olBook.Publishers) > 0 {
		book.Publisher = olBook.Publishers[0].Name
	}

	// Extract publication year if available
	if olBook.PublishDate != "" {
		// Try to extract year from publish date (formats vary)
		for _, yearStr := range regexp.MustCompile(`\b\d{4}\b`).FindAllString(olBook.PublishDate, -1) {
			if year, err := strconv.Atoi(yearStr); err == nil {
				book.YearPublished = year
				break
			}
		}
	}

	// Extract author information if available
	if len(olBook.Authors) > 0 {
		authors := make([]string, 0, len(olBook.Authors))
		for _, author := range olBook.Authors {
			authors = append(authors, author.Name)
		}
		// Only set if we don't already have authors
		if len(book.Authors) == 0 {
			book.Authors = authors
		}
	}

	return book, olBook, fromCache, nil
}
