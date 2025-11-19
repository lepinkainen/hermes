// Package enrichment provides media enrichment from external APIs.
package enrichment

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/lepinkainen/hermes/internal/config"
	"github.com/lepinkainen/hermes/internal/content"
	"github.com/lepinkainen/hermes/internal/errors"
	"github.com/lepinkainen/hermes/internal/fileutil"
	"github.com/lepinkainen/hermes/internal/tmdb"
	"github.com/lepinkainen/hermes/internal/tui"
)

type tmdbClient interface {
	CachedGetMetadataByID(ctx context.Context, mediaID int, mediaType string, force bool) (*tmdb.Metadata, bool, error)
	CachedSearchMovies(ctx context.Context, query string, year int, limit int) ([]tmdb.SearchResult, bool, error)
	CachedSearchMulti(ctx context.Context, query string, year int, limit int) ([]tmdb.SearchResult, bool, error)
	CachedFindByIMDBID(ctx context.Context, imdbID string) (int, string, bool, error)
	GetCoverURLByID(ctx context.Context, mediaID int, mediaType string) (string, error)
	DownloadAndResizeImage(ctx context.Context, imageURL, destPath string, maxWidth int) error
	CachedGetFullMovieDetails(ctx context.Context, movieID int, force bool) (map[string]any, bool, error)
	CachedGetFullTVDetails(ctx context.Context, tvID int, force bool) (map[string]any, bool, error)
}

var newTMDBClient = func(apiKey string) tmdbClient {
	return tmdb.NewClient(apiKey)
}

// TMDBEnrichmentOptions holds options for TMDB enrichment.
type TMDBEnrichmentOptions struct {
	// DownloadCover determines whether to download the cover image
	DownloadCover bool
	// GenerateContent determines whether to generate TMDB content sections
	GenerateContent bool
	// ContentSections specifies which sections to generate (empty = all)
	ContentSections []string
	// AttachmentsDir is the directory where images will be stored
	AttachmentsDir string
	// NoteDir is the directory where the note will be stored
	NoteDir string
	// Interactive enables TUI for multiple matches
	Interactive bool
	// Force forces re-enrichment even when TMDB ID exists
	Force bool
	// MoviesOnly restricts search to movies only (excludes TV shows)
	MoviesOnly bool
	// UseCoverCache enables development cache for TMDB cover images
	UseCoverCache bool
	// CoverCachePath is the directory for cached cover images
	CoverCachePath string
}

// TMDBEnrichment holds TMDB enrichment data.
type TMDBEnrichment struct {
	// TMDBID is the TMDB numeric identifier
	TMDBID int
	// TMDBType is either "movie" or "tv"
	TMDBType string
	// CoverPath is the relative path to the downloaded cover image
	CoverPath string
	// CoverFilename is just the filename of the cover
	CoverFilename string
	// RuntimeMins is the runtime in minutes
	RuntimeMins int
	// TotalEpisodes is the total number of episodes (TV shows only)
	TotalEpisodes int
	// GenreTags are the TMDB genre tags
	GenreTags []string
	// ContentMarkdown is the generated TMDB content
	ContentMarkdown string
}

