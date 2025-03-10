package goodreads

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/lepinkainen/hermes/internal/config"
	"github.com/lepinkainen/hermes/internal/fileutil"
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
		{
			name: "complex_book",
			book: Book{
				ID:                      98765,
				Title:                   "The Complex Narrative: A Journey Through Time and Space",
				Authors:                 []string{"Jane Smith PhD", "Dr. Robert Johnson", "Prof. Emily Williams"},
				YearPublished:           2022,
				OriginalPublicationYear: 2021,
				MyRating:                5.0,
				AverageRating:           4.8,
				DateRead:                "2023-05-22",
				DateAdded:               "2023-01-30",
				NumberOfPages:           842,
				Publisher:               "Academic Press International",
				Binding:                 "Hardcover",
				ISBN:                    "9876543210",
				ISBN13:                  "9789876543210",
				Bookshelves:             []string{"science", "philosophy", "favorites", "to-reread", "best-of-2023"},
				ExclusiveShelf:          "read",
				MyReview:                "This masterpiece of interdisciplinary research blends quantum physics with philosophical inquiry in ways I've never encountered before.<br/>The middle section on temporal paradoxes was particularly enlightening, especially when paired with the appendix on mathematical proofs.<br/><br/>I found myself returning to the chapters on consciousness repeatedly, each time discovering new insights. Highly recommended for anyone interested in the intersection of science and philosophy.",
				PrivateNotes:            "Lent to Alex on June 15, 2023. Need to follow up.\nPotential thesis reference material - especially pages 341-362 on quantum entanglement theory.\nCheck the author's lecture series online for supplementary material.",
				Description:             "A groundbreaking interdisciplinary work that explores the connections between quantum mechanics, consciousness, and philosophical determinism. Drawing from cutting-edge research in physics, neuroscience, and metaphysics, the authors present a unified theory of reality that challenges conventional understanding of time, causality, and human perception. Includes extensive notes, mathematical appendices, and thought experiments designed to illuminate the practical applications of theoretical concepts.",
				Subjects:                []string{"Quantum Physics", "Philosophy of Science", "Consciousness Studies", "Metaphysics", "Theoretical Physics", "Neuroscience", "Determinism", "Free Will", "Time Perception"},
				CoverURL:                "https://example.com/complex_book_cover.jpg",
				Subtitle:                "Exploring the Intersection of Physics, Consciousness, and Metaphysical Determinism",
				SubjectPeople:           []string{"Albert Einstein", "Niels Bohr", "Werner Heisenberg", "Erwin Schrödinger", "David Bohm"},
				CoverID:                 0, // Using CoverURL instead
			},
			wantFile: "complex_book.md",
		},
		{
			name: "minimal_book",
			book: Book{
				ID:             333,
				Title:          "Minimal Book",
				Authors:        []string{"Minimalist Author"},
				YearPublished:  2018,
				ExclusiveShelf: "to-read",
				CoverID:        12345, // Using OpenLibrary cover ID instead of URL
			},
			wantFile: "minimal_book.md",
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
			generatedFilePath := filepath.Join(testDir, fileutil.SanitizeFilename(tc.book.Title)+".md")
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
