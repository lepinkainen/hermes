package steam

import (
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
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
		false,    // fetchAchievements
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

	// Verify markdown output structure
	outputPath := filepath.Join(tempDir, "output")
	files, err := filepath.Glob(filepath.Join(outputPath, "*.md"))
	require.NoError(t, err)
	require.Greater(t, len(files), 0, "Should generate markdown files")
	require.Equal(t, 3, len(files), "Should generate 3 markdown files (one per game)")

	// Sort for deterministic selection
	sort.Strings(files)

	// Read and verify first file
	content, err := os.ReadFile(files[0])
	require.NoError(t, err)
	contentStr := string(content)

	// Verify YAML frontmatter structure
	require.Contains(t, contentStr, "---\n", "Should have YAML frontmatter")
	require.Contains(t, contentStr, "title:", "Should have title field")

	// Steam-specific field checks
	require.Contains(t, contentStr, "type: game")
	require.Contains(t, contentStr, "playtime:")
	require.Contains(t, contentStr, "release_date:")

	// Verify markdown content exists (not just frontmatter)
	require.Regexp(t, `(?m)^#+ `, contentStr, "Should have markdown headers")

	t.Logf("Successfully verified markdown output for %d games", len(files))
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
		err = globalCache.Set("steam_cache", tc.appID, string(detailsJSON), 0)
		require.NoError(t, err)
	}

	t.Logf("Pre-populated cache with %d entries", len(testCases))
}

func TestSteamImportE2E_DatasetteDisabled(t *testing.T) {
	// Setup test environment
	env := testutil.NewTestEnv(t)
	testutil.SetTestConfig(t)
	tempDir := env.RootDir()

	// Setup cache database
	cacheDBPath := filepath.Join(tempDir, "cache.db")
	viper.Set("cache.dbfile", cacheDBPath)
	defer viper.Set("cache.dbfile", "./cache.db")

	// Reset global cache after setting viper config
	resetErr := cache.ResetGlobalCache()
	require.NoError(t, resetErr)
	defer func() { _ = cache.ResetGlobalCache() }()

	// Disable datasette
	prevDatasetteEnabled := viper.GetBool("datasette.enabled")
	viper.Set("datasette.enabled", false)
	defer viper.Set("datasette.enabled", prevDatasetteEnabled)

	// Override markdown output directory to tempDir
	viper.Set("markdownoutputdir", tempDir)
	defer viper.Set("markdownoutputdir", "markdown")

	// Pre-populate cache with game details (similar to main E2E test)
	populateSteamCacheForTesting(t)

	// Mock ImportSteamGames to return test data without hitting the API
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
		false,    // fetchAchievements
	)
	require.NoError(t, err)

	// Verify NO database file was created
	defaultDBPath := filepath.Join(".", "hermes.db")
	require.False(t, fileExists(defaultDBPath),
		"Database file should not be created when datasette is disabled")

	// Verify markdown files WERE created
	outputPath := filepath.Join(tempDir, "output")
	require.DirExists(t, outputPath, "Markdown output directory should exist")

	// Count markdown files
	files, err := filepath.Glob(filepath.Join(outputPath, "*.md"))
	require.NoError(t, err)
	require.Greater(t, len(files), 0, "Markdown files should be generated even when datasette is disabled")
	require.Equal(t, 3, len(files), "Expected 3 markdown files (one per game)")
	t.Logf("Generated %d markdown files with datasette disabled", len(files))
}

