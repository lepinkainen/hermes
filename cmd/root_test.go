package cmd

import (
	"os"
	"testing"

	"github.com/alecthomas/kong"
	"github.com/lepinkainen/hermes/cmd/enhance"
	"github.com/lepinkainen/hermes/cmd/goodreads"
	"github.com/lepinkainen/hermes/cmd/imdb"
	"github.com/lepinkainen/hermes/cmd/letterboxd"
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

	called := false
	orig := parseGoodreads
	parseGoodreads = func(input, output string, json bool, jsonOutput string, overwrite bool) error {
		called = true
		assert.Equal(t, "config.csv", input)
		assert.Equal(t, "goodreads", output)
		assert.False(t, overwrite)
		return nil
	}
	t.Cleanup(func() { parseGoodreads = orig })

	viper.Set("goodreads.csvfile", "config.csv")

	cli, ctx := parseCLI(t, "import", "goodreads")
	updateGlobalConfig(cli)

	err := ctx.Run()
	require.NoError(t, err)
	assert.True(t, called)
}

func TestImportCommandsRequireInput(t *testing.T) {
	resetCmdState(t)

	parseGoodreads = func(_, _ string, _ bool, _ string, _ bool) error {
		t.Fatalf("goodreads parser should not be called")
		return nil
	}
	parseIMDB = func(_, _ string, _ bool, _ string, _ bool, _ bool, _ bool, _ bool, _ bool, _ []string, _ bool, _ string) error {
		t.Fatalf("imdb parser should not be called")
		return nil
	}
	parseLetterboxd = func(_, _ string, _ bool, _ string, _ bool, _ bool, _ bool, _ bool, _ bool, _ []string, _ bool, _ string) error {
		t.Fatalf("letterboxd parser should not be called")
		return nil
	}
	t.Cleanup(func() {
		parseGoodreads = goodreads.ParseGoodreadsWithParams
		parseIMDB = imdb.ParseImdbWithParams
		parseLetterboxd = letterboxd.ParseLetterboxdWithParams
	})

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

	called := false
	orig := parseSteam
	parseSteam = func(steamID, apiKey, output string, json bool, jsonOutput string, overwrite bool) error {
		called = true
		assert.Equal(t, "steam-id", steamID)
		assert.Equal(t, "api-key", apiKey)
		assert.Equal(t, "steam", output)
		assert.False(t, overwrite)
		return nil
	}
	t.Cleanup(func() { parseSteam = orig })

	viper.Set("steam.steamid", "steam-id")
	viper.Set("steam.apikey", "api-key")

	cli, ctx := parseCLI(t, "import", "steam")
	updateGlobalConfig(cli)

	err := ctx.Run()
	require.NoError(t, err)
	assert.True(t, called)
}

func TestEnhanceRunPassesOptions(t *testing.T) {
	resetCmdState(t)

	calledDirs := []string{}
	orig := runEnhancement
	runEnhancement = func(opts enhance.Options) error {
		calledDirs = append(calledDirs, opts.InputDir)
		assert.True(t, opts.Recursive)
		assert.True(t, opts.DryRun)
		assert.True(t, opts.Overwrite)
		assert.True(t, opts.Force)
		assert.Equal(t, []string{"cast"}, opts.TMDBContentSections)
		return nil
	}
	t.Cleanup(func() { runEnhancement = orig })

	cli, ctx := parseCLI(t, "enhance", "--input-dirs", "./notes", "--input-dirs", "./anime", "--recursive", "--dry-run", "--overwrite-tmdb", "--force", "--tmdb-content-sections", "cast")
	updateGlobalConfig(cli)

	err := ctx.Run()
	require.NoError(t, err)
	assert.Equal(t, []string{"./notes", "./anime"}, calledDirs)
}
