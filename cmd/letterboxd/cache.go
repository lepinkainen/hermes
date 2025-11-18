package letterboxd

import (
	"fmt"
	"strings"

	"github.com/lepinkainen/hermes/internal/omdb"
)

// getCachedMovie retrieves movie data from cache or OMDB API
func getCachedMovie(title string, year int) (*Movie, error) {
	cacheKey := fmt.Sprintf("%s_%d", strings.ToLower(strings.TrimSpace(title)), year)

	movie, _, err := omdb.GetCached(cacheKey, func() (*Movie, error) {
		return fetchMovieData(title, year)
	})

	// Note: We don't seed by IMDb ID because letterboxd.Movie and imdb.MovieSeen
	// are different struct types that can't be deserialized interchangeably.

	return movie, err
}
