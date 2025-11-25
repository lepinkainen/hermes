package goodreads

import (
	"fmt"
	"log/slog"
	"strings"
)

// Convert Book to map[string]any for database insertion
func bookToMap(book Book) map[string]any {
	return map[string]any{
		"id":                         book.ID,
		"title":                      book.Title,
		"authors":                    strings.Join(book.Authors, ","),
		"isbn":                       book.ISBN,
		"isbn13":                     book.ISBN13,
		"my_rating":                  book.MyRating,
		"average_rating":             book.AverageRating,
		"publisher":                  book.Publisher,
		"binding":                    book.Binding,
		"number_of_pages":            book.NumberOfPages,
		"year_published":             book.YearPublished,
		"original_publication_year":  book.OriginalPublicationYear,
		"date_read":                  book.DateRead,
		"date_added":                 book.DateAdded,
		"bookshelves":                strings.Join(book.Bookshelves, ","),
		"bookshelves_with_positions": strings.Join(book.BookshelvesWithPositions, ","),
		"exclusive_shelf":            book.ExclusiveShelf,
		"my_review":                  book.MyReview,
		"spoiler":                    book.Spoiler,
		"private_notes":              book.PrivateNotes,
		"read_count":                 book.ReadCount,
		"owned_copies":               book.OwnedCopies,
		"description":                book.Description,
		"subjects":                   strings.Join(book.Subjects, ","),
		"cover_id":                   book.CoverID,
		"cover_url":                  book.CoverURL,
		"subject_people":             strings.Join(book.SubjectPeople, ","),
		"subtitle":                   book.Subtitle,
	}
}

func ParseGoodreads(params ParseParams) error {
	totalBooks, err := countBooksInCSV(params.CSVPath)
	if err != nil {
		return fmt.Errorf("failed to count books in CSV: %w", err)
	}

	books, err := loadBooksFromCSV(params.CSVPath, totalBooks, params.OutputDir)
	if err != nil {
		return err
	}

	processedCount := len(books)
	percentage := "0%"
	if totalBooks > 0 {
		percentage = fmt.Sprintf("%.1f%%", float64(processedCount)/float64(totalBooks)*100)
	}
	slog.Info("Successfully processed all books", "processed", processedCount, "total", totalBooks, "percentage", percentage)

	writeBooksToJSONIfEnabled(books, params.WriteJSON, params.JSONOutput)

	if err := writeBooksToDatasetteIfEnabled(books); err != nil {
		return err
	}

	return nil
}
