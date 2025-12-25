package enrichment

import (
	"path/filepath"
)

// SteamEnrichmentOptions holds options for Steam enrichment.
type SteamEnrichmentOptions struct {
	// DownloadCover determines whether to download the cover image
	DownloadCover bool
	// GenerateContent determines whether to generate Steam content sections
	GenerateContent bool
	// ContentSections specifies which sections to generate (empty = default)
	ContentSections []string
	// AttachmentsDir is the directory where images will be stored
	AttachmentsDir string
	// NoteDir is the directory where the note will be stored
	NoteDir string
	// Interactive enables TUI for multiple matches
	Interactive bool
	// Force forces re-enrichment even when Steam AppID exists
	Force bool
}

// SteamOptionsBuilder provides a fluent builder for SteamEnrichmentOptions.
type SteamOptionsBuilder struct {
	opts SteamEnrichmentOptions
}

// NewSteamOptionsBuilder creates a new options builder with the given output directory.
// It sets up AttachmentsDir as outputDir/attachments and NoteDir as outputDir.
func NewSteamOptionsBuilder(outputDir string) *SteamOptionsBuilder {
	return &SteamOptionsBuilder{
		opts: SteamEnrichmentOptions{
			AttachmentsDir: filepath.Join(outputDir, "attachments"),
			NoteDir:        outputDir,
		},
	}
}

// WithCover enables cover download.
func (b *SteamOptionsBuilder) WithCover(download bool) *SteamOptionsBuilder {
	b.opts.DownloadCover = download
	return b
}

// WithContent enables content generation with optional sections.
func (b *SteamOptionsBuilder) WithContent(generate bool, sections []string) *SteamOptionsBuilder {
	b.opts.GenerateContent = generate
	b.opts.ContentSections = sections
	return b
}

// WithInteractive enables interactive TUI mode.
func (b *SteamOptionsBuilder) WithInteractive(interactive bool) *SteamOptionsBuilder {
	b.opts.Interactive = interactive
	return b
}

// WithForce enables force re-enrichment.
func (b *SteamOptionsBuilder) WithForce(force bool) *SteamOptionsBuilder {
	b.opts.Force = force
	return b
}

// Build returns the configured options.
func (b *SteamOptionsBuilder) Build() SteamEnrichmentOptions {
	return b.opts
}

// SteamEnrichment holds Steam enrichment data.
type SteamEnrichment struct {
	// SteamAppID is the Steam numeric application identifier
	SteamAppID int
	// CoverPath is the relative path to the downloaded cover image
	CoverPath string
	// CoverFilename is just the filename of the cover
	CoverFilename string
	// GenreTags are the Steam genre tags
	GenreTags []string
	// ContentMarkdown is the generated Steam content
	ContentMarkdown string
	// Developers is the list of game developers
	Developers []string
	// Publishers is the list of game publishers
	Publishers []string
	// ReleaseDate is the game release date string
	ReleaseDate string
	// MetacriticScore is the Metacritic score (0-100)
	MetacriticScore int
}
