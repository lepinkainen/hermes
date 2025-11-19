package goodreads

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseGoodreadsWithParams(t *testing.T) {
	dir := t.TempDir()
	csv := filepath.Join(dir, "books.csv")
	requireNoError(t, os.WriteFile(csv, []byte("id,title\n1,Test\n"), 0644))

	var called bool
	parseGoodreadsFunc = func() error {
		called = true
		if csvFile != csv {
			t.Fatalf("csvFile = %s, want %s", csvFile, csv)
		}
		if !strings.Contains(outputDir, dir) {
			t.Fatalf("outputDir = %s, want to contain %s", outputDir, dir)
		}
		if !writeJSON {
			t.Fatalf("writeJSON should be true")
		}
		if jsonOutput == "" {
			t.Fatalf("jsonOutput should be set")
		}
		return nil
	}
	defer func() { parseGoodreadsFunc = ParseGoodreads }()

	err := ParseGoodreadsWithParams(csv, dir, true, filepath.Join(dir, "books.json"), true)
	if err != nil {
		t.Fatalf("ParseGoodreadsWithParams error = %v", err)
	}
	if !called {
		t.Fatalf("expected parseGoodreadsFunc to be called")
	}
}

func requireNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
