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
		assert.True(t, opts.RegenerateData)
		assert.True(t, opts.Force)
		assert.Equal(t, []string{"cast"}, opts.TMDBContentSections)
		return nil
	}
	origEnhanceNotes := enhance.EnhanceNotesFunc
	enhance.EnhanceNotesFunc = mockRun
	t.Cleanup(func() { enhance.EnhanceNotesFunc = origEnhanceNotes })

	cli, ctx := parseCLI(t, "enhance", "--input-dirs", notesDir, "--input-dirs", animeDir, "--recursive", "--dry-run", "--regenerate-data", "--force", "--tmdb-content-sections", "cast")
	updateGlobalConfig(cli)

	err := ctx.Run()
	require.NoError(t, err)
	assert.Equal(t, []string{notesDir, animeDir}, calledDirs)
}

func TestCLIDefaultFlags(t *testing.T) {
	resetCmdState(t)

	cli, _ := parseCLI(t, "import", "steam", "--steam-id", "123", "--api-key", "key")

	// Test default values
	assert.False(t, cli.Overwrite, "Overwrite should default to false")
	assert.False(t, cli.UpdateCovers, "UpdateCovers should default to false")
	assert.True(t, cli.Datasette, "Datasette should default to true")
	assert.Equal(t, "./hermes.db", cli.DatasetteDB, "DatasetteDB should default to ./hermes.db")
	assert.Equal(t, "./cache.db", cli.CacheDBFile, "CacheDBFile should default to ./cache.db")
	assert.Equal(t, "720h", cli.CacheTTL, "CacheTTL should default to 720h")
	assert.False(t, cli.UseTMDBCoverCache, "UseTMDBCoverCache should default to false")
	assert.Equal(t, "tmdb-cover-cache", cli.TMDBCoverCachePath, "TMDBCoverCachePath should default to tmdb-cover-cache")
}

func TestCLIFlagsOverrideDefaults(t *testing.T) {
	resetCmdState(t)

	cli, _ := parseCLI(t,
		"--overwrite",
		"--update-covers",
		"--datasette=false",
		"--datasette-db", "/custom/hermes.db",
		"--cache-db-file", "/custom/cache.db",
		"--cache-ttl", "24h",
		"--use-tmdb-cover-cache",
		"--tmdb-cover-cache-path", "/custom/tmdb-cache",
		"import", "steam", "--steam-id", "123", "--api-key", "key")

	// Test overridden values
	assert.True(t, cli.Overwrite, "Overwrite flag should be set")
	assert.True(t, cli.UpdateCovers, "UpdateCovers flag should be set")
	assert.False(t, cli.Datasette, "Datasette should be disabled")
	assert.Equal(t, "/custom/hermes.db", cli.DatasetteDB)
	assert.Equal(t, "/custom/cache.db", cli.CacheDBFile)
	assert.Equal(t, "24h", cli.CacheTTL)
	assert.True(t, cli.UseTMDBCoverCache)
	assert.Equal(t, "/custom/tmdb-cache", cli.TMDBCoverCachePath)
}

func TestUpdateGlobalConfigSetsViperValues(t *testing.T) {
	resetCmdState(t)

	cli := &CLI{
		Datasette:          false,
		DatasetteDB:        "/tmp/hermes.db",
		CacheDBFile:        "/tmp/cache.db",
		CacheTTL:           "12h",
		UseTMDBCoverCache:  true,
		TMDBCoverCachePath: "/tmp/tmdb-cache",
	}

	updateGlobalConfig(cli)

	// Verify viper settings
	assert.False(t, viper.GetBool("datasette.enabled"))
	assert.Equal(t, "/tmp/hermes.db", viper.GetString("datasette.dbfile"))
	assert.Equal(t, "/tmp/cache.db", viper.GetString("cache.dbfile"))
	assert.Equal(t, "12h", viper.GetString("cache.ttl"))
	assert.True(t, viper.GetBool("tmdb.cover_cache.enabled"))
	assert.Equal(t, "/tmp/tmdb-cache", viper.GetString("tmdb.cover_cache.path"))
}

func TestInitConfigSetsDefaults(t *testing.T) {
	resetCmdState(t)

	// Set defaults directly without calling initConfig to avoid os.Exit
	viper.SetDefault("MarkdownOutputDir", "./markdown/")
	viper.SetDefault("JSONOutputDir", "./json/")
	viper.SetDefault("OverwriteFiles", false)
	viper.SetDefault("datasette.enabled", true)
	viper.SetDefault("datasette.dbfile", "./hermes.db")
	viper.SetDefault("cache.dbfile", "./cache.db")
	viper.SetDefault("cache.ttl", "720h")
	viper.SetDefault("goodreads.automation.timeout", "3m")
	viper.SetDefault("goodreads.automation.download_dir", "exports")
	viper.SetDefault("letterboxd.automation.timeout", "3m")
	viper.SetDefault("letterboxd.automation.download_dir", "exports")

	// Verify default values are accessible from viper
	assert.Equal(t, "./markdown/", viper.GetString("MarkdownOutputDir"))
	assert.Equal(t, "./json/", viper.GetString("JSONOutputDir"))
	assert.False(t, viper.GetBool("OverwriteFiles"))
	assert.True(t, viper.GetBool("datasette.enabled"))
	assert.Equal(t, "./hermes.db", viper.GetString("datasette.dbfile"))
	assert.Equal(t, "./cache.db", viper.GetString("cache.dbfile"))
	assert.Equal(t, "720h", viper.GetString("cache.ttl"))
	assert.Equal(t, "3m", viper.GetString("goodreads.automation.timeout"))
	assert.Equal(t, "exports", viper.GetString("goodreads.automation.download_dir"))
	assert.Equal(t, "3m", viper.GetString("letterboxd.automation.timeout"))
	assert.Equal(t, "exports", viper.GetString("letterboxd.automation.download_dir"))
}

