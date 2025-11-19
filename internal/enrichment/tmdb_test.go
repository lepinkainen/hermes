package enrichment

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/lepinkainen/hermes/internal/config"
	"github.com/lepinkainen/hermes/internal/tmdb"
	"github.com/stretchr/testify/require"
)

type fakeTMDBClient struct {
	onMetadataByID     func(ctx context.Context, id int, mediaType string, force bool) (*tmdb.Metadata, bool, error)
	onSearchMovies     func(ctx context.Context, query string, year int, limit int) ([]tmdb.SearchResult, bool, error)
	onSearchMulti      func(ctx context.Context, query string, year int, limit int) ([]tmdb.SearchResult, bool, error)
	onFindByIMDBID     func(ctx context.Context, imdbID string) (int, string, bool, error)
	onCoverURL         func(ctx context.Context, mediaID int, mediaType string) (string, error)
	onDownload         func(ctx context.Context, imageURL, destPath string, maxWidth int) error
	onFullMovieDetails func(ctx context.Context, movieID int, force bool) (map[string]any, bool, error)
	onFullTVDetails    func(ctx context.Context, tvID int, force bool) (map[string]any, bool, error)
}

func (f *fakeTMDBClient) CachedGetMetadataByID(ctx context.Context, id int, mediaType string, force bool) (*tmdb.Metadata, bool, error) {
	if f.onMetadataByID != nil {
		return f.onMetadataByID(ctx, id, mediaType, force)
	}
	return nil, false, nil
}

func (f *fakeTMDBClient) CachedSearchMovies(ctx context.Context, query string, year int, limit int) ([]tmdb.SearchResult, bool, error) {
	if f.onSearchMovies != nil {
		return f.onSearchMovies(ctx, query, year, limit)
	}
	return nil, false, nil
}

func (f *fakeTMDBClient) CachedSearchMulti(ctx context.Context, query string, year int, limit int) ([]tmdb.SearchResult, bool, error) {
	if f.onSearchMulti != nil {
		return f.onSearchMulti(ctx, query, year, limit)
	}
	return nil, false, nil
}

func (f *fakeTMDBClient) CachedFindByIMDBID(ctx context.Context, imdbID string) (int, string, bool, error) {
	if f.onFindByIMDBID != nil {
		return f.onFindByIMDBID(ctx, imdbID)
	}
	return 0, "", false, nil
}

func (f *fakeTMDBClient) GetCoverURLByID(ctx context.Context, mediaID int, mediaType string) (string, error) {
	if f.onCoverURL != nil {
		return f.onCoverURL(ctx, mediaID, mediaType)
	}
	return "", nil
}

func (f *fakeTMDBClient) DownloadAndResizeImage(ctx context.Context, imageURL, destPath string, maxWidth int) error {
	if f.onDownload != nil {
		return f.onDownload(ctx, imageURL, destPath, maxWidth)
	}
	return nil
}

func (f *fakeTMDBClient) CachedGetFullMovieDetails(ctx context.Context, movieID int, force bool) (map[string]any, bool, error) {
	if f.onFullMovieDetails != nil {
		return f.onFullMovieDetails(ctx, movieID, force)
	}
	return nil, false, nil
}

func (f *fakeTMDBClient) CachedGetFullTVDetails(ctx context.Context, tvID int, force bool) (map[string]any, bool, error) {
	if f.onFullTVDetails != nil {
		return f.onFullTVDetails(ctx, tvID, force)
	}
	return nil, false, nil
}

func withFakeClient(t *testing.T, client tmdbClient) func() {
	t.Helper()
	originalFactory := newTMDBClient
	newTMDBClient = func(string) tmdbClient {
		return client
	}
	return func() { newTMDBClient = originalFactory }
}

func withTMDBAPIKey(t *testing.T, value string) func() {
	t.Helper()
	orig := config.TMDBAPIKey
	config.TMDBAPIKey = value
	return func() { config.TMDBAPIKey = orig }
}

