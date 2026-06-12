package imdb

import (
	"context"
	"fmt"

	"github.com/lepinkainen/hermes/internal/enrichment/omdb"
	"github.com/lepinkainen/hermes/internal/parseutil"
)

func fetchMovieData(imdbID string) (*MovieSeen, error) {
	return fetchMovieDataWithContext(context.Background(), imdbID)
}

func fetchMovieDataWithContext(ctx context.Context, imdbID string) (*MovieSeen, error) {
	resp, err := omdb.FetchByIMDBID(ctx, imdbID)
	if err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, fmt.Errorf("movie not found in OMDB for ID: %s", imdbID)
	}

	return omdbResponseToMovie(resp), nil
}

// omdbResponseToMovie maps an OMDB API response onto the importer's MovieSeen model.
func omdbResponseToMovie(resp *omdb.OMDBResponse) *MovieSeen {
	return &MovieSeen{
		Title:        resp.Title,
		ImdbId:       resp.ImdbID,
		TitleType:    resp.Type,
		IMDbRating:   parseutil.ParseFloat(resp.ImdbRating),
		Plot:         resp.Plot,
		PosterURL:    resp.Poster,
		ContentRated: resp.Rated,
		Awards:       resp.Awards,
		Genres:       parseutil.ParseCommaList(resp.Genre),
		Directors:    parseutil.ParseCommaList(resp.Director),
		RuntimeMins:  parseutil.ParseRuntime(resp.Runtime),
	}
}
