package letterboxd

import (
	"context"
	"fmt"

	"github.com/lepinkainen/hermes/internal/enrichment/omdb"
	"github.com/lepinkainen/hermes/internal/parseutil"
)

// fetchMovieData retrieves movie data from the OMDB API by title and year
func fetchMovieData(title string, year int) (*Movie, error) {
	resp, err := omdb.FetchByTitleYear(context.Background(), title, year)
	if err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, fmt.Errorf("movie not found in OMDB for title: %s (%d)", title, year)
	}

	return omdbResponseToMovie(resp), nil
}

// omdbResponseToMovie maps an OMDB API response onto the importer's Movie model.
func omdbResponseToMovie(resp *omdb.OMDBResponse) *Movie {
	return &Movie{
		Name:        resp.Title,
		Year:        parseutil.ParseYear(resp.Year),
		Director:    resp.Director,
		Cast:        parseutil.ParseCommaList(resp.Actors),
		Genres:      parseutil.ParseCommaList(resp.Genre),
		Runtime:     parseutil.ParseRuntime(resp.Runtime),
		Rating:      parseutil.ParseFloat(resp.ImdbRating),
		PosterURL:   resp.Poster,
		Description: resp.Plot,
		ImdbID:      resp.ImdbID,
	}
}
