package enrichment

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	hermeserrors "github.com/lepinkainen/hermes/internal/errors"
	"github.com/lepinkainen/hermes/internal/tmdb"
	"github.com/lepinkainen/hermes/internal/tui"
)

func resolveTMDBID(ctx context.Context, client tmdbClient, title string, year int, imdbID string, existingTMDBID int, opts TMDBEnrichmentOptions) (int, string, error) {
	if existingTMDBID != 0 && !opts.Force {
		mediaType := opts.StoredMediaType
		if mediaType == "" {
			mediaType = opts.ExpectedMediaType
		}
		if mediaType == "" {
			return 0, "", fmt.Errorf("stored TMDB ID %d found but no media type information available (missing both stored and expected type); use --force to re-enrich", existingTMDBID)
		}

		slog.Debug("Using stored TMDB ID", "tmdb_id", existingTMDBID, "title", title, "type", mediaType)
		return existingTMDBID, mediaType, nil
	}

	return searchTMDBID(ctx, client, title, year, imdbID, opts)
}

func searchTMDBID(ctx context.Context, client tmdbClient, title string, year int, imdbID string, opts TMDBEnrichmentOptions) (int, string, error) {
	if imdbID != "" {
		tmdbID, mediaType := findTMDBIDByIMDBID(ctx, client, imdbID)
		if tmdbID != 0 {
			return tmdbID, mediaType, nil
		}
	}

	var (
		results   []tmdb.SearchResult
		fromCache bool
		err       error
	)

	if opts.MoviesOnly {
		results, fromCache, err = client.CachedSearchMovies(ctx, title, year, 10)
	} else {
		results, fromCache, err = client.CachedSearchMulti(ctx, title, year, 10)
	}
	if err != nil {
		return 0, "", fmt.Errorf("TMDB search failed: %w", err)
	}
	if fromCache {
		slog.Debug("TMDB search result from cache", "title", title, "year", year)
	}

	if len(results) == 0 {
		slog.Debug("No TMDB results found", "title", title)
		return 0, "", nil
	}

	selection, err := selectTMDBResult(results, title, year, opts.Interactive, opts.ExpectedMediaType)
	if err != nil {
		return 0, "", err
	}
	if selection == nil {
		return 0, "", nil
	}

	return selection.ID, selection.MediaType, nil
}

func selectTMDBResult(results []tmdb.SearchResult, title string, year int, interactive bool, expectedMediaType string) (*tmdb.SearchResult, error) {
	// Filter results by vote count (100+ votes required)
	filtered := make([]tmdb.SearchResult, 0, len(results))
	for _, result := range results {
		if result.VoteCount >= 100 {
			filtered = append(filtered, result)
		}
	}

	// If we have an expected media type but no high-vote results of that type,
	// include low-vote results of the expected type as a fallback
	if expectedMediaType != "" {
		hasExpectedType := false
		for _, result := range filtered {
			if result.MediaType == expectedMediaType {
				hasExpectedType = true
				break
			}
		}

		if !hasExpectedType {
			for _, result := range results {
				if result.MediaType == expectedMediaType {
					filtered = append(filtered, result)
				}
			}
		}
	}

	// If still no results, skip
	if len(filtered) == 0 {
		slog.Debug("No TMDB results with 100+ votes found", "title", title)
		return nil, nil
	}

	// Prioritize expected media type after filtering
	filtered = prioritizeMediaType(filtered, expectedMediaType)

	// If only one result after filtering, auto-select it
	if len(filtered) == 1 {
		slog.Debug("Auto-selected single TMDB result", "title", title, "tmdb_id", filtered[0].ID)
		return &filtered[0], nil
	}

	exact := findExactMatch(filtered, title, year)

	if interactive {
		selection, err := tui.Select(title, filtered)
		if err != nil {
			return nil, fmt.Errorf("TUI selection failed: %w", err)
		}

		switch selection.Action {
		case tui.ActionSelected:
			return selection.Selection, nil
		case tui.ActionStopped:
			return nil, hermeserrors.NewStopProcessingError("TMDB selection stopped by user")
		default:
			slog.Debug("User skipped TMDB selection")
			return nil, nil
		}

	}

	// Non-interactive: keep heuristic behavior
	if exact != nil {
		slog.Debug("Auto-selected exact TMDB match", "title", title, "year", year, "tmdb_id", exact.ID)
		return exact, nil
	}

	return &filtered[0], nil
}

// findTMDBIDByIMDBID attempts to find TMDB ID using IMDB ID via the find endpoint.
func findTMDBIDByIMDBID(ctx context.Context, client tmdbClient, imdbID string) (int, string) {
	tmdbID, mediaType, fromCache, err := client.CachedFindByIMDBID(ctx, imdbID)
	if err != nil {
		slog.Warn("Failed to find TMDB ID by IMDB ID", "imdb_id", imdbID, "error", err)
		return 0, ""
	}

	if tmdbID > 0 {
		cacheStatus := "fetched"
		if fromCache {
			cacheStatus = "cached"
		}
		slog.Debug("Found TMDB ID by IMDB ID", "imdb_id", imdbID, "tmdb_id", tmdbID, "media_type", mediaType, "cache", cacheStatus)
	}

	return tmdbID, mediaType
}

// findExactMatch returns a result if exactly one result matches the title and year.
// Returns nil if no match is found or if multiple matches exist (ambiguous).
func findExactMatch(results []tmdb.SearchResult, title string, year int) *tmdb.SearchResult {
	normalizedTitle := strings.ToLower(strings.TrimSpace(title))

	var match *tmdb.SearchResult
	matchCount := 0

	for i := range results {
		result := &results[i]

		resultTitle := result.Title
		if resultTitle == "" {
			resultTitle = result.Name
		}

		resultYear := 0
		dateStr := result.ReleaseDate
		if dateStr == "" {
			dateStr = result.FirstAirDate
		}
		if len(dateStr) >= 4 {
			if y, err := strconv.Atoi(dateStr[:4]); err == nil {
				resultYear = y
			}
		}

		if strings.ToLower(strings.TrimSpace(resultTitle)) == normalizedTitle && resultYear == year {
			match = result
			matchCount++
			if matchCount > 1 {
				return nil
			}
		}
	}

	return match
}

func prioritizeMediaType(results []tmdb.SearchResult, mediaType string) []tmdb.SearchResult {
	if mediaType == "" {
		return results
	}

	preferred := make([]tmdb.SearchResult, 0, len(results))
	others := make([]tmdb.SearchResult, 0, len(results))

	for _, result := range results {
		if result.MediaType == mediaType {
			preferred = append(preferred, result)
		} else {
			others = append(others, result)
		}
	}

	if len(preferred) == 0 {
		return results
	}

	return append(preferred, others...)
}