func TestSteamImportE2E_JSON(t *testing.T) {
	env := testutil.NewTestEnv(t)
	testutil.SetTestConfig(t)
	tempDir := env.RootDir()

	// Setup database (required for JSON output)
	dbPath := filepath.Join(tempDir, "test.db")
	prevDatasetteEnabled := viper.GetBool("datasette.enabled")
	prevDatasetteDB := viper.GetString("datasette.dbfile")
	viper.Set("datasette.enabled", true)
	viper.Set("datasette.dbfile", dbPath)
	defer func() {
		viper.Set("datasette.enabled", prevDatasetteEnabled)
		viper.Set("datasette.dbfile", prevDatasetteDB)
	}()

	// Setup temp cache database and pre-populate it
	cacheDBPath := filepath.Join(tempDir, "cache.db")
	viper.Set("cache.dbfile", cacheDBPath)
	defer viper.Set("cache.dbfile", "./cache.db")

	// Reset the global cache
	resetErr := cache.ResetGlobalCache()
	require.NoError(t, resetErr)
	defer func() { _ = cache.ResetGlobalCache() }()

	// Populate cache
	populateSteamCacheForTesting(t)

	// Override markdown output directory
	viper.Set("markdownoutputdir", tempDir)
	defer viper.Set("markdownoutputdir", "markdown")

	// Mock ImportSteamGames
	prevImportFunc := ImportSteamGamesFunc
	ImportSteamGamesFunc = func(sid, key string) ([]Game, error) {
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
	defer func() { ImportSteamGamesFunc = prevImportFunc }()

	// Enable JSON output
	jsonPath := filepath.Join(tempDir, "output.json")

	// Run importer
	err := ParseSteamWithParams(
		"test-steam-id",
		"test-api-key",
		"output",
		true,     // writeJSON
		jsonPath, // jsonOutput
		false,    // fetchAchievements
	)
	require.NoError(t, err)

	// Verify JSON file exists
	require.FileExists(t, jsonPath)

	// Parse JSON
	content, err := os.ReadFile(jsonPath)
	require.NoError(t, err)

	var items []map[string]interface{}
	err = json.Unmarshal(content, &items)
	require.NoError(t, err)
	require.Len(t, items, 3, "Expected 3 items in JSON output")

	// Verify schema - spot-check first item
	firstItem := items[0]
	require.Contains(t, firstItem, "name")
	require.NotEmpty(t, firstItem["name"])

	// Steam-specific field checks
	require.Contains(t, firstItem, "appid")
	require.Contains(t, firstItem, "playtime_forever")

	t.Logf("Successfully verified JSON output for %d games", len(items))
}

func TestSteamImportE2E_CacheHit(t *testing.T) {
	env := testutil.NewTestEnv(t)
	testutil.SetTestConfig(t)
	tempDir := env.RootDir()

	// Setup cache DB
	cacheDBPath := filepath.Join(tempDir, "cache.db")
	viper.Set("cache.dbfile", cacheDBPath)
	defer viper.Set("cache.dbfile", "./cache.db")

	// Setup datasette DB
	dbPath := filepath.Join(tempDir, "test.db")
	prevDatasetteEnabled := viper.GetBool("datasette.enabled")
	prevDatasetteDB := viper.GetString("datasette.dbfile")
	viper.Set("datasette.enabled", true)
	viper.Set("datasette.dbfile", dbPath)
	defer func() {
		viper.Set("datasette.enabled", prevDatasetteEnabled)
		viper.Set("datasette.dbfile", prevDatasetteDB)
	}()

	// Override markdown output directory
	viper.Set("markdownoutputdir", tempDir)
	defer viper.Set("markdownoutputdir", "markdown")

	// Reset global cache to pick up test DB
	resetErr := cache.ResetGlobalCache()
	require.NoError(t, resetErr)
	defer func() { _ = cache.ResetGlobalCache() }()

	// Pre-populate cache with game details
	populateSteamCacheForTesting(t)

	// Mock ImportSteamGames to return test data
	prevImportFunc := ImportSteamGamesFunc
	ImportSteamGamesFunc = func(sid, key string) ([]Game, error) {
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

	// FIRST RUN: Should use pre-populated cache
	err := ParseSteamWithParams(
		"test-steam-id",
		"test-api-key",
		"output",
		false, // writeJSON
		"",    // jsonOutput
		false, // fetchAchievements
	)
	require.NoError(t, err)

	// Verify cache entries exist
	cacheDB, err := sql.Open("sqlite", cacheDBPath)
	require.NoError(t, err)
	defer func() { _ = cacheDB.Close() }()

	var cacheCount int
	err = cacheDB.QueryRow("SELECT COUNT(*) FROM steam_cache").Scan(&cacheCount)
	require.NoError(t, err)
	require.Equal(t, 3, cacheCount, "Cache should have 3 entries from pre-population")

	initialCacheCount := cacheCount

	// SECOND RUN: Should use cache without adding new entries
	err = ParseSteamWithParams(
		"test-steam-id",
		"test-api-key",
		"output",
		false, // writeJSON
		"",    // jsonOutput
		false, // fetchAchievements
	)
	require.NoError(t, err)

	// Verify cache count unchanged
	err = cacheDB.QueryRow("SELECT COUNT(*) FROM steam_cache").Scan(&cacheCount)
	require.NoError(t, err)
	require.Equal(t, initialCacheCount, cacheCount,
		"Cache count should be unchanged on second run (cache hit)")

	t.Logf("Cache verified: %d Steam entries reused", initialCacheCount)
}

// fileExists checks if a file exists
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
