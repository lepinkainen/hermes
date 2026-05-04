package enrichment

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewTMDBOptionsBuilder_Defaults(t *testing.T) {
	outputDir := filepath.Join("tmp", "notes")

	opts := NewTMDBOptionsBuilder(outputDir).Build()

	require.False(t, opts.DownloadCover)
	require.False(t, opts.GenerateContent)
	require.Nil(t, opts.ContentSections)
	require.Equal(t, filepath.Join(outputDir, "attachments"), opts.AttachmentsDir)
	require.Equal(t, outputDir, opts.NoteDir)
	require.False(t, opts.Interactive)
	require.False(t, opts.Force)
	require.False(t, opts.RefreshCache)
	require.False(t, opts.MoviesOnly)
	require.Empty(t, opts.StoredMediaType)
	require.Equal(t, "movie", opts.ExpectedMediaType)
	require.False(t, opts.UseCoverCache)
	require.Empty(t, opts.CoverCachePath)
	require.Empty(t, opts.SourceURL)
	require.Empty(t, opts.LetterboxdURI)
}

func TestTMDBOptionsBuilder_Setters(t *testing.T) {
	sections := []string{"overview", "info"}

	opts := NewTMDBOptionsBuilder("out").
		WithCover(true).
		WithContent(true, sections).
		WithInteractive(true).
		WithMoviesOnly(true).
		WithStoredType("tv").
		WithExpectedType("tv").
		WithCoverCache(true, filepath.Join("cache", "covers")).
		WithSourceURL("https://letterboxd.com/film/heat/").
		Build()

	require.True(t, opts.DownloadCover)
	require.True(t, opts.GenerateContent)
	require.Equal(t, sections, opts.ContentSections)
	require.Equal(t, filepath.Join("out", "attachments"), opts.AttachmentsDir)
	require.Equal(t, "out", opts.NoteDir)
	require.True(t, opts.Interactive)
	require.True(t, opts.MoviesOnly)
	require.Equal(t, "tv", opts.StoredMediaType)
	require.Equal(t, "tv", opts.ExpectedMediaType)
	require.True(t, opts.UseCoverCache)
	require.Equal(t, filepath.Join("cache", "covers"), opts.CoverCachePath)
	require.Equal(t, "https://letterboxd.com/film/heat/", opts.SourceURL)
}

func TestTMDBOptionsBuilder_SettersCanDisableValues(t *testing.T) {
	opts := NewTMDBOptionsBuilder("out").
		WithCover(true).
		WithContent(true, []string{"overview"}).
		WithInteractive(true).
		WithMoviesOnly(true).
		WithCoverCache(true, "cache").
		WithCover(false).
		WithContent(false, nil).
		WithInteractive(false).
		WithMoviesOnly(false).
		WithCoverCache(false, "").
		Build()

	require.False(t, opts.DownloadCover)
	require.False(t, opts.GenerateContent)
	require.Nil(t, opts.ContentSections)
	require.False(t, opts.Interactive)
	require.False(t, opts.MoviesOnly)
	require.False(t, opts.UseCoverCache)
	require.Empty(t, opts.CoverCachePath)
}
