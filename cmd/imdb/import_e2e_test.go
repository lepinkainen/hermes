package imdb

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

func TestImdbImportE2E(t *testing.T) {
	// Setup test environment
	env := testutil.NewTestEnv(t)
	testutil.SetTestConfig(t)

	// Copy golden CSV to test environment
	csvPath := env.Path("ratings.csv")
	env.CopyFile("testdata/imdb_sample.csv", "ratings.csv")

	// Setup datasette database (with automatic cleanup)
	dbPath := testutil.SetupDatasetteDB(t, env)

	// Save and restore package globals (TODO: refactor to use dependency injection)
	prevCSVFile := csvFile
	prevOutputDir := outputDir
	prevTMDBEnabled := tmdbEnabled
	prevWriteJSON := writeJSON
	csvFile = csvPath
	outputDir = env.Path("output") // Absolute path to temp directory
	tmdbEnabled = false            // Disable TMDB enrichment for offline e2e test
	writeJSON = false
	t.Cleanup(func() {
		csvFile = prevCSVFile
		outputDir = prevOutputDir
		tmdbEnabled = prevTMDBEnabled
		writeJSON = prevWriteJSON
	})

	// Run the parser
	err := ParseImdb()
	require.NoError(t, err)

	// Query the database directly to verify writes
	db, err := sql.Open("sqlite", dbPath)
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	// Verify record count
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM imdb_movies").Scan(&count)
	require.NoError(t, err)
	require.Equal(t, 20, count, "Expected 20 movies from golden CSV")

	// Spot-check first record
	var firstImdbID, firstTitle string
	var firstYear, firstRating int
	err = db.QueryRow(`
		SELECT imdb_id, title, year, my_rating
		FROM imdb_movies
		ORDER BY position ASC
		LIMIT 1
	`).Scan(&firstImdbID, &firstTitle, &firstYear, &firstRating)
	require.NoError(t, err)
	require.NotEmpty(t, firstImdbID, "IMDb ID should be present")
	require.NotEmpty(t, firstTitle, "Title should be present")
	require.Greater(t, firstYear, 1900, "Year should be valid")
	require.GreaterOrEqual(t, firstRating, 1, "Rating should be 1-10")
	require.LessOrEqual(t, firstRating, 10, "Rating should be 1-10")

	// Spot-check a middle record (use median position)
	var middleImdbID, middleTitle string
	var middleYear int
	err = db.QueryRow(`
		SELECT imdb_id, title, year
		FROM imdb_movies
		ORDER BY position ASC
		LIMIT 1 OFFSET 10
	`).Scan(&middleImdbID, &middleTitle, &middleYear)
	require.NoError(t, err)
	require.NotEmpty(t, middleImdbID, "Middle movie IMDb ID should be present")
	require.NotEmpty(t, middleTitle, "Middle movie title should be present")
	require.Greater(t, middleYear, 1900, "Middle movie year should be valid")

	// Verify no TMDB enrichment happened (since tmdbEnabled=false)
	// We can check this by verifying that OMDB data wasn't fetched
	// (since we're offline and caching would not have data for this test CSV)
	var plotCount int
	err = db.QueryRow("SELECT COUNT(*) FROM imdb_movies WHERE plot != ''").Scan(&plotCount)
	require.NoError(t, err)
	require.Equal(t, 0, plotCount, "No plots should be present when enrichment is disabled")

	// Verify markdown output structure
	files, err := filepath.Glob(filepath.Join(outputDir, "*.md"))
	require.NoError(t, err)
	require.Greater(t, len(files), 0, "Should generate markdown files")
	require.Equal(t, 20, len(files), "Should generate 20 markdown files (one per movie)")

	// Sort for deterministic selection
	sort.Strings(files)

	// Read and verify first file
	content, err := os.ReadFile(files[0])
	require.NoError(t, err)
	contentStr := string(content)

	// Verify YAML frontmatter structure
	require.Contains(t, contentStr, "---\n", "Should have YAML frontmatter")
	require.Contains(t, contentStr, "title:", "Should have title field")

	// IMDb-specific field checks
	require.Contains(t, contentStr, "imdb_id:")
	require.Contains(t, contentStr, "year:")
	require.Contains(t, contentStr, "my_rating:")
	require.Contains(t, contentStr, "seen: true")

	// Verify markdown content exists (not just frontmatter)
	require.Contains(t, contentStr, "<!-- IMDB_DATA_START -->", "Should have IMDb content markers")

	t.Logf("Successfully verified markdown output for %d movies", len(files))
}

func TestImdbImportE2E_DatasetteDisabled(t *testing.T) {
	// Setup test environment
	env := testutil.NewTestEnv(t)
	testutil.SetTestConfig(t)

	// Copy golden CSV to test environment
	csvPath := env.Path("ratings.csv")
	env.CopyFile("testdata/imdb_sample.csv", "ratings.csv")

	// Disable datasette and setup markdown output (with automatic cleanup)
	testutil.SetViperValue(t, "datasette.enabled", false)
	testutil.SetupE2EMarkdownOutput(t, env)

	// Save and restore package globals (TODO: refactor to use dependency injection)
	prevCSVFile := csvFile
	prevOutputDir := outputDir
	prevTMDBEnabled := tmdbEnabled
	prevWriteJSON := writeJSON
	csvFile = csvPath
	outputDir = env.Path("output") // Absolute path to temp directory
	tmdbEnabled = false
	writeJSON = false
	t.Cleanup(func() {
		csvFile = prevCSVFile
		outputDir = prevOutputDir
		tmdbEnabled = prevTMDBEnabled
		writeJSON = prevWriteJSON
	})

	// Run importer
	err := ParseImdb()
	require.NoError(t, err)

	// Verify NO database file was created
	defaultDBPath := filepath.Join(".", "hermes.db")
	require.False(t, fileExists(defaultDBPath),
		"Database file should not be created when datasette is disabled")

	// Verify markdown files WERE created
	outputPath := env.Path("output")
	require.DirExists(t, outputPath, "Markdown output directory should exist")

	// Count markdown files
	files, err := filepath.Glob(filepath.Join(outputPath, "*.md"))
	require.NoError(t, err)
	require.Greater(t, len(files), 0, "Markdown files should be generated even when datasette is disabled")
	require.Equal(t, 20, len(files), "Expected 20 markdown files (one per movie)")
	t.Logf("Generated %d markdown files with datasette disabled", len(files))
}

