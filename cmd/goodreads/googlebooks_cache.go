package goodreads

import (
	"github.com/lepinkainen/hermes/internal/cache"
)

// getCachedGoogleBook fetches book data from Google Books with caching
func getCachedGoogleBook(isbn string) (*GoogleBooksBook, bool, error) {
	googleBook, fromCache, err := cache.GetOrFetch("googlebooks_cache", isbn,
		func() (*GoogleBooksBook, error) {
			book, fetchErr := fetchBookDataFromGoogleBooks(isbn)
			return book, fetchErr
		})

	if err != nil {
		return nil, false, err
	}

	return googleBook, fromCache, nil
}
