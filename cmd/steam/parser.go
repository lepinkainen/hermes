package steam

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"

	"github.com/lepinkainen/hermes/internal/datastore"
	"github.com/lepinkainen/hermes/internal/errors"
	"github.com/spf13/viper"
)

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

	games, err := ImportSteamGames(steamID, apiKey)
	if err != nil {
		return fmt.Errorf("error importing games: %w", err)
	}

	var processedGames []GameDetails

	for _, game := range games {
		slog.Debug("Fetching game details", "game", game.Name)

		_, details, err := getCachedGame(strconv.Itoa(game.AppID))
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
	if viper.GetBool("datasette.enabled") {
		slog.Info("Writing Steam games to Datasette")
		mode := viper.GetString("datasette.mode")

		switch mode {
		case "local":
			store := datastore.NewSQLiteStore(viper.GetString("datasette.dbfile"))
			if err := store.Connect(); err != nil {
				slog.Error("Failed to connect to SQLite database", "error", err)
				return err
			}
			defer func() { _ = store.Close() }()

			schema := `CREATE TABLE IF NOT EXISTS steam_games (
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

			if err := store.CreateTable(schema); err != nil {
				slog.Error("Failed to create table", "error", err)
				return err
			}

			records := make([]map[string]any, len(processedGames))
			for i, details := range processedGames {
				records[i] = gameDetailsToMap(details)
			}

			if err := store.BatchInsert("hermes", "steam_games", records); err != nil {
				slog.Error("Failed to insert records", "error", err)
				return err
			}
			slog.Info("Successfully wrote games to SQLite database", "count", len(processedGames))
		case "remote":
			client := datastore.NewDatasetteClient(
				viper.GetString("datasette.remote_url"),
				viper.GetString("datasette.api_token"),
			)
			if err := client.Connect(); err != nil {
				slog.Error("Failed to connect to remote Datasette", "error", err)
				return err
			}
			defer func() { _ = client.Close() }()

			records := make([]map[string]any, len(processedGames))
			for i, details := range processedGames {
				records[i] = gameDetailsToMap(details)
			}

			if err := client.BatchInsert("hermes", "steam_games", records); err != nil {
				slog.Error("Failed to insert records to remote Datasette", "error", err)
				return err
			}
			slog.Info("Successfully wrote games to remote Datasette", "count", len(processedGames))
		default:
			slog.Error("Invalid Datasette mode", "mode", mode)
			return fmt.Errorf("invalid Datasette mode: %s", mode)
		}
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
