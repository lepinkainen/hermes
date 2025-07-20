package imdb

import (
	"log/slog"

	"github.com/lepinkainen/hermes/internal/cache"
	"github.com/lepinkainen/hermes/internal/errors"
)

func getCachedMovie(imdbID string) (*MovieSeen, error) {
	cacheDir := "cache/omdb"

	// Use the generic cache utility
	movie, _, err := cache.GetOrFetch(cacheDir, imdbID, func() (*MovieSeen, error) {
		movieData, fetchErr := fetchMovieData(imdbID)
		if fetchErr != nil {
			// Check if it's a rate limit error
			if _, isRateLimit := fetchErr.(*errors.RateLimitError); isRateLimit {
				slog.Warn("OMDB API rate limit reached, stopping further requests")
				return nil, fetchErr
			}
			slog.Warn("Failed to enrich movie", "error", fetchErr)
			return nil, fetchErr
		}
		return movieData, nil
	})

	return movie, err
}
