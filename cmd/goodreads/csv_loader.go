package goodreads

import (
	"encoding/csv"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strconv"
	"strings"
)

func loadBooksFromCSV(filePath string, totalBooks int, outputDir string) ([]Book, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open CSV file: %w", err)
	}
	defer func() { _ = file.Close() }()

	reader := csv.NewReader(file)
	if _, err := reader.Read(); err != nil {
		return nil, fmt.Errorf("failed to read CSV header: %w", err)
	}

	// Create output directory once before processing
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create output directory %s: %w", outputDir, err)
	}

	var books []Book
	processed := 0

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			slog.Warn("Error reading record", "error", err)
			continue
		}

		book, err := parseBookRecord(record)
		if err != nil {
			slog.Warn("Invalid book record", "error", err)
			continue
		}

		if book.ISBN != "" || book.ISBN13 != "" {
			// Always call both APIs (data is cached after first run)
			if err := enrichBookFromOpenLibrary(book); err != nil {
				slog.Warn("Could not enrich book data from OpenLibrary", "title", book.Title, "error", err)
			}

			if err := enrichBookFromGoogleBooks(book); err != nil {
				slog.Warn("Could not enrich book data from Google Books", "title", book.Title, "error", err)
			}
		}

		if err := writeBookToMarkdown(*book, outputDir); err != nil {
			slog.Error("Error writing markdown for book", "title", book.Title, "error", err)
			continue
		}

		books = append(books, *book)
		processed++
		logBookProgress(processed, totalBooks)
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
		Authors:                  splitString(record[2]),
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
		Bookshelves:              splitString(record[16]),
		BookshelvesWithPositions: splitString(record[17]),
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
