package goodreads

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
)

// countBooksInCSV counts the total number of books in the CSV file
func countBooksInCSV(filePath string) (int, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return 0, fmt.Errorf("failed to open CSV file: %w", err)
	}
	defer func() { _ = file.Close() }()

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
