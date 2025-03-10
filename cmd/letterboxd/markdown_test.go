package letterboxd

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/lepinkainen/hermes/internal/config"
	"github.com/lepinkainen/hermes/internal/fileutil"
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
			wantFile: "Test Movie (2021).md",
		},
		{
			name: "minimal_movie",
			movie: Movie{
				Name:          "Minimal Movie",
				Year:          2020,
				LetterboxdURI: "https://letterboxd.com/user/film/minimal-movie/",
				LetterboxdID:  "minimal-movie",
			},
			wantFile: "Minimal Movie (2020).md",
		},
		{
			name: "movie_with_special_chars",
			movie: Movie{
				Name:          "Movie: With? Special* Chars!",
				Year:          2019,
				LetterboxdURI: "https://letterboxd.com/user/film/movie-with-special-chars/",
				LetterboxdID:  "movie-with-special-chars",
			},
			wantFile: "Movie - With? Special* Chars! (2019).md",
		},
	}

	// Run test cases
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Call the function under test
			err := writeMovieToMarkdown(tc.movie, testDir)
			require.NoError(t, err)

			// Check if the file was created
			filePath := filepath.Join(testDir, tc.wantFile)
			assert.FileExists(t, filePath)

			// Read the file contents
			content, err := os.ReadFile(filePath)
			require.NoError(t, err)

			// Basic validations of file content
			contentStr := string(content)

			// Check for required frontmatter elements
			assert.Contains(t, contentStr, "title: \""+fileutil.SanitizeFilename(tc.movie.Name)+"\"")
			assert.Contains(t, contentStr, "type: movie")
			assert.Contains(t, contentStr, "year: "+fmt.Sprintf("%d", tc.movie.Year))
			assert.Contains(t, contentStr, "letterboxd_uri: \""+tc.movie.LetterboxdURI+"\"")
			assert.Contains(t, contentStr, "letterboxd_id: \""+tc.movie.LetterboxdID+"\"")

			// Check for optional elements based on the test case
			if tc.movie.Rating > 0 {
				assert.Contains(t, contentStr, "letterboxd_rating: ")
			}

			if tc.movie.Runtime > 0 {
				assert.Contains(t, contentStr, "duration: ")
				assert.Contains(t, contentStr, "runtime_mins: ")
			}

			if tc.movie.Director != "" {
				assert.Contains(t, contentStr, "directors:")
				assert.Contains(t, contentStr, "  - \""+tc.movie.Director+"\"")
			}

			if len(tc.movie.Genres) > 0 {
				assert.Contains(t, contentStr, "genres:")
				for _, genre := range tc.movie.Genres {
					assert.Contains(t, contentStr, "  - \""+genre+"\"")
				}
			}

			if tc.movie.ImdbID != "" {
				assert.Contains(t, contentStr, "imdb_id: \""+tc.movie.ImdbID+"\"")
				assert.Contains(t, contentStr, "View on IMDb")
			}

			if tc.movie.PosterURL != "" {
				assert.Contains(t, contentStr, "cover: \""+tc.movie.PosterURL+"\"")
				assert.Contains(t, contentStr, "![]("+tc.movie.PosterURL+")")
			}

			if tc.movie.Description != "" {
				assert.Contains(t, contentStr, "> "+tc.movie.Description)
			}

			if len(tc.movie.Cast) > 0 {
				assert.Contains(t, contentStr, ">[!cast]- Cast")
				for _, actor := range tc.movie.Cast {
					assert.Contains(t, contentStr, "- "+actor)
				}
			}
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
