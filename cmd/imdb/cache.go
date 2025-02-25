package imdb

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/lepinkainen/hermes/internal/errors"
	log "github.com/sirupsen/logrus"
)

func getCachedMovie(imdbID string) (*MovieSeen, error) {
	cacheDir := "cache/omdb"
	cachePath := filepath.Join(cacheDir, imdbID+".json")

	// Check cache first
	if data, err := os.ReadFile(cachePath); err == nil {
		var movie MovieSeen
		if err := json.Unmarshal(data, &movie); err == nil {
			return &movie, nil
		}
	}

	// Fetch from API if not in cache
	movie, err := fetchMovieData(imdbID)
	if err != nil {
		// Check if it's a rate limit error
		if _, isRateLimit := err.(*errors.RateLimitError); isRateLimit {
			log.Warn("OMDB API rate limit reached, stopping further requests")
			return nil, err
		}
		log.Warnf("Failed to enrich movie: %v", err)
		return nil, err
	}

	// Cache the result
	os.MkdirAll(cacheDir, 0755)
	data, _ := json.MarshalIndent(movie, "", "  ")
	os.WriteFile(cachePath, data, 0644)

	return movie, nil
}
