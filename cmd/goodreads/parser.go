package goodreads

import (
	"encoding/csv"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strconv"
	"strings"

	"github.com/lepinkainen/hermes/internal/datastore"
	"github.com/spf13/viper"
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

func ParseGoodreads() error {
	// First, count the total number of books in the CSV file
	totalBooks, err := countBooksInCSV(csvFile)
	if err != nil {
		return fmt.Errorf("failed to count books in CSV: %w", err)
	}

	// Open the CSV file again for processing
	csvFile, err := os.Open(csvFile) // Using the global csvFile variable from cmd.go
	if err != nil {
		return fmt.Errorf("failed to open CSV file: %w", err)
	}
	defer csvFile.Close()

	// Create a new CSV reader
	reader := csv.NewReader(csvFile)

	// Skip the header row (assuming the first row contains column names)
	_, err = reader.Read()
	if err != nil {
		return fmt.Errorf("failed to read CSV header: %w", err)
	}

	var processedCount int
	var books []Book

	// Read each record from the CSV file
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			slog.Warn("Error reading record", "error", err)
			continue
		}

		// Convert string values to appropriate types
		bookID, err := strconv.Atoi(record[0])
		if err != nil {
			slog.Warn("Invalid book ID", "error", err)
			continue
		}

		myRating, err := strconv.ParseFloat(record[7], 64)
		if err != nil {
			myRating = 0.0
		}

		averageRating, err := strconv.ParseFloat(record[8], 64)
		if err != nil {
			averageRating = 0.0
		}

		numberOfPages, err := strconv.Atoi(record[11])
		if err != nil {
			numberOfPages = 0
		}

		yearPublished, err := strconv.Atoi(record[12])
		if err != nil {
			yearPublished = 0
		}

		originalPublicationYear, err := strconv.Atoi(record[13])
		if err != nil {
			originalPublicationYear = 0
		}

		readCount, err := strconv.Atoi(record[22])
		if err != nil {
			readCount = 0
		}

		ownedCopies, err := strconv.Atoi(record[23])
		if err != nil {
			ownedCopies = 0
		}

		// Remove unnecessary quotes from ISBN and ISBN13 (if present)
		isbn := strings.TrimPrefix(strings.TrimSuffix(record[5], "\""), "=\"")
		isbn13 := strings.TrimPrefix(strings.TrimSuffix(record[6], "\""), "=\"")

		// Create a new Book struct
		book := Book{
			ID:                       bookID,
			Title:                    record[1],
			Authors:                  splitString(record[2]),
			ISBN:                     isbn,
			ISBN13:                   isbn13,
			MyRating:                 myRating,
			AverageRating:            averageRating,
			Publisher:                record[9],
			Binding:                  record[10],
			NumberOfPages:            numberOfPages,
			YearPublished:            yearPublished,
			OriginalPublicationYear:  originalPublicationYear,
			DateRead:                 record[14],
			DateAdded:                record[15],
			Bookshelves:              splitString(record[16]),
			BookshelvesWithPositions: splitString(record[17]),
			ExclusiveShelf:           record[18],
			MyReview:                 record[19],
			Spoiler:                  record[20],
			PrivateNotes:             record[21],
			ReadCount:                readCount,
			OwnedCopies:              ownedCopies,
		}

		// Try to enrich the book with OpenLibrary data
		if isbn != "" || isbn13 != "" {
			if err := enrichBookFromOpenLibrary(&book); err != nil {
				slog.Warn("Could not enrich book data", "error", err)
			}
		}

		// Write the book to markdown
		if err := writeBookToMarkdown(book, outputDir); err != nil {
			slog.Error("Error writing markdown for book", "title", book.Title, "error", err)
			continue
		}

		// Add the book to our collection
		books = append(books, book)

		processedCount++
		if processedCount%10 == 0 {
			slog.Info("Processing books",
				"processed", processedCount,
				"total", totalBooks,
				"percentage", fmt.Sprintf("%.1f%%", float64(processedCount)/float64(totalBooks)*100))
		}
	}

	slog.Info("Successfully processed all books", "processed", processedCount, "total", totalBooks, "percentage", "100%")

	// Write to JSON if enabled
	if writeJSON {
		if err := writeBookToJson(books, jsonOutput); err != nil {
			slog.Error("Error writing books to JSON", "error", err)
		}
	}

	// Datasette integration
	if viper.GetBool("datasette.enabled") {
		slog.Info("Writing Goodreads books to Datasette")
		mode := viper.GetString("datasette.mode")

		if mode == "local" {
			store := datastore.NewSQLiteStore(viper.GetString("datasette.dbfile"))
			if err := store.Connect(); err != nil {
				slog.Error("Failed to connect to SQLite database", "error", err)
				return err
			}
			defer store.Close()

			schema := `CREATE TABLE IF NOT EXISTS goodreads_books (
				id INTEGER PRIMARY KEY,
				title TEXT,
				authors TEXT,
				isbn TEXT,
				isbn13 TEXT,
				my_rating REAL,
				average_rating REAL,
				publisher TEXT,
				binding TEXT,
				number_of_pages INTEGER,
				year_published INTEGER,
				original_publication_year INTEGER,
				date_read TEXT,
				date_added TEXT,
				bookshelves TEXT,
				bookshelves_with_positions TEXT,
				exclusive_shelf TEXT,
				my_review TEXT,
				spoiler TEXT,
				private_notes TEXT,
				read_count INTEGER,
				owned_copies INTEGER,
				description TEXT,
				subjects TEXT,
				cover_id INTEGER,
				cover_url TEXT,
				subject_people TEXT,
				subtitle TEXT
			)`

			if err := store.CreateTable(schema); err != nil {
				slog.Error("Failed to create table", "error", err)
				return err
			}

			records := make([]map[string]any, len(books))
			for i, book := range books {
				records[i] = bookToMap(book)
			}

			if err := store.BatchInsert("hermes", "goodreads_books", records); err != nil {
				slog.Error("Failed to insert records", "error", err)
				return err
			}
			slog.Info("Successfully wrote books to SQLite database", "count", len(books))
		} else if mode == "remote" {
			client := datastore.NewDatasetteClient(
				viper.GetString("datasette.remote_url"),
				viper.GetString("datasette.api_token"),
			)
			if err := client.Connect(); err != nil {
				slog.Error("Failed to connect to remote Datasette", "error", err)
				return err
			}
			defer client.Close()

			records := make([]map[string]any, len(books))
			for i, book := range books {
				records[i] = bookToMap(book)
			}

			if err := client.BatchInsert("hermes", "goodreads_books", records); err != nil {
				slog.Error("Failed to insert records to remote Datasette", "error", err)
				return err
			}
			slog.Info("Successfully wrote books to remote Datasette", "count", len(books))
		} else {
			slog.Error("Invalid Datasette mode", "mode", mode)
			return fmt.Errorf("invalid Datasette mode: %s", mode)
		}
	}

	return nil
}

