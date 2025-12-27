package letterboxd

import (
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/lepinkainen/hermes/internal/testutil"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

func TestLoadExistingTMDBID(t *testing.T) {
	env := testutil.NewTestEnv(t)
	tempDir := env.RootDir()
	filePath := filepath.Join(tempDir, "Heat (1995).md")

	content := `---
title: "Heat"
tmdb_id: 949
tmdb_type: movie
imdb_id: tt0113277
tags:
  - letterboxd/movie
  - genre/Action
  - genre/Crime
---

Movie body
`
	err := os.WriteFile(filePath, []byte(content), 0644)
	require.NoError(t, err)

	m := Movie{
		Name: "Heat",
		Year: 1995,
	}

	loadExistingTMDBID(&m, tempDir)

	require.NotNil(t, m.TMDBEnrichment, "tmdb enrichment should be initialized from frontmatter")
	require.Equal(t, 949, m.TMDBEnrichment.TMDBID)
	require.Equal(t, "movie", m.TMDBEnrichment.TMDBType)
	require.Equal(t, "tt0113277", m.ImdbID, "existing imdb id should be reused")
}

func TestWriteMoviesToJSON_UsesExistingTMDBIDWhenSkippingEnrich(t *testing.T) {
	env := testutil.NewTestEnv(t)
	tempDir := env.RootDir()
	filePath := filepath.Join(tempDir, "Heat (1995).md")

	content := `---
title: "Heat"
tmdb_id: 949
tmdb_type: movie
imdb_id: tt0113277
---

Movie body
`
	err := os.WriteFile(filePath, []byte(content), 0644)
	require.NoError(t, err)

	prevOutputDir := outputDir
	prevSkipEnrich := skipEnrich
	prevOverwrite := overwrite
	outputDir = tempDir
	skipEnrich = true
	overwrite = true
	defer func() {
		outputDir = prevOutputDir
		skipEnrich = prevSkipEnrich
		overwrite = prevOverwrite
	}()

	movies := []Movie{
		{Name: "Heat", Year: 1995},
	}

	jsonPath := filepath.Join(tempDir, "movies.json")

	err = writeMoviesToJSON(movies, jsonPath)
	require.NoError(t, err)

	data, err := os.ReadFile(jsonPath)
	require.NoError(t, err)

	var saved []Movie
	err = json.Unmarshal(data, &saved)
	require.NoError(t, err)
	require.Len(t, saved, 1)
	require.NotNil(t, saved[0].TMDBEnrichment, "tmdb enrichment should be propagated from existing note")
	require.Equal(t, 949, saved[0].TMDBEnrichment.TMDBID)
	require.Equal(t, "movie", saved[0].TMDBEnrichment.TMDBType)
	require.Equal(t, "tt0113277", saved[0].ImdbID)
}

func TestParseMovieRecord(t *testing.T) {
	tests := []struct {
		name        string
		record      []string
		wantMovie   Movie
		wantErr     bool
		skipInvalid bool
	}{
		{
			name: "valid record without trailing slash",
			record: []string{
				"2024-01-15",
				"The Matrix",
				"1999",
				"https://letterboxd.com/user/film/the-matrix",
			},
			wantMovie: Movie{
				Date:          "2024-01-15",
				Name:          "The Matrix",
				Year:          1999,
				LetterboxdURI: "https://letterboxd.com/user/film/the-matrix",
				LetterboxdID:  "the-matrix",
			},
			wantErr: false,
		},
		{
			name: "missing fields",
			record: []string{
				"2024-01-15",
				"The Matrix",
			},
			wantErr: true,
		},
		{
			name: "invalid year with skipInvalid",
			record: []string{
				"2024-01-15",
				"Test Movie",
				"invalid-year",
				"https://letterboxd.com/user/film/test-movie",
			},
			wantMovie: Movie{
				Date:          "2024-01-15",
				Name:          "Test Movie",
				Year:          0,
				LetterboxdURI: "https://letterboxd.com/user/film/test-movie",
				LetterboxdID:  "test-movie",
			},
			skipInvalid: true,
			wantErr:     false,
		},
		{
			name: "invalid year without skipInvalid",
			record: []string{
				"2024-01-15",
				"Test Movie",
				"invalid-year",
				"https://letterboxd.com/user/film/test-movie/",
			},
			skipInvalid: false,
			wantErr:     true,
		},
		{
			name: "empty letterboxd URI",
			record: []string{
				"2024-01-15",
				"Test Movie",
				"2020",
				"",
			},
			wantMovie: Movie{
				Date:          "2024-01-15",
				Name:          "Test Movie",
				Year:          2020,
				LetterboxdURI: "",
				LetterboxdID:  "",
			},
			wantErr: false,
		},
		{
			name: "URI with trailing slash",
			record: []string{
				"2024-01-15",
				"Test Movie",
				"2020",
				"https://letterboxd.com/user/film/test-movie/",
			},
			wantMovie: Movie{
				Date:          "2024-01-15",
				Name:          "Test Movie",
				Year:          2020,
				LetterboxdURI: "https://letterboxd.com/user/film/test-movie/",
				LetterboxdID:  "", // Empty because URI ends with slash
			},
			wantErr: false,
		},
		{
			name: "record with rating",
			record: []string{
				"2024-02-01",
				"Rated Movie",
				"2022",
				"https://letterboxd.com/user/film/rated-movie",
				"4.5",
			},
			wantMovie: Movie{
				Date:          "2024-02-01",
				Name:          "Rated Movie",
				Year:          2022,
				LetterboxdURI: "https://letterboxd.com/user/film/rated-movie",
				LetterboxdID:  "rated-movie",
				Rating:        4.5,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save and restore global state
			prevSkipInvalid := skipInvalid
			skipInvalid = tt.skipInvalid
			defer func() { skipInvalid = prevSkipInvalid }()

			movie, err := parseMovieRecord(tt.record)
			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tt.wantMovie.Date, movie.Date)
			require.Equal(t, tt.wantMovie.Name, movie.Name)
			require.Equal(t, tt.wantMovie.Year, movie.Year)
			require.Equal(t, tt.wantMovie.LetterboxdURI, movie.LetterboxdURI)
			require.Equal(t, tt.wantMovie.LetterboxdID, movie.LetterboxdID)
			require.Equal(t, tt.wantMovie.Rating, movie.Rating)
		})
	}
}

