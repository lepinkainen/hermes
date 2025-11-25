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
		book.SubjectPeople = getSubjects(olBook.SubjectPeople)

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

// enrichBookFromGoogleBooks enriches book data from Google Books API
// Only fills in empty fields - never overwrites existing data
func enrichBookFromGoogleBooks(book *Book) error {
	// Try ISBN13 first, then ISBN
	searchISBN := book.ISBN13
	if searchISBN == "" {
		searchISBN = book.ISBN
	}

	if searchISBN == "" {
		return fmt.Errorf("no ISBN available")
	}

	// Try cache first
	googleBook, cacheHit, err := getCachedGoogleBook(searchISBN)
	if err != nil {
		return fmt.Errorf("failed to get Google Books data: %w", err)
	}

	// Only fill in empty fields - never overwrite existing data
	if book.Description == "" && googleBook.VolumeInfo.Description != "" {
		book.Description = googleBook.VolumeInfo.Description
	}

	if book.Subtitle == "" && googleBook.VolumeInfo.Subtitle != "" {
		book.Subtitle = googleBook.VolumeInfo.Subtitle
	}

	if book.Publisher == "" && googleBook.VolumeInfo.Publisher != "" {
		book.Publisher = googleBook.VolumeInfo.Publisher
	}

	if book.NumberOfPages == 0 && googleBook.VolumeInfo.PageCount > 0 {
		book.NumberOfPages = googleBook.VolumeInfo.PageCount
	}

	if book.CoverURL == "" && googleBook.VolumeInfo.ImageLinks.Thumbnail != "" {
		book.CoverURL = googleBook.VolumeInfo.ImageLinks.Thumbnail
	}

	// Append categories to subjects if we don't have subjects yet
	if len(book.Subjects) == 0 && len(googleBook.VolumeInfo.Categories) > 0 {
		book.Subjects = googleBook.VolumeInfo.Categories
	}

	if !cacheHit {
		// Add a small delay only when we had to make an API call
		time.Sleep(100 * time.Millisecond)
	}

	return nil
}
