package fileutil

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

// GetMarkdownFilePath returns the expected markdown file path for a given name
func GetMarkdownFilePath(name string, directory string) string {
	// Clean the filename first
	filename := SanitizeFilename(name)
	return filepath.Join(directory, filename+".md")
}

// SanitizeFilename cleans a filename by replacing problematic characters
func SanitizeFilename(name string) string {
	// Replace problematic characters
	name = strings.ReplaceAll(name, ":", " -")
	name = strings.ReplaceAll(name, "/", "-")
	name = strings.ReplaceAll(name, "\\", "-")
	return name
}

// FileExists checks if a file exists at the given path
func FileExists(filePath string) bool {
	info, err := os.Stat(filePath)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

// WriteFileWithOverwrite writes data to a file, respecting the overwrite flag
// Returns true if the file was written, false if it was skipped
func WriteFileWithOverwrite(filePath string, data []byte, perm os.FileMode, overwrite bool) (bool, error) {
	// Check if file exists
	if FileExists(filePath) && !overwrite {
		// Skip writing if file exists and overwrite is false
		return false, nil
	}

	// Create directory if it doesn't exist
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return false, err
	}

	// Write the file
	if err := os.WriteFile(filePath, data, perm); err != nil {
		return false, err
	}

	return true, nil
}

// WriteJSONFile writes data as JSON to a file, respecting the overwrite flag
// Returns true if the file was written, false if it was skipped
func WriteJSONFile(data interface{}, filePath string, overwrite bool) (bool, error) {
	// Check if file exists and we shouldn't overwrite
	if FileExists(filePath) && !overwrite {
		slog.Info("JSON file already exists, skipping", "filename", filePath, "overwrite", overwrite)
		return false, nil
	}

	// Marshal data to JSON
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return false, fmt.Errorf("failed to marshal JSON: %w", err)
	}

	// Create directory if it doesn't exist
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return false, fmt.Errorf("failed to create directory: %w", err)
	}

	// Write the file
	slog.Info("Writing JSON file", "filename", filePath, "overwrite", overwrite)
	if err := os.WriteFile(filePath, jsonData, 0644); err != nil {
		return false, fmt.Errorf("failed to write JSON file: %w", err)
	}

	return true, nil
}
