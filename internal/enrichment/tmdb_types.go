package enrichment

import (
	"context"

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
}

// TMDBEnrichment holds TMDB enrichment data.
type TMDBEnrichment struct {
	// TMDBID is the TMDB numeric identifier
	TMDBID int
	// TMDBType is either "movie" or "tv"
	TMDBType string
	// CoverPath is the relative path to the downloaded cover image
	CoverPath string
	// CoverFilename is just the filename of the cover
	CoverFilename string
	// RuntimeMins is the runtime in minutes
	RuntimeMins int
	// TotalEpisodes is the total number of episodes (TV shows only)
	TotalEpisodes int
	// GenreTags are the TMDB genre tags
	GenreTags []string
	// ContentMarkdown is the generated TMDB content
	ContentMarkdown string
	// Finished indicates if a TV show has ended (true for "Ended" or "Canceled" status)
	Finished *bool
}
