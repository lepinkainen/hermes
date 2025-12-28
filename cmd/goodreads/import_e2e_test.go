package goodreads

import (
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/lepinkainen/hermes/internal/automation"
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
}
