package steam

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"

	"github.com/lepinkainen/hermes/internal/cache"
	"github.com/lepinkainen/hermes/internal/cmdutil"
	"github.com/lepinkainen/hermes/internal/errors"
	"github.com/spf13/viper"
)

const steamGamesSchema = `CREATE TABLE IF NOT EXISTS steam_games (
		appid INTEGER PRIMARY KEY,
		name TEXT,
		playtime_forever INTEGER,
		playtime_recent INTEGER,
		last_played TEXT,
		details_fetched BOOLEAN,
		description TEXT,
		short_desc TEXT,
		header_image TEXT,
		screenshots TEXT,
		developers TEXT,
		publishers TEXT,
		release_date TEXT,
		coming_soon BOOLEAN,
		categories TEXT,
		genres TEXT,
		metacritic_score INTEGER,
		metacritic_url TEXT
	)`

// Convert GameDetails to map[string]any for database insertion
func gameDetailsToMap(details GameDetails) map[string]any {
	return map[string]any{
		"appid":            details.AppID,
		"name":             details.Name,
		"playtime_forever": details.PlaytimeForever,
		"playtime_recent":  details.PlaytimeRecent,
		"last_played":      details.LastPlayed.String(),
		"details_fetched":  details.DetailsFetched,
		"description":      details.Description,
		"short_desc":       details.ShortDesc,
		"header_image":     details.HeaderImage,
		"screenshots":      "", // Could serialize as JSON if needed
		"developers":       strings.Join(details.Developers, ","),
		"publishers":       strings.Join(details.Publishers, ","),
		"release_date":     details.ReleaseDate.Date,
		"coming_soon":      details.ReleaseDate.ComingSoon,
		"categories":       "", // Could serialize as JSON if needed
		"genres":           "", // Could serialize as JSON if needed
		"metacritic_score": details.Metacritic.Score,
		"metacritic_url":   details.Metacritic.URL,
	}
}

func ParseSteam() error {
	// Ensure output directory exists
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("error creating output directory: %w", err)
	}

	games, err := ImportSteamGamesFunc(steamID, apiKey)
	if err != nil {
		return fmt.Errorf("error importing games: %w", err)
	}

	var processedGames []GameDetails

	for _, game := range games {
		slog.Debug("Fetching game details", "game", game.Name)

		appID := strconv.Itoa(game.AppID)
		// Use the generic cache utility with SQLite backend
		details, _, err := cache.GetOrFetch("steam_cache", appID, func() (*GameDetails, error) {
			_, detailsData, fetchErr := fetchGameData(appID)
			if fetchErr != nil {
				return nil, fetchErr
			}
			// Ensure the AppID is set before caching
			detailsData.AppID = game.AppID
			return detailsData, nil
		})
		if err != nil {
			if strings.Contains(err.Error(), "status code 429") {
				return errors.NewRateLimitError("Rate limit reached. Please try again later (usually after a few minutes)")
			}
			slog.Warn("Error fetching game details", "game", game.Name, "error", err)
			continue
		}

		// Ensure we preserve the original game's AppID and other important fields
		details.AppID = game.AppID
		details.PlaytimeForever = game.PlaytimeForever
		details.PlaytimeRecent = game.PlaytimeRecent
		details.LastPlayed = game.LastPlayed
		details.DetailsFetched = true

		if err := CreateMarkdownFile(game, details, outputDir); err != nil {
			slog.Error("Error creating markdown", "game", game.Name, "error", err)
			continue
		}

		processedGames = append(processedGames, *details)
	}

	// Datasette integration
	if err := cmdutil.WriteToDatastore(processedGames, steamGamesSchema, "steam_games", "Steam games", gameDetailsToMap); err != nil {
		return err
	}

	// Write to JSON if enabled
	if writeJSON {
		if err := writeGameToJson(processedGames, jsonOutput); err != nil {
			slog.Error("Error writing games to JSON", "error", err)
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
