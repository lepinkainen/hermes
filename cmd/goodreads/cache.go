package goodreads

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
)

func getCachedBook(isbn string) (*Book, *OpenLibraryBook, bool, error) {
	cacheDir := "cache/goodreads"
	cachePath := filepath.Join(cacheDir, isbn+".json")

	// Check cache first
	if data, err := os.ReadFile(cachePath); err == nil {
		var olBook OpenLibraryBook
		if err := json.Unmarshal(data, &olBook); err == nil {
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

			return book, &olBook, true, nil
		}
	}

	// Fetch from API if not in cache
	book, olBook, err := fetchBookData(isbn)
	if err != nil {
		return nil, nil, false, err
	}

	// Cache the result
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		// Log error but continue - caching failure shouldn't stop the process
		fmt.Printf("Warning: Failed to create cache directory: %v\n", err)
	} else {
		data, err := json.MarshalIndent(olBook, "", "  ")
		if err != nil {
			fmt.Printf("Warning: Failed to marshal book data: %v\n", err)
		} else {
			if err := os.WriteFile(cachePath, data, 0644); err != nil {
				fmt.Printf("Warning: Failed to write cache file: %v\n", err)
			}
		}
	}

	return book, olBook, false, nil
}
