package csvutil

import (
	"os"
	"path/filepath"
	"testing"
)

func TestProcessCSV(t *testing.T) {
	// Create a temporary CSV file for testing
	tmpDir := t.TempDir()
	csvPath := filepath.Join(tmpDir, "test.csv")

	csvContent := `name,age,city
Alice,30,NYC
Bob,25,LA
Charlie,35,Chicago
`
	if err := os.WriteFile(csvPath, []byte(csvContent), 0644); err != nil {
		t.Fatalf("failed to write test CSV: %v", err)
	}

	type Person struct {
		Name string
		Age  string
		City string
	}

	parser := func(record []string) (Person, error) {
		return Person{
			Name: record[0],
			Age:  record[1],
			City: record[2],
		}, nil
	}

	opts := ProcessorOptions{}
	people, err := ProcessCSV(csvPath, parser, opts)
	if err != nil {
		t.Fatalf("ProcessCSV() error = %v", err)
	}

	if len(people) != 3 {
		t.Errorf("expected 3 people, got %d", len(people))
	}

	expected := []Person{
		{"Alice", "30", "NYC"},
		{"Bob", "25", "LA"},
		{"Charlie", "35", "Chicago"},
	}

	for i, p := range people {
		if p != expected[i] {
			t.Errorf("people[%d] = %v, want %v", i, p, expected[i])
		}
	}
}

func TestProcessCSV_EmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	csvPath := filepath.Join(tmpDir, "empty.csv")

	if err := os.WriteFile(csvPath, []byte(""), 0644); err != nil {
		t.Fatalf("failed to write test CSV: %v", err)
	}

	parser := func(record []string) (string, error) {
		return record[0], nil
	}

	_, err := ProcessCSV(csvPath, parser, ProcessorOptions{})
	if err == nil {
		t.Error("expected error for empty file, got nil")
	}
}

func TestProcessCSV_FileNotFound(t *testing.T) {
	parser := func(record []string) (string, error) {
		return record[0], nil
	}

	_, err := ProcessCSV("/nonexistent/file.csv", parser, ProcessorOptions{})
	if err == nil {
		t.Error("expected error for nonexistent file, got nil")
	}
}
