package goodreads

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"

	"github.com/lepinkainen/hermes/internal/csvutil"
	"github.com/lepinkainen/hermes/internal/parseutil"
)

// loadBooksFromCSV parses all book records from a Goodreads export CSV.
// Invalid records are logged and skipped.
func loadBooksFromCSV(filePath string) ([]*Book, error) {
	return csvutil.ProcessCSV(filePath, parseBookRecord, csvutil.ProcessorOptions{SkipInvalid: true})
}

// enrichAndWriteBooks enriches each parsed book and writes its markdown note.
// Returns the books that were successfully written.
func enrichAndWriteBooks(parsed []*Book, outputDir string) ([]Book, error) {
	// Create output directory once before processing
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create output directory %s: %w", outputDir, err)
	}

	books := make([]Book, 0, len(parsed))
	for _, book := range parsed {
		if book.ISBN != "" || book.ISBN13 != "" {
			// Use the enricher system to fetch and merge data from all sources
			enrichBookWithEnrichers(context.Background(), book)
		}

		if err := writeBookToMarkdown(*book, outputDir); err != nil {
			slog.Error("Error writing markdown for book", "title", book.Title, "error", err)
			continue
		}

		books = append(books, *book)
		logBookProgress(len(books), len(parsed))
	}

	return books, nil
}

func parseBookRecord(record []string) (*Book, error) {
	const minColumns = 24
	if len(record) < minColumns {
		return nil, fmt.Errorf("record has %d columns, want at least %d", len(record), minColumns)
	}

	bookID, err := strconv.Atoi(record[0])
	if err != nil {
		return nil, fmt.Errorf("invalid book ID: %w", err)
	}

	book := &Book{
		ID:                       bookID,
		Title:                    record[1],
		Authors:                  parseutil.ParseCommaList(record[2]),
		ISBN:                     sanitizeISBNValue(record[5]),
		ISBN13:                   sanitizeISBNValue(record[6]),
		MyRating:                 parseFloatField(record[7]),
		AverageRating:            parseFloatField(record[8]),
		Publisher:                record[9],
		Binding:                  record[10],
		NumberOfPages:            parseIntField(record[11]),
		YearPublished:            parseIntField(record[12]),
		OriginalPublicationYear:  parseIntField(record[13]),
		DateRead:                 record[14],
		DateAdded:                record[15],
		Bookshelves:              parseutil.ParseCommaList(record[16]),
		BookshelvesWithPositions: parseutil.ParseCommaList(record[17]),
		ExclusiveShelf:           record[18],
		MyReview:                 record[19],
		Spoiler:                  record[20],
		PrivateNotes:             record[21],
		ReadCount:                parseIntField(record[22]),
		OwnedCopies:              parseIntField(record[23]),
	}

	return book, nil
}

func parseIntField(value string) int {
	result, err := strconv.Atoi(value)
	if err != nil {
		return 0
	}
	return result
}

func parseFloatField(value string) float64 {
	result, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0
	}
	return result
}

func sanitizeISBNValue(value string) string {
	trimmed := strings.TrimSuffix(value, "\"")
	trimmed = strings.TrimPrefix(trimmed, "=\"")
	return trimmed
}

func logBookProgress(processed, total int) {
	if processed == 0 || processed%10 != 0 {
		return
	}

	percentage := "0%"
	if total > 0 {
		percentage = fmt.Sprintf("%.1f%%", float64(processed)/float64(total)*100)
	}

	slog.Info("Processing books",
		"processed", processed,
		"total", total,
		"percentage", percentage,
	)
}
