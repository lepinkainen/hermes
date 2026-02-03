package enhance

import (
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/lepinkainen/hermes/cmd/steam"
	"github.com/lepinkainen/hermes/internal/cache"
	"github.com/lepinkainen/hermes/internal/content"
	"github.com/lepinkainen/hermes/internal/fileutil"
	"github.com/lepinkainen/hermes/internal/obsidian"
	"github.com/lepinkainen/hermes/internal/testutil"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

func TestSteamWorkflowE2E_ImportEnhanceExport(t *testing.T) {
	env := testutil.NewTestEnv(t)
	testutil.SetTestConfig(t)

	dbPath := testutil.SetupDatasetteDB(t, env)
	testutil.SetupE2EMarkdownOutput(t, env)

	cacheDBPath := env.Path("cache.db")
	testutil.SetViperValue(t, "cache.dbfile", cacheDBPath)

	require.NoError(t, cache.ResetGlobalCache())
	t.Cleanup(func() { _ = cache.ResetGlobalCache() })
	populateSteamCacheForTesting(t)

	games := loadSteamFixtureGames(t)
	outputDir := filepath.Join(env.RootDir(), "steam")
	require.NoError(t, os.MkdirAll(filepath.Join(outputDir, "attachments"), 0o755))
	for _, game := range games {
		coverName := fileutil.BuildCoverFilename(game.Name)
		coverPath := filepath.Join(outputDir, "attachments", coverName)
		require.NoError(t, os.WriteFile(coverPath, []byte("cover"), 0o644))
	}

	prevImportFunc := steam.ImportSteamGamesFunc
	steam.ImportSteamGamesFunc = func(sid, key string) ([]steam.Game, error) {
		return games, nil
	}
	t.Cleanup(func() { steam.ImportSteamGamesFunc = prevImportFunc })

	require.NoError(t, steam.ParseSteamWithParams("12345", "api-key", "steam", false, "", false))

	db, err := sql.Open("sqlite", dbPath)
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	var count int
	require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM steam_games").Scan(&count))
	require.Equal(t, len(games), count)

	require.NotEmpty(t, games)
	first := games[0]
	notePath := fileutil.GetMarkdownFilePath(first.Name, outputDir)
	noteContent, err := os.ReadFile(notePath)
	require.NoError(t, err)

	note, err := obsidian.ParseMarkdown(noteContent)
	require.NoError(t, err)
	note.Frontmatter.Set("steam_appid", first.AppID)

	updated, err := note.Build()
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(notePath, updated, 0o644))

	require.NoError(t, EnhanceNotes(Options{
		InputDir:           outputDir,
		Recursive:          false,
		DryRun:             false,
		RegenerateData:     true,
		Force:              true,
		RefreshCache:       false,
		TMDBDownloadCover:  true,
		TMDBInteractive:    false,
		UseTMDBCoverCache:  false,
		TMDBCoverCachePath: filepath.Join(outputDir, "tmdb-cover-cache"),
		OMDBEnrich:         false,
	}))

	enhancedContent, err := os.ReadFile(notePath)
	require.NoError(t, err)

	enhancedNote, err := obsidian.ParseMarkdown(enhancedContent)
	require.NoError(t, err)
	require.Equal(t, first.AppID, enhancedNote.Frontmatter.GetInt("steam_appid"))
	require.True(t, content.HasSteamContentMarkers(enhancedNote.Body))
}

func loadSteamFixtureGames(t *testing.T) []steam.Game {
	t.Helper()

	fixtureData, err := os.ReadFile("../steam/testdata/owned_games_response.json")
	require.NoError(t, err)

	var steamResp steam.SteamResponse
	require.NoError(t, json.Unmarshal(fixtureData, &steamResp))

	return steamResp.Response.Games
}

func populateSteamCacheForTesting(t *testing.T) {
	t.Helper()

	globalCache, err := cache.GetGlobalCache()
	require.NoError(t, err)
	require.NotNil(t, globalCache)

	testCases := []struct {
		appID   string
		fixture string
	}{
		{"12345", "../steam/testdata/app_details_success.json"},
		{"67890", "../steam/testdata/app_details_minimal.json"},
		{"11111", "../steam/testdata/app_details_success.json"},
	}

	for _, tc := range testCases {
		fixtureData, err := os.ReadFile(tc.fixture)
		require.NoError(t, err)

		var result map[string]struct {
			Success bool              `json:"success"`
			Data    steam.GameDetails `json:"data"`
		}
		require.NoError(t, json.Unmarshal(fixtureData, &result))

		var gameDetails *steam.GameDetails
		for _, v := range result {
			if v.Success {
				gameDetails = &v.Data
				break
			}
		}
		require.NotNil(t, gameDetails)

		detailsJSON, err := json.Marshal(gameDetails)
		require.NoError(t, err)

		require.NoError(t, globalCache.Set("steam_cache", tc.appID, string(detailsJSON), 0))
	}
}