// countBooksInCSV counts the total number of books in the CSV file
func countBooksInCSV(filePath string) (int, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return 0, fmt.Errorf("failed to open CSV file: %w", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)

	// Skip header row
	_, err = reader.Read()
	if err != nil {
		return 0, fmt.Errorf("failed to read CSV header: %w", err)
	}

	count := 0
	for {
		_, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			// Just skip invalid records when counting
			continue
		}
		count++
	}

	return count, nil
}

// Helper function to split comma-separated strings
func splitString(str string) []string {
	if str == "" {
		return nil
	}
	var splitStrings = strings.Split(str, ",")
	for i, s := range splitStrings {
		splitStrings[i] = strings.TrimSpace(s)
	}
	return splitStrings
}

// Helper function to handle the description field
func getDescription(desc any) string {
	switch v := desc.(type) {
	case string:
		return v
	case map[string]any:
		if value, ok := v["value"].(string); ok {
			return value
		}
	}
	return ""
}

// Helper function to handle subjects
func getSubjects(subjects []any) []string {
	result := make([]string, 0)
	for _, subject := range subjects {
		switch v := subject.(type) {
		case string:
			result = append(result, v)
		case map[string]any:
			if name, ok := v["name"].(string); ok {
				result = append(result, name)
			}
		}
	}
	return result
}

// Helper function for subject people
func getSubjectPeople(subjects []any) []string {
	result := make([]string, 0)
	for _, subject := range subjects {
		switch v := subject.(type) {
		case string:
			result = append(result, v)
		case map[string]any:
			if name, ok := v["name"].(string); ok {
				result = append(result, name)
			}
		}
	}
	return result
}
