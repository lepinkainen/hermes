package goodreads

import (
	"encoding/json"
	"os"
	"path/filepath"
)

func getCachedBook(isbn string) (*Book, *OpenLibraryBook, bool, error) {
	cacheDir := "cache/goodreads"
	cachePath := filepath.Join(cacheDir, isbn+".json")

	// Check cache first
	if data, err := os.ReadFile(cachePath); err == nil {
		var olBook OpenLibraryBook
		if err := json.Unmarshal(data, &olBook); err == nil {
			book := &Book{
				Title:    olBook.Title,
				ISBN:     isbn,
				Subtitle: olBook.Subtitle,
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
	os.MkdirAll(cacheDir, 0755)
	data, _ := json.MarshalIndent(olBook, "", "  ")
	os.WriteFile(cachePath, data, 0644)

	return book, olBook, false, nil
}
