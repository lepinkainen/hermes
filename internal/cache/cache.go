package cache

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
)

// FetchFunc represents a function that fetches data from an external source
type FetchFunc[T any] func() (T, error)

// GetOrFetch retrieves data from cache or fetches it using the provided function
// T is the type of data being cached
// cacheDir is the directory where cache files are stored
// cacheKey is the unique identifier for this cache entry (e.g., ISBN, IMDb ID)
// fetchFunc is called if the data is not found in cache
func GetOrFetch[T any](cacheDir, cacheKey string, fetchFunc FetchFunc[T]) (T, bool, error) {
	var zero T

	cachePath := filepath.Join(cacheDir, cacheKey+".json")

	// Check cache first
	if data, err := os.ReadFile(cachePath); err == nil {
		var cached T
		if err := json.Unmarshal(data, &cached); err == nil {
			slog.Debug("Cache hit", "cache_path", cachePath)
			return cached, true, nil
		} else {
			slog.Warn("Failed to unmarshal cached data, will refetch", "cache_path", cachePath, "error", err)
		}
	}

	// Fetch from external source if not in cache
	slog.Debug("Cache miss, fetching data", "cache_path", cachePath)
	data, err := fetchFunc()
	if err != nil {
		return zero, false, fmt.Errorf("failed to fetch data: %w", err)
	}

	// Cache the result
	if err := cacheData(cachePath, data); err != nil {
		// Log error but don't fail - caching failure shouldn't stop the process
		slog.Warn("Failed to cache data", "cache_path", cachePath, "error", err)
	} else {
		slog.Debug("Data cached successfully", "cache_path", cachePath)
	}

	return data, false, nil
}

// cacheData writes data to the cache file
func cacheData[T any](cachePath string, data T) error {
	// Create cache directory if it doesn't exist
	cacheDir := filepath.Dir(cachePath)
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	// Marshal data to JSON
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal data: %w", err)
	}

	// Write to cache file
	if err := os.WriteFile(cachePath, jsonData, 0644); err != nil {
		return fmt.Errorf("failed to write cache file: %w", err)
	}

	return nil
}

// ClearCache removes all cache files from the specified directory
func ClearCache(cacheDir string) error {
	if _, err := os.Stat(cacheDir); os.IsNotExist(err) {
		// Directory doesn't exist, nothing to clear
		return nil
	}

	entries, err := os.ReadDir(cacheDir)
	if err != nil {
		return fmt.Errorf("failed to read cache directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".json" {
			cachePath := filepath.Join(cacheDir, entry.Name())
			if err := os.Remove(cachePath); err != nil {
				slog.Warn("Failed to remove cache file", "path", cachePath, "error", err)
			}
		}
	}

	slog.Info("Cache cleared", "directory", cacheDir)
	return nil
}

// CacheExists checks if a cache file exists for the given key
func CacheExists(cacheDir, cacheKey string) bool {
	cachePath := filepath.Join(cacheDir, cacheKey+".json")
	_, err := os.Stat(cachePath)
	return err == nil
}
