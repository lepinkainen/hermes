package goodreads

import (
	"regexp"
	"strconv"

	"github.com/lepinkainen/hermes/internal/cache"
)

func getCachedBook(isbn string) (*Book, *OpenLibraryBook, bool, error) {
	cacheDir := "cache/goodreads"

	// Use the generic cache utility
	olBook, fromCache, err := cache.GetOrFetch(cacheDir, isbn, func() (*OpenLibraryBook, error) {
		_, olBookData, err := fetchBookData(isbn)
		return olBookData, err
	})
	if err != nil {
		return nil, nil, false, err
	}

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
