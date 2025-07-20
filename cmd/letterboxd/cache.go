package letterboxd

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/lepinkainen/hermes/internal/cache"
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

// createSafeFilename creates a safe filename from title
func createSafeFilename(title string) string {
	safeTitle := strings.ReplaceAll(title, "/", "_")
	safeTitle = strings.ReplaceAll(safeTitle, "\\", "_")
	safeTitle = strings.ReplaceAll(safeTitle, ":", "_")
	safeTitle = strings.ReplaceAll(safeTitle, "*", "_")
	safeTitle = strings.ReplaceAll(safeTitle, "?", "_")
	safeTitle = strings.ReplaceAll(safeTitle, "\"", "_")
	safeTitle = strings.ReplaceAll(safeTitle, "<", "_")
	safeTitle = strings.ReplaceAll(safeTitle, ">", "_")
	safeTitle = strings.ReplaceAll(safeTitle, "|", "_")
	return safeTitle
}

// checkOMDbCacheForMovie tries to find movie data in the OMDB cache
func checkOMDbCacheForMovie(title string, year int) (*Movie, bool) {
	omdbCacheDir := "cache/omdb"
	safeTitle := createSafeFilename(title)

	// Try to find in the IMDB/OMDB cache by searching for files that might match our movie
	files, err := filepath.Glob(filepath.Join(omdbCacheDir, "*.json"))
	if err != nil {
		return nil, false
	}

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

		// Simple check to see if this might be our movie
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

			return movie, true
		}
	}

	return nil, false
}

// getCachedMovie retrieves movie data from cache or OMDB API
func getCachedMovie(title string, year int) (*Movie, error) {
	letterboxdCacheDir := "cache/letterboxd"
	safeTitle := createSafeFilename(title)
	cacheKey := fmt.Sprintf("%s_%d", safeTitle, year)

	// Use the generic cache utility with custom fetch logic
	movie, _, err := cache.GetOrFetch(letterboxdCacheDir, cacheKey, func() (*Movie, error) {
		// First, try to find in the OMDB cache
		if movie, found := checkOMDbCacheForMovie(title, year); found {
			return movie, nil
		}

		// If not found in OMDB cache, fetch from API
		movieData, fetchErr := fetchMovieData(title, year)
		if fetchErr != nil {
			// Check if it's a rate limit error
			if _, isRateLimit := fetchErr.(*errors.RateLimitError); isRateLimit {
				slog.Warn("OMDB API rate limit reached, stopping further requests")
				return nil, fetchErr
			}
			slog.Warn("Failed to enrich movie", "error", fetchErr)
			return nil, fetchErr
		}

		return movieData, nil
	})

	return movie, err
}
