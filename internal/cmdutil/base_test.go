package cmdutil

import (
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

func TestSetupOutputDirCreatesMarkdownAndJSONPaths(t *testing.T) {
	t.Cleanup(viper.Reset)

	tempDir := t.TempDir()
	viper.Set("markdownoutputdir", filepath.Join(tempDir, "markdown"))
	viper.Set("jsonoutputdir", filepath.Join(tempDir, "json"))

	cfg := &BaseCommandConfig{
		OutputDir: "",
		ConfigKey: "goodreads",
		WriteJSON: true,
	}

	err := SetupOutputDir(cfg)
	require.NoError(t, err)

	expectedMarkdown := filepath.Join(tempDir, "markdown", "goodreads")
	expectedJSON := filepath.Join(tempDir, "json", "goodreads.json")

	require.Equal(t, expectedMarkdown, cfg.OutputDir)
	require.DirExists(t, cfg.OutputDir)
	require.Equal(t, expectedJSON, cfg.JSONOutput)
	require.DirExists(t, filepath.Dir(cfg.JSONOutput))
}

func TestSetupOutputDirUsesProvidedOutputDir(t *testing.T) {
	t.Cleanup(viper.Reset)

	tempDir := t.TempDir()
	viper.Set("markdownoutputdir", tempDir)

	cfg := &BaseCommandConfig{
		OutputDir: "custom",
		ConfigKey: "ignored",
	}

	err := SetupOutputDir(cfg)
	require.NoError(t, err)

	expectedPath := filepath.Join(tempDir, "custom")
	require.Equal(t, expectedPath, cfg.OutputDir)
	require.DirExists(t, cfg.OutputDir)
}
