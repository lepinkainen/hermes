package cache

import (
	"fmt"
	"log/slog"

	"github.com/spf13/viper"
)

// InvalidateCacheCmd represents the cache invalidate subcommand
type InvalidateCacheCmd struct {
	Source string `arg:"" help:"Cache source to invalidate: tmdb, omdb, steam, letterboxd, openlibrary, openlibrary_search, rawg" required:""`
}

// validSources maps a cache invalidation source name to the cache table(s)
// it manages. Most sources map 1:1 to "<source>_cache"; RAWG manages two
// tables (title search results + per-game details, mirroring
// rawg_search_cache/rawg_cache in schema.go) so "rawg" invalidates both.
var validSources = map[string][]string{
	"tmdb":               {"tmdb_cache"},
	"omdb":               {"omdb_cache"},
	"steam":              {"steam_cache"},
	"letterboxd":         {"letterboxd_cache"},
	"openlibrary":        {"openlibrary_cache"},
	"openlibrary_search": {"openlibrary_search_cache"},
	"rawg":               {"rawg_search_cache", "rawg_cache"},
}

// Run executes the cache invalidate command.
func (i *InvalidateCacheCmd) Run() error {
	cacheDB := viper.GetString("cache.dbfile")

	slog.Info("Invalidating cache", "source", i.Source, "database", cacheDB)

	tableNames, ok := validSources[i.Source]
	if !ok {
		return fmt.Errorf("invalid cache source '%s'; valid sources are: tmdb, omdb, steam, letterboxd, openlibrary, openlibrary_search, rawg", i.Source)
	}

	// Get or create cache database
	cacheInstance, err := GetGlobalCache()
	if err != nil {
		return fmt.Errorf("failed to open cache database: %w", err)
	}

	var totalRowsDeleted int64
	for _, tableName := range tableNames {
		rowsDeleted, err := cacheInstance.InvalidateSource(tableName)
		if err != nil {
			return fmt.Errorf("failed to invalidate cache: %w", err)
		}
		totalRowsDeleted += rowsDeleted
	}

	slog.Info("Cache invalidated", "source", i.Source, "rows_deleted", totalRowsDeleted)
	return nil
}
