package steam

import (
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/lepinkainen/hermes/internal/cache"
	"github.com/lepinkainen/hermes/internal/testutil"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

func TestSteamImportE2E(t *testing.T) {
	// Setup test environment
	env := testutil.NewTestEnv(t)
	testutil.SetTestConfig(t)
	tempDir := env.RootDir()

	// Setup temp database
	dbPath := filepath.Join(tempDir, "test.db")

	// Setup temp cache database and pre-populate it with test data
	// This allows the test to run without making actual Steam API calls
	cacheDBPath := filepath.Join(tempDir, "cache.db")

	// Save and restore viper settings BEFORE populating cache
	prevDatasetteEnabled := viper.GetBool("datasette.enabled")
	prevDatasetteDB := viper.GetString("datasette.dbfile")
	prevCacheDB := viper.GetString("cache.dbfile")
	viper.Set("datasette.enabled", true)
	viper.Set("datasette.dbfile", dbPath)
	viper.Set("cache.dbfile", cacheDBPath)
	defer func() {
		viper.Set("datasette.enabled", prevDatasetteEnabled)
		viper.Set("datasette.dbfile", prevDatasetteDB)
		viper.Set("cache.dbfile", prevCacheDB)
	}()

	// Reset the global cache singleton so it picks up our test cache DB
	// This is necessary because the cache is initialized once with the config value
	resetErr := cache.ResetGlobalCache()
	require.NoError(t, resetErr)
	defer func() {
		// Reset again at the end so subsequent tests get a fresh cache
		_ = cache.ResetGlobalCache()
	}()

	// Populate the cache AFTER resetting so the cache system creates the tables
	populateSteamCacheForTesting(t)

	// Override markdown output directory to tempDir
	viper.Set("markdownoutputdir", tempDir)
	defer viper.Set("markdownoutputdir", "markdown")

	// Save and restore package-level variables
	prevSteamID := steamID
	prevAPIKey := apiKey
	prevOutputDir := outputDir
	prevWriteJSON := writeJSON
	prevJSONOutput := jsonOutput
	defer func() {
		steamID = prevSteamID
		apiKey = prevAPIKey
		outputDir = prevOutputDir
		writeJSON = prevWriteJSON
		jsonOutput = prevJSONOutput
	}()

	// Mock ImportSteamGames to return test data without hitting the API
	// We load the fixtures and return them directly
	prevImportFunc := ImportSteamGamesFunc
	ImportSteamGamesFunc = func(sid, key string) ([]Game, error) {
		// Load test fixture
		fixtureData, err := os.ReadFile("testdata/owned_games_response.json")
		if err != nil {
			return nil, err
		}

		var steamResp SteamResponse
		err = json.Unmarshal(fixtureData, &steamResp)
		if err != nil {
			return nil, err
		}

		return steamResp.Response.Games, nil
	}
	defer func() {
		ImportSteamGamesFunc = prevImportFunc
	}()

	// Run importer using ParseSteamWithParams
	err := ParseSteamWithParams(
		"test-steam-id",
		"test-api-key",
		"output", // relative path - will become tempDir/output
		false,    // writeJSON
		"",       // jsonOutput
	)
	require.NoError(t, err)

	// Query the database directly to verify writes
	db, err := sql.Open("sqlite", dbPath)
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	// Verify record count (owned_games_response.json has 3 games)
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM steam_games").Scan(&count)
	require.NoError(t, err)
	require.Equal(t, 3, count, "Expected 3 games from golden fixture")

	// Spot-check first game
	var firstAppID int
	var firstName string
	var firstPlaytime int
	err = db.QueryRow(`
		SELECT appid, name, playtime_forever
		FROM steam_games
		ORDER BY appid ASC
		LIMIT 1
	`).Scan(&firstAppID, &firstName, &firstPlaytime)
	require.NoError(t, err)
	require.Equal(t, 11111, firstAppID, "First game should have appid 11111")
	require.NotEmpty(t, firstName, "Name should be present")
	require.Greater(t, firstPlaytime, 0, "Playtime should be greater than 0")

	// Spot-check a middle record
	var middleAppID int
	var middleName string
	err = db.QueryRow(`
		SELECT appid, name
		FROM steam_games
		ORDER BY appid ASC
		LIMIT 1 OFFSET 1
	`).Scan(&middleAppID, &middleName)
	require.NoError(t, err)
	require.NotEmpty(t, middleName, "Middle game name should be present")

	// Verify that game details were fetched from cache
	var detailsCount int
	err = db.QueryRow("SELECT COUNT(*) FROM steam_games WHERE details_fetched = true").Scan(&detailsCount)
	require.NoError(t, err)
	require.Greater(t, detailsCount, 0, "At least some games should have details fetched from cache")
	t.Logf("Games with details fetched: %d out of %d", detailsCount, count)
}

// populateSteamCacheForTesting populates the cache with test data using the cache API
func populateSteamCacheForTesting(t *testing.T) {
	t.Helper()

	// Get the global cache instance (this will create tables if they don't exist)
	globalCache, err := cache.GetGlobalCache()
	require.NoError(t, err)
	require.NotNil(t, globalCache)

	// Pre-populate cache with test responses for each game
	// This allows the E2E test to run without making actual API calls
	// Note: The cache stores GameDetails objects as JSON, not the raw API response
	testCases := []struct {
		appID   string
		fixture string
	}{
		{"12345", "testdata/app_details_success.json"},
		{"67890", "testdata/app_details_minimal.json"},
		{"11111", "testdata/app_details_success.json"}, // Reuse success fixture
	}

	for _, tc := range testCases {
		fixtureData, err := os.ReadFile(tc.fixture)
		require.NoError(t, err)

		// Parse the fixture to extract the GameDetails
		var result map[string]struct {
			Success bool        `json:"success"`
			Data    GameDetails `json:"data"`
		}
		err = json.Unmarshal(fixtureData, &result)
		require.NoError(t, err)

		// Get the first (and only) entry from the map
		var gameDetails *GameDetails
		for _, v := range result {
			if v.Success {
				gameDetails = &v.Data
				break
			}
		}
		require.NotNil(t, gameDetails, "Failed to parse game details from fixture")

		// Serialize the GameDetails to JSON for caching
		detailsJSON, err := json.Marshal(gameDetails)
		require.NoError(t, err)

		// Use the cache API to store the data
		err = globalCache.Set("steam_cache", tc.appID, string(detailsJSON))
		require.NoError(t, err)
	}

	t.Logf("Pre-populated cache with %d entries", len(testCases))
}
