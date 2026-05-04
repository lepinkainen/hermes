package enrichment

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewSteamOptionsBuilder_Defaults(t *testing.T) {
	outputDir := filepath.Join("tmp", "notes")

	opts := NewSteamOptionsBuilder(outputDir).Build()

	require.False(t, opts.DownloadCover)
	require.False(t, opts.GenerateContent)
	require.Nil(t, opts.ContentSections)
	require.Equal(t, filepath.Join(outputDir, "attachments"), opts.AttachmentsDir)
	require.Equal(t, outputDir, opts.NoteDir)
	require.False(t, opts.Interactive)
	require.False(t, opts.Force)
}

func TestSteamOptionsBuilder_Setters(t *testing.T) {
	sections := []string{"overview", "details"}

	opts := NewSteamOptionsBuilder("out").
		WithCover(true).
		WithContent(true, sections).
		WithInteractive(true).
		WithForce(true).
		Build()

	require.True(t, opts.DownloadCover)
	require.True(t, opts.GenerateContent)
	require.Equal(t, sections, opts.ContentSections)
	require.Equal(t, filepath.Join("out", "attachments"), opts.AttachmentsDir)
	require.Equal(t, "out", opts.NoteDir)
	require.True(t, opts.Interactive)
	require.True(t, opts.Force)
}

func TestSteamOptionsBuilder_SettersCanDisableValues(t *testing.T) {
	opts := NewSteamOptionsBuilder("out").
		WithCover(true).
		WithContent(true, []string{"overview"}).
		WithInteractive(true).
		WithForce(true).
		WithCover(false).
		WithContent(false, nil).
		WithInteractive(false).
		WithForce(false).
		Build()

	require.False(t, opts.DownloadCover)
	require.False(t, opts.GenerateContent)
	require.Nil(t, opts.ContentSections)
	require.False(t, opts.Interactive)
	require.False(t, opts.Force)
}
