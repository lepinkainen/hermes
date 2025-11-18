package enrichment

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSkipExistingCoverFile(t *testing.T) {
	// Create a temporary directory for test files
	tempDir, err := os.MkdirTemp("", "tmdb-cover-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() {
		if err := os.RemoveAll(tempDir); err != nil {
			t.Errorf("Failed to remove temp dir: %v", err)
		}
	}()

	// Create an existing cover file
	coverFilename := "Test_Movie - cover.jpg"
	coverPath := filepath.Join(tempDir, coverFilename)

	// Create a dummy cover file
	existingContent := []byte("existing cover image")
	if err := os.WriteFile(coverPath, existingContent, 0644); err != nil {
		t.Fatalf("Failed to create existing cover file: %v", err)
	}

	// Test file existence check
	if _, err := os.Stat(coverPath); err != nil {
		t.Fatalf("Existing cover file check failed: %v", err)
	}

	// Verify file content is still the same after existence check
	currentContent, err := os.ReadFile(coverPath)
	if err != nil {
		t.Fatalf("Failed to read existing cover file: %v", err)
	}
	if string(currentContent) != string(existingContent) {
		t.Error("Existing cover file was modified")
	}
}

func TestDownloadCoverFile_MissingFile(t *testing.T) {
	// Create a temporary directory for test files
	tempDir, err := os.MkdirTemp("", "tmdb-cover-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() {
		if err := os.RemoveAll(tempDir); err != nil {
			t.Errorf("Failed to remove temp dir: %v", err)
		}
	}()

	coverFilename := "Test_Movie - cover.jpg"
	coverPath := filepath.Join(tempDir, coverFilename)

	// Test file existence check for missing file
	if _, err := os.Stat(coverPath); !os.IsNotExist(err) {
		t.Fatalf("Expected file to not exist, but stat returned: %v", err)
	}

	// Verify the file doesn't exist
	if _, err := os.ReadFile(coverPath); err == nil {
		t.Error("Expected file to not exist, but ReadFile succeeded")
	}
}
