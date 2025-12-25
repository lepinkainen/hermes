package enrichment

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/lepinkainen/hermes/internal/config"
	"github.com/lepinkainen/hermes/internal/fileutil"
)

func ensureSteamCoverAssets(_ context.Context, opts SteamEnrichmentOptions, title string, headerImageURL string) (string, string) {
	if headerImageURL == "" {
		slog.Debug("No Steam header image URL provided", "title", title)
		return "", ""
	}

	coverFilename := fileutil.SanitizeFilename(title) + " - cover.jpg"
	coverPath := filepath.Join(opts.AttachmentsDir, coverFilename)

	// Check if cover already exists
	if _, err := os.Stat(coverPath); err == nil && !config.UpdateCovers {
		slog.Debug("Steam cover already exists, skipping download", "path", coverPath)
		return coverFilename, relativeCoverPath(opts.NoteDir, coverPath)
	}

	// Use the existing DownloadCover function
	result, err := fileutil.DownloadCover(fileutil.CoverDownloadOptions{
		URL:          headerImageURL,
		OutputDir:    filepath.Dir(opts.AttachmentsDir), // Parent of attachments
		Filename:     coverFilename,
		UpdateCovers: config.UpdateCovers,
	})

	if err != nil {
		slog.Warn("Failed to download Steam cover", "error", err, "url", headerImageURL)
		return "", ""
	}

	if result == nil {
		return "", ""
	}

	slog.Info("Downloaded Steam cover", "path", result.LocalPath)
	return coverFilename, result.RelativePath
}
