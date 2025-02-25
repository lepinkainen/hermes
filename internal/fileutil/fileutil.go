package fileutil

import (
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