// EnrichFromTMDB enriches a movie/TV show with TMDB data.
// It searches TMDB, optionally shows TUI for selection, downloads cover, and generates content.
// If existingTMDBID is provided and Force is false, it skips search and uses the ID directly.
func EnrichFromTMDB(ctx context.Context, title string, year int, imdbID string, existingTMDBID int, opts TMDBEnrichmentOptions) (*TMDBEnrichment, error) {
	if config.TMDBAPIKey == "" {
		slog.Debug("TMDB API key not configured, skipping TMDB enrichment")
		return nil, nil
	}

	client := newTMDBClient(config.TMDBAPIKey)

	var tmdbID int
	var mediaType string

	// Use existing TMDB ID if available and not forcing re-enrichment
	if existingTMDBID != 0 && !opts.Force {
		slog.Debug("Using stored TMDB ID", "tmdb_id", existingTMDBID, "title", title)
		tmdbID = existingTMDBID
		// We don't know the media type yet, will be determined when fetching metadata
		// For now, try movie first, then TV
		metadata, _, err := client.CachedGetMetadataByID(ctx, tmdbID, "movie", false)
		if err != nil {
			// Try TV if movie fails
			metadata, _, err = client.CachedGetMetadataByID(ctx, tmdbID, "tv", false)
			if err != nil {
				return nil, fmt.Errorf("failed to fetch TMDB metadata for ID %d: %w", tmdbID, err)
			}
			mediaType = "tv"
		} else {
			mediaType = "movie"
		}
		_ = metadata // Will be fetched again below
	} else {
		// First try to find TMDB ID using IMDB ID if available
		if imdbID != "" {
			tmdbID, mediaType = findTMDBIDByIMDBID(ctx, client, imdbID)
		}

		// If not found by IMDB ID, search by title (with year hint)
		if tmdbID == 0 {
			var results []tmdb.SearchResult
			var fromCache bool
			var err error
			if opts.MoviesOnly {
				results, fromCache, err = client.CachedSearchMovies(ctx, title, year, 10)
			} else {
				results, fromCache, err = client.CachedSearchMulti(ctx, title, year, 10)
			}
			if err != nil {
				return nil, fmt.Errorf("TMDB search failed: %w", err)
			}
			if fromCache {
				slog.Debug("TMDB search result from cache", "title", title, "year", year)
			}

			if len(results) == 0 {
				slog.Debug("No TMDB results found", "title", title)
				return nil, nil
			}

			// Filter out items with less than 100 votes
			filteredResults := make([]tmdb.SearchResult, 0, len(results))
			for _, result := range results {
				if result.VoteCount >= 100 {
					filteredResults = append(filteredResults, result)
				}
			}

			// If no items meet the vote threshold, return
			if len(filteredResults) == 0 {
				slog.Debug("No TMDB results with 100+ votes found", "title", title)
				return nil, nil
			}

			results = filteredResults

			// If multiple results and interactive mode, show TUI
			var selectedResult tmdb.SearchResult
			if len(results) == 1 {
				selectedResult = results[0]
			} else if exactMatch := findExactMatch(results, title, year); exactMatch != nil {
				// Auto-select exact title+year match
				slog.Debug("Auto-selected exact TMDB match", "title", title, "year", year, "tmdb_id", exactMatch.ID)
				selectedResult = *exactMatch
			} else if opts.Interactive {
				selection, err := tui.Select(title, results)
				if err != nil {
					return nil, fmt.Errorf("TUI selection failed: %w", err)
				}

				switch selection.Action {
				case tui.ActionSelected:
					selectedResult = *selection.Selection
				case tui.ActionStopped:
					return nil, errors.NewStopProcessingError("TMDB selection stopped by user")
				default:
					slog.Debug("User skipped TMDB selection")
					return nil, nil
				}
			} else {
				// Non-interactive: use first result
				selectedResult = results[0]
			}

			tmdbID = selectedResult.ID
			mediaType = selectedResult.MediaType
		}
	}

	enrichment := &TMDBEnrichment{
		TMDBID:   tmdbID,
		TMDBType: mediaType,
	}

	// Fetch metadata
	metadata, fromCache, err := client.CachedGetMetadataByID(ctx, tmdbID, mediaType, opts.Force)
	if err != nil {
		slog.Warn("Failed to fetch TMDB metadata", "error", err)
	} else {
		if fromCache {
			slog.Debug("TMDB metadata from cache", "tmdb_id", tmdbID)
		}
		if metadata.Runtime != nil {
			enrichment.RuntimeMins = *metadata.Runtime
		}
		if metadata.TotalEpisodes != nil {
			enrichment.TotalEpisodes = *metadata.TotalEpisodes
		}
		enrichment.GenreTags = metadata.GenreTags
	}

	// Download cover if requested
	if opts.DownloadCover {
		coverURL, err := client.GetCoverURLByID(ctx, tmdbID, mediaType)
		if err != nil {
			slog.Warn("Failed to get TMDB cover URL", "error", err)
		} else {
			coverFilename := fileutil.SanitizeFilename(title) + " - cover.jpg"
			coverPath := filepath.Join(opts.AttachmentsDir, coverFilename)

			// Use caching if enabled
			if opts.UseCoverCache && opts.CoverCachePath != "" {
				// Use TMDB ID for cache filename (more stable than title)
				cacheFilename := fmt.Sprintf("%s_%d.jpg", mediaType, tmdbID)
				cachePath := filepath.Join(opts.CoverCachePath, cacheFilename)

				// Check cache for hit
				if _, err := os.Stat(cachePath); err == nil {
					// Cache hit
					slog.Debug("TMDB cover cache hit", "cache_path", cachePath, "tmdb_id", tmdbID)

					// Check if we need to copy to attachments
					if _, err := os.Stat(coverPath); err == nil && !config.UpdateCovers {
						slog.Debug("TMDB cover already exists in attachments, skipping copy", "path", coverPath)
					} else {
						// Copy from cache to attachments
						if err := copyFile(cachePath, coverPath); err != nil {
							slog.Warn("Failed to copy cover from cache", "error", err)
						} else {
							slog.Debug("Copied TMDB cover from cache", "cache_path", cachePath, "dest_path", coverPath)
						}
					}

					enrichment.CoverFilename = coverFilename
					if opts.NoteDir != "" {
						relPath, err := fileutil.RelativeTo(opts.NoteDir, coverPath)
						if err == nil {
							enrichment.CoverPath = relPath
						}
					}
				} else {
					// Cache miss - download to cache, then copy to attachments
					slog.Debug("TMDB cover cache miss", "cache_path", cachePath, "tmdb_id", tmdbID)

					if err := client.DownloadAndResizeImage(ctx, coverURL, cachePath, 1000); err != nil {
						slog.Warn("Failed to download TMDB cover to cache", "error", err)
					} else {
						slog.Info("Downloaded TMDB cover to cache", "cache_path", cachePath)

						// Copy from cache to attachments
						if err := copyFile(cachePath, coverPath); err != nil {
							slog.Warn("Failed to copy cover from cache to attachments", "error", err)
						} else {
							slog.Debug("Copied TMDB cover to attachments", "path", coverPath)
						}

						enrichment.CoverFilename = coverFilename
						if opts.NoteDir != "" {
							relPath, err := fileutil.RelativeTo(opts.NoteDir, coverPath)
							if err == nil {
								enrichment.CoverPath = relPath
							}
						}
					}
				}
			} else {
				// No caching - use original behavior
				if _, err := os.Stat(coverPath); err == nil {
					slog.Debug("TMDB cover already exists, skipping download", "path", coverPath)
					enrichment.CoverFilename = coverFilename

					// Calculate relative path from note to cover
					if opts.NoteDir != "" {
						relPath, err := fileutil.RelativeTo(opts.NoteDir, coverPath)
						if err == nil {
							enrichment.CoverPath = relPath
						}
					}
				} else if err := client.DownloadAndResizeImage(ctx, coverURL, coverPath, 1000); err != nil {
					slog.Warn("Failed to download TMDB cover", "error", err)
				} else {
					slog.Info("Downloaded TMDB cover", "path", coverPath)
					enrichment.CoverFilename = coverFilename

					// Calculate relative path from note to cover
					if opts.NoteDir != "" {
						relPath, err := fileutil.RelativeTo(opts.NoteDir, coverPath)
						if err == nil {
							enrichment.CoverPath = relPath
						}
					}
				}
			}
		}
	}

	// Generate content if requested
	if opts.GenerateContent {
		var details map[string]any
		var err error
		var detailsFromCache bool
		if mediaType == "movie" {
			details, detailsFromCache, err = client.CachedGetFullMovieDetails(ctx, tmdbID, opts.Force)
		} else {
			details, detailsFromCache, err = client.CachedGetFullTVDetails(ctx, tmdbID, opts.Force)
		}

		if err != nil {
			slog.Warn("Failed to fetch TMDB details for content generation", "error", err)
		} else {
			if detailsFromCache {
				slog.Debug("TMDB full details from cache", "tmdb_id", tmdbID)
			}
			tmdbContent := content.BuildTMDBContent(details, mediaType, opts.ContentSections)

			// Prepend cover image embed if cover was downloaded
			if enrichment.CoverFilename != "" {
				coverEmbed := content.BuildCoverImageEmbed(enrichment.CoverFilename)
				if coverEmbed != "" {
					enrichment.ContentMarkdown = coverEmbed + "\n\n" + tmdbContent
				} else {
					enrichment.ContentMarkdown = tmdbContent
				}
			} else {
				enrichment.ContentMarkdown = tmdbContent
			}
		}
	}

	return enrichment, nil
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

		// Get the result's title and year
		resultTitle := result.Title
		if resultTitle == "" {
			resultTitle = result.Name // TV shows use Name
		}

		// Get year from release date
		resultYear := 0
		dateStr := result.ReleaseDate
		if dateStr == "" {
			dateStr = result.FirstAirDate // TV shows use FirstAirDate
		}
		if len(dateStr) >= 4 {
			if y, err := strconv.Atoi(dateStr[:4]); err == nil {
				resultYear = y
			}
		}

		// Check for exact match (case-insensitive title, exact year)
		if strings.ToLower(strings.TrimSpace(resultTitle)) == normalizedTitle && resultYear == year {
			match = result
			matchCount++
			if matchCount > 1 {
				// Multiple exact matches - ambiguous, return nil
				return nil
			}
		}
	}

	return match
}

// copyFile copies a file from src to dst
func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer func() { _ = srcFile.Close() }()

	// Create destination directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	dstFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer func() { _ = dstFile.Close() }()

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return fmt.Errorf("failed to copy file contents: %w", err)
	}

	return nil
}
