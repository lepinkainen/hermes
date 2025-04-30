package letterboxd

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/lepinkainen/hermes/internal/errors"
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

	url := fmt.Sprintf("http://www.omdbapi.com/?t=%s&y=%d&apikey=%s", escapedTitle, year, apiKey)

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

	// If the movie has an IMDB ID, also cache it in the OMDB cache format
	// This will benefit future IMDB imports as well as Letterboxd imports
	if omdbMovie.ImdbID != "" {
		// We should save this to the OMDB cache as well
		omdbCacheDir := "cache/omdb"
		omdbCachePath := filepath.Join(omdbCacheDir, omdbMovie.ImdbID+".json")

		// Only if it doesn't already exist
		if _, err := os.Stat(omdbCachePath); os.IsNotExist(err) {
			// Create a structure that's compatible with the IMDB importer
			imdbMovie := struct {
				Title        string   `json:"Title"`
				ImdbId       string   `json:"ImdbId"`
				Plot         string   `json:"Plot"`
				PosterURL    string   `json:"Poster URL"`
				ContentRated string   `json:"Content Rated"`
				Awards       string   `json:"Awards"`
				Genres       []string `json:"Genres"`
				Directors    []string `json:"Directors"`
				RuntimeMins  int      `json:"Runtime (mins)"`
				IMDbRating   float64  `json:"IMDb Rating"`
			}{
				Title:        omdbMovie.Title,
				ImdbId:       omdbMovie.ImdbID,
				Plot:         omdbMovie.Plot,
				PosterURL:    omdbMovie.Poster,
				ContentRated: omdbMovie.Rated,
				Awards:       omdbMovie.Awards,
				Genres:       parseCommaList(omdbMovie.Genre),
				Directors:    parseCommaList(omdbMovie.Director),
				RuntimeMins:  parseRuntime(omdbMovie.Runtime),
				IMDbRating:   parseFloat(omdbMovie.ImdbRating),
			}

			// Save to OMDB cache
			os.MkdirAll(omdbCacheDir, 0755)
			imdbData, _ := json.MarshalIndent(imdbMovie, "", "  ")
			if err := os.WriteFile(omdbCachePath, imdbData, 0644); err != nil {
				slog.Warn("Failed to save to OMDB cache", "error", err)
			} else {
				slog.Info("Cached movie in OMDB cache", "title", title, "imdb_id", omdbMovie.ImdbID)
			}
		}
	}

	// Create a new Movie with enriched data
	movie := &Movie{
		Name:        omdbMovie.Title,
		Year:        parseYear(omdbMovie.Year),
		Director:    omdbMovie.Director,
		Cast:        parseCommaList(omdbMovie.Actors),
		Genres:      parseCommaList(omdbMovie.Genre),
		Runtime:     parseRuntime(omdbMovie.Runtime),
		Rating:      parseFloat(omdbMovie.ImdbRating),
		PosterURL:   omdbMovie.Poster,
		Description: omdbMovie.Plot,
		ImdbID:      omdbMovie.ImdbID,
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

func parseYear(year string) int {
	// Handle cases like "2019-2022" (for series)
	if strings.Contains(year, "–") || strings.Contains(year, "-") {
		parts := strings.FieldsFunc(year, func(r rune) bool {
			return r == '–' || r == '-'
		})
		if len(parts) > 0 {
			val, _ := strconv.Atoi(parts[0])
			return val
		}
	}
	val, _ := strconv.Atoi(year)
	return val
}

func parseCommaList(list string) []string {
	if list == "" || list == "N/A" {
		return nil
	}
	items := strings.Split(list, ", ")
	return items
}
