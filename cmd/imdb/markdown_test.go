package imdb

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/lepinkainen/hermes/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriteMovieToMarkdown(t *testing.T) {
	// Setup test directory
	testDir := t.TempDir()

	// Force overwrite for testing
	config.SetOverwriteFiles(true)

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
		// Add more test cases as needed
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create golden file path
			goldenFilePath := filepath.Join("testdata", tc.wantFile)

			// Write movie to markdown in test directory
			err := writeMovieToMarkdown(tc.movie, testDir)
			require.NoError(t, err)

			// Read the generated file
			generatedFilePath := filepath.Join(testDir, sanitizeTitle(tc.movie.Title)+".md")
			generated, err := os.ReadFile(generatedFilePath)
			require.NoError(t, err)

			// Check if we need to update golden files (useful during development)
			if os.Getenv("UPDATE_GOLDEN") == "true" {
				err = os.MkdirAll(filepath.Dir(goldenFilePath), 0755)
				require.NoError(t, err)
				err = os.WriteFile(goldenFilePath, generated, 0644)
				require.NoError(t, err)
			}

			// Read the golden file
			golden, err := os.ReadFile(goldenFilePath)
			require.NoError(t, err)

			// Compare generated content with golden file
			assert.Equal(t, string(golden), string(generated))
		})
	}
}
