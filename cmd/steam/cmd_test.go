package steam

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lepinkainen/hermes/internal/testutil"
)

func TestParseSteamWithParams(t *testing.T) {
	env := testutil.NewTestEnv(t)
	dir := env.RootDir()

	// Mock the actual ParseSteam function (ParseSteamFunc points to this)
	mockParseSteam := func() error {
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
	origParseSteamFunc := ParseSteamFunc
	ParseSteamFunc = mockParseSteam
	defer func() { ParseSteamFunc = origParseSteamFunc }() // Restore original after test

	jsonPath := filepath.Join(dir, "steam.json")
	if err := os.WriteFile(jsonPath, []byte("{}"), 0644); err != nil {
		t.Fatalf("write json path: %v", err)
	}

	// Call ParseSteamWithParams, which will internally call the mocked ParseSteamFunc
	if err := ParseSteamWithParams("123", "key", dir, true, jsonPath, true); err != nil {
		t.Fatalf("ParseSteamWithParams error = %v", err)
	}
}

