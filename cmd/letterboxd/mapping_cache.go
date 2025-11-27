package letterboxd

import (
	"database/sql"
	"fmt"
	"log/slog"

	"github.com/lepinkainen/hermes/internal/cache"
)

// LetterboxdMapping represents a cached mapping from Letterboxd URI to TMDB/IMDB IDs
type LetterboxdMapping struct {
	LetterboxdURI string
	TMDBID        int
	TMDBType      string
	ImdbID        string
}

// GetLetterboxdMapping retrieves a cached mapping for the given Letterboxd URI
func GetLetterboxdMapping(letterboxdURI string) (*LetterboxdMapping, error) {
	if letterboxdURI == "" {
		return nil, nil
	}

	cacheDB, err := cache.GetGlobalCache()
	if err != nil {
		return nil, fmt.Errorf("failed to get cache: %w", err)
	}

	query := `
		SELECT tmdb_id, tmdb_type, imdb_id
		FROM letterboxd_mapping_cache
		WHERE letterboxd_uri = ?
	`

	var mapping LetterboxdMapping
	mapping.LetterboxdURI = letterboxdURI

	var tmdbID sql.NullInt64
	var tmdbType sql.NullString
	var imdbID sql.NullString

	err = cacheDB.QueryRow(query, letterboxdURI).Scan(&tmdbID, &tmdbType, &imdbID)
	if err == sql.ErrNoRows {
		return nil, nil // No cached mapping found
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query mapping cache: %w", err)
	}

	if tmdbID.Valid {
		mapping.TMDBID = int(tmdbID.Int64)
	}
	if tmdbType.Valid {
		mapping.TMDBType = tmdbType.String
	}
	if imdbID.Valid {
		mapping.ImdbID = imdbID.String
	}

	slog.Debug("Retrieved Letterboxd mapping from cache",
		"letterboxd_uri", letterboxdURI,
		"tmdb_id", mapping.TMDBID,
		"tmdb_type", mapping.TMDBType,
		"imdb_id", mapping.ImdbID)

	return &mapping, nil
}

// SetLetterboxdMapping stores a mapping in the cache
func SetLetterboxdMapping(mapping LetterboxdMapping) error {
	if mapping.LetterboxdURI == "" {
		return nil // Nothing to cache
	}

	cacheDB, err := cache.GetGlobalCache()
	if err != nil {
		return fmt.Errorf("failed to get cache: %w", err)
	}

	query := `
		INSERT OR REPLACE INTO letterboxd_mapping_cache
		(letterboxd_uri, tmdb_id, tmdb_type, imdb_id, cached_at)
		VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP)
	`

	var tmdbID any
	if mapping.TMDBID != 0 {
		tmdbID = mapping.TMDBID
	}

	var tmdbType any
	if mapping.TMDBType != "" {
		tmdbType = mapping.TMDBType
	}

	var imdbID any
	if mapping.ImdbID != "" {
		imdbID = mapping.ImdbID
	}

	if err := cacheDB.Exec(query, mapping.LetterboxdURI, tmdbID, tmdbType, imdbID); err != nil {
		return fmt.Errorf("failed to store mapping in cache: %w", err)
	}

	slog.Debug("Stored Letterboxd mapping in cache",
		"letterboxd_uri", mapping.LetterboxdURI,
		"tmdb_id", mapping.TMDBID,
		"tmdb_type", mapping.TMDBType,
		"imdb_id", mapping.ImdbID)

	return nil
}
