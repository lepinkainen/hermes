package letterboxd

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/lepinkainen/hermes/internal/fileutil"
)

// writeJSONFile writes the movies to a JSON file
func writeJSONFile(movies []Movie, filename string) error {
	// Check if file exists and we shouldn't overwrite
	if fileutil.FileExists(filename) && !overwrite {
		slog.Info("JSON file already exists, skipping", "filename", filename, "overwrite", overwrite)
		return nil
	}

	data, err := json.MarshalIndent(movies, "", "  ")
	if err != nil {
		return err
	}

	// Ensure the output directory exists
	dir := filepath.Dir(filename)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %v", err)
	}

	slog.Info("Writing JSON file", "filename", filename, "overwrite", overwrite)
	return os.WriteFile(filename, data, 0644)
}
