package goodreads

import (
	"fmt"
	"time"
)

// Helper functions for OpenLibrary API interactions
func enrichBookFromOpenLibrary(book *Book) error {
	// Try ISBN13 first, then ISBN
	searchISBN := book.ISBN13
	if searchISBN == "" {
		searchISBN = book.ISBN
	}

	if searchISBN == "" {
		return fmt.Errorf("no ISBN available")
	}

	// Try cache first
	_, olBook, cache_hit, err := getCachedBook(searchISBN)
	if err == nil {
		if desc := getDescription(olBook.Description); desc != "" {
			book.Description = desc
		}
		if len(olBook.Covers) > 0 {
			book.CoverID = olBook.Covers[0]
		}
		if olBook.Cover.Large != "" {
			book.CoverURL = olBook.Cover.Large
		}
		if olBook.Subtitle != "" {
			book.Subtitle = olBook.Subtitle
		}
		book.Subjects = getSubjects(olBook.Subjects)
		book.SubjectPeople = getSubjectPeople(olBook.SubjectPeople)

		if !cache_hit {
			// Add a small delay only when we had to make an API call
			time.Sleep(100 * time.Millisecond)
		}
		return nil
	}

	return fmt.Errorf("failed to get book data: %v", err)
}
