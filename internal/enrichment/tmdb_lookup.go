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
		slog.Debug("Using stored TMDB ID", "tmdb_id", existingTMDBID, "title", title)
		mediaType, err := determineMediaTypeFromStoredID(ctx, client, existingTMDBID)
		if err != nil {
			return 0, "", err
		}
		return existingTMDBID, mediaType, nil
	}

	return searchTMDBID(ctx, client, title, year, imdbID, opts)
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

	filtered := make([]tmdb.SearchResult, 0, len(results))
	for _, result := range results {
		if result.VoteCount >= 100 {
			filtered = append(filtered, result)
		}
	}
	if len(filtered) == 0 {
		slog.Debug("No TMDB results with 100+ votes found", "title", title)
		return 0, "", nil
	}

	selection, err := selectTMDBResult(filtered, title, year, opts.Interactive)
	if err != nil {
		return 0, "", err
	}
	if selection == nil {
		return 0, "", nil
	}

	return selection.ID, selection.MediaType, nil
}

func selectTMDBResult(results []tmdb.SearchResult, title string, year int, interactive bool) (*tmdb.SearchResult, error) {
	if len(results) == 1 {
		return &results[0], nil
	}

	if exact := findExactMatch(results, title, year); exact != nil {
		slog.Debug("Auto-selected exact TMDB match", "title", title, "year", year, "tmdb_id", exact.ID)
		return exact, nil
	}

	if !interactive {
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
