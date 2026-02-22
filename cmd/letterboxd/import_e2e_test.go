//go:build integration

package letterboxd

import (
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/lepinkainen/hermes/internal/cache"
	"github.com/lepinkainen/hermes/internal/testutil"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

func TestLetterboxdImportE2E(t *testing.T) {
	// Setup test environment
	env := testutil.NewTestEnv(t)
	testutil.SetTestConfig(t)

	// Copy golden CSV to test environment
	csvPath := env.Path("movies.csv")
	env.CopyFile("testdata/letterboxd_sample.csv", "movies.csv")

	// Setup datasette database (with automatic cleanup)
	dbPath := testutil.SetupDatasetteDB(t, env)

	// Save and restore package globals (TODO: refactor to use dependency injection)
	prevCSVFile := csvFile
	prevOutputDir := outputDir
	prevTMDBEnabled := tmdbEnabled
	prevOverwrite := overwrite
	prevWriteJSON := writeJSON
	csvFile = csvPath
	outputDir = env.Path("output") // Absolute path to temp directory
	tmdbEnabled = false            // Disable enrichment for offline e2e test
	overwrite = true
	writeJSON = false
	t.Cleanup(func() {
		csvFile = prevCSVFile
		outputDir = prevOutputDir
		tmdbEnabled = prevTMDBEnabled
		overwrite = prevOverwrite
		writeJSON = prevWriteJSON
	})

	// Run the parser
	err := ParseLetterboxd()
	require.NoError(t, err)

	// Query the database directly to verify writes
	db, err := sql.Open("sqlite", dbPath)
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	// Verify record count (letterboxd_sample.csv has 20 movies)
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM letterboxd_movies").Scan(&count)
	require.NoError(t, err)
	require.Equal(t, 20, count, "Expected 20 movies from golden CSV")

	// Spot-check first record
	var firstName, firstLetterboxdID string
	var firstYear int
	var firstRating sql.NullFloat64
	err = db.QueryRow(`
		SELECT name, year, letterboxd_id, rating
		FROM letterboxd_movies
		ORDER BY date ASC
		LIMIT 1
	`).Scan(&firstName, &firstYear, &firstLetterboxdID, &firstRating)
	require.NoError(t, err)
	require.NotEmpty(t, firstName, "Name should be present")
	require.Greater(t, firstYear, 1900, "Year should be valid")
	require.NotEmpty(t, firstLetterboxdID, "Letterboxd ID should be present")

	// Spot-check a middle record
	var middleName string
	var middleYear int
	err = db.QueryRow(`
		SELECT name, year
		FROM letterboxd_movies
		ORDER BY date ASC
		LIMIT 1 OFFSET 10
	`).Scan(&middleName, &middleYear)
	require.NoError(t, err)
	require.NotEmpty(t, middleName, "Middle movie name should be present")
	require.Greater(t, middleYear, 1900, "Middle movie year should be valid")

	// Verify that ratings from CSV are preserved
	// (some movies should have ratings, some shouldn't)
	var ratedCount int
	err = db.QueryRow("SELECT COUNT(*) FROM letterboxd_movies WHERE rating > 0").Scan(&ratedCount)
	require.NoError(t, err)
	t.Logf("Movies with ratings: %d out of %d", ratedCount, count)

	// Verify no enrichment happened (since tmdbEnabled=false)
	// Community rating should be 0 for all movies when enrichment is disabled
	var enrichedCount int
	err = db.QueryRow("SELECT COUNT(*) FROM letterboxd_movies WHERE community_rating > 0").Scan(&enrichedCount)
	require.NoError(t, err)
	require.Equal(t, 0, enrichedCount, "No community ratings should be present when enrichment is disabled")

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

	// Letterboxd-specific field checks
	require.Contains(t, contentStr, "type: movie")
	require.Contains(t, contentStr, "letterboxd_uri:")
	require.Contains(t, contentStr, "year:")
	require.Contains(t, contentStr, "date_watched:")

	// Verify markdown content exists (not just frontmatter)
	require.Contains(t, contentStr, "<!-- LETTERBOXD_DATA_START -->", "Should have Letterboxd content markers")

	t.Logf("Successfully verified markdown output for %d movies", len(files))
}

func TestLetterboxdImportE2E_DatasetteDisabled(t *testing.T) {
	// Setup test environment
	env := testutil.NewTestEnv(t)
	testutil.SetTestConfig(t)

	// Copy golden CSV to test environment
	csvPath := env.Path("movies.csv")
	env.CopyFile("testdata/letterboxd_sample.csv", "movies.csv")

	// Disable datasette and setup markdown output (with automatic cleanup)
	testutil.SetViperValue(t, "datasette.enabled", false)
	testutil.SetupE2EMarkdownOutput(t, env)

	// Save and restore package globals (TODO: refactor to use dependency injection)
	prevCSVFile := csvFile
	prevOutputDir := outputDir
	prevTMDBEnabled := tmdbEnabled
	prevOverwrite := overwrite
	prevWriteJSON := writeJSON
	csvFile = csvPath
	outputDir = env.Path("output") // Absolute path to temp directory
	tmdbEnabled = false
	overwrite = true
	writeJSON = false
	t.Cleanup(func() {
		csvFile = prevCSVFile
		outputDir = prevOutputDir
		tmdbEnabled = prevTMDBEnabled
		overwrite = prevOverwrite
		writeJSON = prevWriteJSON
	})

	// Run importer
	err := ParseLetterboxd()
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

func TestLetterboxdImportE2E_JSON(t *testing.T) {
	env := testutil.NewTestEnv(t)
	testutil.SetTestConfig(t)

	// Copy fixture
	csvPath := env.Path("movies.csv")
	env.CopyFile("testdata/letterboxd_sample.csv", "movies.csv")

	// Setup datasette database (with automatic cleanup)
	testutil.SetupDatasetteDB(t, env)

	// Save and restore package globals (TODO: refactor to use dependency injection)
	prevCSVFile := csvFile
	prevOutputDir := outputDir
	prevTMDBEnabled := tmdbEnabled
	prevOverwrite := overwrite
	prevWriteJSON := writeJSON
	prevJSONOutput := jsonOutput
	csvFile = csvPath
	outputDir = env.Path("output")
	jsonOutput = env.Path("output", "output.json")
	tmdbEnabled = false
	overwrite = true
	writeJSON = true
	t.Cleanup(func() {
		csvFile = prevCSVFile
		outputDir = prevOutputDir
		tmdbEnabled = prevTMDBEnabled
		overwrite = prevOverwrite
		writeJSON = prevWriteJSON
		jsonOutput = prevJSONOutput
	})

	// Run importer
	err := ParseLetterboxd()
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
	require.Contains(t, firstItem, "name")
	require.NotEmpty(t, firstItem["name"])

	// Letterboxd-specific field checks
	require.Contains(t, firstItem, "letterboxdId")
	require.Contains(t, firstItem, "year")

	t.Logf("Successfully verified JSON output for %d movies", len(items))
}

func TestLetterboxdImportE2E_CacheHit(t *testing.T) {
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
	csvPath := env.Path("diary.csv")
	env.CopyFile("testdata/letterboxd_sample.csv", "diary.csv")

	// FIRST RUN: Populate cache
	err := ParseLetterboxdWithParams(
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

	// Verify cache entries created (letterboxd_mapping_cache)
	cacheDB, err := sql.Open("sqlite", cacheDBPath)
	require.NoError(t, err)
	defer func() { _ = cacheDB.Close() }()

	var cacheCount int
	err = cacheDB.QueryRow("SELECT COUNT(*) FROM letterboxd_mapping_cache").Scan(&cacheCount)
	require.NoError(t, err)
	require.Greater(t, cacheCount, 0, "Cache should have entries after first run")

	initialCacheCount := cacheCount

	// SECOND RUN: Should use cache
	err = ParseLetterboxdWithParams(
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
	err = cacheDB.QueryRow("SELECT COUNT(*) FROM letterboxd_mapping_cache").Scan(&cacheCount)
	require.NoError(t, err)
	require.Equal(t, initialCacheCount, cacheCount,
		"Cache count should be unchanged on second run (cache hit)")

	t.Logf("Cache verified: %d Letterboxd mapping entries reused", initialCacheCount)
}

// fileExists checks if a file exists
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
