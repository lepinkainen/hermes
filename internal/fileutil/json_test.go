package fileutil

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/lepinkainen/hermes/internal/testutil"
)

type TestJSONData struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

func TestWriteJSONFile_NewFile(t *testing.T) {
	// Setup
	env := testutil.NewTestEnv(t)
	tempDir := env.RootDir()
	filePath := filepath.Join(tempDir, "test.json")
	testData := []TestJSONData{
		{ID: 1, Name: "Test 1"},
		{ID: 2, Name: "Test 2"},
	}

	// Test
	written, err := WriteJSONFile(testData, filePath, true)

	// Assertions
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if !written {
		t.Error("Expected file to be written")
	}

	// Verify file exists and has correct content
	if !FileExists(filePath) {
		t.Error("Expected file to exist")
	}

	// Read and verify content
	data, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	var result []TestJSONData
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	if len(result) != 2 {
		t.Errorf("Expected 2 items, got %d", len(result))
	}
	if result[0].ID != 1 || result[0].Name != "Test 1" {
		t.Errorf("Expected first item to be {1, 'Test 1'}, got %+v", result[0])
	}
}

func TestWriteJSONFile_OverwriteTrue(t *testing.T) {
	// Setup
	env := testutil.NewTestEnv(t)
	tempDir := env.RootDir()
	filePath := filepath.Join(tempDir, "test.json")

	// Create existing file
	existingData := []TestJSONData{{ID: 99, Name: "Old"}}
	_, _ = WriteJSONFile(existingData, filePath, true)

	// Test overwriting
	newData := []TestJSONData{{ID: 1, Name: "New"}}
	written, err := WriteJSONFile(newData, filePath, true)

	// Assertions
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if !written {
		t.Error("Expected file to be written")
	}

	// Verify content was overwritten
	data, _ := os.ReadFile(filePath)
	var result []TestJSONData
	_ = json.Unmarshal(data, &result)

	if len(result) != 1 || result[0].ID != 1 || result[0].Name != "New" {
		t.Errorf("Expected file to be overwritten with new data, got %+v", result)
	}
}

func TestWriteJSONFile_OverwriteFalse(t *testing.T) {
	// Setup
	env := testutil.NewTestEnv(t)
	tempDir := env.RootDir()
	filePath := filepath.Join(tempDir, "test.json")

	// Create existing file
	existingData := []TestJSONData{{ID: 99, Name: "Old"}}
	_, _ = WriteJSONFile(existingData, filePath, true)

	// Test not overwriting
	newData := []TestJSONData{{ID: 1, Name: "New"}}
	written, err := WriteJSONFile(newData, filePath, false)

	// Assertions
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if written {
		t.Error("Expected file not to be written")
	}

	// Verify content was not changed
	data, _ := os.ReadFile(filePath)
	var result []TestJSONData
	_ = json.Unmarshal(data, &result)

	if len(result) != 1 || result[0].ID != 99 || result[0].Name != "Old" {
		t.Errorf("Expected file to remain unchanged, got %+v", result)
	}
}

func TestWriteJSONFile_CreateDirectory(t *testing.T) {
	// Setup
	env := testutil.NewTestEnv(t)
	tempDir := env.RootDir()
	filePath := filepath.Join(tempDir, "subdir", "nested", "test.json")
	testData := TestJSONData{ID: 1, Name: "Test"}

	// Test
	written, err := WriteJSONFile(testData, filePath, true)

	// Assertions
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if !written {
		t.Error("Expected file to be written")
	}

	// Verify directory was created
	if !FileExists(filePath) {
		t.Error("Expected file to exist")
	}

	// Verify directory structure
	dirPath := filepath.Join(tempDir, "subdir", "nested")
	if _, err := os.Stat(dirPath); os.IsNotExist(err) {
		t.Error("Expected directory to be created")
	}
}

func TestWriteJSONFile_InvalidData(t *testing.T) {
	// Setup
	env := testutil.NewTestEnv(t)
	tempDir := env.RootDir()
	filePath := filepath.Join(tempDir, "test.json")

	// Test with data that can't be marshaled (channel)
	invalidData := make(chan int)

	// Test
	written, err := WriteJSONFile(invalidData, filePath, true)

	// Assertions
	if err == nil {
		t.Fatal("Expected error for invalid data")
	}
	if written {
		t.Error("Expected file not to be written")
	}
	if FileExists(filePath) {
		t.Error("Expected file not to exist")
	}
}

func TestWriteJSONFile_SingleObject(t *testing.T) {
	// Setup
	env := testutil.NewTestEnv(t)
	tempDir := env.RootDir()
	filePath := filepath.Join(tempDir, "single.json")
	testData := TestJSONData{ID: 42, Name: "Single"}

	// Test
	written, err := WriteJSONFile(testData, filePath, true)

	// Assertions
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if !written {
		t.Error("Expected file to be written")
	}

	// Verify content
	data, _ := os.ReadFile(filePath)
	var result TestJSONData
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	if result.ID != 42 || result.Name != "Single" {
		t.Errorf("Expected {42, 'Single'}, got %+v", result)
	}
}
