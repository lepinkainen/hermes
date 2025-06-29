package steam

import (
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
)

func getCachedGame(appID string) (*Game, *GameDetails, error) {
	cacheDir := "cache/steam"
	cachePath := filepath.Join(cacheDir, appID+".json")

	appIDInt, _ := strconv.Atoi(appID)

	// Check cache first
	if data, err := os.ReadFile(cachePath); err == nil {
		var details GameDetails
		if err := json.Unmarshal(data, &details); err == nil {
			// Ensure the cached details have the correct AppID
			details.AppID = appIDInt
			game := &Game{
				AppID:           appIDInt,
				Name:            details.Name,
				PlaytimeForever: details.PlaytimeForever,
				PlaytimeRecent:  details.PlaytimeRecent,
				LastPlayed:      details.LastPlayed,
				DetailsFetched:  true,
			}
			return game, &details, nil
		}
	}

	// Fetch from API if not in cache
	game, details, err := fetchGameData(appID)
	if err != nil {
		return nil, nil, err
	}

	// Ensure the AppID is set before caching
	details.AppID = appIDInt
	game.AppID = appIDInt

	// Cache the result
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		slog.Warn("Failed to create cache directory", "error", err)
	} else {
		data, _ := json.MarshalIndent(details, "", "  ")
		if err := os.WriteFile(cachePath, data, 0644); err != nil {
			slog.Warn("Failed to write cache file", "error", err)
		}
	}

	return game, details, nil
}
