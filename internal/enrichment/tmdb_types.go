package enrichment

import (
	"context"
	"path/filepath"

	"github.com/lepinkainen/hermes/internal/tmdb"
)

type tmdbClient interface {
	CachedGetMetadataByID(ctx context.Context, mediaID int, mediaType string, force bool) (*tmdb.Metadata, bool, error)
	CachedSearchMovies(ctx context.Context, query string, year int, limit int) ([]tmdb.SearchResult, bool, error)
	CachedSearchMulti(ctx context.Context, query string, year int, limit int) ([]tmdb.SearchResult, bool, error)
	CachedFindByIMDBID(ctx context.Context, imdbID string) (int, string, bool, error)
	GetCoverURLByID(ctx context.Context, mediaID int, mediaType string) (string, error)
	DownloadAndResizeImage(ctx context.Context, imageURL, destPath string, maxWidth int) error
	CachedGetFullMovieDetails(ctx context.Context, movieID int, force bool) (map[string]any, bool, error)
	CachedGetFullTVDetails(ctx context.Context, tvID int, force bool) (map[string]any, bool, error)
}

var newTMDBClient = func(apiKey string) tmdbClient {
	return tmdb.NewClient(apiKey)
}

// TMDBEnrichmentOptions holds options for TMDB enrichment.
type TMDBEnrichmentOptions struct {
	// DownloadCover determines whether to download the cover image
	DownloadCover bool
	// GenerateContent determines whether to generate TMDB content sections
	GenerateContent bool
	// ContentSections specifies which sections to generate (empty = all)
	ContentSections []string
	// AttachmentsDir is the directory where images will be stored
	AttachmentsDir string
	// NoteDir is the directory where the note will be stored
	NoteDir string
	// Interactive enables TUI for multiple matches
	Interactive bool
	// Force forces re-enrichment even when TMDB ID exists
	Force bool
	// RefreshCache refreshes TMDB cache without re-searching for matches
	RefreshCache bool
	// MoviesOnly restricts search to movies only (excludes TV shows)
	MoviesOnly bool
	// StoredMediaType is the tmdb_type already present in the note (if any)
	StoredMediaType string
	// ExpectedMediaType is an optional hint (movie or tv) used to resolve mismatches
	// between cached TMDB IDs and the note's intended type.
	ExpectedMediaType string
	// UseCoverCache enables development cache for TMDB cover images
	UseCoverCache bool
	// CoverCachePath is the directory for cached cover images
	CoverCachePath string
	// SourceURL is an optional URL to display in the TUI (e.g., Letterboxd URL)
	// to help users verify which exact item they're selecting
	SourceURL string
	// LetterboxdURI is an optional Letterboxd URI to include in TMDB content output
	LetterboxdURI string
}

// TMDBOptionsBuilder provides a fluent builder for TMDBEnrichmentOptions.
// This reduces boilerplate in importers that configure TMDB enrichment.
type TMDBOptionsBuilder struct {
	opts TMDBEnrichmentOptions
}

// NewTMDBOptionsBuilder creates a new options builder with the given output directory.
// It sets up AttachmentsDir as outputDir/attachments and NoteDir as outputDir.
func NewTMDBOptionsBuilder(outputDir string) *TMDBOptionsBuilder {
	return &TMDBOptionsBuilder{
		opts: TMDBEnrichmentOptions{
			AttachmentsDir:    filepath.Join(outputDir, "attachments"),
			NoteDir:           outputDir,
			ExpectedMediaType: "movie", // Default for most importers
		},
	}
}

// WithCover enables cover download.
func (b *TMDBOptionsBuilder) WithCover(download bool) *TMDBOptionsBuilder {
	b.opts.DownloadCover = download
	return b
}

// WithContent enables content generation with optional sections.
func (b *TMDBOptionsBuilder) WithContent(generate bool, sections []string) *TMDBOptionsBuilder {
	b.opts.GenerateContent = generate
	b.opts.ContentSections = sections
	return b
}

// WithInteractive enables interactive TUI mode.
func (b *TMDBOptionsBuilder) WithInteractive(interactive bool) *TMDBOptionsBuilder {
	b.opts.Interactive = interactive
	return b
}

// WithMoviesOnly restricts search to movies only (no TV shows).
func (b *TMDBOptionsBuilder) WithMoviesOnly(moviesOnly bool) *TMDBOptionsBuilder {
	b.opts.MoviesOnly = moviesOnly
	return b
}

// WithStoredType sets the stored media type from an existing note.
func (b *TMDBOptionsBuilder) WithStoredType(storedType string) *TMDBOptionsBuilder {
	b.opts.StoredMediaType = storedType
	return b
}

// WithExpectedType sets the expected media type hint.
func (b *TMDBOptionsBuilder) WithExpectedType(expectedType string) *TMDBOptionsBuilder {
	b.opts.ExpectedMediaType = expectedType
	return b
}

// WithCoverCache enables the cover cache.
func (b *TMDBOptionsBuilder) WithCoverCache(enabled bool, cachePath string) *TMDBOptionsBuilder {
	b.opts.UseCoverCache = enabled
	b.opts.CoverCachePath = cachePath
	return b
}

// WithSourceURL sets the source URL to display in the TUI.
func (b *TMDBOptionsBuilder) WithSourceURL(sourceURL string) *TMDBOptionsBuilder {
	b.opts.SourceURL = sourceURL
	return b
}

// Build returns the configured options.
func (b *TMDBOptionsBuilder) Build() TMDBEnrichmentOptions {
	return b.opts
}

// TMDBEnrichment holds TMDB enrichment data.
type TMDBEnrichment struct {
	// TMDBID is the TMDB numeric identifier
	TMDBID int `json:"tmdbId"`
	// TMDBType is either "movie" or "tv"
	TMDBType string `json:"tmdbType"`
	// IMDBID is the IMDb identifier from TMDB external_ids (e.g., "tt1234567")
	IMDBID string `json:"imdbId,omitempty"`
	// CoverPath is the relative path to the downloaded cover image
	CoverPath string `json:"coverPath,omitempty"`
	// CoverFilename is just the filename of the cover
	CoverFilename string `json:"coverFilename,omitempty"`
	// RuntimeMins is the runtime in minutes
	RuntimeMins int `json:"runtimeMins,omitempty"`
	// TotalEpisodes is the total number of episodes (TV shows only)
	TotalEpisodes int `json:"totalEpisodes,omitempty"`
	// GenreTags are the TMDB genre tags
	GenreTags []string `json:"genreTags,omitempty"`
	// ContentMarkdown is the generated TMDB content
	ContentMarkdown string `json:"contentMarkdown,omitempty"`
	// Finished indicates if a TV show has ended (true for "Ended" or "Canceled" status)
	Finished *bool `json:"finished,omitempty"`
}
