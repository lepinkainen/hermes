package imdb

import (
	"github.com/lepinkainen/hermes/internal/omdb"
)

func getCachedMovie(imdbID string) (*MovieSeen, error) {
	movie, _, err := omdb.GetCached(imdbID, func() (*MovieSeen, error) {
		return fetchMovieData(imdbID)
	})
	return movie, err
}
