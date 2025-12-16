package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/alecthomas/kong"
	"github.com/lepinkainen/hermes/cmd/enhance"
	"github.com/lepinkainen/hermes/cmd/goodreads"
	"github.com/lepinkainen/hermes/cmd/steam"
	"github.com/lepinkainen/hermes/internal/config"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func resetCmdState(t *testing.T) {
	origOverwrite := config.OverwriteFiles
	origUpdate := config.UpdateCovers

	t.Cleanup(func() {
		config.OverwriteFiles = origOverwrite
		config.UpdateCovers = origUpdate
		viper.Reset()
	})

	viper.Reset()
	// Explicitly unset or set to false any environment variables that might interfere with tests
	t.Setenv("GOODREADS_AUTOMATED", "false")
}

func parseCLI(t *testing.T, args ...string) (*CLI, *kong.Context) {
	t.Helper()

	originalArgs := os.Args
	os.Args = append([]string{"hermes"}, args...)
	t.Cleanup(func() { os.Args = originalArgs })

	cli := &CLI{}
	ctx := kong.Parse(cli,
		kong.Name("hermes"),
		kong.Description("A tool to import data from various sources into a unified format."),
		kong.UsageOnError(),
		kong.Exit(func(code int) {
			t.Fatalf("unexpected Kong exit %d", code)
		}),
	)

	return cli, ctx
}

func TestUpdateGlobalConfig(t *testing.T) {
	resetCmdState(t)

	cli := &CLI{
		Overwrite:    true,
		UpdateCovers: true,
		Datasette:    false,
		DatasetteDB:  "/tmp/hermes.db",
		CacheDBFile:  "/tmp/cache.db",
		CacheTTL:     "12h",
	}

	updateGlobalConfig(cli)

	assert.True(t, config.OverwriteFiles)
	assert.True(t, config.UpdateCovers)
	assert.False(t, viper.GetBool("datasette.enabled"))
	assert.Equal(t, "/tmp/hermes.db", viper.GetString("datasette.dbfile"))
	assert.Equal(t, "/tmp/cache.db", viper.GetString("cache.dbfile"))
	assert.Equal(t, "12h", viper.GetString("cache.ttl"))
}

func TestGoodreadsRunUsesConfigFallback(t *testing.T) {
	resetCmdState(t)

	mockRun := func(params goodreads.ParseParams) error {
		assert.Equal(t, "config.csv", params.CSVPath)
		assert.Equal(t, "markdown/goodreads", params.OutputDir)
		assert.False(t, params.Automated)
		return nil
	}
	origParseGoodreads := goodreads.DefaultParseGoodreadsFunc
	goodreads.DefaultParseGoodreadsFunc = mockRun
	t.Cleanup(func() { goodreads.DefaultParseGoodreadsFunc = origParseGoodreads })

	viper.Set("goodreads.csvfile", "config.csv")

	cli, ctx := parseCLI(t, "import", "goodreads")
	updateGlobalConfig(cli)

	err := ctx.Run()
	require.NoError(t, err)
}

func TestImportCommandsRequireInput(t *testing.T) {
	resetCmdState(t)

	tests := []struct {
		name string
		args []string
		want string
	}{
		{
			name: "goodreads missing input",
			args: []string{"import", "goodreads"},
			want: "input CSV file is required",
		},
		{
			name: "imdb missing input",
			args: []string{"import", "imdb"},
			want: "input CSV file is required",
		},
		{
			name: "letterboxd missing input",
			args: []string{"import", "letterboxd"},
			want: "input CSV file is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cli, ctx := parseCLI(t, tt.args...)
			updateGlobalConfig(cli)
			err := ctx.Run()
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.want)
		})
	}
}

func TestSteamRunUsesConfig(t *testing.T) {
	resetCmdState(t)

	mockRun := func(steamID, apiKey, output string, json bool, jsonOutput string, overwrite bool) error {
		assert.Equal(t, "steam-id", steamID)
		assert.Equal(t, "api-key", apiKey)
		assert.Equal(t, "steam", output)
		assert.False(t, overwrite)
		return nil
	}
	origParseSteam := steam.ParseSteamWithParamsFunc
	steam.ParseSteamWithParamsFunc = mockRun
	t.Cleanup(func() { steam.ParseSteamWithParamsFunc = origParseSteam })

	viper.Set("steam.steamid", "steam-id")
	viper.Set("steam.apikey", "api-key")

	cli, ctx := parseCLI(t, "import", "steam")
	updateGlobalConfig(cli)

	err := ctx.Run()
	require.NoError(t, err)
}

func TestEnhanceRunPassesOptions(t *testing.T) {
	resetCmdState(t)

	// Create temporary directories
	tempDir := t.TempDir()
	notesDir := filepath.Join(tempDir, "notes")
	animeDir := filepath.Join(tempDir, "anime")
	require.NoError(t, os.MkdirAll(notesDir, 0755))
	require.NoError(t, os.MkdirAll(animeDir, 0755))

	// Mock config.TMDBAPIKey
	origTMDBAPIKey := config.TMDBAPIKey
	config.TMDBAPIKey = "test-key"
	t.Cleanup(func() { config.TMDBAPIKey = origTMDBAPIKey })

	calledDirs := []string{}
	mockRun := func(opts enhance.Options) error {
		t.Logf("mockRun called with InputDir: %s", opts.InputDir)
		calledDirs = append(calledDirs, opts.InputDir)
		assert.True(t, opts.Recursive)
		assert.True(t, opts.DryRun)
		assert.True(t, opts.Overwrite)
		assert.True(t, opts.Force)
		assert.Equal(t, []string{"cast"}, opts.TMDBContentSections)
		return nil
	}
	origEnhanceNotes := enhance.EnhanceNotesFunc
	enhance.EnhanceNotesFunc = mockRun
	t.Cleanup(func() { enhance.EnhanceNotesFunc = origEnhanceNotes })

	cli, ctx := parseCLI(t, "enhance", "--input-dirs", notesDir, "--input-dirs", animeDir, "--recursive", "--dry-run", "--overwrite-tmdb", "--force", "--tmdb-content-sections", "cast")
	updateGlobalConfig(cli)

	err := ctx.Run()
	require.NoError(t, err)
	assert.Equal(t, []string{notesDir, animeDir}, calledDirs)
}
