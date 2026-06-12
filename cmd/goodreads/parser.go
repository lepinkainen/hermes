package goodreads

import (
	"fmt"
	"log/slog"

	"github.com/lepinkainen/hermes/internal/cmdutil"
)

// Convert Book to map[string]any for database insertion
func bookToMap(book Book) map[string]any {
	return cmdutil.StructToMap(book, cmdutil.StructToMapOptions{
		JoinStringSlices: true,
	})
}

// ParseGoodreads imports the configured Goodreads CSV and writes the requested outputs.
func ParseGoodreads(params ParseParams) error {
	parsed, err := loadBooksFromCSV(params.CSVPath)
	if err != nil {
		return fmt.Errorf("failed to load books from CSV: %w", err)
	}

	books, err := enrichAndWriteBooks(parsed, params.OutputDir)
	if err != nil {
		return err
	}

	totalBooks := len(parsed)
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
