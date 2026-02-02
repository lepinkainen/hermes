package goodreads

import (
	"fmt"
	"log/slog"
	"time"
)

var _ = enrichBookFromOpenLibrary

func openLibrarySearchISBN(book *Book) (string, error) {
	searchISBN := book.ISBN13
	if searchISBN == "" {
		searchISBN = book.ISBN
	}

	if searchISBN == "" {
		return "", fmt.Errorf("no ISBN available")
	}

	return searchISBN, nil
}

func fetchOpenLibraryBook(searchISBN string) (*OpenLibraryBook, bool, error) {
	_, olBook, cacheHit, err := getCachedBook(searchISBN)
	if err != nil {
		return nil, false, err
	}

	return olBook, cacheHit, nil
}

func applyOpenLibraryDescription(book *Book, olBook *OpenLibraryBook) {
	if desc := getDescription(olBook.Description); desc != "" {
		book.Description = desc
	}
}

func applyOpenLibraryCover(book *Book, olBook *OpenLibraryBook) {
	if len(olBook.Covers) > 0 {
		book.CoverID = olBook.Covers[0]

		if coverURL, err := fetchCoverImage(olBook.Covers[0]); err == nil && coverURL != "" {
			book.CoverURL = coverURL
		}
	}

	if olBook.Cover.Large != "" {
		book.CoverURL = olBook.Cover.Large
	}
}

func applyOpenLibrarySubtitle(book *Book, olBook *OpenLibraryBook) {
	if olBook.Subtitle != "" {
		book.Subtitle = olBook.Subtitle
	}
}

func applyOpenLibraryPublisher(book *Book, olBook *OpenLibraryBook) {
	if len(olBook.Publishers) > 0 && book.Publisher == "" {
		book.Publisher = olBook.Publishers[0].Name
	}
}

func applyOpenLibrarySubjects(book *Book, olBook *OpenLibraryBook) {
	book.Subjects = getSubjects(olBook.Subjects)
	book.SubjectPeople = getSubjects(olBook.SubjectPeople)
}

func applyOpenLibraryBookData(book *Book, olBook *OpenLibraryBook) {
	applyOpenLibraryDescription(book, olBook)
	applyOpenLibraryCover(book, olBook)
	applyOpenLibrarySubtitle(book, olBook)
	applyOpenLibraryPublisher(book, olBook)
	applyOpenLibrarySubjects(book, olBook)
}

func applyOpenLibraryEditionData(book *Book, editionData *OpenLibraryEdition) {
	if editionData.Number_of_pages > 0 {
		book.NumberOfPages = editionData.Number_of_pages
	}

	if len(editionData.Publishers) > 0 && book.Publisher == "" {
		book.Publisher = editionData.Publishers[0]
	}
}

func applyOpenLibraryEditionFallback(book *Book, searchISBN string) {
	if book.NumberOfPages != 0 {
		return
	}

	editionData, err := fetchEditionData(searchISBN)
	if err != nil || editionData == nil {
		return
	}

	applyOpenLibraryEditionData(book, editionData)
}

// enrichBookFromOpenLibrary is kept for backward compatibility and tests.
// New code should use enrichBookWithEnrichers instead.
func enrichBookFromOpenLibrary(book *Book) error {
	searchISBN, err := openLibrarySearchISBN(book)
	if err != nil {
		return err
	}

	olBook, cacheHit, err := fetchOpenLibraryBook(searchISBN)
	if err != nil {
		return fmt.Errorf("failed to get book data: %v", err)
	}

	applyOpenLibraryBookData(book, olBook)
	applyOpenLibraryEditionFallback(book, searchISBN)

	if !cacheHit {
		time.Sleep(100 * time.Millisecond)
	}

	return nil
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
	slog.Debug("Google Books enrichment", "isbn", searchISBN, "cache_hit", cacheHit, "error", err, "book_nil", googleBook == nil)
	if err != nil {
		// If book not found, just return without error (nothing to enrich with)
		return fmt.Errorf("failed to get Google Books data: %w", err)
	}

	// If googleBook is nil (shouldn't happen given the error check above, but be defensive)
	if googleBook == nil {
		slog.Debug("Google Books returned nil, skipping enrichment", "isbn", searchISBN)
		return nil
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
