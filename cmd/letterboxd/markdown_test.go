package letterboxd

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/alecthomas/assert/v2"
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
		movie    Movie
		wantFile string
	}{
		{
			name: "basic_movie",
			movie: Movie{
				Name:          "Test Movie",
				Year:          2021,
				Date:          "2021-10-15",
				LetterboxdURI: "https://letterboxd.com/user/film/test-movie/",
				LetterboxdID:  "test-movie",
				Rating:        4.5,
				Runtime:       120,
				Director:      "Test Director",
				Genres:        []string{"Action", "Drama"},
				Cast:          []string{"Actor 1", "Actor 2"},
				PosterURL:     "https://example.com/poster.jpg",
				Description:   "This is a test movie description.",
				ImdbID:        "tt1234567",
			},
			wantFile: "basic_movie.md",
		},
		{
			name: "complex_movie",
			movie: Movie{
				Name:          "The Masterpiece of Cinema: A Director's Vision",
				Year:          2023,
				Date:          "2023-08-15",
				LetterboxdURI: "https://letterboxd.com/cinephile/film/the-masterpiece-of-cinema/",
				LetterboxdID:  "the-masterpiece-of-cinema",
				Rating:        5.0,
				Runtime:       187,
				Director:      "Visionary Auteur",
				Genres:        []string{"Drama", "Psychological Thriller", "Art House", "Experimental", "Historical Fiction"},
				Cast:          []string{"Award-winning Actor", "Breakthrough Performer", "Character Actor Veteran", "Method Acting Master", "Critically Acclaimed Actress"},
				PosterURL:     "https://example.com/masterpiece_poster.jpg",
				Description:   "A groundbreaking cinematic achievement that weaves together multiple timelines and perspectives to create a tapestry of human experience. Set against the backdrop of pivotal historical events, the film explores themes of memory, identity, and the nature of reality itself through innovative visual storytelling techniques and transcendent performances.",
				ImdbID:        "tt8765432",
			},
			wantFile: "complex_movie.md",
		},
		{
			name: "minimal_movie",
			movie: Movie{
				Name:          "Minimal Movie",
				Year:          2020,
				LetterboxdURI: "https://letterboxd.com/user/film/minimal-movie/",
				LetterboxdID:  "minimal-movie",
			},
			wantFile: "minimal_movie.md",
		},
	}

	// Run test cases
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Write movie to markdown in test directory
			err := writeMovieToMarkdown(tc.movie, env.RootDir())
			require.NoError(t, err)

			// Read the generated file
			expectedFilename := fmt.Sprintf("%s (%d).md", fileutil.SanitizeFilename(tc.movie.Name), tc.movie.Year)
			generatedFilePath := filepath.Join(env.RootDir(), expectedFilename)
			generated := env.ReadFile(generatedFilePath[len(env.RootDir())+1:])

			// Compare with golden file (handles UPDATE_GOLDEN automatically)
			golden.AssertGolden(tc.wantFile, generated)
		})
	}
}

func TestFormatDuration(t *testing.T) {
	testCases := []struct {
		minutes  int
		expected string
	}{
		{120, "2h 0m"},
		{90, "1h 30m"},
		{45, "0h 45m"},
		{0, "0h 0m"},
		{135, "2h 15m"},
		{180, "3h 0m"},
	}

	for _, tc := range testCases {
		t.Run(tc.expected, func(t *testing.T) {
			result := fileutil.FormatDuration(tc.minutes)
			assert.Equal(t, tc.expected, result)
		})
	}
}
