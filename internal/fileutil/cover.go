package fileutil

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// coverHTTPClient is reused across cover downloads to share connection pools.
var coverHTTPClient = &http.Client{
	Timeout: 30 * time.Second,
}

// SetCoverHTTPClient swaps the cover-download HTTP client and returns a
// restore function. Test-only seam.
func SetCoverHTTPClient(c *http.Client) func() {
	prev := coverHTTPClient
	coverHTTPClient = c
	return func() { coverHTTPClient = prev }
}

// CoverDownloadOptions holds options for downloading cover images.
type CoverDownloadOptions struct {
	// URL is the source URL of the cover image
	URL string
	// OutputDir is the directory where the cover will be saved
	OutputDir string
	// Filename is the name of the cover file (e.g., "Title - cover.jpg")
	Filename string
	// UpdateCovers forces re-downloading even if cover exists
	UpdateCovers bool
}

// CoverDownloadResult holds the result of a cover download operation.
type CoverDownloadResult struct {
	// Downloaded indicates if a new file was downloaded
	Downloaded bool
	// LocalPath is the full path to the downloaded cover
	LocalPath string
	// RelativePath is the path relative to the note (e.g., "attachments/Title - cover.jpg")
	RelativePath string
	// Filename is just the filename
	Filename string
}

// DownloadCover downloads a cover image to the local attachments directory.
// It skips downloading if the file already exists and UpdateCovers is false.
func DownloadCover(ctx context.Context, opts CoverDownloadOptions) (*CoverDownloadResult, error) {
	if opts.URL == "" {
		return nil, nil
	}

	// Create attachments directory
	attachmentsDir := filepath.Join(opts.OutputDir, "attachments")
	if err := os.MkdirAll(attachmentsDir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create attachments directory: %w", err)
	}

	localPath := filepath.Join(attachmentsDir, opts.Filename)
	relativePath := filepath.Join("attachments", opts.Filename)

	result := &CoverDownloadResult{
		LocalPath:    localPath,
		RelativePath: relativePath,
		Filename:     opts.Filename,
	}

	// Check if file already exists
	if FileExists(localPath) && !opts.UpdateCovers {
		slog.Debug("Cover already exists, skipping download", "path", localPath)
		return result, nil
	}

	// Download the image
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, opts.URL, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("failed to build cover request: %w", err)
	}
	resp, err := coverHTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to download cover: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("unexpected status %d downloading cover from %s", resp.StatusCode, opts.URL)
	}

	// Create the file
	file, err := os.Create(localPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create cover file: %w", err)
	}
	defer func() { _ = file.Close() }()

	// Copy the response body to the file
	_, err = io.Copy(file, resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to write cover file: %w", err)
	}

	slog.Info("Downloaded cover", "path", localPath)
	result.Downloaded = true

	return result, nil
}

// BuildCoverFilename creates a standard cover filename from a title.
// Returns: "Title - cover.jpg"
func BuildCoverFilename(title string) string {
	return SanitizeFilename(title) + " - cover.jpg"
}
