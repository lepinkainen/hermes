package imdb

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/lepinkainen/hermes/internal/errors"
	"github.com/spf13/viper"
)

func fetchMovieData(imdbID string) (*MovieSeen, error) {
	apiKey := viper.GetString("imdb.omdb_api_key")
	if apiKey == "" {
		return nil, fmt.Errorf("imdb.omdb_api_key not set in config")
	}

	slog.Info("Fetching movie data", "imdb_id", imdbID)

	url := fmt.Sprintf("http://www.omdbapi.com/?i=%s&apikey=%s", imdbID, apiKey)

	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch data: %w", err)
	}
	defer resp.Body.Close()

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
		return nil, fmt.Errorf("OMDB API returned non-200 status code: %d for ID: %s", resp.StatusCode, imdbID)
	}

	var omdbMovie OMDbMovie
	if err := json.NewDecoder(resp.Body).Decode(&omdbMovie); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Check if we got a valid response with actual data
	if omdbMovie.ImdbID == "" || omdbMovie.Title == "" {
		return nil, fmt.Errorf("invalid or empty response from OMDB API for ID: %s", imdbID)
	}

	// Enhanced conversion
	movie := &MovieSeen{
		Title:        omdbMovie.Title,
		ImdbId:       omdbMovie.ImdbID,
		TitleType:    omdbMovie.Type,
		IMDbRating:   parseFloat(omdbMovie.ImdbRating),
		Plot:         omdbMovie.Plot,
		PosterURL:    omdbMovie.Poster,
		ContentRated: omdbMovie.Rated,
		Awards:       omdbMovie.Awards,
		// Parse genres from comma-separated string
		Genres: strings.Split(omdbMovie.Genre, ", "),
		// Parse directors from comma-separated string
		Directors: strings.Split(omdbMovie.Director, ", "),
		// Parse runtime to minutes
		RuntimeMins: parseRuntime(omdbMovie.Runtime),
	}

	return movie, nil
}

// Helper functions to parse OMDB data
func parseFloat(s string) float64 {
	f, _ := strconv.ParseFloat(s, 64)
	return f
}

func parseRuntime(runtime string) int {
	// Convert "123 min" to 123
	mins := strings.TrimSuffix(runtime, " min")
	val, _ := strconv.Atoi(mins)
	return val
}
