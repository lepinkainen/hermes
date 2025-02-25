package steam

import (
	"encoding/json"
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
	os.MkdirAll(cacheDir, 0755)
	data, _ := json.MarshalIndent(details, "", "  ")
	os.WriteFile(cachePath, data, 0644)

	return game, details, nil
}