func TestMovieToMap(t *testing.T) {
	movie := Movie{
		Date:            "2024-01-15",
		Name:            "The Matrix",
		Year:            1999,
		LetterboxdID:    "the-matrix",
		LetterboxdURI:   "https://letterboxd.com/user/film/the-matrix/",
		ImdbID:          "tt0133093",
		Director:        "The Wachowskis",
		Cast:            []string{"Keanu Reeves", "Laurence Fishburne"},
		Genres:          []string{"Action", "Sci-Fi"},
		Runtime:         136,
		Rating:          4.5,
		CommunityRating: 8.7,
		PosterURL:       "https://example.com/poster.jpg",
		Description:     "A computer hacker learns from mysterious rebels about the true nature of his reality and his role in the war against its controllers.",
	}

	result := movieToMap(movie)

	require.Equal(t, "2024-01-15", result["date"])
	require.Equal(t, "The Matrix", result["name"])
	require.Equal(t, 1999, result["year"])
	require.Equal(t, "the-matrix", result["letterboxd_id"])
	require.Equal(t, "https://letterboxd.com/user/film/the-matrix/", result["letterboxd_uri"])
	require.Equal(t, "tt0133093", result["imdb_id"])
	require.Equal(t, "The Wachowskis", result["director"])
	require.Equal(t, "Keanu Reeves,Laurence Fishburne", result["cast"])
	require.Equal(t, "Action,Sci-Fi", result["genres"])
	require.Equal(t, 136, result["runtime"])
	require.Equal(t, 4.5, result["rating"])
	require.Equal(t, 8.7, result["community_rating"])
	require.Equal(t, "https://example.com/poster.jpg", result["poster_url"])
	require.Contains(t, result["description"], "computer hacker")
}

