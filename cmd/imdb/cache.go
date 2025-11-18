package imdb

import (
	"log/slog"

	"github.com/lepinkainen/hermes/internal/cache"
	"github.com/lepinkainen/hermes/internal/errors"
)

func getCachedMovie(imdbID string) (*MovieSeen, error) {
	if !omdbRequestsAllowed() {
		return nil, errors.NewRateLimitError("OMDB API request limit reached")
	}

	// Use the generic cache utility with SQLite backend
	movie, _, err := cache.GetOrFetch("omdb_cache", imdbID, func() (*MovieSeen, error) {
		movieData, fetchErr := fetchMovieData(imdbID)
		if fetchErr != nil {
			// Check if it's a rate limit error
			if _, isRateLimit := fetchErr.(*errors.RateLimitError); isRateLimit {
				slog.Warn("OMDB API rate limit reached; skipping further OMDB requests for this run")
				markOmdbRateLimitReached()
				return nil, fetchErr
			}
			slog.Warn("Failed to enrich movie", "error", fetchErr)
			return nil, fetchErr
		}
		return movieData, nil
	})

	if _, isRateLimit := err.(*errors.RateLimitError); isRateLimit {
		markOmdbRateLimitReached()
	}

	return movie, err
}
