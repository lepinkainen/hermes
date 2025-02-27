package fileutil

import (
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
	name = strings.ReplaceAll(name, ":", " - ")
	name = strings.ReplaceAll(name, "/", "-")
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
