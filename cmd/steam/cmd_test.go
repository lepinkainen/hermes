package steam

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseSteamWithParams(t *testing.T) {
	dir := t.TempDir()
	parseSteamFunc = func() error {
		if steamID != "123" || apiKey != "key" {
			t.Fatalf("steamID/apiKey not set")
		}
		if !strings.Contains(outputDir, dir) {
			t.Fatalf("outputDir = %s, want to contain %s", outputDir, dir)
		}
		if !writeJSON || !overwrite {
			t.Fatalf("flags not propagated")
		}
		if jsonOutput == "" {
			t.Fatalf("jsonOutput should be set")
		}
		return nil
	}
	defer func() { parseSteamFunc = ParseSteam }()

	jsonPath := filepath.Join(dir, "steam.json")
	if err := os.WriteFile(jsonPath, []byte("{}"), 0644); err != nil {
		t.Fatalf("write json path: %v", err)
	}

	if err := ParseSteamWithParams("123", "key", dir, true, jsonPath, true); err != nil {
		t.Fatalf("ParseSteamWithParams error = %v", err)
	}
}
