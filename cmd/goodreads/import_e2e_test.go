package goodreads

import (
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/lepinkainen/hermes/internal/automation"
	"github.com/lepinkainen/hermes/internal/cache"
	"github.com/lepinkainen/hermes/internal/testutil"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

func TestGoodreadsImportE2E(t *testing.T) {
	// Setup test environment
	env := testutil.NewTestEnv(t)
	testutil.SetTestConfig(t)
	tempDir := env.RootDir()

	// Copy golden CSV to test environment
	csvPath := filepath.Join(tempDir, "books.csv")
	env.CopyFile("testdata/goodreads_sample.csv", "books.csv")

	// Setup temp database
	dbPath := filepath.Join(tempDir, "test.db")

	// Save and restore viper settings
	prevDatasetteEnabled := viper.GetBool("datasette.enabled")
	prevDatasetteDB := viper.GetString("datasette.dbfile")
	viper.Set("datasette.enabled", true)
	viper.Set("datasette.dbfile", dbPath)
	defer func() {
		viper.Set("datasette.enabled", prevDatasetteEnabled)
		viper.Set("datasette.dbfile", prevDatasetteDB)
	}()

	// Run importer using ParseGoodreadsWithParams
	// Note: OutputDir must be a relative path that will be joined with the base markdown dir
	// We override markdownoutputdir to point to our temp directory to avoid creating files in the repo
	viper.Set("markdownoutputdir", tempDir)
	defer viper.Set("markdownoutputdir", "markdown")

	err := ParseGoodreadsWithParams(
		ParseParams{
			CSVPath:    csvPath,
			OutputDir:  "output", // Relative path - will become tempDir/output
			WriteJSON:  false,
			JSONOutput: "",
		},
		ParseGoodreads,
		DefaultDownloadGoodreadsCSVFunc,
		&automation.DefaultCDPRunner{},
	)
	require.NoError(t, err)

	// Query the database directly to verify writes
	db, err := sql.Open("sqlite", dbPath)
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	// Verify record count (goodreads_sample.csv has 20 books)
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM goodreads_books").Scan(&count)
	require.NoError(t, err)
	require.Equal(t, 20, count, "Expected 20 books from golden CSV")

	// Spot-check first record
	var firstTitle, firstAuthors string
	var firstID int
	err = db.QueryRow(`
		SELECT id, title, authors
		FROM goodreads_books
		ORDER BY id ASC
		LIMIT 1
	`).Scan(&firstID, &firstTitle, &firstAuthors)
	require.NoError(t, err)
	require.NotEmpty(t, firstTitle, "Title should be present")
	require.NotEmpty(t, firstAuthors, "Authors should be present")

	// Spot-check a middle record
	var middleTitle, middleAuthors string
	err = db.QueryRow(`
		SELECT title, authors
		FROM goodreads_books
		ORDER BY id ASC
		LIMIT 1 OFFSET 10
	`).Scan(&middleTitle, &middleAuthors)
	require.NoError(t, err)
	require.NotEmpty(t, middleTitle, "Middle book title should be present")
	require.NotEmpty(t, middleAuthors, "Middle book authors should be present")

	// Verify that ISBN data is present for at least some books
	var booksWithISBN int
	err = db.QueryRow("SELECT COUNT(*) FROM goodreads_books WHERE isbn != '' OR isbn13 != ''").Scan(&booksWithISBN)
	require.NoError(t, err)
	require.Greater(t, booksWithISBN, 0, "At least some books should have ISBN data")
	t.Logf("Books with ISBN data: %d out of %d", booksWithISBN, count)

	// Verify markdown output structure
	outputPath := filepath.Join(tempDir, "output")
	files, err := filepath.Glob(filepath.Join(outputPath, "*.md"))
	require.NoError(t, err)
	require.Greater(t, len(files), 0, "Should generate markdown files")
	require.Equal(t, 20, len(files), "Should generate 20 markdown files (one per book)")

	// Sort for deterministic selection
	sort.Strings(files)

	// Read and verify first file
	content, err := os.ReadFile(files[0])
	require.NoError(t, err)
	contentStr := string(content)

	// Verify YAML frontmatter structure
	require.Contains(t, contentStr, "---\n", "Should have YAML frontmatter")
	require.Contains(t, contentStr, "title:", "Should have title field")

	// Goodreads-specific field checks
	require.Contains(t, contentStr, "type: book")
	require.Contains(t, contentStr, "goodreads_id:")
	require.Contains(t, contentStr, "authors:")
	require.Contains(t, contentStr, "year:")
	// Note: my_rating is optional (only present if book has been rated)
	// Note: markdown headers (like ## Review) are optional - only present if book has review/notes

	t.Logf("Successfully verified markdown output for %d books", len(files))
}

func TestGoodreadsImportE2E_DatasetteDisabled(t *testing.T) {
	// Setup test environment
	env := testutil.NewTestEnv(t)
	testutil.SetTestConfig(t)
	tempDir := env.RootDir()

	// Copy golden CSV to test environment
	csvPath := filepath.Join(tempDir, "books.csv")
	env.CopyFile("testdata/goodreads_sample.csv", "books.csv")

	// Disable datasette
	prevDatasetteEnabled := viper.GetBool("datasette.enabled")
	viper.Set("datasette.enabled", false)
	defer viper.Set("datasette.enabled", prevDatasetteEnabled)

	// Override markdown output directory to tempDir
	viper.Set("markdownoutputdir", tempDir)
	defer viper.Set("markdownoutputdir", "markdown")

	// Run importer
	err := ParseGoodreadsWithParams(
		ParseParams{
			CSVPath:    csvPath,
			OutputDir:  "output", // Relative path - will become tempDir/output
			WriteJSON:  false,
			JSONOutput: "",
		},
		ParseGoodreads,
		DefaultDownloadGoodreadsCSVFunc,
		&automation.DefaultCDPRunner{},
	)
	require.NoError(t, err)

	// Verify NO database file was created
	// (We don't set a dbfile path when datasette is disabled, so check the default)
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
	require.Equal(t, 20, len(files), "Expected 20 markdown files (one per book)")
	t.Logf("Generated %d markdown files with datasette disabled", len(files))
}

func TestGoodreadsImportE2E_JSON(t *testing.T) {
	env := testutil.NewTestEnv(t)
	testutil.SetTestConfig(t)
	tempDir := env.RootDir()

	// Copy fixture
	csvPath := filepath.Join(tempDir, "books.csv")
	env.CopyFile("testdata/goodreads_sample.csv", "books.csv")

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

	// Override markdown output directory
	viper.Set("markdownoutputdir", tempDir)
	defer viper.Set("markdownoutputdir", "markdown")

	// Enable JSON output
	jsonPath := filepath.Join(tempDir, "output.json")

	err := ParseGoodreadsWithParams(
		ParseParams{
			CSVPath:    csvPath,
			OutputDir:  "output",
			WriteJSON:  true,
			JSONOutput: jsonPath,
		},
		ParseGoodreads,
		DefaultDownloadGoodreadsCSVFunc,
		&automation.DefaultCDPRunner{},
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
	require.Len(t, items, 20, "Expected 20 items in JSON output")

	// Verify schema - spot-check first item
	firstItem := items[0]
	require.Contains(t, firstItem, "title")
	require.NotEmpty(t, firstItem["title"])

	// Goodreads-specific field checks
	require.Contains(t, firstItem, "bookId")
	require.Contains(t, firstItem, "authors")
	require.Contains(t, firstItem, "yearPublished")

	t.Logf("Successfully verified JSON output for %d books", len(items))
}

func TestGoodreadsImportE2E_CacheHit(t *testing.T) {
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

	// Copy fixture
	csvPath := filepath.Join(tempDir, "books.csv")
	env.CopyFile("testdata/goodreads_sample.csv", "books.csv")

	// FIRST RUN: Populate cache
	err := ParseGoodreadsWithParams(
		ParseParams{
			CSVPath:    csvPath,
			OutputDir:  "output",
			WriteJSON:  false,
			JSONOutput: "",
		},
		ParseGoodreads,
		DefaultDownloadGoodreadsCSVFunc,
		&automation.DefaultCDPRunner{},
	)
	require.NoError(t, err)

	// Verify cache entries created (openlibrary or googlebooks)
	cacheDB, err := sql.Open("sqlite", cacheDBPath)
	require.NoError(t, err)
	defer func() { _ = cacheDB.Close() }()

	// Check openlibrary_cache
	var openlibraryCount int
	_ = cacheDB.QueryRow("SELECT COUNT(*) FROM openlibrary_cache").Scan(&openlibraryCount)

	// Check googlebooks_cache
	var googlebooksCount int
	_ = cacheDB.QueryRow("SELECT COUNT(*) FROM googlebooks_cache").Scan(&googlebooksCount)

	totalCacheCount := openlibraryCount + googlebooksCount
	require.Greater(t, totalCacheCount, 0, "Cache should have entries after first run")

	// SECOND RUN: Should use cache
	err = ParseGoodreadsWithParams(
		ParseParams{
			CSVPath:    csvPath,
			OutputDir:  "output",
			WriteJSON:  false,
			JSONOutput: "",
		},
		ParseGoodreads,
		DefaultDownloadGoodreadsCSVFunc,
		&automation.DefaultCDPRunner{},
	)
	require.NoError(t, err)

	// Verify cache count unchanged (cache reused)
	var newOpenlibraryCount, newGooglebooksCount int
	_ = cacheDB.QueryRow("SELECT COUNT(*) FROM openlibrary_cache").Scan(&newOpenlibraryCount)
	_ = cacheDB.QueryRow("SELECT COUNT(*) FROM googlebooks_cache").Scan(&newGooglebooksCount)

	newTotalCount := newOpenlibraryCount + newGooglebooksCount
	require.Equal(t, totalCacheCount, newTotalCount,
		"Cache count should be unchanged on second run (cache hit)")

	t.Logf("Cache verified: %d OpenLibrary + %d GoogleBooks entries reused",
		openlibraryCount, googlebooksCount)
}

// fileExists checks if a file exists
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
