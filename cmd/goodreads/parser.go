package goodreads

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	log "github.com/sirupsen/logrus"
)

func ParseGoodreads() error {
	// Open the CSV file
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

	// Read each record from the CSV file
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Printf("Warning: Error reading record: %v\n", err)
			continue
		}

		// Convert string values to appropriate types
		bookID, err := strconv.Atoi(record[0])
		if err != nil {
			log.Printf("Warning: Invalid book ID: %v\n", err)
			continue
		}

		myRating, err := strconv.ParseFloat(record[8], 64)
		if err != nil {
			myRating = 0.0
		}

		averageRating, err := strconv.ParseFloat(record[9], 64)
		if err != nil {
			averageRating = 0.0
		}

		numberOfPages, err := strconv.Atoi(record[12])
		if err != nil {
			numberOfPages = 0
		}

		yearPublished, err := strconv.Atoi(record[13])
		if err != nil {
			yearPublished = 0
		}

		originalPublicationYear, err := strconv.Atoi(record[14])
		if err != nil {
			originalPublicationYear = 0
		}

		readCount, err := strconv.Atoi(record[20])
		if err != nil {
			readCount = 0
		}

		ownedCopies, err := strconv.Atoi(record[21])
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
			Publisher:                record[10],
			Binding:                  record[11],
			NumberOfPages:            numberOfPages,
			YearPublished:            yearPublished,
			OriginalPublicationYear:  originalPublicationYear,
			DateRead:                 record[14],
			DateAdded:                record[15],
			Bookshelves:              splitString(record[16]),
			BookshelvesWithPositions: splitString(record[17]),
			ExclusiveShelf:           record[17],
			MyReview:                 record[18],
			Spoiler:                  record[19],
			PrivateNotes:             record[20],
			ReadCount:                readCount,
			OwnedCopies:              ownedCopies,
		}

		// Try to enrich the book with OpenLibrary data
		if isbn != "" || isbn13 != "" {
			if err := enrichBookFromOpenLibrary(&book); err != nil {
				log.Warnf("Could not enrich book data: %v\n", err)
			}
		}

		// Write the book to markdown
		if err := writeBookToMarkdown(book, outputDir); err != nil {
			log.Errorf("Error writing markdown for book %s: %v\n", book.Title, err)
			continue
		}

		processedCount++
		if processedCount%10 == 0 {
			log.Printf("Processed %d books...\n", processedCount)
		}
	}

	log.Printf("Successfully processed %d books\n", processedCount)
	return nil
}

// Helper function to split comma-separated strings
func splitString(str string) []string {
	if str == "" {
		return nil
	}
	return strings.Split(str, ",")
}

// Helper function to handle the description field
func getDescription(desc interface{}) string {
	switch v := desc.(type) {
	case string:
		return v
	case map[string]interface{}:
		if value, ok := v["value"].(string); ok {
			return value
		}
	}
	return ""
}

// Helper function to handle subjects
func getSubjects(subjects []interface{}) []string {
	result := make([]string, 0)
	for _, subject := range subjects {
		switch v := subject.(type) {
		case string:
			result = append(result, v)
		case map[string]interface{}:
			if name, ok := v["name"].(string); ok {
				result = append(result, name)
			}
		}
	}
	return result
}

// Helper function for subject people
func getSubjectPeople(subjects []interface{}) []string {
	result := make([]string, 0)
	for _, subject := range subjects {
		switch v := subject.(type) {
		case string:
			result = append(result, v)
		case map[string]interface{}:
			if name, ok := v["name"].(string); ok {
				result = append(result, name)
			}
		}
	}
	return result
}
