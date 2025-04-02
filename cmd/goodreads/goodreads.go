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
		// Extract basic information
		if desc := getDescription(olBook.Description); desc != "" {
			book.Description = desc
		}
		if len(olBook.Covers) > 0 {
			book.CoverID = olBook.Covers[0]

			// Generate cover URL if we have a cover ID
			if coverURL, err := fetchCoverImage(olBook.Covers[0]); err == nil && coverURL != "" {
				book.CoverURL = coverURL
			}
		}
		if olBook.Cover.Large != "" {
			book.CoverURL = olBook.Cover.Large
		}
		if olBook.Subtitle != "" {
			book.Subtitle = olBook.Subtitle
		}

		// Extract publisher information
		if len(olBook.Publishers) > 0 && book.Publisher == "" {
			book.Publisher = olBook.Publishers[0].Name
		}

		// Extract subjects and subject people
		book.Subjects = getSubjects(olBook.Subjects)
		book.SubjectPeople = getSubjectPeople(olBook.SubjectPeople)

		// Try to get additional edition data if we don't have page count
		if book.NumberOfPages == 0 {
			if editionData, err := fetchEditionData(searchISBN); err == nil && editionData != nil {
				if editionData.Number_of_pages > 0 {
					book.NumberOfPages = editionData.Number_of_pages
				}

				// Use publisher from edition data if not already set
				if len(editionData.Publishers) > 0 && book.Publisher == "" {
					book.Publisher = editionData.Publishers[0]
				}
			}
		}

		if !cache_hit {
			// Add a small delay only when we had to make an API call
			time.Sleep(100 * time.Millisecond)
		}
		return nil
	}

	return fmt.Errorf("failed to get book data: %v", err)
}
