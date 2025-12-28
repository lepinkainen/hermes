package imdb

import (
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/lepinkainen/hermes/internal/testutil"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

func TestImdbImportE2E(t *testing.T) {
	// Setup test environment
	env := testutil.NewTestEnv(t)
	testutil.SetTestConfig(t)
	tempDir := env.RootDir()

	// Copy golden CSV to test environment
	csvPath := filepath.Join(tempDir, "ratings.csv")
	env.CopyFile("testdata/imdb_sample.csv", "ratings.csv")

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
	prevWriteJSON := writeJSON
	csvFile = csvPath
	outputDir = filepath.Join(tempDir, "output") // Absolute path to temp directory
	tmdbEnabled = false                          // Disable TMDB enrichment for offline e2e test
	writeJSON = false
	defer func() {
		csvFile = prevCSVFile
		outputDir = prevOutputDir
		tmdbEnabled = prevTMDBEnabled
		writeJSON = prevWriteJSON
	}()

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
}
