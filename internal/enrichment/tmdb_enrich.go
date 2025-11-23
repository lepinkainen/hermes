package enrichment

import (
	"context"
	"log/slog"

	"github.com/lepinkainen/hermes/internal/config"
)

// EnrichFromTMDB enriches a movie/TV show with TMDB data.
// It searches TMDB, optionally shows TUI for selection, downloads cover, and generates content.
// If existingTMDBID is provided and Force is false, it skips search and uses the ID directly.
func EnrichFromTMDB(ctx context.Context, title string, year int, imdbID string, existingTMDBID int, opts TMDBEnrichmentOptions) (*TMDBEnrichment, error) {
	if config.TMDBAPIKey == "" {
		slog.Debug("TMDB API key not configured, skipping TMDB enrichment")
		return nil, nil
	}

	client := newTMDBClient(config.TMDBAPIKey)

	tmdbID, mediaType, err := resolveTMDBID(ctx, client, title, year, imdbID, existingTMDBID, opts)
	if err != nil {
		return nil, err
	}
	if tmdbID == 0 {
		return nil, nil
	}

	enrichment := &TMDBEnrichment{
		TMDBID:   tmdbID,
		TMDBType: mediaType,
	}

	applyPrimaryMetadata(ctx, client, enrichment, opts.Force)

	if opts.DownloadCover {
		coverFilename, coverPath := ensureCoverAssets(ctx, client, opts, title, mediaType, tmdbID)
		enrichment.CoverFilename = coverFilename
		enrichment.CoverPath = coverPath
	}

	if opts.GenerateContent {
		enrichment.ContentMarkdown = buildContentMarkdown(ctx, client, opts, mediaType, tmdbID, enrichment.CoverFilename)
	}

	return enrichment, nil
}

func applyPrimaryMetadata(ctx context.Context, client tmdbClient, enrichment *TMDBEnrichment, force bool) {
	metadata, fromCache, err := client.CachedGetMetadataByID(ctx, enrichment.TMDBID, enrichment.TMDBType, force)
	if err != nil {
		slog.Warn("Failed to fetch TMDB metadata", "error", err)
		return
	}

	if fromCache {
		slog.Debug("TMDB metadata from cache", "tmdb_id", enrichment.TMDBID)
	}

	if metadata.Runtime != nil {
		enrichment.RuntimeMins = *metadata.Runtime
	}
	if metadata.TotalEpisodes != nil {
		enrichment.TotalEpisodes = *metadata.TotalEpisodes
	}
	enrichment.GenreTags = metadata.GenreTags

	// For TV shows, determine if the show has finished based on status
	if enrichment.TMDBType == "tv" && metadata.Status != "" {
		finished := metadata.Status == "Ended" || metadata.Status == "Canceled" || metadata.Status == "Cancelled"
		enrichment.Finished = &finished
	}
}
