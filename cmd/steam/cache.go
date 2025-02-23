package steam

import (
	"encoding/json"
	"os"
	"path/filepath"
)

func getCachedGame(appID string) (*Game, *GameDetails, error) {
	cacheDir := "cache/steam"
	cachePath := filepath.Join(cacheDir, appID+".json")

	// Check cache first
	if data, err := os.ReadFile(cachePath); err == nil {
		var details GameDetails
		if err := json.Unmarshal(data, &details); err == nil {
			game := &Game{
				AppID:          details.AppID,
				Name:           details.Name,
				DetailsFetched: true,
			}
			return game, &details, nil
		}
	}

	// Fetch from API if not in cache
	game, details, err := fetchGameData(appID)
	if err != nil {
		return nil, nil, err
	}

	// Cache the result
	os.MkdirAll(cacheDir, 0755)
	data, _ := json.Marshal(details)
	os.WriteFile(cachePath, data, 0644)

	return game, details, nil
}