func TestWriteJSONFile(t *testing.T) {
	env := testutil.NewTestEnv(t)
	tempDir := env.RootDir()

	prevOverwrite := overwrite
	overwrite = true
	defer func() { overwrite = prevOverwrite }()

	movies := []Movie{
		{
			Name:          "The Matrix",
			Year:          1999,
			LetterboxdID:  "the-matrix",
			LetterboxdURI: "https://letterboxd.com/user/film/the-matrix/",
		},
		{
			Name:          "Inception",
			Year:          2010,
			LetterboxdID:  "inception",
			LetterboxdURI: "https://letterboxd.com/user/film/inception/",
		},
	}

	jsonPath := filepath.Join(tempDir, "movies.json")
	err := writeJSONFile(movies, jsonPath)
	require.NoError(t, err)

	// Verify file was created
	data, err := os.ReadFile(jsonPath)
	require.NoError(t, err)

	// Verify JSON is valid
	var saved []Movie
	err = json.Unmarshal(data, &saved)
	require.NoError(t, err)
	require.Len(t, saved, 2)
	require.Equal(t, "The Matrix", saved[0].Name)
	require.Equal(t, "Inception", saved[1].Name)
}

func TestProcessCSVFile(t *testing.T) {
	env := testutil.NewTestEnv(t)
	tempDir := env.RootDir()

	// Create test CSV file
	csvContent := `Date,Name,Year,Letterboxd URI
2024-01-15,The Matrix,1999,https://letterboxd.com/user/film/the-matrix/
2024-01-16,Inception,2010,https://letterboxd.com/user/film/inception/
`
	csvPath := filepath.Join(tempDir, "test.csv")
	err := os.WriteFile(csvPath, []byte(csvContent), 0644)
	require.NoError(t, err)

	movies, err := processCSVFile(csvPath)
	require.NoError(t, err)
	require.Len(t, movies, 2)
	require.Equal(t, "The Matrix", movies[0].Name)
	require.Equal(t, 1999, movies[0].Year)
	require.Equal(t, "Inception", movies[1].Name)
	require.Equal(t, 2010, movies[1].Year)
}

func TestProcessCSVFile_InvalidFile(t *testing.T) {
	_, err := processCSVFile("/nonexistent/file.csv")
	require.Error(t, err)
}

func TestProcessCSVFile_SkipInvalidRecords(t *testing.T) {
	env := testutil.NewTestEnv(t)
	tempDir := env.RootDir()

	// Create CSV with some invalid records (missing fields)
	csvContent := `Date,Name,Year,Letterboxd URI
2024-01-15,The Matrix,1999,https://letterboxd.com/user/film/the-matrix
2024-01-15,Incomplete
2024-01-16,Inception,2010,https://letterboxd.com/user/film/inception
`
	csvPath := filepath.Join(tempDir, "mixed.csv")
	err := os.WriteFile(csvPath, []byte(csvContent), 0644)
	require.NoError(t, err)

	// Test with skipInvalid = true (should skip bad records)
	prevSkipInvalid := skipInvalid
	skipInvalid = true
	defer func() { skipInvalid = prevSkipInvalid }()

	movies, err := processCSVFile(csvPath)
	require.NoError(t, err)
	require.Len(t, movies, 2, "should have 2 valid movies when skipping invalid records")
	require.Equal(t, "The Matrix", movies[0].Name)
	require.Equal(t, "Inception", movies[1].Name)
}

func TestParseLetterboxdCSV_GoldenFile(t *testing.T) {
	csvPath := filepath.Join("testdata", "letterboxd_sample.csv")

	// Verify the file exists
	_, err := os.Stat(csvPath)
	require.NoError(t, err, "golden file should exist")

	// Process the CSV file
	movies, err := processCSVFile(csvPath)
	require.NoError(t, err)

	// Verify we got all 20 movies
	require.Len(t, movies, 20, "should parse all 20 movies from golden file")

	// Verify first movie (Ocean's Eight)
	require.Equal(t, "2019-03-09", movies[0].Date)
	require.Equal(t, "Ocean's Eight", movies[0].Name)
	require.Equal(t, 2018, movies[0].Year)
	require.Equal(t, "https://boxd.it/eaai", movies[0].LetterboxdURI)
	require.Equal(t, "eaai", movies[0].LetterboxdID)

	// Verify second movie (Akira)
	require.Equal(t, "2019-03-09", movies[1].Date)
	require.Equal(t, "Akira", movies[1].Name)
	require.Equal(t, 1988, movies[1].Year)
	require.Equal(t, "https://boxd.it/2b1i", movies[1].LetterboxdURI)
	require.Equal(t, "2b1i", movies[1].LetterboxdID)

	// Verify another movie (They Shall Not Grow Old)
	require.Equal(t, "2019-04-06", movies[2].Date)
	require.Equal(t, "They Shall Not Grow Old", movies[2].Name)
	require.Equal(t, 2018, movies[2].Year)
	require.Equal(t, "https://boxd.it/jP62", movies[2].LetterboxdURI)
	require.Equal(t, "jP62", movies[2].LetterboxdID)

	// Verify a movie from the middle (Slumdog Millionaire)
	var slumdogMovie *Movie
	for i := range movies {
		if movies[i].Name == "Slumdog Millionaire" {
			slumdogMovie = &movies[i]
			break
		}
	}
	require.NotNil(t, slumdogMovie, "should find Slumdog Millionaire in the parsed movies")
	require.Equal(t, "Slumdog Millionaire", slumdogMovie.Name)
	require.Equal(t, 2008, slumdogMovie.Year)
	require.Equal(t, "2019-04-12", slumdogMovie.Date)
	require.Equal(t, "https://boxd.it/1S3E", slumdogMovie.LetterboxdURI)
	require.Equal(t, "1S3E", slumdogMovie.LetterboxdID)
}

