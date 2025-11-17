package cache

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

type TestData struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

func TestGetOrFetch_CacheHit(t *testing.T) {
	// Setup
	tempDir := t.TempDir()
	cacheKey := "test-key"
	expectedData := TestData{ID: 1, Name: "Test"}

	// Pre-populate cache
	cachePath := filepath.Join(tempDir, cacheKey+".json")
	jsonData, _ := json.MarshalIndent(expectedData, "", "  ")
	_ = os.WriteFile(cachePath, jsonData, 0644)

	// Test
	fetchCalled := false
	fetchFunc := func() (TestData, error) {
		fetchCalled = true
		return TestData{}, nil
	}

	result, fromCache, err := GetOrFetch(tempDir, cacheKey, fetchFunc)

	// Assertions
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if !fromCache {
		t.Error("Expected fromCache to be true")
	}
	if fetchCalled {
		t.Error("Expected fetch function not to be called")
	}
	if result.ID != expectedData.ID || result.Name != expectedData.Name {
		t.Errorf("Expected %+v, got %+v", expectedData, result)
	}
}

func TestGetOrFetch_CacheMiss(t *testing.T) {
	// Setup
	tempDir := t.TempDir()
	cacheKey := "test-key"
	expectedData := TestData{ID: 2, Name: "Fetched"}

	// Test
	fetchCalled := false
	fetchFunc := func() (TestData, error) {
		fetchCalled = true
		return expectedData, nil
	}

	result, fromCache, err := GetOrFetch(tempDir, cacheKey, fetchFunc)

	// Assertions
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if fromCache {
		t.Error("Expected fromCache to be false")
	}
	if !fetchCalled {
		t.Error("Expected fetch function to be called")
	}
	if result.ID != expectedData.ID || result.Name != expectedData.Name {
		t.Errorf("Expected %+v, got %+v", expectedData, result)
	}

	// Verify data was cached
	cachePath := filepath.Join(tempDir, cacheKey+".json")
	if _, err := os.Stat(cachePath); os.IsNotExist(err) {
		t.Error("Expected cache file to be created")
	}
}

func TestGetOrFetch_FetchError(t *testing.T) {
	// Setup
	tempDir := t.TempDir()
	cacheKey := "test-key"
	expectedError := errors.New("fetch failed")

	// Test
	fetchFunc := func() (TestData, error) {
		return TestData{}, expectedError
	}

	result, fromCache, err := GetOrFetch(tempDir, cacheKey, fetchFunc)

	// Assertions
	if err == nil {
		t.Fatal("Expected error, got nil")
	}
	if !errors.Is(err, expectedError) {
		t.Errorf("Expected error to contain %v, got %v", expectedError, err)
	}
	if fromCache {
		t.Error("Expected fromCache to be false")
	}
	if result.ID != 0 || result.Name != "" {
		t.Errorf("Expected zero value, got %+v", result)
	}
}

func TestGetOrFetch_CorruptedCache(t *testing.T) {
	// Setup
	tempDir := t.TempDir()
	cacheKey := "test-key"
	expectedData := TestData{ID: 3, Name: "Refetched"}

	// Create corrupted cache file
	cachePath := filepath.Join(tempDir, cacheKey+".json")
	_ = os.WriteFile(cachePath, []byte("invalid json"), 0644)

	// Test
	fetchCalled := false
	fetchFunc := func() (TestData, error) {
		fetchCalled = true
		return expectedData, nil
	}

	result, fromCache, err := GetOrFetch(tempDir, cacheKey, fetchFunc)

	// Assertions
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if fromCache {
		t.Error("Expected fromCache to be false")
	}
	if !fetchCalled {
		t.Error("Expected fetch function to be called due to corrupted cache")
	}
	if result.ID != expectedData.ID || result.Name != expectedData.Name {
		t.Errorf("Expected %+v, got %+v", expectedData, result)
	}
}

func TestClearCache(t *testing.T) {
	// Setup
	tempDir := t.TempDir()

	// Create some cache files
	testFiles := []string{"file1.json", "file2.json", "file3.txt", "file4.json"}
	for _, file := range testFiles {
		filePath := filepath.Join(tempDir, file)
		_ = os.WriteFile(filePath, []byte("{}"), 0644)
	}

	// Test
	err := ClearCache(tempDir)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Verify only .json files were removed
	entries, _ := os.ReadDir(tempDir)
	remainingFiles := make([]string, 0)
	for _, entry := range entries {
		remainingFiles = append(remainingFiles, entry.Name())
	}

	if len(remainingFiles) != 1 {
		t.Errorf("Expected 1 remaining file, got %d: %v", len(remainingFiles), remainingFiles)
	}
	if remainingFiles[0] != "file3.txt" {
		t.Errorf("Expected file3.txt to remain, got %v", remainingFiles)
	}
}

func TestClearCache_NonExistentDirectory(t *testing.T) {
	// Test clearing cache from non-existent directory
	err := ClearCache("/non/existent/directory")
	if err != nil {
		t.Fatalf("Expected no error for non-existent directory, got %v", err)
	}
}

func TestCacheExists(t *testing.T) {
	// Setup
	tempDir := t.TempDir()
	existingKey := "existing"
	nonExistingKey := "non-existing"

	// Create one cache file
	cachePath := filepath.Join(tempDir, existingKey+".json")
	_ = os.WriteFile(cachePath, []byte("{}"), 0644)

	// Test existing file
	if !CacheExists(tempDir, existingKey) {
		t.Error("Expected cache to exist for existing key")
	}

	// Test non-existing file
	if CacheExists(tempDir, nonExistingKey) {
		t.Error("Expected cache not to exist for non-existing key")
	}
}