func TestEnrichFromTMDB_UsesExistingIDAndDownloadsCover(t *testing.T) {
	restoreClient := withFakeClient(t, &fakeTMDBClient{
		onMetadataByID: func(ctx context.Context, id int, mediaType string, force bool) (*tmdb.Metadata, bool, error) {
			require.Equal(t, 949, id)
			require.Equal(t, "movie", mediaType)
			runtime := 170
			return &tmdb.Metadata{Runtime: &runtime, GenreTags: []string{"Action", "Crime"}}, false, nil
		},
		onCoverURL: func(ctx context.Context, mediaID int, mediaType string) (string, error) {
			return "https://example.com/cover.jpg", nil
		},
		onDownload: func(ctx context.Context, imageURL, destPath string, maxWidth int) error {
			require.Equal(t, "https://example.com/cover.jpg", imageURL)
			return nil
		},
		onFullMovieDetails: func(ctx context.Context, movieID int, force bool) (map[string]any, bool, error) {
			return map[string]any{
				"overview":     "Classic saga.",
				"status":       "Released",
				"vote_average": 8.2,
				"vote_count":   1234,
			}, false, nil
		},
	})
	defer restoreClient()
	restoreKey := withTMDBAPIKey(t, "test-key")
	defer restoreKey()

	tempDir := t.TempDir()
	opts := TMDBEnrichmentOptions{
		DownloadCover:   true,
		GenerateContent: true,
		AttachmentsDir:  filepath.Join(tempDir, "attachments"),
		NoteDir:         tempDir,
		ContentSections: []string{"overview", "info"},
	}

	enrichment, err := EnrichFromTMDB(context.Background(), "Heat", 1995, "tt0113277", 949, opts)
	require.NoError(t, err)
	require.NotNil(t, enrichment)

	require.Equal(t, 949, enrichment.TMDBID)
	require.Equal(t, "movie", enrichment.TMDBType)
	require.Equal(t, "Heat - cover.jpg", enrichment.CoverFilename)
	require.Equal(t, filepath.Join("attachments", "Heat - cover.jpg"), filepath.ToSlash(enrichment.CoverPath))
	require.Contains(t, enrichment.ContentMarkdown, "![[Heat - cover.jpg|250]]")
	require.Contains(t, enrichment.ContentMarkdown, "## Movie Info")
	require.Equal(t, 170, enrichment.RuntimeMins)
	require.ElementsMatch(t, []string{"Action", "Crime"}, enrichment.GenreTags)
}

func TestEnrichFromTMDB_SearchesWhenNoExistingID(t *testing.T) {
	var searchedMulti bool
	var lookedUpMetadata int
	restoreClient := withFakeClient(t, &fakeTMDBClient{
		onFindByIMDBID: func(ctx context.Context, imdbID string) (int, string, bool, error) {
			return 0, "", false, nil
		},
		onSearchMulti: func(ctx context.Context, query string, year int, limit int) ([]tmdb.SearchResult, bool, error) {
			searchedMulti = true
			return []tmdb.SearchResult{
				{ID: 1, MediaType: "movie", Title: "Alpha", VoteCount: 50},
				{ID: 2, MediaType: "movie", Title: "Alpha", VoteCount: 150},
			}, false, nil
		},
		onMetadataByID: func(ctx context.Context, id int, mediaType string, force bool) (*tmdb.Metadata, bool, error) {
			lookedUpMetadata = id
			require.Equal(t, "movie", mediaType)
			runtime := 120
			return &tmdb.Metadata{Runtime: &runtime}, false, nil
		},
	})
	defer restoreClient()
	restoreKey := withTMDBAPIKey(t, "test-key")
	defer restoreKey()

	enrichment, err := EnrichFromTMDB(context.Background(), "Alpha", 2024, "", 0, TMDBEnrichmentOptions{
		GenerateContent: false,
	})
	require.NoError(t, err)
	require.NotNil(t, enrichment)

	require.True(t, searchedMulti, "expected multi search to run")
	require.Equal(t, 2, lookedUpMetadata, "should ignore low-vote result")
	require.Equal(t, 2, enrichment.TMDBID)
	require.Equal(t, "movie", enrichment.TMDBType)
	require.Equal(t, 120, enrichment.RuntimeMins)
	require.Equal(t, "", enrichment.ContentMarkdown)
}

func TestFindTMDBIDByIMDBIDHandlesErrors(t *testing.T) {
	client := &fakeTMDBClient{
		onFindByIMDBID: func(ctx context.Context, imdbID string) (int, string, bool, error) {
			return 0, "", false, errors.New("boom")
		},
	}

	id, mediaType := findTMDBIDByIMDBID(context.Background(), client, "tt0")
	require.Equal(t, 0, id)
	require.Equal(t, "", mediaType)
}
