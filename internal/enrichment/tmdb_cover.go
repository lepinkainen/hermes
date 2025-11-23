package enrichment

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/lepinkainen/hermes/internal/config"
	"github.com/lepinkainen/hermes/internal/fileutil"
)

func ensureCoverAssets(ctx context.Context, client tmdbClient, opts TMDBEnrichmentOptions, title, mediaType string, tmdbID int) (string, string) {
	coverURL, err := client.GetCoverURLByID(ctx, tmdbID, mediaType)
	if err != nil {
		slog.Warn("Failed to get TMDB cover URL", "error", err)
		return "", ""
	}

	coverFilename := fileutil.SanitizeFilename(title) + " - cover.jpg"
	coverPath := filepath.Join(opts.AttachmentsDir, coverFilename)

	var (
		relative string
		success  bool
	)

	if opts.UseCoverCache && opts.CoverCachePath != "" {
		relative, success = useCoverCache(ctx, client, opts, mediaType, tmdbID, coverURL, coverPath)
	} else {
		relative, success = downloadCoverDirect(ctx, client, coverURL, coverPath, opts.NoteDir)
	}

	if !success {
		return "", ""
	}

	return coverFilename, relative
}

func useCoverCache(ctx context.Context, client tmdbClient, opts TMDBEnrichmentOptions, mediaType string, tmdbID int, coverURL, coverPath string) (string, bool) {
	cacheFilename := fmt.Sprintf("%s_%d.jpg", mediaType, tmdbID)
	cachePath := filepath.Join(opts.CoverCachePath, cacheFilename)

	if _, err := os.Stat(cachePath); err == nil {
		slog.Debug("TMDB cover cache hit", "cache_path", cachePath, "tmdb_id", tmdbID)

		if _, err := os.Stat(coverPath); err == nil && !config.UpdateCovers {
			slog.Debug("TMDB cover already exists in attachments, skipping copy", "path", coverPath)
			return relativeCoverPath(opts.NoteDir, coverPath), true
		}

		if err := copyFile(cachePath, coverPath); err != nil {
			slog.Warn("Failed to copy cover from cache", "error", err)
			return "", false
		}
		slog.Debug("Copied TMDB cover from cache", "cache_path", cachePath, "dest_path", coverPath)
		return relativeCoverPath(opts.NoteDir, coverPath), true
	}

	slog.Debug("TMDB cover cache miss", "cache_path", cachePath, "tmdb_id", tmdbID)

	if err := client.DownloadAndResizeImage(ctx, coverURL, cachePath, 1000); err != nil {
		slog.Warn("Failed to download TMDB cover to cache", "error", err)
		return "", false
	}
	slog.Info("Downloaded TMDB cover to cache", "cache_path", cachePath)

	if err := copyFile(cachePath, coverPath); err != nil {
		slog.Warn("Failed to copy cover from cache to attachments", "error", err)
		return "", false
	}
	slog.Debug("Copied TMDB cover to attachments", "path", coverPath)

	return relativeCoverPath(opts.NoteDir, coverPath), true
}

func downloadCoverDirect(ctx context.Context, client tmdbClient, coverURL, coverPath, noteDir string) (string, bool) {
	if _, err := os.Stat(coverPath); err == nil && !config.UpdateCovers {
		slog.Debug("TMDB cover already exists, skipping download", "path", coverPath)
		return relativeCoverPath(noteDir, coverPath), true
	}

	if err := client.DownloadAndResizeImage(ctx, coverURL, coverPath, 1000); err != nil {
		slog.Warn("Failed to download TMDB cover", "error", err)
		return "", false
	}
	slog.Info("Downloaded TMDB cover", "path", coverPath)
	return relativeCoverPath(noteDir, coverPath), true
}

func relativeCoverPath(noteDir, coverPath string) string {
	if noteDir == "" {
		return ""
	}

	relPath, err := fileutil.RelativeTo(noteDir, coverPath)
	if err != nil {
		return ""
	}
	return relPath
}

func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer func() { _ = srcFile.Close() }()

	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
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
