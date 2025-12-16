package fileutil

import (
	"encoding/json"
	"fmt"
	"io"
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
	// Also handle other invalid characters
	invalid := []string{`<`, `>`, `"`, `|`, `?`, `*`}
	for _, ch := range invalid {
		name = strings.ReplaceAll(name, ch, "_")
	}
	name = strings.Trim(name, ". ")
	if len(name) > 200 {
		return name[:200]
	}
	return name
}

// EnsureDir creates a directory and all necessary parent directories.
func EnsureDir(path string) error {
	return os.MkdirAll(path, 0o755)
}

// RelativeTo returns the relative path from base to target.
func RelativeTo(base, target string) (string, error) {
	rel, err := filepath.Rel(base, target)
	if err != nil {
		return "", err
	}
	return filepath.ToSlash(rel), nil
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

// WriteMarkdownFile writes markdown content to a file with logging.
// It respects the overwrite flag and logs the result.
func WriteMarkdownFile(filePath string, content string, overwrite bool) error {
	written, err := WriteFileWithOverwrite(filePath, []byte(content), 0644, overwrite)
	if err != nil {
		return err
	}

	LogFileWriteResult(written, filePath)

	return nil
}

// LogFileWriteResult logs the result of a file write operation using standardized keys.
// This ensures consistent logging across all importers.
func LogFileWriteResult(written bool, filePath string) {
	if !written {
		slog.Debug("Skipped existing file", "path", filePath)
	} else {
		slog.Info("Wrote file", "path", filePath)
	}
}

// WriteJSONFile writes data as JSON to a file, respecting the overwrite flag
// Returns true if the file was written, false if it was skipped
func WriteJSONFile(data interface{}, filePath string, overwrite bool) (bool, error) {
	// Check if file exists and we shouldn't overwrite
	if FileExists(filePath) && !overwrite {
		slog.Debug("Skipped existing JSON file", "path", filePath)
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
	if err := os.WriteFile(filePath, jsonData, 0644); err != nil {
		return false, fmt.Errorf("failed to write JSON file: %w", err)
	}

	return true, nil
}

// CopyFile copies a file from src to dst
func CopyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() { _ = in.Close() }()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer func() { _ = out.Close() }()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}

	return out.Sync()
}
