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
		{
			name: "complex_game",
			game: Game{
				AppID:           987654,
				Name:            "Epic RPG: Deluxe Edition",
				PlaytimeForever: 3630, // 60.5 hours
			},
			details: &GameDetails{
				ReleaseDate: struct {
					ComingSoon bool   `json:"coming_soon"`
					Date       string `json:"date"`
				}{
					Date: "10 November, 2022",
				},
				HeaderImage: "https://example.com/epic_rpg_header.jpg",
				Developers:  []string{"Legendary Game Studio", "Fantasy Interactive", "RPG Masters Inc."},
				Publishers:  []string{"Premium Publisher", "Global Games Distribution"},
				Categories: []Category{
					{Description: "Single-player"},
					{Description: "Multi-player"},
					{Description: "Co-op"},
					{Description: "Online Co-op"},
					{Description: "Steam Achievements"},
					{Description: "Full Controller Support"},
					{Description: "Steam Cloud"},
					{Description: "Steam Workshop"},
					{Description: "Steam Trading Cards"},
					{Description: "Captions available"},
					{Description: "VR Support"},
				},
				Genres: []Genre{
					{Description: "RPG"},
					{Description: "Open World"},
					{Description: "Fantasy"},
					{Description: "Adventure"},
					{Description: "Action"},
					{Description: "Story Rich"},
				},
				Description: "Embark on an epic journey through a vast open world filled with danger, mystery, and adventure. Create your character, choose your class, and forge your own path through a living, breathing fantasy realm where your choices matter and shape the world around you.\n\nWith hundreds of quests, countless customization options, and a deep, engaging combat system, Epic RPG Deluxe Edition offers over 100 hours of immersive gameplay. Features all previously released DLC content, including the critically acclaimed 'Realm of Shadows' expansion.\n\n• Massive open world with diverse environments\n• Complex character development system with 12 character classes\n• Real-time combat with tactical pause feature\n• Dynamic weather and day/night cycle\n• Over 1,000 unique items and equipment pieces\n• Advanced crafting system\n• Player housing with customization\n• Companion system with relationship development\n• Multiple endings based on player choices",
				Metacritic: MetacriticData{
					Score: 94,
					URL:   "https://www.metacritic.com/game/pc/epic-rpg-deluxe-edition",
				},
				Screenshots: []Screenshot{
					{PathURL: "https://example.com/epic_rpg_screenshot1.jpg"},
					{PathURL: "https://example.com/epic_rpg_screenshot2.jpg"},
					{PathURL: "https://example.com/epic_rpg_screenshot3.jpg"},
					{PathURL: "https://example.com/epic_rpg_screenshot4.jpg"},
					{PathURL: "https://example.com/epic_rpg_screenshot5.jpg"},
				},
			},
			wantFile: "complex_game.md",
		},
		{
			name: "minimal_game",
			game: Game{
				AppID:           55555,
				Name:            "Indie Puzzle",
				PlaytimeForever: 30, // 30 minutes
			},
			details: &GameDetails{
				ReleaseDate: struct {
					ComingSoon bool   `json:"coming_soon"`
					Date       string `json:"date"`
				}{
					Date: "2023-08-01",
				},
				HeaderImage: "https://example.com/indie_puzzle_header.jpg",
				Developers:  []string{"Solo Developer"},
				Categories: []Category{
					{Description: "Single-player"},
				},
				Genres: []Genre{
					{Description: "Puzzle"},
					{Description: "Indie"},
				},
				Description: "A minimalist puzzle game.",
			},
			wantFile: "minimal_game.md",
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
