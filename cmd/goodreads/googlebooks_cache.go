package goodreads

import (
	"fmt"
	"strings"
	"time"

	"github.com/lepinkainen/hermes/internal/cache"
)

// getCachedGoogleBook fetches book data from Google Books with caching and negative caching support
func getCachedGoogleBook(isbn string) (*GoogleBooksBook, bool, error) {
	cached, fromCache, err := cache.GetOrFetchWithTTL("googlebooks_cache", isbn,
		func() (*CachedGoogleBooksBook, error) {
			book, fetchErr := fetchBookDataFromGoogleBooks(isbn)
			if fetchErr != nil {
				// Check if this is a "not found" error
				if strings.Contains(fetchErr.Error(), "no data found in Google Books") {
					// Cache the "not found" response
					return &CachedGoogleBooksBook{
						Book:     nil,
						NotFound: true,
					}, nil
				}
				// Other errors (network, etc.) should not be cached
				return nil, fetchErr
			}
			// Cache the successful response
			return &CachedGoogleBooksBook{
				Book:     book,
				NotFound: false,
			}, nil
		}, func(result *CachedGoogleBooksBook) time.Duration {
			// Use shorter TTL for "not found" responses
			if result.NotFound {
				return negativeCacheTTL // 7 days
			}
			return normalCacheTTL // 30 days
		})

	if err != nil {
		return nil, false, err
	}

	// If it's a "not found" cache entry, return the appropriate error
	if cached.NotFound {
		return nil, fromCache, fmt.Errorf("no data found in Google Books for ISBN: %s", isbn)
	}

	return cached.Book, fromCache, nil
}
