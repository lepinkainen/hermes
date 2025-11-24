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
	forcePrompt := false
	if existingTMDBID != 0 && !opts.Force {
		slog.Debug("Using stored TMDB ID", "tmdb_id", existingTMDBID, "title", title)
		mediaType, err := determineMediaTypeFromStoredID(ctx, client, existingTMDBID)
		if err != nil {
			return 0, "", err
		}
		if opts.ExpectedMediaType != "" && mediaType != opts.ExpectedMediaType {
			forcePrompt = opts.Interactive
			slog.Info("Stored TMDB type differs from expected; re-running TMDB search",
				"title", title,
				"tmdb_id", existingTMDBID,
				"cached_type", mediaType,
				"expected_type", opts.ExpectedMediaType,
				"interactive", opts.Interactive,
			)

			tmdbID, resolvedType, err := searchTMDBID(ctx, client, title, year, imdbID, opts, forcePrompt)
			if err != nil {
				return 0, "", err
			}
			if tmdbID != 0 {
				return tmdbID, resolvedType, nil
			}

			slog.Debug("Keeping stored TMDB ID after mismatch prompt", "tmdb_id", existingTMDBID)
		}
		return existingTMDBID, mediaType, nil
	}

	return searchTMDBID(ctx, client, title, year, imdbID, opts, forcePrompt)
}

func determineMediaTypeFromStoredID(ctx context.Context, client tmdbClient, tmdbID int) (string, error) {
	movieMeta, _, movieErr := client.CachedGetMetadataByID(ctx, tmdbID, "movie", false)
	tvMeta, _, tvErr := client.CachedGetMetadataByID(ctx, tmdbID, "tv", false)

	if movieErr != nil && tvErr != nil {
		return "", fmt.Errorf("failed to fetch TMDB metadata for ID %d: %w", tmdbID, movieErr)
	}

	movieScore := 0
	if movieErr == nil && movieMeta != nil {
		if movieMeta.Runtime != nil && *movieMeta.Runtime > 0 {
			movieScore += 10
		}
		if len(movieMeta.GenreTags) > 0 {
			movieScore++
		}
	}

	tvScore := 0
	if tvErr == nil && tvMeta != nil {
		if tvMeta.TotalEpisodes != nil && *tvMeta.TotalEpisodes > 0 {
			tvScore += 10
		}
		if len(tvMeta.GenreTags) > 0 {
			tvScore++
		}
	}

	var mediaType string
	switch {
	case tvScore > movieScore:
		mediaType = "tv"
	case movieScore > 0:
		mediaType = "movie"
	case tvErr == nil:
		mediaType = "tv"
	default:
		mediaType = "movie"
	}

	slog.Debug("Determined media type from TMDB data",
		"tmdb_id", tmdbID,
		"type", mediaType,
		"movie_score", movieScore,
		"tv_score", tvScore,
	)

	return mediaType, nil
}

func searchTMDBID(ctx context.Context, client tmdbClient, title string, year int, imdbID string, opts TMDBEnrichmentOptions, forcePrompt bool) (int, string, error) {
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

	filtered := make([]tmdb.SearchResult, 0, len(results))
	for _, result := range results {
		if result.VoteCount >= 100 || (opts.ExpectedMediaType != "" && result.MediaType == opts.ExpectedMediaType) {
			filtered = append(filtered, result)
		}
	}
	filtered = prioritizeMediaType(filtered, opts.ExpectedMediaType)
	if len(filtered) == 0 {
		slog.Debug("No TMDB results with 100+ votes found", "title", title)
		return 0, "", nil
	}

	selection, err := selectTMDBResult(filtered, title, year, opts.Interactive, forcePrompt)
	if err != nil {
		return 0, "", err
	}
	if selection == nil {
		return 0, "", nil
	}

	return selection.ID, selection.MediaType, nil
}

func selectTMDBResult(results []tmdb.SearchResult, title string, year int, interactive bool, forcePrompt bool) (*tmdb.SearchResult, error) {
	exact := findExactMatch(results, title, year)

	if interactive {
		// Only auto-select when there's exactly one result and it is an exact match
		if !forcePrompt && len(results) == 1 && exact != nil {
			slog.Debug("Auto-selected exact TMDB match", "title", title, "year", year, "tmdb_id", exact.ID)
			return &results[0], nil
		}

		selection, err := tui.Select(title, results)
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
	if len(results) == 1 {
		return &results[0], nil
	}

	if exact != nil && !forcePrompt {
		slog.Debug("Auto-selected exact TMDB match", "title", title, "year", year, "tmdb_id", exact.ID)
		return exact, nil
	}

	return &results[0], nil
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
