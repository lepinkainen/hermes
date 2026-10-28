package obsidian

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/lepinkainen/hermes/internal/config"
	"github.com/lepinkainen/hermes/internal/fileutil"
)

// CoverSource describes where a note's cover image may come from.
//
// Sources are tried in priority order: a cover that enrichment has already
// downloaded (TMDBCoverPath/TMDBCoverFilename) wins outright; otherwise
// FallbackURL is downloaded into the note directory's attachments folder.
type CoverSource struct {
	// TMDBCoverPath is the relative path of a cover that enrichment has
	// already downloaded (e.g. "attachments/Title - cover.jpg").
	TMDBCoverPath string
	// TMDBCoverFilename is the bare filename of that pre-downloaded cover.
	TMDBCoverFilename string
	// FallbackURL is a source-specific poster/header image URL to download
	// when no pre-downloaded cover exists.
	FallbackURL string
	// FallbackToURLOnError makes a failed download fall back to referencing
	// FallbackURL directly instead of leaving the note without a cover.
	FallbackToURLOnError bool
}

// CoverResult is a resolved cover ready to be applied to a note.
type CoverResult struct {
	// Filename is the local cover filename, used for the Obsidian embed
	// (![[filename|width]]). Empty when the cover is a remote URL.
	Filename string
	// FrontmatterValue is what goes in the frontmatter "cover" field: the
	// relative attachment path, or the remote URL for a remote cover.
	FrontmatterValue string
	// RemoteURL is set when the cover could not be downloaded and the note
	// should reference the image by URL instead.
	RemoteURL string
}

// Embed returns the markdown that embeds the cover in the note body, followed
// by a blank line. Local covers use Obsidian's ![[file|width]] syntax; remote
// covers use a standard markdown image link.
func (r *CoverResult) Embed(width int) string {
	if r.RemoteURL != "" {
		return fmt.Sprintf("![](%s)\n\n", r.RemoteURL)
	}
	return fmt.Sprintf("![[%s|%d]]\n\n", r.Filename, width)
}

// ResolveCover resolves a cover image for a note from the available sources.
//
// Priority: a pre-downloaded enrichment cover, then a download of FallbackURL
// into directory/attachments (named after title). Returns nil when no cover is
// available, including when the download fails and FallbackToURLOnError is
// not set.
func ResolveCover(source CoverSource, title, directory string) *CoverResult {
	if source.TMDBCoverFilename != "" {
		return &CoverResult{
			Filename:         source.TMDBCoverFilename,
			FrontmatterValue: source.TMDBCoverPath,
		}
	}

	if source.FallbackURL == "" {
		return nil
	}

	result, err := fileutil.DownloadCover(context.Background(), fileutil.CoverDownloadOptions{
		URL:          source.FallbackURL,
		OutputDir:    directory,
		Filename:     fileutil.BuildCoverFilename(title),
		UpdateCovers: config.UpdateCovers,
	})
	if err != nil {
		slog.Warn("Failed to download cover", "title", title, "error", err)
		if source.FallbackToURLOnError {
			return &CoverResult{
				FrontmatterValue: source.FallbackURL,
				RemoteURL:        source.FallbackURL,
			}
		}
		return nil
	}
	if result == nil {
		return nil
	}

	return &CoverResult{
		Filename:         result.Filename,
		FrontmatterValue: result.RelativePath,
	}
}
