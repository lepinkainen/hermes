package enrichment

import (
	"context"
	"log/slog"

	"github.com/lepinkainen/hermes/cmd/steam"
)

// EnrichFromSteam enriches a game with Steam data.
// It searches Steam, optionally shows TUI for selection, downloads cover, and generates content.
// If existingSteamAppID is provided and Force is false, it skips search and uses the ID directly.
func EnrichFromSteam(ctx context.Context, title string, existingSteamAppID int, opts SteamEnrichmentOptions) (*SteamEnrichment, error) {
	appID, err := resolveSteamAppID(ctx, title, existingSteamAppID, opts)
	if err != nil {
		return nil, err
	}
	if appID == 0 {
		slog.Debug("No Steam AppID found", "title", title)
		return nil, nil
	}

	// Fetch game details
	details, err := steam.GetGameDetails(appID)
	if err != nil {
		slog.Warn("Failed to fetch Steam game details", "appid", appID, "error", err)
		return nil, nil
	}

	enrichment := &SteamEnrichment{
		SteamAppID:  appID,
		Developers:  details.Developers,
		Publishers:  details.Publishers,
		ReleaseDate: details.ReleaseDate.Date,
	}

	// Extract genre tags
	enrichment.GenreTags = extractGenreTags(details)

	// Set metacritic score
	enrichment.MetacriticScore = details.Metacritic.Score

	// Download cover if requested
	if opts.DownloadCover {
		coverFilename, coverPath := ensureSteamCoverAssets(ctx, opts, title, details.HeaderImage)
		enrichment.CoverFilename = coverFilename
		enrichment.CoverPath = coverPath
	}

	// Generate content if requested
	if opts.GenerateContent {
		enrichment.ContentMarkdown = buildSteamContentMarkdown(details, enrichment.CoverFilename, opts.ContentSections)
	}

	return enrichment, nil
}
