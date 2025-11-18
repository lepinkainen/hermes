package omdb

import (
	"encoding/json"
	"log/slog"

	"github.com/lepinkainen/hermes/internal/cache"
	"github.com/lepinkainen/hermes/internal/errors"
)

// GetCached retrieves data from the OMDB cache, checking rate limits.
// The fetcher function is called if the data is not in the cache.
// Returns the data, whether it was from cache, and any error.
func GetCached[T any](cacheKey string, fetcher func() (*T, error)) (*T, bool, error) {
	if !RequestsAllowed() {
		return nil, false, errors.NewRateLimitError("OMDB API request limit reached")
	}

	data, fromCache, err := cache.GetOrFetch("omdb_cache", cacheKey, func() (*T, error) {
		result, fetchErr := fetcher()
		if fetchErr != nil {
			if errors.IsRateLimitError(fetchErr) {
				MarkRateLimitReached()
				return nil, fetchErr
			}
			slog.Warn("Failed to fetch from OMDB", "error", fetchErr)
			return nil, fetchErr
		}
		return result, nil
	})

	if errors.IsRateLimitError(err) {
		MarkRateLimitReached()
	}

	return data, fromCache, err
}

// SeedCacheByID stores data in the cache by an additional IMDb ID key.
// This allows data fetched by title/year to be found by IMDb ID later.
func SeedCacheByID(imdbID string, data any) error {
	if imdbID == "" {
		return nil
	}

	cacheDB, err := cache.GetGlobalCache()
	if err != nil {
		slog.Warn("Failed to get cache for seeding", "imdb_id", imdbID, "error", err)
		return err
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		slog.Warn("Failed to marshal data for cache seeding", "imdb_id", imdbID, "error", err)
		return err
	}

	if err := cacheDB.Set("omdb_cache", imdbID, string(jsonData)); err != nil {
		slog.Warn("Failed to seed OMDB cache by IMDb ID", "imdb_id", imdbID, "error", err)
		return err
	}

	return nil
}