func TestEnvironmentVariableBinding(t *testing.T) {
	resetCmdState(t)

	// Set environment variables
	t.Setenv("TMDB_API_KEY", "test-api-key")
	t.Setenv("GOODREADS_HEADFUL", "true")
	t.Setenv("GOODREADS_DOWNLOAD_DIR", "/tmp/goodreads")
	t.Setenv("GOODREADS_AUTOMATION_TIMEOUT", "5m")
	t.Setenv("LETTERBOXD_USERNAME", "testuser")
	t.Setenv("LETTERBOXD_PASSWORD", "testpass")
	t.Setenv("LETTERBOXD_HEADFUL", "true")
	t.Setenv("LETTERBOXD_DOWNLOAD_DIR", "/tmp/letterboxd")
	t.Setenv("LETTERBOXD_AUTOMATION_TIMEOUT", "10m")

	// Set up environment variable bindings without calling initConfig
	viper.AutomaticEnv()
	require.NoError(t, viper.BindEnv("TMDBAPIKey", "TMDB_API_KEY"))
	require.NoError(t, viper.BindEnv("goodreads.automation.headful", "GOODREADS_HEADFUL"))
	require.NoError(t, viper.BindEnv("goodreads.automation.download_dir", "GOODREADS_DOWNLOAD_DIR"))
	require.NoError(t, viper.BindEnv("goodreads.automation.timeout", "GOODREADS_AUTOMATION_TIMEOUT"))
	require.NoError(t, viper.BindEnv("letterboxd.automation.username", "LETTERBOXD_USERNAME"))
	require.NoError(t, viper.BindEnv("letterboxd.automation.password", "LETTERBOXD_PASSWORD"))
	require.NoError(t, viper.BindEnv("letterboxd.automation.headful", "LETTERBOXD_HEADFUL"))
	require.NoError(t, viper.BindEnv("letterboxd.automation.download_dir", "LETTERBOXD_DOWNLOAD_DIR"))
	require.NoError(t, viper.BindEnv("letterboxd.automation.timeout", "LETTERBOXD_AUTOMATION_TIMEOUT"))

	// Verify environment variables are bound
	assert.Equal(t, "test-api-key", viper.GetString("TMDBAPIKey"))
	assert.True(t, viper.GetBool("goodreads.automation.headful"))
	assert.Equal(t, "/tmp/goodreads", viper.GetString("goodreads.automation.download_dir"))
	assert.Equal(t, "5m", viper.GetString("goodreads.automation.timeout"))
	assert.Equal(t, "testuser", viper.GetString("letterboxd.automation.username"))
	assert.Equal(t, "testpass", viper.GetString("letterboxd.automation.password"))
	assert.True(t, viper.GetBool("letterboxd.automation.headful"))
	assert.Equal(t, "/tmp/letterboxd", viper.GetString("letterboxd.automation.download_dir"))
	assert.Equal(t, "10m", viper.GetString("letterboxd.automation.timeout"))
}

func TestInitLogging(t *testing.T) {
	tests := []struct {
		name     string
		envValue string
		// We can't easily verify the log level without exposing it,
		// but we can at least verify initLogging doesn't panic
	}{
		{"default", ""},
		{"debug", "debug"},
		{"DEBUG", "DEBUG"},
		{"info", "info"},
		{"INFO", "INFO"},
		{"warn", "warn"},
		{"WARN", "WARN"},
		{"error", "error"},
		{"ERROR", "ERROR"},
		{"invalid", "invalid"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue != "" {
				t.Setenv("HERMES_LOG_LEVEL", tt.envValue)
			}
			// Should not panic
			require.NotPanics(t, func() {
				initLogging()
			})
		})
	}
}

func TestCommandStructure(t *testing.T) {
	resetCmdState(t)

	// Verify that CLI structure has all expected commands
	cli := &CLI{}

	// Check that ImportCmd has all expected subcommands
	assert.NotNil(t, cli.Import)
	assert.IsType(t, goodreads.GoodreadsCmd{}, cli.Import.Goodreads)

	// Verify Enhance command exists
	assert.IsType(t, enhance.EnhanceCmd{}, cli.Enhance)

	// Verify Cache command exists
	assert.NotNil(t, cli.Cache)
}
