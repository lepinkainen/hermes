package imdb

import (
	"path/filepath"
	"testing"

	"github.com/lepinkainen/hermes/internal/fileutil"
	"github.com/lepinkainen/hermes/internal/testutil"
	"github.com/stretchr/testify/require"
)

func TestWriteMovieToMarkdown(t *testing.T) {
	// Setup test environment with automatic config management
	env := testutil.NewTestEnv(t)
	testutil.SetTestConfig(t)
	golden := testutil.NewGoldenHelper(t, "testdata")

	// Create test cases
	testCases := []struct {
		name     string
		movie    MovieSeen
		wantFile string
	}{
		{
			name: "basic_movie",
			movie: MovieSeen{
				ImdbId:        "tt1234567",
				Title:         "Test Movie",
				OriginalTitle: "Original Test Movie",
				TitleType:     "Movie",
				Year:          2020,
				IMDbRating:    8.5,
				MyRating:      9,
				DateRated:     "2023-01-15",
				RuntimeMins:   120,
				Genres:        []string{"Action", "Adventure"},
				Directors:     []string{"Director One", "Director Two"},
				Plot:          "This is a test movie plot.",
				PosterURL:     "https://example.com/poster.jpg",
				URL:           "https://www.imdb.com/title/tt1234567/",
				ContentRated:  "PG-13",
				Awards:        "Won 2 Oscars.",
			},
			wantFile: "basic_movie.md",
		},
		{
			name: "complex_movie",
			movie: MovieSeen{
				ImdbId:        "tt9876543",
				Title:         "The Cinematic Masterpiece: Director's Extended Cut",
				OriginalTitle: "La Chef-d'œuvre Cinématographique",
				TitleType:     "Movie",
				Year:          2022,
				IMDbRating:    9.8,
				MyRating:      10,
				DateRated:     "2023-06-15",
				RuntimeMins:   195,
				Genres:        []string{"Drama", "Science Fiction", "Thriller", "Mystery", "Psychological"},
				Directors:     []string{"Renowned Director", "Co-Director Vision"},
				Plot:          "In a dystopian future where memories can be transferred between individuals, a memory detective becomes entangled in a complex web of deception after encountering a rare memory sequence that contains the key to unlocking humanity's greatest mystery. As political forces and corporate interests race to obtain this knowledge, the detective must question their own reality while navigating a landscape of shifting alliances and manufactured truths. The journey leads to a profound revelation about consciousness and the nature of human experience that challenges everything society has built itself upon.",
				PosterURL:     "https://example.com/masterpiece_poster.jpg",
				URL:           "https://www.imdb.com/title/tt9876543/",
				ContentRated:  "R",
				Awards:        "Won 7 Oscars including Best Picture, Best Director, and Best Original Screenplay. Nominated for 5 additional categories. 125 wins & 86 nominations total at various international film festivals and awards ceremonies.",
			},
			wantFile: "complex_movie.md",
		},
		{
			name: "tv_series",
			movie: MovieSeen{
				ImdbId:       "tt5555555",
				Title:        "Epic Fantasy Series",
				TitleType:    "TV Series",
				Year:         2021,
				IMDbRating:   9.2,
				MyRating:     8,
				DateRated:    "2022-12-10",
				RuntimeMins:  55,
				Genres:       []string{"Fantasy", "Adventure", "Drama"},
				Directors:    []string{"Show Runner"},
				Plot:         "A television adaptation of the popular fantasy book series.",
				PosterURL:    "https://example.com/series_poster.jpg",
				URL:          "https://www.imdb.com/title/tt5555555/",
				ContentRated: "TV-MA",
			},
			wantFile: "tv_series.md",
		},
		// Add more test cases as needed
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Write movie to markdown in test directory
			err := writeMovieToMarkdown(tc.movie, env.RootDir())
			require.NoError(t, err)

			// Read the generated file
			generatedFilePath := filepath.Join(env.RootDir(), fileutil.SanitizeFilename(tc.movie.Title)+".md")
			generated := env.ReadFile(generatedFilePath[len(env.RootDir())+1:])

			// Compare with golden file (handles UPDATE_GOLDEN automatically)
			golden.AssertGolden(tc.wantFile, generated)
		})
	}
}
