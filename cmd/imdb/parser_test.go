package imdb

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/lepinkainen/hermes/internal/enrichment"
	"github.com/lepinkainen/hermes/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseMovieRecord(t *testing.T) {
	tests := []struct {
		name      string
		record    []string
		wantMovie MovieSeen
		wantErr   bool
	}{
		{
			name: "valid complete record",
			record: []string{
				"tt0133093",                             // IMDb ID
				"5",                                     // Your Rating
				"2024-01-15",                            // Date Rated
				"The Matrix",                            // Title
				"The Matrix",                            // Original Title
				"https://www.imdb.com/title/tt0133093/", // URL
				"movie",                                 // Title Type
				"8.7",                                   // IMDb Rating
				"136",                                   // Runtime
				"1999",                                  // Year
				"Action, Sci-Fi",                        // Genres
				"2000000",                               // Num Votes
				"1999-03-31",                            // Release Date
				"Lana Wachowski,Lilly Wachowski",        // Directors
			},
			wantMovie: MovieSeen{
				ImdbId:        "tt0133093",
				MyRating:      5,
				DateRated:     "2024-01-15",
				Title:         "The Matrix",
				OriginalTitle: "The Matrix",
				URL:           "https://www.imdb.com/title/tt0133093/",
				TitleType:     "movie",
				IMDbRating:    8.7,
				RuntimeMins:   136,
				Year:          1999,
				Genres:        []string{"Action", "Sci-Fi"},
				NumVotes:      2000000,
				ReleaseDate:   "1999-03-31",
				Directors:     []string{"Lana Wachowski", "Lilly Wachowski"},
			},
			wantErr: false,
		},
		{
			name: "minimal record with empty optional fields",
			record: []string{
				"tt0111161",                             // IMDb ID
				"5",                                     // Your Rating
				"2024-01-15",                            // Date Rated
				"The Shawshank Redemption",              // Title
				"The Shawshank Redemption",              // Original Title
				"https://www.imdb.com/title/tt0111161/", // URL
				"movie",                                 // Title Type
				"",                                      // IMDb Rating (empty)
				"",                                      // Runtime (empty)
				"",                                      // Year (empty)
				"",                                      // Genres (empty)
				"",                                      // Num Votes (empty)
				"",                                      // Release Date (empty)
				"",                                      // Directors (empty)
			},
			wantMovie: MovieSeen{
				ImdbId:        "tt0111161",
				MyRating:      5,
				DateRated:     "2024-01-15",
				Title:         "The Shawshank Redemption",
				OriginalTitle: "The Shawshank Redemption",
				URL:           "https://www.imdb.com/title/tt0111161/",
				TitleType:     "movie",
				IMDbRating:    0,
				RuntimeMins:   0,
				Year:          0,
				Genres:        nil,
				NumVotes:      0,
				ReleaseDate:   "",
				Directors:     nil,
			},
			wantErr: false,
		},
		{
			name: "null IMDb rating",
			record: []string{
				"tt1234567",
				"4",
				"2024-01-15",
				"Test Movie",
				"Test Movie",
				"https://www.imdb.com/title/tt1234567/",
				"movie",
				"null", // null rating
				"120",
				"2020",
				"Drama",
				"1000",
				"2020-01-01",
				"Test Director",
			},
			wantMovie: MovieSeen{
				ImdbId:        "tt1234567",
				MyRating:      4,
				DateRated:     "2024-01-15",
				Title:         "Test Movie",
				OriginalTitle: "Test Movie",
				URL:           "https://www.imdb.com/title/tt1234567/",
				TitleType:     "movie",
				IMDbRating:    0, // null becomes 0
				RuntimeMins:   120,
				Year:          2020,
				Genres:        []string{"Drama"},
				NumVotes:      1000,
				ReleaseDate:   "2020-01-01",
				Directors:     []string{"Test Director"},
			},
			wantErr: false,
		},
		{
			name: "invalid rating",
			record: []string{
				"tt0133093",
				"invalid", // Invalid rating
				"2024-01-15",
				"The Matrix",
				"The Matrix",
				"https://www.imdb.com/title/tt0133093/",
				"movie",
				"8.7",
				"136",
				"1999",
				"Action, Sci-Fi",
				"2000000",
				"1999-03-31",
				"Lana Wachowski,Lilly Wachowski",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			movie, err := parseMovieRecord(tt.record)
			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantMovie.ImdbId, movie.ImdbId)
			assert.Equal(t, tt.wantMovie.MyRating, movie.MyRating)
			assert.Equal(t, tt.wantMovie.DateRated, movie.DateRated)
			assert.Equal(t, tt.wantMovie.Title, movie.Title)
			assert.Equal(t, tt.wantMovie.OriginalTitle, movie.OriginalTitle)
			assert.Equal(t, tt.wantMovie.URL, movie.URL)
			assert.Equal(t, tt.wantMovie.TitleType, movie.TitleType)
			assert.Equal(t, tt.wantMovie.IMDbRating, movie.IMDbRating)
			assert.Equal(t, tt.wantMovie.RuntimeMins, movie.RuntimeMins)
			assert.Equal(t, tt.wantMovie.Year, movie.Year)
			assert.Equal(t, tt.wantMovie.Genres, movie.Genres)
			assert.Equal(t, tt.wantMovie.NumVotes, movie.NumVotes)
			assert.Equal(t, tt.wantMovie.ReleaseDate, movie.ReleaseDate)
			assert.Equal(t, tt.wantMovie.Directors, movie.Directors)
		})
	}
}

