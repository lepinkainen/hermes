package csvutil

import (
	"testing"

	"github.com/lepinkainen/hermes/internal/testutil"
)

func TestProcessCSV(t *testing.T) {
	// Create a sandboxed test environment
	env := testutil.NewTestEnv(t)

	// Create a temporary CSV file for testing
	csvContent := `name,age,city
Alice,30,NYC
Bob,25,LA
Charlie,35,Chicago
`
	env.WriteFileString("test.csv", csvContent)
	csvPath := env.Path("test.csv")

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
	env := testutil.NewTestEnv(t)

	env.WriteFileString("empty.csv", "")
	csvPath := env.Path("empty.csv")

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
