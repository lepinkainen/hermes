package letterboxd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseLetterboxdWithParams(t *testing.T) {
	dir := t.TempDir()
	csv := filepath.Join(dir, "log.csv")
	if err := os.WriteFile(csv, []byte("Date,Name,Year\n"), 0644); err != nil {
		t.Fatalf("write csv: %v", err)
	}

	var called bool
	parseLetterboxdFunc = func() error {
		called = true
		if csvFile != csv {
			t.Fatalf("csvFile = %s, want %s", csvFile, csv)
		}
		if !strings.Contains(outputDir, dir) {
			t.Fatalf("outputDir = %s, want to contain %s", outputDir, dir)
		}
		if !tmdbEnabled || !tmdbGenerateContent {
			t.Fatalf("tmdb flags expected true")
		}
		return nil
	}
	defer func() { parseLetterboxdFunc = ParseLetterboxd }()

	err := ParseLetterboxdWithParams(csv, dir, false, "", true, true, false, true, true, []string{"overview"})
	if err != nil {
		t.Fatalf("ParseLetterboxdWithParams error = %v", err)
	}
	if !called {
		t.Fatalf("expected parser to run")
	}
}
