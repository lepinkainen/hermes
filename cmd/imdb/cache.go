package imdb

import (
	"encoding/json"
	"os"
	"path/filepath"
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
		return nil, err
	}

	// Cache the result
	os.MkdirAll(cacheDir, 0755)
	data, _ := json.Marshal(movie)
	os.WriteFile(cachePath, data, 0644)

	return movie, nil
}
