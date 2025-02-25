package goodreads

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

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