func TestMovieToMap(t *testing.T) {
	movie := MovieSeen{
		Position:      1,
		ImdbId:        "tt0133093",
		MyRating:      5,
		DateRated:     "2024-01-15",
		Created:       "2024-01-01",
		Modified:      "2024-01-15",
		Description:   "A computer hacker learns from mysterious rebels",
		Title:         "The Matrix",
		OriginalTitle: "The Matrix",
		URL:           "https://www.imdb.com/title/tt0133093/",
		TitleType:     "movie",
		IMDbRating:    8.7,
		RuntimeMins:   136,
		Year:          1999,
		Genres:        []string{"Action", "Sci-Fi"},
		NumVotes:      2000000,
		ReleaseDate:   "1999-03-31",
		Directors:     []string{"Lana Wachowski", "Lilly Wachowski"},
		Plot:          "A computer hacker learns about the true nature of reality",
		ContentRated:  "R",
		Awards:        "Won 4 Oscars",
		PosterURL:     "https://example.com/poster.jpg",
	}

	result := movieToMap(movie)

	assert.Equal(t, 1, result["position"])
	assert.Equal(t, "tt0133093", result["imdb_id"])
	assert.Equal(t, 5, result["my_rating"])
	assert.Equal(t, "2024-01-15", result["date_rated"])
	assert.Equal(t, "2024-01-01", result["created"])
	assert.Equal(t, "2024-01-15", result["modified"])
	assert.Equal(t, "A computer hacker learns from mysterious rebels", result["description"])
	assert.Equal(t, "The Matrix", result["title"])
	assert.Equal(t, "The Matrix", result["original_title"])
	assert.Equal(t, "https://www.imdb.com/title/tt0133093/", result["url"])
	assert.Equal(t, "movie", result["title_type"])
	assert.Equal(t, 8.7, result["imdb_rating"])
	assert.Equal(t, 136, result["runtime_mins"])
	assert.Equal(t, 1999, result["year"])
	assert.Equal(t, "Action,Sci-Fi", result["genres"])
	assert.Equal(t, 2000000, result["num_votes"])
	assert.Equal(t, "1999-03-31", result["release_date"])
	assert.Equal(t, "Lana Wachowski,Lilly Wachowski", result["directors"])
	assert.Equal(t, "A computer hacker learns about the true nature of reality", result["plot"])
	assert.Equal(t, "R", result["content_rated"])
	assert.Equal(t, "Won 4 Oscars", result["awards"])
	assert.Equal(t, "https://example.com/poster.jpg", result["poster_url"])
}

