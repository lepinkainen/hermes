// Package enrichment provides media enrichment from external APIs.
package enrichment

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"

	"github.com/lepinkainen/hermes/internal/config"
	"github.com/lepinkainen/hermes/internal/content"
	"github.com/lepinkainen/hermes/internal/fileutil"
	"github.com/lepinkainen/hermes/internal/tmdb"
	"github.com/lepinkainen/hermes/internal/tui"
)

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

	client := tmdb.NewClient(config.TMDBAPIKey)

	var tmdbID int
	var mediaType string

	// Use existing TMDB ID if available and not forcing re-enrichment
	if existingTMDBID != 0 && !opts.Force {
		slog.Debug("Using stored TMDB ID", "tmdb_id", existingTMDBID, "title", title)
		tmdbID = existingTMDBID
		// We don't know the media type yet, will be determined when fetching metadata
		// For now, try movie first, then TV
		metadata, err := client.GetMetadataByID(ctx, tmdbID, "movie")
		if err != nil {
			// Try TV if movie fails
			metadata, err = client.GetMetadataByID(ctx, tmdbID, "tv")
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

		// If not found by IMDB ID, search by title
		if tmdbID == 0 {
			query := title
			if year > 0 {
				query = fmt.Sprintf("%s %d", title, year)
			}

			results, err := client.SearchMulti(ctx, query, 5)
			if err != nil {
				return nil, fmt.Errorf("TMDB search failed: %w", err)
			}

			if len(results) == 0 {
				slog.Debug("No TMDB results found", "title", title)
				return nil, nil
			}

			// If multiple results and interactive mode, show TUI
			var selectedResult tmdb.SearchResult
			if len(results) == 1 {
				selectedResult = results[0]
			} else if opts.Interactive {
				selection, err := tui.Select(title, results)
				if err != nil {
					return nil, fmt.Errorf("TUI selection failed: %w", err)
				}
				if selection.Action != tui.ActionSelected {
					slog.Debug("User skipped TMDB selection")
					return nil, nil
				}
				selectedResult = *selection.Selection
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
	metadata, err := client.GetMetadataByID(ctx, tmdbID, mediaType)
	if err != nil {
		slog.Warn("Failed to fetch TMDB metadata", "error", err)
	} else {
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

			if err := client.DownloadAndResizeImage(ctx, coverURL, coverPath, 1000); err != nil {
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

	// Generate content if requested
	if opts.GenerateContent {
		var details map[string]any
		var err error
		if mediaType == "movie" {
			details, err = client.GetFullMovieDetails(ctx, tmdbID)
		} else {
			details, err = client.GetFullTVDetails(ctx, tmdbID)
		}

		if err != nil {
			slog.Warn("Failed to fetch TMDB details for content generation", "error", err)
		} else {
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

// findTMDBIDByIMDBID attempts to find TMDB ID using IMDB ID via external_ids endpoint.
func findTMDBIDByIMDBID(ctx context.Context, client *tmdb.Client, _ string) (int, string) {
	// TODO: TMDB doesn't have a direct IMDB ID lookup in the simple API
	// For now, we'll skip this and rely on search
	_ = ctx
	_ = client

	return 0, ""
}
