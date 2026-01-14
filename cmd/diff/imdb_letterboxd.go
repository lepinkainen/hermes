package diff

import (
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	"github.com/lepinkainen/hermes/internal/fileutil"
	"github.com/lepinkainen/hermes/internal/obsidian"
	_ "modernc.org/sqlite"
)

// BuildDiffReport builds a diff report comparing IMDb and Letterboxd entries.
// The returned diffReport can be used to generate either markdown or HTML output.
func BuildDiffReport(mainDBPath, cacheDBPath string, now time.Time) (*diffReport, error) {
	if !fileutil.FileExists(mainDBPath) {
		return nil, fmt.Errorf("main database not found: %s", mainDBPath)
	}

	mainDB, err := sql.Open("sqlite", mainDBPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open main database: %w", err)
	}
	defer func() { _ = mainDB.Close() }()

	imdbMovies, err := loadImdbMovies(mainDB)
	if err != nil {
		return nil, err
	}

	letterboxdMovies, err := loadLetterboxdMovies(mainDB)
	if err != nil {
		return nil, err
	}

	mapping := map[string]string{}
	if cacheDBPath != "" && fileutil.FileExists(cacheDBPath) {
		mapping, err = loadLetterboxdMappings(cacheDBPath)
		if err != nil {
			return nil, err
		}
	} else if cacheDBPath != "" {
		slog.Warn("Cache database not found, continuing without letterboxd mappings", "path", cacheDBPath)
	}

	applyLetterboxdMappings(letterboxdMovies, mapping)

	imdbOnly, letterboxdOnly, resolvedTitleYear := diffIMDbLetterboxd(imdbMovies, letterboxdMovies)
	slog.Info(
		"Diff summary",
		"imdb_only", len(imdbOnly),
		"letterboxd_only", len(letterboxdOnly),
	)

	stats := diffStats{
		imdbOnlyCount:       len(imdbOnly),
		letterboxdOnlyCount: len(letterboxdOnly),
		resolvedTitleYear:   resolvedTitleYear,
	}
	for _, item := range imdbOnly {
		if len(item.FuzzyMatches) > 0 {
			stats.imdbOnlyWithFuzzy++
		}
	}
	for _, item := range letterboxdOnly {
		if len(item.FuzzyMatches) > 0 {
			stats.letterboxdOnlyWithFuzzy++
		}
	}

	return &diffReport{
		ImdbOnly:       imdbOnly,
		LetterboxdOnly: letterboxdOnly,
		Stats:          stats,
		GeneratedAt:    now,
		MainDBPath:     mainDBPath,
		CacheDBPath:    cacheDBPath,
	}, nil
}

// BuildIMDbLetterboxdReport builds an Obsidian markdown report comparing IMDb and Letterboxd entries.
func BuildIMDbLetterboxdReport(mainDBPath, cacheDBPath string, now time.Time) (*obsidian.Note, error) {
	report, err := BuildDiffReport(mainDBPath, cacheDBPath, now)
	if err != nil {
		return nil, err
	}
	return buildDiffNote(report.ImdbOnly, report.LetterboxdOnly, report.Stats, report.GeneratedAt, report.MainDBPath, report.CacheDBPath), nil
}
