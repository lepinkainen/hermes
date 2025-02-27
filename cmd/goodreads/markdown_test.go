package goodreads

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/lepinkainen/hermes/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriteBookToMarkdown(t *testing.T) {
	// Setup test directory
	testDir := t.TempDir()

	// Force overwrite for testing
	config.SetOverwriteFiles(true)

	// Create test cases
	testCases := []struct {
		name     string
		book     Book
		wantFile string
	}{
		{
			name: "basic_book",
			book: Book{
				ID:                      12345,
				Title:                   "Test Book",
				Authors:                 []string{"Author One", "Author Two"},
				YearPublished:           2020,
				OriginalPublicationYear: 2019,
				MyRating:                4.5,
				AverageRating:           4.2,
				DateRead:                "2023-01-15",
				DateAdded:               "2022-12-01",
				NumberOfPages:           300,
				Publisher:               "Test Publisher",
				Binding:                 "Paperback",
				ISBN:                    "1234567890",
				ISBN13:                  "1234567890123",
				Bookshelves:             []string{"fiction", "favorites"},
				ExclusiveShelf:          "read",
				MyReview:                "This is my review of the book.",
				PrivateNotes:            "These are my private notes.",
				Description:             "This is a description of the book.",
				Subjects:                []string{"Fiction", "Adventure"},
				CoverURL:                "https://example.com/cover.jpg",
				Subtitle:                "A Test Subtitle",
				SubjectPeople:           []string{"Character One", "Character Two"},
			},
			wantFile: "basic_book.md",
		},
		// Add more test cases as needed
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create golden file path
			goldenFilePath := filepath.Join("testdata", tc.wantFile)

			// Write book to markdown in test directory
			err := writeBookToMarkdown(tc.book, testDir)
			require.NoError(t, err)

			// Read the generated file
			generatedFilePath := filepath.Join(testDir, sanitizeGoodreadsTitle(tc.book.Title)+".md")
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
