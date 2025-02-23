package steam

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/spf13/viper"
)

func ParseSteam() error {
	// Ensure output directory exists
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("error creating output directory: %w", err)
	}

	games, err := ImportSteamGames(steamID, apiKey)
	if err != nil {
		return fmt.Errorf("error importing games: %w", err)
	}

	for _, game := range games {
		//filePath := getGameFilePath(game.Name, outputDir)

		// Check if file already exists
		//if _, err := os.Stat(filePath); err == nil {
		//	fmt.Printf("Skipping %s: File already exists\n", game.Name)
		//	continue
		//}

		fmt.Printf("Fetching details for: %s\n", game.Name)

		_, details, err := getCachedGame(strconv.Itoa(game.AppID))
		if err != nil {
			if strings.Contains(err.Error(), "status code 429") {
				return fmt.Errorf("rate limit reached. Please try again later (usually after a few minutes)")
			}
			fmt.Printf("Error fetching details for %s: %v\n", game.Name, err)
			continue
		}

		if err := CreateMarkdownFile(game, details, outputDir); err != nil {
			fmt.Printf("Error creating markdown for %s: %v\n", game.Name, err)
			continue
		}

		fmt.Printf("Created markdown file for: %s\n", game.Name)
		fmt.Println("---")
	}

	return nil
}

// Helper function to get the expected file path for a game
func getGameFilePath(gameName string, directory string) string {
	// Clean the filename first
	filename := sanitizeFilename(gameName)
	return filepath.Join(directory, filename+".md")
}

// Helper function to sanitize filename
func sanitizeFilename(name string) string {
	// Replace problematic characters
	name = strings.ReplaceAll(name, ":", " - ")
	name = strings.ReplaceAll(name, "/", "-")
	return name
}

func fetchGameData(appID string) (*Game, *GameDetails, error) {
	apiKey := viper.GetString("steam.apikey")
	if apiKey == "" {
		return nil, nil, fmt.Errorf("steam.apikey not set in config")
	}

	appIDInt, _ := strconv.Atoi(appID)
	details, err := GetGameDetails(appIDInt)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get game details: %w", err)
	}

	game := &Game{
		AppID:          details.AppID,
		Name:           details.Name,
		DetailsFetched: true,
	}

	return game, details, nil
}

func enrichGameData(game *Game) error {
	// Skip if we already have enriched data
	if game.DetailsFetched {
		return nil
	}

	enriched, _, err := getCachedGame(strconv.Itoa(game.AppID))
	if err != nil {
		return fmt.Errorf("failed to enrich game data: %w", err)
	}

	// Copy enriched data
	*game = *enriched
	game.DetailsFetched = true
	return nil
}
