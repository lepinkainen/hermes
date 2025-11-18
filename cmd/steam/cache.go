package steam

import (
	"strconv"

	"github.com/lepinkainen/hermes/internal/cache"
)

func getCachedGame(appID string) (*Game, *GameDetails, error) {
	appIDInt, _ := strconv.Atoi(appID)

	// Use the generic cache utility with SQLite backend
	details, _, err := cache.GetOrFetch("steam_cache", appID, func() (*GameDetails, error) {
		_, detailsData, fetchErr := fetchGameData(appID)
		if fetchErr != nil {
			return nil, fetchErr
		}
		// Ensure the AppID is set before caching
		detailsData.AppID = appIDInt
		return detailsData, nil
	})
	if err != nil {
		return nil, nil, err
	}

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

	return game, details, nil
}
