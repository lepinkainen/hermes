package enrichment

import (
	"context"
	"log/slog"

	"github.com/lepinkainen/hermes/internal/content"
)

func buildContentMarkdown(ctx context.Context, client tmdbClient, opts TMDBEnrichmentOptions, mediaType string, tmdbID int, coverFilename string) string {
	var (
		details         map[string]any
		err             error
		detailFromCache bool
	)

	if mediaType == "movie" {
		details, detailFromCache, err = client.CachedGetFullMovieDetails(ctx, tmdbID, opts.Force)
	} else {
		details, detailFromCache, err = client.CachedGetFullTVDetails(ctx, tmdbID, opts.Force)
	}

	if err != nil {
		slog.Warn("Failed to fetch TMDB details for content generation", "error", err)
		return ""
	}

	if detailFromCache {
		slog.Debug("TMDB full details from cache", "tmdb_id", tmdbID)
	}

	tmdbContent := content.BuildTMDBContent(details, mediaType, opts.ContentSections)
	coverEmbed := ""
	if coverFilename != "" {
		coverEmbed = content.BuildCoverImageEmbed(coverFilename)
	}

	if coverEmbed == "" {
		return tmdbContent
	}
	if tmdbContent == "" {
		return coverEmbed
	}
	return coverEmbed + "\n\n" + tmdbContent
}