// TestUserRatingPreservedInDatabase verifies that when importing from CSV,
// the user's personal Letterboxd rating (0.5-5 scale) is stored in the rating column
// and NOT overwritten by OMDB/TMDB community ratings.
func TestUserRatingPreservedInDatabase(t *testing.T) {
	env := testutil.NewTestEnv(t)
	tempDir := env.RootDir()

	// Create CSV with known ratings
	csvContent := `Date,Name,Year,Letterboxd URI,Rating
2024-01-15,The Matrix,1999,https://letterboxd.com/user/film/the-matrix,4.5
2024-01-16,Inception,2010,https://letterboxd.com/user/film/inception,5
2024-01-17,Unrated Movie,2020,https://letterboxd.com/user/film/unrated,
`
	csvPath := filepath.Join(tempDir, "test_ratings.csv")
	err := os.WriteFile(csvPath, []byte(csvContent), 0644)
	require.NoError(t, err)

	// Set up temp database
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
	prevSkipEnrich := skipEnrich
	prevOverwrite := overwrite
	csvFile = csvPath
	outputDir = tempDir
	skipEnrich = true // Skip enrichment to test rating preservation without OMDB
	overwrite = true
	defer func() {
		csvFile = prevCSVFile
		outputDir = prevOutputDir
		skipEnrich = prevSkipEnrich
		overwrite = prevOverwrite
	}()

	// Run the parser
	err = ParseLetterboxd()
	require.NoError(t, err)

	// Query the database directly to verify ratings
	db, err := sql.Open("sqlite", dbPath)
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	// Check The Matrix rating (should be 4.5)
	var matrixRating sql.NullFloat64
	err = db.QueryRow("SELECT rating FROM letterboxd_movies WHERE name = 'The Matrix'").Scan(&matrixRating)
	require.NoError(t, err)
	require.True(t, matrixRating.Valid, "Matrix rating should be present")
	require.Equal(t, 4.5, matrixRating.Float64, "Matrix user rating should be 4.5")

	// Check Inception rating (should be 5.0)
	var inceptionRating sql.NullFloat64
	err = db.QueryRow("SELECT rating FROM letterboxd_movies WHERE name = 'Inception'").Scan(&inceptionRating)
	require.NoError(t, err)
	require.True(t, inceptionRating.Valid, "Inception rating should be present")
	require.Equal(t, 5.0, inceptionRating.Float64, "Inception user rating should be 5.0")

	// Check unrated movie (should be 0 or null)
	var unratedRating sql.NullFloat64
	err = db.QueryRow("SELECT rating FROM letterboxd_movies WHERE name = 'Unrated Movie'").Scan(&unratedRating)
	require.NoError(t, err)
	require.Equal(t, 0.0, unratedRating.Float64, "Unrated movie should have 0 rating")

	// Verify community_rating column exists and is separate
	var matrixCommunityRating sql.NullFloat64
	err = db.QueryRow("SELECT community_rating FROM letterboxd_movies WHERE name = 'The Matrix'").Scan(&matrixCommunityRating)
	require.NoError(t, err)
	require.Equal(t, 0.0, matrixCommunityRating.Float64, "Community rating should be 0 when enrichment is skipped")
}
