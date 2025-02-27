package steam

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/lepinkainen/hermes/internal/errors"
	"github.com/spf13/viper"

	log "github.com/sirupsen/logrus"
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

	var processedGames []GameDetails

	for _, game := range games {
		log.Debugf("Fetching details for: %s\n", game.Name)

		_, details, err := getCachedGame(strconv.Itoa(game.AppID))
		if err != nil {
			if strings.Contains(err.Error(), "status code 429") {
				return errors.NewRateLimitError("Rate limit reached. Please try again later (usually after a few minutes)")
			}
			log.Warnf("Error fetching details for %s: %v\n", game.Name, err)
			continue
		}

		// Ensure we preserve the original game's AppID and other important fields
		details.AppID = game.AppID
		details.PlaytimeForever = game.PlaytimeForever
		details.PlaytimeRecent = game.PlaytimeRecent
		details.LastPlayed = game.LastPlayed
		details.DetailsFetched = true

		if err := CreateMarkdownFile(game, details, outputDir); err != nil {
			log.Errorf("Error creating markdown for %s: %v\n", game.Name, err)
			continue
		}

		processedGames = append(processedGames, *details)
		//log.Infof("Created markdown file for: %s\n", game.Name)
	}

	// Write to JSON if enabled
	if writeJSON {
		if err := writeGameToJson(processedGames, jsonOutput); err != nil {
			log.Errorf("Error writing games to JSON: %v\n", err)
		}
	}

	return nil
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

	// Create a game object with the correct AppID and other details
	game := &Game{
		AppID:           appIDInt,
		Name:            details.Name,
		PlaytimeForever: details.PlaytimeForever,
		PlaytimeRecent:  details.PlaytimeRecent,
		LastPlayed:      details.LastPlayed,
		DetailsFetched:  true,
	}

	// Ensure the details also have the correct AppID
	details.AppID = appIDInt

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
