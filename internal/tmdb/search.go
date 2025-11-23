package tmdb

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
)

// SearchMovies performs a movie-specific search on TMDB.
// If year > 0, it is passed as a hint to TMDB (but results are otherwise left in API order).
func (c *Client) SearchMovies(ctx context.Context, query string, year int, limit int) ([]SearchResult, error) {
	if limit <= 0 {
		limit = 1
	}

	params := url.Values{}
	params.Set("api_key", c.apiKey)
	params.Set("query", query)
	params.Set("include_adult", "false")
	if year > 0 {
		params.Set("year", strconv.Itoa(year))
	}

	endpoint := fmt.Sprintf("%s/search/movie?%s", c.baseURL, params.Encode())

	var response struct {
		Results []struct {
			ID               int     `json:"id"`
			Title            string  `json:"title"`
			PosterPath       string  `json:"poster_path"`
			Overview         string  `json:"overview"`
			ReleaseDate      string  `json:"release_date"`
			VoteAverage      float64 `json:"vote_average"`
			VoteCount        int     `json:"vote_count"`
			Popularity       float64 `json:"popularity"`
			Runtime          int     `json:"runtime"`
			OriginalLanguage string  `json:"original_language"`
		} `json:"results"`
	}

	if err := c.getJSON(ctx, endpoint, &response); err != nil {
		return nil, err
	}

	results := make([]SearchResult, 0, limit)

	for _, item := range response.Results {
		if len(results) >= limit {
			break
		}
		// Filter out results with 0.0 score (upcoming/unrated movies)
		if item.VoteAverage == 0.0 {
			continue
		}

		results = append(results, SearchResult{
			ID:           item.ID,
			MediaType:    "movie",
			Title:        item.Title,
			PosterPath:   item.PosterPath,
			Overview:     item.Overview,
			ReleaseDate:  item.ReleaseDate,
			VoteAverage:  item.VoteAverage,
			VoteCount:    item.VoteCount,
			Popularity:   item.Popularity,
			Runtime:      item.Runtime,
			OriginalLang: item.OriginalLanguage,
		})
	}

	return results, nil
}

// SearchMulti performs a multi-search on TMDB for movies and TV shows.
// If year > 0, it is passed as a hint to TMDB (but results are otherwise left in API order).
func (c *Client) SearchMulti(ctx context.Context, query string, year int, limit int) ([]SearchResult, error) {
	if limit <= 0 {
		limit = 1
	}

	params := url.Values{}
	params.Set("api_key", c.apiKey)
	params.Set("query", query)
	params.Set("include_adult", "false")
	if year > 0 {
		params.Set("year", strconv.Itoa(year))
		params.Set("first_air_date_year", strconv.Itoa(year))
	}

	endpoint := fmt.Sprintf("%s/search/multi?%s", c.baseURL, params.Encode())

	var response struct {
		Results []struct {
			ID               int     `json:"id"`
			MediaType        string  `json:"media_type"`
			Title            string  `json:"title"`
			Name             string  `json:"name"`
			PosterPath       string  `json:"poster_path"`
			Overview         string  `json:"overview"`
			ReleaseDate      string  `json:"release_date"`
			FirstAirDate     string  `json:"first_air_date"`
			VoteAverage      float64 `json:"vote_average"`
			VoteCount        int     `json:"vote_count"`
			Popularity       float64 `json:"popularity"`
			Runtime          int     `json:"runtime"`
			OriginalLanguage string  `json:"original_language"`
		} `json:"results"`
	}

	if err := c.getJSON(ctx, endpoint, &response); err != nil {
		return nil, err
	}

	results := make([]SearchResult, 0, limit)

	for _, item := range response.Results {
		if len(results) >= limit {
			break
		}
		if item.MediaType != "movie" && item.MediaType != "tv" {
			continue
		}
		// Filter out results with 0.0 score (upcoming/unrated movies)
		if item.VoteAverage == 0.0 {
			continue
		}

		results = append(results, SearchResult{
			ID:           item.ID,
			MediaType:    item.MediaType,
			Title:        item.Title,
			Name:         item.Name,
			PosterPath:   item.PosterPath,
			Overview:     item.Overview,
			ReleaseDate:  item.ReleaseDate,
			FirstAirDate: item.FirstAirDate,
			VoteAverage:  item.VoteAverage,
			VoteCount:    item.VoteCount,
			Popularity:   item.Popularity,
			Runtime:      item.Runtime,
			OriginalLang: item.OriginalLanguage,
		})
	}

	return results, nil
}

// FindByIMDBID finds a TMDB entry by its IMDB ID using the /find endpoint.
// Returns the TMDB ID, media type ("movie" or "tv"), and any error.
// Returns (0, "", nil) if no match is found.
func (c *Client) FindByIMDBID(ctx context.Context, imdbID string) (int, string, error) {
	if imdbID == "" {
		return 0, "", nil
	}

	params := url.Values{}
	params.Set("api_key", c.apiKey)
	params.Set("external_source", "imdb_id")

	endpoint := fmt.Sprintf("%s/find/%s?%s", c.baseURL, url.PathEscape(imdbID), params.Encode())

	var response struct {
		MovieResults []struct {
			ID int `json:"id"`
		} `json:"movie_results"`
		TVResults []struct {
			ID int `json:"id"`
		} `json:"tv_results"`
	}

	if err := c.getJSON(ctx, endpoint, &response); err != nil {
		return 0, "", err
	}

	// Prefer movie results over TV results
	if len(response.MovieResults) > 0 {
		return response.MovieResults[0].ID, "movie", nil
	}
	if len(response.TVResults) > 0 {
		return response.TVResults[0].ID, "tv", nil
	}

	return 0, "", nil
}