func TestImdbImportE2E_JSON(t *testing.T) {
	env := testutil.NewTestEnv(t)
	testutil.SetTestConfig(t)

	// Copy fixture
	csvPath := env.Path("ratings.csv")
	env.CopyFile("testdata/imdb_sample.csv", "ratings.csv")

	// Setup datasette database (with automatic cleanup)
	testutil.SetupDatasetteDB(t, env)

	// Save and restore package globals (TODO: refactor to use dependency injection)
	prevCSVFile := csvFile
	prevOutputDir := outputDir
	prevTMDBEnabled := tmdbEnabled
	prevWriteJSON := writeJSON
	prevJSONOutput := jsonOutput
	csvFile = csvPath
	outputDir = env.Path("output")
	tmdbEnabled = false
	writeJSON = true
	jsonOutput = env.Path("output.json")
	t.Cleanup(func() {
		csvFile = prevCSVFile
		outputDir = prevOutputDir
		tmdbEnabled = prevTMDBEnabled
		writeJSON = prevWriteJSON
		jsonOutput = prevJSONOutput
	})

	// Run importer
	err := ParseImdb()
	require.NoError(t, err)

	// Verify JSON file exists
	require.FileExists(t, jsonOutput)

	// Parse JSON
	content, err := os.ReadFile(jsonOutput)
	require.NoError(t, err)

	var items []map[string]interface{}
	err = json.Unmarshal(content, &items)
	require.NoError(t, err)
	require.Len(t, items, 20, "Expected 20 items in JSON output")

	// Verify schema - spot-check first item
	firstItem := items[0]
	require.Contains(t, firstItem, "title")
	require.NotEmpty(t, firstItem["title"])

	// IMDb-specific field checks
	require.Contains(t, firstItem, "imdbId")
	require.Contains(t, firstItem, "year")
	require.Contains(t, firstItem, "myRating")

	t.Logf("Successfully verified JSON output for %d movies", len(items))
}

func TestImdbImportE2E_CacheHit(t *testing.T) {
	env := testutil.NewTestEnv(t)
	testutil.SetTestConfig(t)

	// Setup cache database in test environment
	cacheDBPath := env.Path("cache.db")
	testutil.SetViperValue(t, "cache.dbfile", cacheDBPath)

	// Setup datasette database and markdown output (with automatic cleanup)
	testutil.SetupDatasetteDB(t, env)
	testutil.SetupE2EMarkdownOutput(t, env)

	// Reset global cache to pick up test DB
	resetErr := cache.ResetGlobalCache()
	require.NoError(t, resetErr)
	t.Cleanup(func() { _ = cache.ResetGlobalCache() })

	// Copy fixture
	csvPath := env.Path("ratings.csv")
	env.CopyFile("testdata/imdb_sample.csv", "ratings.csv")

	// FIRST RUN: Populate cache
	err := ParseImdbWithParams(
		csvPath,
		"output",
		false, // writeJSON
		"",    // jsonOutput
		false, // tmdbEnabled
		false, // tmdbDownloadCover
		false, // tmdbGenerateContent
		false, // tmdbInteractive
		nil,   // tmdbContentSections
		false, // useTMDBCoverCache
		"",    // tmdbCoverCachePath
	)
	require.NoError(t, err)

	// Check if OMDB is configured
	omdbConfigured := viper.GetString("imdb.omdb_api_key") != "" || viper.GetString("omdb.api_key") != ""

	if !omdbConfigured {
		t.Skip("OMDB API key not configured, skipping cache verification test")
	}

	// Verify cache entries created
	cacheDB, err := sql.Open("sqlite", cacheDBPath)
	require.NoError(t, err)
	defer func() { _ = cacheDB.Close() }()

	var cacheCount int
	err = cacheDB.QueryRow("SELECT COUNT(*) FROM omdb_cache").Scan(&cacheCount)
	require.NoError(t, err)
	require.Greater(t, cacheCount, 0, "Cache should have entries after first run")

	initialCacheCount := cacheCount

	// SECOND RUN: Should use cache
	err = ParseImdbWithParams(
		csvPath,
		"output",
		false, // writeJSON
		"",    // jsonOutput
		false, // tmdbEnabled
		false, // tmdbDownloadCover
		false, // tmdbGenerateContent
		false, // tmdbInteractive
		nil,   // tmdbContentSections
		false, // useTMDBCoverCache
		"",    // tmdbCoverCachePath
	)
	require.NoError(t, err)

	// Verify cache count unchanged
	err = cacheDB.QueryRow("SELECT COUNT(*) FROM omdb_cache").Scan(&cacheCount)
	require.NoError(t, err)
	require.Equal(t, initialCacheCount, cacheCount,
		"Cache count should be unchanged on second run (cache hit)")

	t.Logf("Cache verified: %d OMDB entries reused", initialCacheCount)
}

// fileExists checks if a file exists
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
