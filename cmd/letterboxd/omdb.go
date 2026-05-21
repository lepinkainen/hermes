package letterboxd

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/lepinkainen/hermes/internal/errors"
	"github.com/lepinkainen/hermes/internal/parseutil"
	"github.com/spf13/viper"
)

// OMDbMovie represents a movie from the OMDB API
type OMDbMovie struct {
	Title      string   `json:"Title"`
	Year       string   `json:"Year"`
	Rated      string   `json:"Rated"`
	Released   string   `json:"Released"`
	Runtime    string   `json:"Runtime"`
	Genre      string   `json:"Genre"`
	Director   string   `json:"Director"`
	Writer     string   `json:"Writer"`
	Actors     string   `json:"Actors"`
	Plot       string   `json:"Plot"`
	Language   string   `json:"Language"`
	Country    string   `json:"Country"`
	Awards     string   `json:"Awards"`
	Poster     string   `json:"Poster"`
	Ratings    []Rating `json:"Ratings"`
	ImdbRating string   `json:"imdbRating"`
	ImdbVotes  string   `json:"imdbVotes"`
	ImdbID     string   `json:"imdbID"`
	Type       string   `json:"Type"`
}

// Rating represents a rating from a specific source
type Rating struct {
	Source string `json:"Source"`
	Value  string `json:"Value"`
}

// HTTP seams — package vars so tests can redirect to an httptest.Server.
var (
	omdbBaseURL = "http://www.omdbapi.com"
	omdbHTTPGet = http.Get
)

// fetchMovieData retrieves movie data from the OMDB API by title and year
func fetchMovieData(title string, year int) (*Movie, error) {
	apiKey := viper.GetString("omdb.api_key")
	if apiKey == "" {
		apiKey = viper.GetString("imdb.omdb_api_key") // Fallback to imdb key
		if apiKey == "" {
			return nil, fmt.Errorf("omdb.api_key or imdb.omdb_api_key not set in config")
		}
	}

	slog.Info("Fetching OMDB data", "title", title, "year", year)

	// Encode the title for URL
	escapedTitle := strings.ReplaceAll(title, " ", "+")

	url := fmt.Sprintf("%s/?t=%s&y=%d&apikey=%s", omdbBaseURL, escapedTitle, year, apiKey)

	resp, err := omdbHTTPGet(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch data: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Check HTTP status code
	if resp.StatusCode != http.StatusOK {
		// Read the response body
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			slog.Warn("Failed to read error response body", "error", err)
		} else {
			// Parse error response
			var errorResp struct {
				Response string `json:"Response"`
				Error    string `json:"Error"`
			}
			if err := json.Unmarshal(body, &errorResp); err == nil {
				if errorResp.Error == "Request limit reached!" {
					return nil, errors.NewRateLimitError("OMDB API request limit reached")
				}
				slog.Warn("OMDB API error", "error", errorResp.Error)
			}
			slog.Warn("OMDB API response body", "body", string(body))
		}
		return nil, fmt.Errorf("OMDB API returned non-200 status code: %d for title: %s (%d)", resp.StatusCode, title, year)
	}

	var omdbMovie OMDbMovie
	if err := json.NewDecoder(resp.Body).Decode(&omdbMovie); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Check if we got a valid response with actual data
	if omdbMovie.Title == "" {
		return nil, fmt.Errorf("invalid or empty response from OMDB API for title: %s (%d)", title, year)
	}

	// Create a new Movie with enriched data
	movie := &Movie{
		Name:        omdbMovie.Title,
		Year:        parseutil.ParseYear(omdbMovie.Year),
		Director:    omdbMovie.Director,
		Cast:        parseutil.ParseCommaList(omdbMovie.Actors),
		Genres:      parseutil.ParseCommaList(omdbMovie.Genre),
		Runtime:     parseutil.ParseRuntime(omdbMovie.Runtime),
		Rating:      parseutil.ParseFloat(omdbMovie.ImdbRating),
		PosterURL:   omdbMovie.Poster,
		Description: omdbMovie.Plot,
		ImdbID:      omdbMovie.ImdbID,
	}

	return movie, nil
}
