package goodreads

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"
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

func fetchBookData(isbn string) (*Book, *OpenLibraryBook, error) {
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	url := fmt.Sprintf("https://openlibrary.org/api/books?bibkeys=ISBN:%s&format=json&jscmd=data", isbn)
	resp, err := client.Get(url)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()

	var result map[string]OpenLibraryBook
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, nil, err
	}

	if len(result) == 0 {
		return nil, nil, fmt.Errorf("no data found in OpenLibrary for ISBN: %s", isbn)
	}

	olBook := result["ISBN:"+isbn]
	book := &Book{
		Title:    olBook.Title,
		ISBN:     isbn,
		Subtitle: olBook.Subtitle,
	}

	return book, &olBook, nil
}
