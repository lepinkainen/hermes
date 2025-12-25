// Package enhance provides functionality to enrich existing markdown notes with TMDB data.
package enhance

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"path/filepath"

	"github.com/lepinkainen/hermes/cmd/letterboxd"
	"github.com/lepinkainen/hermes/internal/config"
	"github.com/lepinkainen/hermes/internal/enrichment"
	"github.com/lepinkainen/hermes/internal/fileutil"
	fm "github.com/lepinkainen/hermes/internal/frontmatter"
	"github.com/spf13/viper"
)

// EnhanceCmd represents the enhance command
type EnhanceCmd struct {
	InputDirs           []string `short:"d" help:"Directories containing markdown files to enhance (can specify multiple)" required:""`
	Recursive           bool     `short:"r" help:"Scan subdirectories recursively" default:"false"`
	DryRun              bool     `help:"Show what would be done without making changes" default:"false"`
	RegenerateData      bool     `help:"Regenerate data sections (TMDB/Steam) even if they already exist" default:"false"`
	Force               bool     `short:"f" help:"Force re-enrichment even when TMDB ID exists in frontmatter" default:"false"`
	RefreshCache        bool     `help:"Refresh TMDB cache without re-searching for matches" default:"false"`
	TMDBNoInteractive   bool     `help:"Disable interactive TUI for TMDB selection (auto-select first result)" default:"false"`
	TMDBContentSections []string `help:"Specific TMDB content sections to generate (empty = all)"`
}

func (e *EnhanceCmd) Run() error {
	for _, inputDir := range e.InputDirs {
		opts := Options{
			InputDir:            inputDir,
			Recursive:           e.Recursive,
			DryRun:              e.DryRun,
			RegenerateData:      e.RegenerateData,
			Force:               e.Force,
			RefreshCache:        e.RefreshCache,
			TMDBDownloadCover:   true,                 // Always download covers
			TMDBInteractive:     !e.TMDBNoInteractive, // Invert: default is interactive
			TMDBContentSections: e.TMDBContentSections,
			UseTMDBCoverCache:   viper.GetBool("tmdb.cover_cache.enabled"),
			TMDBCoverCachePath:  viper.GetString("tmdb.cover_cache.path"),
		}

		if err := EnhanceNotesFunc(opts); err != nil {
			return err
		}
	}

	return nil
}

var EnhanceNotesFunc = EnhanceNotes

