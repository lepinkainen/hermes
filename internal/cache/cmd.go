package cache

import (
	"fmt"
	"log/slog"

	"github.com/spf13/viper"
)

// InvalidateCacheCmd represents the cache invalidate subcommand
type InvalidateCacheCmd struct {
	Source string `arg:"" help:"Cache source to invalidate: tmdb, omdb, steam, letterboxd, openlibrary" required:""`
}

func (i *InvalidateCacheCmd) Run() error {
	cacheDB := viper.GetString("cache.dbfile")

	slog.Info("Invalidating cache", "source", i.Source, "database", cacheDB)

	// Map source name to cache table name
	tableName := i.Source + "_cache"

	// Validate source
	validSources := map[string]bool{
		"tmdb":        true,
		"omdb":        true,
		"steam":       true,
		"letterboxd":  true,
		"openlibrary": true,
	}

	if !validSources[i.Source] {
		return fmt.Errorf("invalid cache source '%s'; valid sources are: tmdb, omdb, steam, letterboxd, openlibrary", i.Source)
	}

	// Get or create cache database
	cacheInstance, err := GetGlobalCache()
	if err != nil {
		return fmt.Errorf("failed to open cache database: %w", err)
	}

	rowsDeleted, err := cacheInstance.InvalidateSource(tableName)
	if err != nil {
		return fmt.Errorf("failed to invalidate cache: %w", err)
	}

	slog.Info("Cache invalidated", "source", i.Source, "rows_deleted", rowsDeleted)
	return nil
}
