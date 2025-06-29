package letterboxd

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/lepinkainen/hermes/internal/errors"
)

// Common OMDb data structure that we need to extract from IMDB cache files
type imdbOMDbCache struct {
	Plot        string   `json:"Plot"`
	PosterURL   string   `json:"Poster URL"` // IMDB cache uses this field
	Genres      []string `json:"Genres"`
	Directors   []string `json:"Directors"`
	RuntimeMins int      `json:"Runtime (mins)"`
	IMDbRating  float64  `json:"IMDb Rating"`
}

// getCachedMovie retrieves movie data from cache or OMDB API
func getCachedMovie(title string, year int) (*Movie, error) {
	letterboxdCacheDir := "cache/letterboxd"
	omdbCacheDir := "cache/omdb"

	// Create a safe filename for the letterboxd cache
	safeTitle := strings.ReplaceAll(title, "/", "_")
	safeTitle = strings.ReplaceAll(safeTitle, "\\", "_")
	safeTitle = strings.ReplaceAll(safeTitle, ":", "_")
	safeTitle = strings.ReplaceAll(safeTitle, "*", "_")
	safeTitle = strings.ReplaceAll(safeTitle, "?", "_")
	safeTitle = strings.ReplaceAll(safeTitle, "\"", "_")
	safeTitle = strings.ReplaceAll(safeTitle, "<", "_")
	safeTitle = strings.ReplaceAll(safeTitle, ">", "_")
	safeTitle = strings.ReplaceAll(safeTitle, "|", "_")

	letterboxdCachePath := filepath.Join(letterboxdCacheDir, fmt.Sprintf("%s_%d.json", safeTitle, year))

	// First check letterboxd cache
	if data, err := os.ReadFile(letterboxdCachePath); err == nil {
		var movie Movie
		if err := json.Unmarshal(data, &movie); err == nil {
			slog.Debug("Found movie in letterboxd cache", "title", title)
			return &movie, nil
		}
	}

	// Then, try to find in the IMDB/OMDB cache by searching for files that might match our movie
	// This is an optimization to avoid unnecessary API calls
	files, err := filepath.Glob(filepath.Join(omdbCacheDir, "*.json"))
	if err == nil {
		for _, file := range files {
			// Read the file
			data, err := os.ReadFile(file)
			if err != nil {
				continue
			}

			// Try to unmarshal it
			var imdbCache imdbOMDbCache
			if err := json.Unmarshal(data, &imdbCache); err != nil {
				continue
			}

			// Simple check to see if this might be our movie - not perfect but better than nothing
			// We could make this more sophisticated by checking multiple fields
			if strings.Contains(strings.ToLower(file), strings.ToLower(safeTitle)) {
				slog.Info("Found potential match in OMDB cache", "title", title)

				// Use the data from IMDB cache to create a Movie object
				movie := &Movie{
					Name:        title,
					Year:        year,
					Description: imdbCache.Plot,
					PosterURL:   imdbCache.PosterURL,
					Genres:      imdbCache.Genres,
					Director:    strings.Join(imdbCache.Directors, ", "),
					Runtime:     imdbCache.RuntimeMins,
					Rating:      imdbCache.IMDbRating,
					ImdbID:      filepath.Base(file[:len(file)-5]), // Extract IMDB ID from filename
				}

				// Cache it in our format too
				if err := os.MkdirAll(letterboxdCacheDir, 0755); err != nil {
					slog.Warn("Failed to create cache directory", "error", err)
				} else {
					movieData, _ := json.MarshalIndent(movie, "", "  ")
					if err := os.WriteFile(letterboxdCachePath, movieData, 0644); err != nil {
						slog.Warn("Failed to write cache file", "error", err)
					}
				}

				return movie, nil
			}
		}
	}

	// If not found in either cache, fetch from API
	movie, err := fetchMovieData(title, year)
	if err != nil {
		// Check if it's a rate limit error
		if _, isRateLimit := err.(*errors.RateLimitError); isRateLimit {
			slog.Warn("OMDB API rate limit reached, stopping further requests")
			return nil, err
		}
		slog.Warn("Failed to enrich movie", "error", err)
		return nil, err
	}

	// Cache the result in letterboxd cache
	if err := os.MkdirAll(letterboxdCacheDir, 0755); err != nil {
		slog.Warn("Failed to create cache directory", "error", err)
	} else {
		data, _ := json.MarshalIndent(movie, "", "  ")
		if err := os.WriteFile(letterboxdCachePath, data, 0644); err != nil {
			slog.Warn("Failed to write cache file", "error", err)
		}
	}

	return movie, nil
}
