package csvutil

import (
	"encoding/csv"
	"fmt"
	"io"
	"log/slog"
	"os"
)

// ProcessorOptions configures CSV processing behavior.
type ProcessorOptions struct {
	// FieldsPerRecord sets the expected number of fields per record.
	// If 0, it's set to the number of fields in the first record.
	FieldsPerRecord int

	// SkipInvalid controls whether to skip invalid records or return an error.
	SkipInvalid bool
}

// ProcessCSV reads a CSV file and parses each record into type T.
// The parser function converts a CSV record ([]string) into the target type.
// Returns a slice of parsed items or an error.
func ProcessCSV[T any](filename string, parser func([]string) (T, error), opts ProcessorOptions) ([]T, error) {
	csvFile, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open CSV file: %v", err)
	}
	defer func() { _ = csvFile.Close() }()

	// File existence check
	if fi, err := csvFile.Stat(); err != nil || fi.Size() == 0 {
		return nil, fmt.Errorf("CSV file is empty or cannot be read")
	}

	reader := csv.NewReader(csvFile)
	if opts.FieldsPerRecord > 0 {
		reader.FieldsPerRecord = opts.FieldsPerRecord
	}

	// Skip header
	if _, err := reader.Read(); err != nil {
		return nil, fmt.Errorf("failed to read header: %v", err)
	}

	var items []T

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			slog.Warn("Error reading record", "error", err)
			continue
		}

		item, err := parser(record)
		if err != nil {
			if opts.SkipInvalid {
				slog.Warn("Skipping invalid record", "error", err)
				continue
			}
			return nil, fmt.Errorf("invalid record: %v", err)
		}

		items = append(items, item)
	}

	return items, nil
}
