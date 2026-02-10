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
	"github.com/lepinkainen/hermes/internal/enrichment/omdb"
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
	OMDBNoEnrich        bool     `help:"Disable OMDB ratings enrichment" default:"false"`
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
			OMDBEnrich:          !e.OMDBNoEnrich, // Invert: default is enabled
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
	// OMDBEnrich enables OMDB ratings enrichment (default: true)
	OMDBEnrich bool
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
		// Get note directory for file existence checks
		noteDir := filepath.Dir(file)

		// Smart needs detection: determine what needs to be updated
		needsCover := note.NeedsCover(noteDir)
		needsContent := note.NeedsContent()
		needsMetadata := note.TMDBID == 0 // Missing tmdb_id in frontmatter

		// Only consider OMDB needed if: OMDB enabled + has IMDB ID + doesn't have OMDB data yet
		// AND cache doesn't already show no ratings available (to avoid reprocessing loop)
		needsOMDB := false
		if opts.OMDBEnrich && note.IMDBID != "" && !note.HasOMDBData() {
			cacheStatus := omdb.CheckCacheStatus(note.IMDBID)
			switch cacheStatus {
			case omdb.CacheStatusNotCached:
				// Not cached - need to fetch
				needsOMDB = true
			case omdb.CacheStatusHasRatings:
				// Cached with ratings - need to update note
				needsOMDB = true
			case omdb.CacheStatusNotFound:
				// Movie not in OMDB (invalid/incorrect ID) - skip (re-check after 7 day TTL)
				slog.Debug("Skipping OMDB (not in OMDB)", "imdb_id", note.IMDBID)
			case omdb.CacheStatusNoRatings:
				// Movie exists but no ratings - skip (re-check after 24h TTL)
				slog.Debug("Skipping OMDB (no ratings yet)", "imdb_id", note.IMDBID)
			}
		}

		// Skip if already has everything and not forcing/regenerating/overwriting/refreshing
		// Note: metadata is always fetched to ensure all fields are current (uses cache for efficiency)
		if !opts.Force && !opts.RefreshCache && !opts.RegenerateData && !config.OverwriteFiles && !needsCover && !needsContent && !needsMetadata && !needsOMDB {
			// Build concise status message
			status := "TMDB OK"
			if note.HasOMDBData() {
				status += ", OMDB OK"
			}
			slog.Info("Skipping file ("+status+")", "path", file, "tmdb_id", note.TMDBID)
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

		// Convert frontmatter to map for DetectMediaTypeFromTags
		frontmatterMap := make(map[string]any)
		for _, key := range note.Frontmatter.Keys() {
			if val, ok := note.Frontmatter.Get(key); ok {
				frontmatterMap[key] = val
			}
		}
		expectedType := fm.DetectMediaTypeFromTags(frontmatterMap)
		if expectedType == "" {
			expectedType = note.Type
		}
		storedType := note.Frontmatter.GetString("tmdb_type")

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
		searchTitle := note.Title
		searchYear := note.Year
		if searchYear == 0 {
			if parsedTitle, parsedYear, ok := parseTitleYearFromTitle(note.Title); ok {
				searchTitle = parsedTitle
				searchYear = parsedYear
				slog.Debug("Parsed year from title", "title", note.Title, "parsed_title", searchTitle, "year", searchYear)
			}
		}
		tmdbData, err := enrichment.EnrichFromTMDB(ctx, searchTitle, searchYear, note.IMDBID, note.TMDBID, enrichOpts)
		if err != nil {
			slog.Warn("Failed to enrich from TMDB", "title", note.Title, "error", err)
			errorCount++
			continue
		}

		if tmdbData == nil {
			slog.Warn("No TMDB data found for file", "title", note.Title, "path", file)
			skipCount++
			continue
		}

		// Enrich with OMDB ratings if enabled and IMDb ID available
		// Use IMDB ID from TMDB enrichment if available, otherwise fall back to note's existing ID
		var omdbRatings *omdb.RatingsEnrichment
		imdbID := tmdbData.IMDBID
		if imdbID == "" {
			imdbID = note.IMDBID
		}
		if opts.OMDBEnrich && imdbID != "" {
			omdbRatings, err = omdb.EnrichFromOMDB(ctx, imdbID)
			if err != nil {
				slog.Warn("OMDB enrichment failed", "imdb_id", imdbID, "error", err)
				// Don't fail the whole process, just continue without OMDB data
			}
		} else if opts.OMDBEnrich && imdbID == "" {
			slog.Debug("Skipping OMDB enrichment (no IMDb ID)", "title", note.Title)
		}

		// Update the note with TMDB data and OMDB ratings
		if err := updateNoteWithTMDBData(file, note, tmdbData, omdbRatings, opts.RegenerateData); err != nil {
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

// updateNoteWithTMDBData updates the note file with TMDB enrichment data and OMDB ratings.
func updateNoteWithTMDBData(filePath string, note *Note, tmdbData *enrichment.TMDBEnrichment, omdbRatings *omdb.RatingsEnrichment, overwrite bool) error {
	// Read the original file
	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// Update frontmatter with TMDB data
	note.AddTMDBData(tmdbData)

	// Update frontmatter with OMDB ratings
	note.AddOMDBData(omdbRatings)

	// Build the new file content
	newContent := note.BuildMarkdown(string(content), tmdbData, omdbRatings, overwrite)

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
	if uri := note.Frontmatter.GetString("letterboxd_uri"); uri != "" {
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
	noteDir := filepath.Dir(file)

	needsCover := note.NeedsCover(noteDir)
	needsContent := note.NeedsSteamContent()

	// Skip if already has everything and not forcing/regenerating/overwriting/refreshing
	if !opts.Force && !opts.RefreshCache && !opts.RegenerateData && !config.OverwriteFiles && !needsCover && !needsContent {
		slog.Info("Skipping file (Steam OK)", "path", file, "steam_appid", note.SteamAppID)
		return false, true // skip
	}

	if opts.DryRun {
		slog.Info("Would enhance game", "title", note.Title, "file", file,
			"needs_cover", needsCover, "needs_content", needsContent)
		return true, false // success
	}

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
		slog.Warn("No Steam data found for game", "title", note.Title, "path", file)
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
