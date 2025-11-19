// Package enhance provides functionality to enrich existing markdown notes with TMDB data.
package enhance

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/lepinkainen/hermes/internal/config"
	"github.com/lepinkainen/hermes/internal/enrichment"
	"github.com/lepinkainen/hermes/internal/fileutil"
)

// Options holds configuration for the enhance command.
type Options struct {
	// InputDir is the directory containing markdown files to enhance
	InputDir string
	// Recursive determines whether to scan subdirectories
	Recursive bool
	// TMDBDownloadCover determines whether to download cover images
	TMDBDownloadCover bool
	// TMDBGenerateContent determines whether to generate TMDB content sections
	TMDBGenerateContent bool
	// TMDBInteractive enables TUI for multiple TMDB matches
	TMDBInteractive bool
	// TMDBContentSections specifies which sections to generate (empty = all)
	TMDBContentSections []string
	// DryRun shows what would be done without making changes
	DryRun bool
	// Overwrite determines whether to overwrite existing TMDB content
	Overwrite bool
	// Force forces re-enrichment even when TMDB ID exists
	Force bool
	// UseTMDBCoverCache enables development cache for TMDB cover images
	UseTMDBCoverCache bool
	// TMDBCoverCachePath is the directory for cached cover images
	TMDBCoverCachePath string
}

// EnhanceNotes processes markdown files and enriches them with TMDB data.
func EnhanceNotes(opts Options) error {
	ctx := context.Background()

	if config.TMDBAPIKey == "" {
		return fmt.Errorf("TMDB API key not configured (set in config.yaml or TMDB_API_KEY environment variable)")
	}

	slog.Info("Starting enhance process", "dir", opts.InputDir, "recursive", opts.Recursive)

	// Find all markdown files
	files, err := findMarkdownFiles(opts.InputDir, opts.Recursive)
	if err != nil {
		return fmt.Errorf("failed to find markdown files: %w", err)
	}

	if len(files) == 0 {
		slog.Info("No markdown files found in directory")
		return nil
	}

	slog.Info("Found markdown files to process", "count", len(files))

	// Prepare root-level attachments directory
	attachmentsDir := filepath.Join(opts.InputDir, "attachments")
	if err := os.MkdirAll(attachmentsDir, 0755); err != nil {
		return fmt.Errorf("failed to create attachments directory: %w", err)
	}

	// Prepare cover cache directory if enabled
	if opts.UseTMDBCoverCache {
		if err := os.MkdirAll(opts.TMDBCoverCachePath, 0755); err != nil {
			return fmt.Errorf("failed to create cover cache directory: %w", err)
		}
		slog.Info("Using TMDB cover cache", "path", opts.TMDBCoverCachePath)
	}

	successCount := 0
	skipCount := 0
	errorCount := 0

	for _, file := range files {
		slog.Debug("Processing file", "path", file)

		note, err := parseNoteFile(file)
		if err != nil {
			slog.Warn("Failed to parse file", "path", file, "error", err)
			errorCount++
			continue
		}

		// Smart needs detection: determine what needs to be updated
		needsCover := note.NeedsCover()
		needsContent := note.NeedsContent()
		needsMetadata := note.NeedsMetadata()

		// Skip if already has everything and not forcing
		if !opts.Force && !needsCover && !needsContent && !needsMetadata {
			slog.Info("Skipping file (already has all TMDB data)", "path", file, "tmdb_id", note.TMDBID)
			skipCount++
			continue
		}

		// Skip if already has TMDB data and not overwriting (for backward compatibility)
		if !opts.Overwrite && note.HasTMDBData() && !opts.Force {
			slog.Info("Skipping file (already has TMDB data)", "path", file, "tmdb_id", note.TMDBID)
			skipCount++
			continue
		}

		if opts.DryRun {
			slog.Info("Would enhance", "title", note.Title, "year", note.Year, "file", file,
				"needs_cover", needsCover, "needs_content", needsContent, "needs_metadata", needsMetadata)
			successCount++
			continue
		}

		// Prepare enrichment options based on what's needed
		noteDir := filepath.Dir(file)

		enrichOpts := enrichment.TMDBEnrichmentOptions{
			DownloadCover:   opts.TMDBDownloadCover && needsCover,
			GenerateContent: opts.TMDBGenerateContent && needsContent,
			ContentSections: opts.TMDBContentSections,
			AttachmentsDir:  attachmentsDir,
			NoteDir:         noteDir,
			Interactive:     opts.TMDBInteractive,
			Force:           opts.Force,
			UseCoverCache:   opts.UseTMDBCoverCache,
			CoverCachePath:  opts.TMDBCoverCachePath,
		}

		// Enrich with TMDB data (pass existing TMDB ID if present)
		tmdbData, err := enrichment.EnrichFromTMDB(ctx, note.Title, note.Year, note.IMDBID, note.TMDBID, enrichOpts)
		if err != nil {
			slog.Warn("Failed to enrich from TMDB", "title", note.Title, "error", err)
			errorCount++
			continue
		}

		if tmdbData == nil {
			slog.Debug("No TMDB data found", "title", note.Title)
			skipCount++
			continue
		}

		// Update the note with TMDB data
		if err := updateNoteWithTMDBData(file, note, tmdbData, opts.Overwrite); err != nil {
			slog.Warn("Failed to update note", "path", file, "error", err)
			errorCount++
			continue
		}

		slog.Info("Enhanced note", "title", note.Title, "tmdb_id", tmdbData.TMDBID)
		successCount++
	}

	slog.Info("Enhancement complete",
		"total", len(files),
		"enhanced", successCount,
		"skipped", skipCount,
		"errors", errorCount)

	return nil
}

// findMarkdownFiles finds all markdown files in the given directory.
func findMarkdownFiles(dir string, recursive bool) ([]string, error) {
	var files []string

	if recursive {
		err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() && filepath.Ext(path) == ".md" {
				files = append(files, path)
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	} else {
		entries, err := os.ReadDir(dir)
		if err != nil {
			return nil, err
		}
		for _, entry := range entries {
			if !entry.IsDir() && filepath.Ext(entry.Name()) == ".md" {
				files = append(files, filepath.Join(dir, entry.Name()))
			}
		}
	}

	return files, nil
}

// updateNoteWithTMDBData updates the note file with TMDB enrichment data.
func updateNoteWithTMDBData(filePath string, note *Note, tmdbData *enrichment.TMDBEnrichment, overwrite bool) error {
	// Read the original file
	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// Update frontmatter with TMDB data
	note.AddTMDBData(tmdbData)

	// Build the new file content
	newContent := note.BuildMarkdown(string(content), tmdbData, overwrite)

	// Write back to file
	_, err = fileutil.WriteFileWithOverwrite(filePath, []byte(newContent), 0644, true)
	if err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}
