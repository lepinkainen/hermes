package letterboxd

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/lepinkainen/hermes/internal/cache"
	"github.com/lepinkainen/hermes/internal/errors"
)

// getCachedMovie retrieves movie data from cache or OMDB API
func getCachedMovie(title string, year int) (*Movie, error) {
	cacheKey := fmt.Sprintf("%s_%d", strings.ToLower(strings.TrimSpace(title)), year)

	if !omdbRequestsAllowed() {
		return nil, errors.NewRateLimitError("OMDB API request limit reached")
	}

	// Use the generic cache utility with SQLite backend
	movie, fromCache, err := cache.GetOrFetch("omdb_cache", cacheKey, func() (*Movie, error) {
		// Fetch from API when cache is missing/expired
		movieData, fetchErr := fetchMovieData(title, year)
		if fetchErr != nil {
			// Check if it's a rate limit error
			if errors.IsRateLimitError(fetchErr) {
				slog.Warn("OMDB API rate limit reached; skipping further OMDB requests for this run")
				markOmdbRateLimitReached()
				return nil, fetchErr
			}
			slog.Warn("Failed to enrich movie", "error", fetchErr)
			return nil, fetchErr
		}

		return movieData, nil
	})

	// Seed the omdb_cache by IMDb ID too so IMDb importer can reuse the data
	if !fromCache && movie != nil && movie.ImdbID != "" {
		if cacheDB, dbErr := cache.GetGlobalCache(); dbErr == nil {
			if movieJSON, marshalErr := json.Marshal(movie); marshalErr == nil {
				if setErr := cacheDB.Set("omdb_cache", movie.ImdbID, string(movieJSON)); setErr != nil {
					slog.Warn("Failed to seed OMDB cache by IMDb ID", "imdb_id", movie.ImdbID, "error", setErr)
				}
			} else {
				slog.Warn("Failed to marshal movie for IMDb cache seed", "imdb_id", movie.ImdbID, "error", marshalErr)
			}
		} else {
			slog.Warn("Failed to seed OMDB cache by IMDb ID", "imdb_id", movie.ImdbID, "error", dbErr)
		}
	}

	return movie, err
}
