package imdb

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseImdbWithParams(t *testing.T) {
	dir := t.TempDir()
	csv := filepath.Join(dir, "ratings.csv")
	if err := os.WriteFile(csv, []byte("const,yourRating\n1,10\n"), 0644); err != nil {
		t.Fatalf("write csv: %v", err)
	}

	var called bool
	parseImdbFunc = func() error {
		called = true
		if csvFile != csv {
			t.Fatalf("csvFile = %s, want %s", csvFile, csv)
		}
		if !strings.Contains(outputDir, dir) {
			t.Fatalf("outputDir = %s, want to contain %s", outputDir, dir)
		}
		if !tmdbEnabled || !tmdbDownloadCover || tmdbInteractive {
			t.Fatalf("tmdb flags not propagated")
		}
		if tmdbContentSections[0] != "overview" {
			t.Fatalf("tmdbContentSections not set")
		}
		return nil
	}
	defer func() { parseImdbFunc = ParseImdb }()

	err := ParseImdbWithParams(csv, dir, true, filepath.Join(dir, "imdb.json"), true, true, true, true, false, []string{"overview"})
	if err != nil {
		t.Fatalf("ParseImdbWithParams error = %v", err)
	}
	if !called {
		t.Fatalf("expected parseImdbFunc to be called")
	}
}