func TestProcessCSVFile(t *testing.T) {
	env := testutil.NewTestEnv(t)
	tempDir := env.RootDir()

	// Create test CSV file (IMDb exports are comma-separated)
	csvContent := `Const,Your Rating,Date Rated,Title,Original Title,URL,Title Type,IMDb Rating,Runtime (mins),Year,Genres,Num Votes,Release Date,Directors
tt0133093,5,2024-01-15,"The Matrix","The Matrix",https://www.imdb.com/title/tt0133093/,Movie,8.7,136,1999,"Action, Sci-Fi",2000000,1999-03-31,"Lana Wachowski,Lilly Wachowski"
tt0111161,5,2024-01-16,"The Shawshank Redemption","The Shawshank Redemption",https://www.imdb.com/title/tt0111161/,Movie,9.3,142,1994,Drama,2800000,1994-09-23,"Frank Darabont"
`

	csvPath := filepath.Join(tempDir, "ratings.csv")
	err := os.WriteFile(csvPath, []byte(csvContent), 0644)
	require.NoError(t, err)

	movies, err := processCSVFile(csvPath)
	require.NoError(t, err)
	require.Len(t, movies, 2)
	assert.Equal(t, "The Matrix", movies[0].Title)
	assert.Equal(t, 1999, movies[0].Year)
	assert.Equal(t, "The Shawshank Redemption", movies[1].Title)
	assert.Equal(t, 1994, movies[1].Year)
}

func TestProcessCSVFile_InvalidFile(t *testing.T) {
	_, err := processCSVFile("/nonexistent/file.tsv")
	require.Error(t, err)
}

func TestLoadExistingMediaIDs(t *testing.T) {
	env := testutil.NewTestEnv(t)
	tempDir := env.RootDir()
	filePath := filepath.Join(tempDir, "The Matrix.md")

	content := `---
title: "The Matrix"
tmdb_id: 603
tmdb_type: movie
imdb_id: tt0133093
---

Movie body
`
	err := os.WriteFile(filePath, []byte(content), 0644)
	require.NoError(t, err)

	movie := &MovieSeen{
		Title: "The Matrix",
	}

	loadExistingMediaIDs(movie, tempDir)

	assert.Equal(t, "tt0133093", movie.ImdbId, "should load IMDb ID from existing file")
	require.NotNil(t, movie.TMDBEnrichment, "should initialize TMDB enrichment from file")
	assert.Equal(t, 603, movie.TMDBEnrichment.TMDBID)
	assert.Equal(t, "movie", movie.TMDBEnrichment.TMDBType)
}

func TestLoadExistingMediaIDs_NoFile(t *testing.T) {
	env := testutil.NewTestEnv(t)
	tempDir := env.RootDir()

	movie := &MovieSeen{
		Title: "Nonexistent Movie",
	}

	// Should not panic or error when file doesn't exist
	loadExistingMediaIDs(movie, tempDir)

	assert.Equal(t, "", movie.ImdbId)
	assert.Nil(t, movie.TMDBEnrichment)
}

func TestLoadExistingMediaIDs_PreservesExistingID(t *testing.T) {
	env := testutil.NewTestEnv(t)
	tempDir := env.RootDir()
	filePath := filepath.Join(tempDir, "The Matrix.md")

	content := `---
title: "The Matrix"
imdb_id: tt9999999
tmdb_id: 603
tmdb_type: movie
---

Movie body
`
	err := os.WriteFile(filePath, []byte(content), 0644)
	require.NoError(t, err)

	movie := &MovieSeen{
		Title:  "The Matrix",
		ImdbId: "tt0133093", // Already has an ID
	}

	loadExistingMediaIDs(movie, tempDir)

	// Should preserve existing IMDb ID instead of overwriting
	assert.Equal(t, "tt0133093", movie.ImdbId, "should not overwrite existing IMDb ID")
	require.NotNil(t, movie.TMDBEnrichment)
	assert.Equal(t, 603, movie.TMDBEnrichment.TMDBID)
}

func TestLoadExistingMediaIDs_NilMovie(t *testing.T) {
	env := testutil.NewTestEnv(t)
	tempDir := env.RootDir()

	// Should not panic when movie is nil
	require.NotPanics(t, func() {
		loadExistingMediaIDs(nil, tempDir)
	})
}

