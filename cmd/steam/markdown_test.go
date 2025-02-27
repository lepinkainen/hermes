package steam

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/lepinkainen/hermes/internal/config"
	"github.com/lepinkainen/hermes/internal/fileutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateMarkdownFile(t *testing.T) {
	// Setup test directory
	testDir := t.TempDir()

	// Force overwrite for testing
	config.SetOverwriteFiles(true)

	// Create test cases
	testCases := []struct {
		name     string
		game     Game
		details  *GameDetails
		wantFile string
	}{
		{
			name: "basic_game",
			game: Game{
				AppID:           12345,
				Name:            "Test Game",
				PlaytimeForever: 120, // 2 hours
			},
			details: &GameDetails{
				ReleaseDate: struct {
					ComingSoon bool   `json:"coming_soon"`
					Date       string `json:"date"`
				}{
					Date: "15 Jan, 2023",
				},
				HeaderImage: "https://example.com/header.jpg",
				Developers:  []string{"Developer One", "Developer Two"},
				Publishers:  []string{"Publisher One", "Publisher Two"},
				Categories: []Category{
					{Description: "Single-player"},
					{Description: "Multi-player"},
				},
				Genres: []Genre{
					{Description: "Action"},
					{Description: "Adventure"},
				},
				Description: "This is a test game description.",
				Metacritic: MetacriticData{
					Score: 85,
					URL:   "https://www.metacritic.com/game/test-game",
				},
				Screenshots: []Screenshot{
					{PathURL: "https://example.com/screenshot1.jpg"},
					{PathURL: "https://example.com/screenshot2.jpg"},
				},
			},
			wantFile: "basic_game.md",
		},
		// Add more test cases as needed
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create golden file path
			goldenFilePath := filepath.Join("testdata", tc.wantFile)

			// Write game to markdown in test directory
			err := CreateMarkdownFile(tc.game, tc.details, testDir)
			require.NoError(t, err)

			// Read the generated file
			generatedFilePath := filepath.Join(testDir, fileutil.SanitizeFilename(tc.game.Name)+".md")
			generated, err := os.ReadFile(generatedFilePath)
			require.NoError(t, err)

			// Check if we need to update golden files (useful during development)
			if os.Getenv("UPDATE_GOLDEN") == "true" {
				err = os.MkdirAll(filepath.Dir(goldenFilePath), 0755)
				require.NoError(t, err)
				err = os.WriteFile(goldenFilePath, generated, 0644)
				require.NoError(t, err)
			}

			// Read the golden file
			golden, err := os.ReadFile(goldenFilePath)
			require.NoError(t, err)

			// Compare generated content with golden file
			assert.Equal(t, string(golden), string(generated))
		})
	}
}
