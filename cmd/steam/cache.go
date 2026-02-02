package steam

import (
	"fmt"
	"strconv"
	"time"

	"github.com/lepinkainen/hermes/internal/cache"
	"github.com/lepinkainen/hermes/internal/errors"
)

// CachedOwnedGames wraps a list of owned games for caching
type CachedOwnedGames struct {
	Games []Game `json:"games"`
}

// GetCachedGameDetails fetches game details with caching
// Returns: details, fromCache, error
func GetCachedGameDetails(appID int) (*GameDetails, bool, error) {
	cacheKey := strconv.Itoa(appID)

	details, fromCache, err := cache.GetOrFetch("steam_cache", cacheKey, func() (*GameDetails, error) {
		detailsData, fetchErr := getGameDetails(appID)
		if fetchErr != nil {
			return nil, fetchErr
		}
		// Ensure the AppID is set before caching
		detailsData.AppID = appID
		return detailsData, nil
	})

	return details, fromCache, err
}

// GetCachedOwnedGames fetches owned games list with 24-hour caching
// Returns: games, fromCache, error
func GetCachedOwnedGames(steamID string, apiKey string) ([]Game, bool, error) {
	// Cache key format: steamid
	cacheKey := steamID

	// Use 24-hour TTL for owned games list
	ttl := 24 * time.Hour

	cached, fromCache, err := cache.GetOrFetchWithTTL(
		"steam_owned_games_cache",
		cacheKey,
		func() (*CachedOwnedGames, error) {
			// Use ImportSteamGamesFunc to allow mocking in tests
			games, fetchErr := ImportSteamGamesFunc(steamID, apiKey)
			if fetchErr != nil {
				return nil, fetchErr
			}
			return &CachedOwnedGames{Games: games}, nil
		},
		func(_ *CachedOwnedGames) time.Duration {
			return ttl
		},
	)

	if err != nil {
		return nil, false, err
	}

	return cached.Games, fromCache, nil
}

// getCachedAchievements fetches achievements with negative caching
// Returns: achievements, fromCache, error
func getCachedAchievements(steamID string, apiKey string, appID int) ([]Achievement, bool, error) {
	// Cache key format: steamid_appid
	cacheKey := fmt.Sprintf("%s_%d", steamID, appID)

	// Use negative caching with TTL selector
	cached, fromCache, err := cache.GetOrFetchWithTTL(
		"steam_achievements_cache",
		cacheKey,
		func() (*CachedAchievements, error) {
			achievements, fetchErr := GetPlayerAchievements(steamID, apiKey, appID)
			if fetchErr != nil {
				// Profile access errors should not be cached (user might fix config)
				if errors.IsSteamProfileError(fetchErr) {
					return nil, fetchErr
				}
				// Network/other errors also shouldn't be cached
				return nil, fetchErr
			}

			// GetPlayerAchievements returns nil for "no achievements"
			if achievements == nil {
				return &CachedAchievements{
					Achievements:   nil,
					NoAchievements: true,
				}, nil
			}

			// Cache the successful response
			return &CachedAchievements{
				Achievements:   achievements,
				NoAchievements: false,
			}, nil
		},
		cache.SelectNegativeCacheTTL(func(r *CachedAchievements) bool {
			return r.NoAchievements
		}),
	)

	if err != nil {
		return nil, false, err
	}

	// Return empty slice for "no achievements" case
	if cached.NoAchievements {
		return []Achievement{}, fromCache, nil
	}

	return cached.Achievements, fromCache, nil
}