func TestLoadExistingMediaIDs_InitializesTMDBEnrichment(t *testing.T) {
	env := testutil.NewTestEnv(t)
	tempDir := env.RootDir()
	filePath := filepath.Join(tempDir, "Inception.md")

	content := `---
title: "Inception"
tmdb_id: 27205
tmdb_type: movie
---

Movie body
`
	err := os.WriteFile(filePath, []byte(content), 0644)
	require.NoError(t, err)

	movie := &MovieSeen{
		Title: "Inception",
		TMDBEnrichment: &enrichment.TMDBEnrichment{
			TMDBID:   12345, // Existing ID that should be overwritten
			TMDBType: "tv",  // Existing type that should be overwritten
		},
	}

	loadExistingMediaIDs(movie, tempDir)

	require.NotNil(t, movie.TMDBEnrichment)
	assert.Equal(t, 27205, movie.TMDBEnrichment.TMDBID, "should overwrite existing TMDB ID")
	assert.Equal(t, "movie", movie.TMDBEnrichment.TMDBType, "should overwrite existing TMDB type")
}

func TestParseIMDbCSV_GoldenFile(t *testing.T) {
	csvPath := filepath.Join("testdata", "imdb_sample.csv")

	// Verify the file exists
	_, err := os.Stat(csvPath)
	require.NoError(t, err, "golden file should exist")

	// Process the CSV file
	movies, err := processCSVFile(csvPath)
	require.NoError(t, err)

	// Verify we got all 20 movies
	require.Len(t, movies, 20, "should parse all 20 movies from golden file")

	// Verify first movie (S.W.A.T.)
	require.Equal(t, "tt0257076", movies[0].ImdbId)
	require.Equal(t, "S.W.A.T.", movies[0].Title)
	require.Equal(t, "S.W.A.T.", movies[0].OriginalTitle)
	require.Equal(t, 6, movies[0].MyRating)
	require.Equal(t, "2025-11-17", movies[0].DateRated)
	require.Equal(t, "Movie", movies[0].TitleType)
	require.Equal(t, 6.1, movies[0].IMDbRating)
	require.Equal(t, 117, movies[0].RuntimeMins)
	require.Equal(t, 2003, movies[0].Year)
	require.Equal(t, []string{"Crime", "Thriller", "Adventure", "Action"}, movies[0].Genres)
	require.Equal(t, []string{"Clark Johnson"}, movies[0].Directors)

	// Verify second movie (Dark Blue)
	require.Equal(t, "tt0279331", movies[1].ImdbId)
	require.Equal(t, "Dark Blue", movies[1].Title)
	require.Equal(t, 7, movies[1].MyRating)
	require.Equal(t, 2002, movies[1].Year)

	// Verify a movie with higher rating (Frankenstein)
	var frankensteinMovie *MovieSeen
	for i := range movies {
		if movies[i].ImdbId == "tt1312221" {
			frankensteinMovie = &movies[i]
			break
		}
	}
	require.NotNil(t, frankensteinMovie, "should find Frankenstein in the parsed movies")
	require.Equal(t, "Frankenstein", frankensteinMovie.Title)
	require.Equal(t, 9, frankensteinMovie.MyRating)
	require.Equal(t, 2025, frankensteinMovie.Year)
	require.Equal(t, []string{"Horror", "Sci-Fi", "Drama", "Fantasy"}, frankensteinMovie.Genres)
	require.Equal(t, []string{"Guillermo del Toro"}, frankensteinMovie.Directors)

	// Verify a TV Movie (Return to Christmas Creek)
	var christmasMovie *MovieSeen
	for i := range movies {
		if movies[i].ImdbId == "tt9103028" {
			christmasMovie = &movies[i]
			break
		}
	}
	require.NotNil(t, christmasMovie, "should find Return to Christmas Creek in the parsed movies")
	require.Equal(t, "Return to Christmas Creek", christmasMovie.Title)
	require.Equal(t, "TV Movie", christmasMovie.TitleType)
	require.Equal(t, 6, christmasMovie.MyRating)
}