// Options holds configuration for the enhance command.
type Options struct {
	// InputDir is the directory containing markdown files to enhance
	InputDir string
	// Recursive determines whether to scan subdirectories
	Recursive bool
	// TMDBDownloadCover determines whether to download cover images
	TMDBDownloadCover bool
	// TMDBInteractive enables TUI for multiple TMDB matches
	TMDBInteractive bool
	// TMDBContentSections specifies which sections to generate (empty = all)
	TMDBContentSections []string
	// DryRun shows what would be done without making changes
	DryRun bool
	// RegenerateData determines whether to regenerate data sections even if they exist
	RegenerateData bool
	// Force forces re-enrichment even when TMDB ID exists
	Force bool
	// RefreshCache refreshes TMDB cache without re-searching for matches
	RefreshCache bool
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

		// Route to appropriate enrichment based on media type
		if note.IsGame() {
			// Handle game enrichment with Steam
			success, skip := processGameNote(ctx, file, note, opts, attachmentsDir)
			if success {
				successCount++
			} else if skip {
				skipCount++
			} else {
				errorCount++
			}
			continue
		}

		// Handle movie/TV enrichment with TMDB
		// Smart needs detection: determine what needs to be updated
		needsCover := note.NeedsCover()
		needsContent := note.NeedsContent()

		// Skip if already has everything and not forcing/regenerating
		// Note: metadata is always fetched to ensure all fields are current (uses cache for efficiency)
		if !opts.Force && !opts.RegenerateData && !needsCover && !needsContent {
			slog.Info("Skipping file (already has all TMDB data)", "path", file, "tmdb_id", note.TMDBID)
			skipCount++
			continue
		}

		if opts.DryRun {
			slog.Info("Would enhance", "title", note.Title, "year", note.Year, "file", file,
				"needs_cover", needsCover, "needs_content", needsContent)
			successCount++
			continue
		}

		// Prepare enrichment options based on what's needed
		noteDir := filepath.Dir(file)
		expectedType := fm.DetectMediaTypeFromTags(note.RawFrontmatter)
		if expectedType == "" {
			expectedType = note.Type
		}
		storedType := fm.StringFromAny(note.RawFrontmatter["tmdb_type"])

		enrichOpts := enrichment.TMDBEnrichmentOptions{
			DownloadCover:     opts.TMDBDownloadCover && (needsCover || opts.RegenerateData),
			GenerateContent:   needsContent || opts.RegenerateData,
			ContentSections:   opts.TMDBContentSections,
			AttachmentsDir:    attachmentsDir,
			NoteDir:           noteDir,
			Interactive:       opts.TMDBInteractive,
			Force:             opts.Force,
			RefreshCache:      opts.RefreshCache,
			StoredMediaType:   storedType,
			ExpectedMediaType: expectedType,
			UseCoverCache:     opts.UseTMDBCoverCache,
			CoverCachePath:    opts.TMDBCoverCachePath,
		}

		// Resolve Letterboxd URI using 3-tier strategy
		letterboxdURI := resolveLetterboxdURI(note, enrichOpts.StoredMediaType, enrichOpts.ExpectedMediaType)
		enrichOpts.LetterboxdURI = letterboxdURI

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
		if err := updateNoteWithTMDBData(file, note, tmdbData, opts.RegenerateData); err != nil {
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

// resolveLetterboxdURI attempts to find a Letterboxd URI using a 3-tier strategy:
// 1. Check frontmatter for letterboxd_uri
// 2. If TMDB ID exists, check cache for reverse lookup
// 3. Generate search URL as fallback
func resolveLetterboxdURI(note *Note, storedType, expectedType string) string {
	// Tier 1: Check frontmatter first
	if uri := fm.StringFromAny(note.RawFrontmatter["letterboxd_uri"]); uri != "" {
		slog.Debug("Using Letterboxd URI from frontmatter", "uri", uri)
		return uri
	}

	// Tier 2: Check cache if we have TMDB ID
	if note.TMDBID != 0 {
		mediaType := storedType
		if mediaType == "" {
			mediaType = expectedType
		}
		if mediaType == "" {
			mediaType = "movie" // default
		}

		if cachedURI, err := letterboxd.GetLetterboxdURIByTMDB(note.TMDBID, mediaType); err == nil && cachedURI != "" {
			slog.Debug("Found Letterboxd URI in cache", "tmdb_id", note.TMDBID, "uri", cachedURI)
			return cachedURI
		}
	}

	// Tier 3: Generate search URL as fallback
	if note.Title != "" {
		searchURL := generateLetterboxdSearchURL(note.Title)
		slog.Debug("Generated Letterboxd search URL", "title", note.Title, "url", searchURL)
		return searchURL
	}

	return ""
}

// generateLetterboxdSearchURL creates a Letterboxd search URL for the given title.
func generateLetterboxdSearchURL(title string) string {
	return fmt.Sprintf("https://letterboxd.com/search/%s/", url.PathEscape(title))
}

// processGameNote handles enrichment for game notes using Steam.
// Returns (success, skip) booleans.
func processGameNote(ctx context.Context, file string, note *Note, opts Options, attachmentsDir string) (bool, bool) {
	needsCover := note.NeedsCover()
	needsContent := note.NeedsSteamContent()

	// Skip if already has everything and not forcing/regenerating
	if !opts.Force && !opts.RegenerateData && !needsCover && !needsContent {
		slog.Info("Skipping game file (already has all Steam data)", "path", file, "steam_appid", note.SteamAppID)
		return false, true // skip
	}

	if opts.DryRun {
		slog.Info("Would enhance game", "title", note.Title, "file", file,
			"needs_cover", needsCover, "needs_content", needsContent)
		return true, false // success
	}

	noteDir := filepath.Dir(file)

	steamOpts := enrichment.SteamEnrichmentOptions{
		DownloadCover:   opts.TMDBDownloadCover && (needsCover || opts.RegenerateData),
		GenerateContent: needsContent || opts.RegenerateData,
		AttachmentsDir:  attachmentsDir,
		NoteDir:         noteDir,
		Interactive:     opts.TMDBInteractive,
		Force:           opts.Force,
	}

	// Enrich with Steam data
	steamData, err := enrichment.EnrichFromSteam(ctx, note.Title, note.SteamAppID, steamOpts)
	if err != nil {
		slog.Warn("Failed to enrich game from Steam", "title", note.Title, "error", err)
		return false, false // error
	}

	if steamData == nil {
		slog.Debug("No Steam data found", "title", note.Title)
		return false, true // skip
	}

	// Update the note with Steam data
	if err := updateNoteWithSteamData(file, note, steamData, opts.RegenerateData); err != nil {
		slog.Warn("Failed to update game note", "path", file, "error", err)
		return false, false // error
	}

	slog.Info("Enhanced game note", "title", note.Title, "steam_appid", steamData.SteamAppID)
	return true, false // success
}

// updateNoteWithSteamData updates the note file with Steam enrichment data.
func updateNoteWithSteamData(filePath string, note *Note, steamData *enrichment.SteamEnrichment, overwrite bool) error {
	// Read the original file
	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// Update frontmatter with Steam data
	note.AddSteamData(steamData)

	// Build the new file content
	newContent := note.BuildMarkdownForSteam(string(content), steamData, overwrite)

	// Write back to file
	_, err = fileutil.WriteFileWithOverwrite(filePath, []byte(newContent), 0644, true)
	if err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}
