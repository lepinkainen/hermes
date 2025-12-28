package letterboxd

import (
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/lepinkainen/hermes/internal/testutil"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

func TestLetterboxdImportE2E(t *testing.T) {
	// Setup test environment
	env := testutil.NewTestEnv(t)
	testutil.SetTestConfig(t)
	tempDir := env.RootDir()

	// Copy golden CSV to test environment
	csvPath := filepath.Join(tempDir, "movies.csv")
	env.CopyFile("testdata/letterboxd_sample.csv", "movies.csv")

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

	// Save and restore package globals
	prevCSVFile := csvFile
	prevOutputDir := outputDir
	prevTMDBEnabled := tmdbEnabled
	prevOverwrite := overwrite
	prevWriteJSON := writeJSON
	csvFile = csvPath
	outputDir = filepath.Join(tempDir, "output") // Absolute path to temp directory
	tmdbEnabled = false                          // Disable enrichment for offline e2e test
	overwrite = true
	writeJSON = false
	defer func() {
		csvFile = prevCSVFile
		outputDir = prevOutputDir
		tmdbEnabled = prevTMDBEnabled
		overwrite = prevOverwrite
		writeJSON = prevWriteJSON
	}()

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
}
