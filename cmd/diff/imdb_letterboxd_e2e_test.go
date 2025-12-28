package diff

import (
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	"github.com/lepinkainen/hermes/cmd/imdb"
	"github.com/lepinkainen/hermes/cmd/letterboxd"
	"github.com/lepinkainen/hermes/internal/testutil"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

func TestIMDbLetterboxdDiffE2E(t *testing.T) {
	env := testutil.NewTestEnv(t)
	tempDir := env.RootDir()

	// Create fixture databases
	mainDBPath := filepath.Join(tempDir, "hermes.db")
	createFixtureMainDB(t, mainDBPath)

	cacheDBPath := filepath.Join(tempDir, "cache.db")
	createFixtureCacheDB(t, cacheDBPath)

	// Run diff
	report, err := BuildDiffReport(mainDBPath, cacheDBPath, time.Now())
	require.NoError(t, err)

	// Verify diff results
	// Note: "Fuzzy Match Movie" (2019) appears in both databases with same title+year,
	// so it's auto-matched via title+year resolution, not in IMDb-only/Letterboxd-only lists
	require.Len(t, report.ImdbOnly, 2, "Should find 2 IMDb-only movies")
	require.Len(t, report.LetterboxdOnly, 1, "Should find 1 Letterboxd-only movie")
	require.Equal(t, 3, report.Stats.resolvedTitleYear, "Should match 3 movies (2 via IMDb ID + 1 via title+year)")

	// Count fuzzy matches (there may not be any in this test data since exact title+year matching takes precedence)
	var fuzzyMatchCount int
	for _, item := range report.ImdbOnly {
		fuzzyMatchCount += len(item.FuzzyMatches)
	}
	// Note: Fuzzy matches are only shown for items that don't have exact title+year matches

	// Test markdown report
	note, err := BuildIMDbLetterboxdReport(mainDBPath, cacheDBPath, time.Now())
	require.NoError(t, err)
	require.Contains(t, note.Body, "## IMDb-only")
	require.Contains(t, note.Body, "## Letterboxd-only")
	require.Contains(t, note.Body, "Auto-resolved title+year matches:")

	// Test HTML report
	htmlBytes, err := renderDiffHTML(report)
	require.NoError(t, err)
	require.Contains(t, string(htmlBytes), "<html")
	require.Contains(t, string(htmlBytes), "IMDb-only")

	t.Logf("Diff report: %d matched, %d IMDb-only, %d Letterboxd-only, %d fuzzy matches",
		report.Stats.resolvedTitleYear, len(report.ImdbOnly), len(report.LetterboxdOnly), fuzzyMatchCount)
}

func createFixtureMainDB(t *testing.T, dbPath string) {
	t.Helper()

	db, err := sql.Open("sqlite", dbPath)
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	// Use EXPORTED production schemas - NO DRIFT!
	_, err = db.Exec(imdb.IMDbMoviesSchema)
	require.NoError(t, err)

	_, err = db.Exec(letterboxd.LetterboxdMoviesSchema)
	require.NoError(t, err)

	// Matched movies (2): Same IMDb ID
	_, err = db.Exec(`INSERT INTO imdb_movies
		(imdb_id, title, year, my_rating, url, title_type) VALUES
		('tt1234567', 'The Matrix', 1999, 9, 'https://imdb.com/title/tt1234567', 'movie'),
		('tt7654321', 'Inception', 2010, 10, 'https://imdb.com/title/tt7654321', 'movie')`)
	require.NoError(t, err)

	_, err = db.Exec(`INSERT INTO letterboxd_movies
		(letterboxd_id, name, year, imdb_id, rating, letterboxd_uri) VALUES
		('the-matrix-1999', 'The Matrix', 1999, 'tt1234567', 4.5, 'https://letterboxd.com/film/the-matrix/'),
		('inception-2010', 'Inception', 2010, 'tt7654321', 5.0, 'https://letterboxd.com/film/inception/')`)
	require.NoError(t, err)

	// IMDb-only movies (3)
	_, err = db.Exec(`INSERT INTO imdb_movies
		(imdb_id, title, year, my_rating, url, title_type) VALUES
		('tt1111111', 'IMDb Only 1', 2020, 8, 'https://imdb.com/title/tt1111111', 'movie'),
		('tt2222222', 'IMDb Only 2', 2021, 7, 'https://imdb.com/title/tt2222222', 'movie'),
		('tt3333333', 'Fuzzy Match Movie', 2019, 6, 'https://imdb.com/title/tt3333333', 'movie')`)
	require.NoError(t, err)

	// Letterboxd-only movies (2)
	_, err = db.Exec(`INSERT INTO letterboxd_movies
		(letterboxd_id, name, year, rating, letterboxd_uri) VALUES
		('lb-only-1-2022', 'Letterboxd Only 1', 2022, 4.0, 'https://letterboxd.com/film/lb-only-1/'),
		('fuzzy-match-movie-2019', 'Fuzzy Match Movie', 2019, 3.5, 'https://letterboxd.com/film/fuzzy-match-movie/')`)
	require.NoError(t, err)

	t.Logf("Created main database with 5 IMDb movies and 4 Letterboxd movies")
}

func createFixtureCacheDB(t *testing.T, dbPath string) {
	t.Helper()

	db, err := sql.Open("sqlite", dbPath)
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	// Create letterboxd_mapping_cache table
	_, err = db.Exec(`CREATE TABLE letterboxd_mapping_cache (
		letterboxd_uri TEXT PRIMARY KEY NOT NULL,
		tmdb_id INTEGER,
		tmdb_type TEXT,
		imdb_id TEXT,
		cached_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`)
	require.NoError(t, err)

	// Insert cache mappings (for matched movies)
	_, err = db.Exec(`INSERT INTO letterboxd_mapping_cache
		(letterboxd_uri, imdb_id, tmdb_id, tmdb_type) VALUES
		('https://letterboxd.com/film/the-matrix/', 'tt1234567', 603, 'movie'),
		('https://letterboxd.com/film/inception/', 'tt7654321', 27205, 'movie')`)
	require.NoError(t, err)

	t.Logf("Created cache database with 2 mapping entries")
}
