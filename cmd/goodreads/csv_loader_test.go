package goodreads

import (
	"encoding/csv"
	"os"
	"path/filepath"
	"testing"

	"github.com/lepinkainen/hermes/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseBookRecord(t *testing.T) {
	tests := []struct {
		name     string
		record   []string
		wantBook *Book
		wantErr  bool
	}{
		{
			name: "valid complete record",
			record: []string{
				"12345",                               // 0: Book Id
				"The Hobbit",                          // 1: Title
				"J.R.R. Tolkien, Christopher Tolkien", // 2: Author (comma-separated)
				"",                                    // 3: Additional Authors (unused)
				"",                                    // 4: Unused
				"9780547928227",                       // 5: ISBN
				"=\"9780547928227\"",                  // 6: ISBN13
				"5",                                   // 7: My Rating
				"4.28",                                // 8: Average Rating
				"Harper Collins",                      // 9: Publisher
				"Binding",                             // 10: Binding
				"310",                                 // 11: Number of Pages
				"1937",                                // 12: Year Published
				"1937",                                // 13: Original Publication Year
				"2024-01-15",                          // 14: Date Read
				"2024-01-10",                          // 15: Date Added
				"fantasy",                             // 16: Bookshelves
				"fantasy-shelf",                       // 17: Bookshelves with positions
				"Exclusive Shelf",                     // 18: Exclusive Shelf
				"Great book!",                         // 19: My Review
				"Bilbo Baggins",                       // 20: Spoiler
				"Read",                                // 21: Private Notes
				"100000",                              // 22: Read Count
				"2",                                   // 23: Owned Copies
			},
			wantBook: &Book{
				ID:                       12345,
				Title:                    "The Hobbit",
				Authors:                  []string{"J.R.R. Tolkien", "Christopher Tolkien"},
				ISBN:                     "9780547928227",
				ISBN13:                   "9780547928227",
				MyRating:                 5,
				AverageRating:            4.28,
				Publisher:                "Harper Collins",
				Binding:                  "Binding",
				NumberOfPages:            310,
				YearPublished:            1937,
				OriginalPublicationYear:  1937,
				DateRead:                 "2024-01-15",
				DateAdded:                "2024-01-10",
				Bookshelves:              []string{"fantasy"},
				BookshelvesWithPositions: []string{"fantasy-shelf"},
				ExclusiveShelf:           "Exclusive Shelf",
				MyReview:                 "Great book!",
				Spoiler:                  "Bilbo Baggins",
				PrivateNotes:             "Read",
				ReadCount:                100000,
				OwnedCopies:              2,
			},
			wantErr: false,
		},
		{
			name: "minimal record with empty optional fields",
			record: []string{
				"1",           // Book Id
				"Test Book",   // Title
				"Test Author", // Author
				"",            // Unused
				"",            // Additional Authors
				"",            // ISBN
				"",            // ISBN13
				"0",           // My Rating
				"",            // Average Rating
				"",            // Publisher
				"",            // Binding
				"",            // Number of Pages
				"",            // Year Published
				"",            // Original Publication Year
				"",            // Date Read
				"2024-01-01",  // Date Added
				"",            // Bookshelves
				"",            // Bookshelves with positions
				"to-read",     // Exclusive Shelf
				"",            // My Review
				"",            // Spoiler
				"",            // Private Notes
				"",            // Read Count
				"",            // Owned Copies
			},
			wantBook: &Book{
				ID:             1,
				Title:          "Test Book",
				Authors:        []string{"Test Author"},
				DateAdded:      "2024-01-01",
				ExclusiveShelf: "to-read",
			},
			wantErr: false,
		},
		{
			name: "ISBN with formula prefix",
			record: []string{
				"1", "Book", "Author", "", "", "=\"1234567890\"", "=\"9781234567890\"",
				"0", "", "", "", "", "", "", "", "2024-01-01",
				"", "", "read", "", "", "", "", "",
			},
			wantBook: &Book{
				ID:             1,
				Title:          "Book",
				Authors:        []string{"Author"},
				ISBN:           "1234567890",
				ISBN13:         "9781234567890",
				DateAdded:      "2024-01-01",
				ExclusiveShelf: "read",
			},
			wantErr: false,
		},
		{
			name: "invalid book ID",
			record: []string{
				"invalid", // Invalid Book Id
				"Test Book",
				"Test Author",
			},
			wantErr: true,
		},
		{
			name: "too few fields",
			record: []string{
				"1",
				"Test Book",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			book, err := parseBookRecord(tt.record)
			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, book)
			assert.Equal(t, tt.wantBook.ID, book.ID)
			assert.Equal(t, tt.wantBook.Title, book.Title)
			assert.Equal(t, tt.wantBook.Authors, book.Authors)
			assert.Equal(t, tt.wantBook.ISBN, book.ISBN)
			assert.Equal(t, tt.wantBook.ISBN13, book.ISBN13)
			assert.Equal(t, tt.wantBook.MyRating, book.MyRating)
			assert.Equal(t, tt.wantBook.AverageRating, book.AverageRating)
			assert.Equal(t, tt.wantBook.NumberOfPages, book.NumberOfPages)
			assert.Equal(t, tt.wantBook.YearPublished, book.YearPublished)
			assert.Equal(t, tt.wantBook.ExclusiveShelf, book.ExclusiveShelf)
		})
	}
}

func TestParseIntField(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int
	}{
		{"valid integer", "123", 123},
		{"zero", "0", 0},
		{"negative", "-5", -5},
		{"empty string", "", 0},
		{"invalid string", "abc", 0},
		{"decimal", "12.5", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseIntField(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseFloatField(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected float64
	}{
		{"valid float", "4.28", 4.28},
		{"integer", "5", 5.0},
		{"zero", "0", 0.0},
		{"negative", "-3.5", -3.5},
		{"empty string", "", 0.0},
		{"invalid string", "abc", 0.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseFloatField(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSanitizeISBNValue(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"normal ISBN", "9780547928227", "9780547928227"},
		{"with formula prefix", "=\"9780547928227\"", "9780547928227"},
		{"with formula prefix and quotes", "=\"1234567890\"", "1234567890"},
		{"empty string", "", ""},
		{"only formula prefix", "=\"\"", ""},
		{"no quotes", "1234567890", "1234567890"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeISBNValue(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCountBooksInCSV(t *testing.T) {
	env := testutil.NewTestEnv(t)
	tempDir := env.RootDir()

	// Create test CSV file
	csvContent := `Book Id,Title,Author,Additional Authors,ISBN,ISBN13,My Rating,Average Rating,Publisher,Binding,Number of Pages,Year Published,Original Publication Year,Date Read,Date Added,Bookshelves,Bookshelves with positions,Exclusive Shelf,My Review,Spoiler,Private Notes,Read Count,Owned Copies
1,Book 1,Author 1,,,123456,0,4.5,,,200,2020,2020,,2024-01-01,,,read,,,,0,0
2,Book 2,Author 2,,,234567,5,4.0,,,300,2021,2021,2024-01-15,2024-01-01,,,read,,,,1,1
3,Book 3,Author 3,,,345678,0,3.5,,,150,2019,2019,,2024-01-01,,,to-read,,,,0,0
`

	csvPath := filepath.Join(tempDir, "books.csv")
	err := os.WriteFile(csvPath, []byte(csvContent), 0644)
	require.NoError(t, err)

	count, err := countBooksInCSV(csvPath)
	require.NoError(t, err)
	assert.Equal(t, 3, count, "should count 3 books in CSV")
}

func TestCountBooksInCSV_InvalidFile(t *testing.T) {
	_, err := countBooksInCSV("/nonexistent/file.csv")
	require.Error(t, err)
}

func TestCountBooksInCSV_EmptyFile(t *testing.T) {
	env := testutil.NewTestEnv(t)
	tempDir := env.RootDir()

	// Create empty CSV file (only header)
	csvContent := `Book Id,Title,Author,Additional Authors,ISBN,ISBN13,My Rating,Average Rating,Publisher,Binding,Number of Pages,Year Published,Original Publication Year,Date Read,Date Added,Bookshelves,Bookshelves with positions,Exclusive Shelf,My Review,Spoiler,Private Notes,Read Count,Owned Copies
`

	csvPath := filepath.Join(tempDir, "empty.csv")
	err := os.WriteFile(csvPath, []byte(csvContent), 0644)
	require.NoError(t, err)

	count, err := countBooksInCSV(csvPath)
	require.NoError(t, err)
	assert.Equal(t, 0, count, "should count 0 books in empty CSV")
}

func TestBookToMap(t *testing.T) {
	book := Book{
		ID:                       12345,
		Title:                    "The Hobbit",
		Authors:                  []string{"J.R.R. Tolkien", "Christopher Tolkien"},
		ISBN:                     "9780547928227",
		ISBN13:                   "9780547928227",
		MyRating:                 5.0,
		AverageRating:            4.28,
		Publisher:                "Harper Collins",
		Binding:                  "Paperback",
		NumberOfPages:            310,
		YearPublished:            1937,
		OriginalPublicationYear:  1937,
		DateRead:                 "2024-01-15",
		DateAdded:                "2024-01-10",
		Bookshelves:              []string{"Fantasy", "Adventure"},
		BookshelvesWithPositions: []string{"fantasy-shelf", "adventure-shelf"},
		ExclusiveShelf:           "read",
		MyReview:                 "Excellent adventure!",
		Spoiler:                  "Dragons!",
		PrivateNotes:             "First read",
		ReadCount:                1,
		OwnedCopies:              2,
	}

	result := bookToMap(book)

	assert.Equal(t, 12345, result["id"])
	assert.Equal(t, "The Hobbit", result["title"])
	assert.Equal(t, "J.R.R. Tolkien,Christopher Tolkien", result["authors"])
	assert.Equal(t, "9780547928227", result["isbn"])
	assert.Equal(t, "9780547928227", result["isbn13"])
	assert.Equal(t, 5.0, result["my_rating"])
	assert.Equal(t, 4.28, result["average_rating"])
	assert.Equal(t, "Harper Collins", result["publisher"])
	assert.Equal(t, "Paperback", result["binding"])
	assert.Equal(t, 310, result["number_of_pages"])
	assert.Equal(t, 1937, result["year_published"])
	assert.Equal(t, 1937, result["original_publication_year"])
	assert.Equal(t, "2024-01-15", result["date_read"])
	assert.Equal(t, "2024-01-10", result["date_added"])
	assert.Equal(t, "Fantasy,Adventure", result["bookshelves"])
	assert.Equal(t, "fantasy-shelf,adventure-shelf", result["bookshelves_with_positions"])
	assert.Equal(t, "read", result["exclusive_shelf"])
	assert.Equal(t, "Excellent adventure!", result["my_review"])
	assert.Equal(t, "Dragons!", result["spoiler"])
	assert.Equal(t, "First read", result["private_notes"])
	assert.Equal(t, 1, result["read_count"])
	assert.Equal(t, 2, result["owned_copies"])
}

func TestParseGoodreadsCSV_GoldenFile(t *testing.T) {
	csvPath := filepath.Join("testdata", "goodreads_sample.csv")

	// Verify the file exists
	_, err := os.Stat(csvPath)
	require.NoError(t, err, "golden file should exist")

	// Count books in the CSV
	count, err := countBooksInCSV(csvPath)
	require.NoError(t, err)
	assert.Equal(t, 20, count, "golden file should have exactly 20 books")

	// Open and parse the CSV
	file, err := os.Open(csvPath)
	require.NoError(t, err)
	defer func() { _ = file.Close() }()

	reader := csv.NewReader(file)

	// Skip header
	_, err = reader.Read()
	require.NoError(t, err)

	// Parse all records
	books := []Book{}
	for {
		record, err := reader.Read()
		if err != nil {
			break
		}

		book, err := parseBookRecord(record)
		require.NoError(t, err, "should successfully parse book record")
		require.NotNil(t, book)
		books = append(books, *book)
	}

	// Verify we got all 20 books
	assert.Len(t, books, 20, "should parse all 20 books from golden file")

	// Verify first book (Salvager)
	assert.Equal(t, 218224452, books[0].ID)
	assert.Equal(t, "Salvager: A Military Science Fiction Adventure (Tall Boys)", books[0].Title)
	assert.Equal(t, []string{"Scott Moon"}, books[0].Authors)
	assert.Equal(t, "to-read", books[0].ExclusiveShelf)

	// Verify a book with rating (Wool)
	var woolBook *Book
	for i := range books {
		if books[i].ID == 12287209 {
			woolBook = &books[i]
			break
		}
	}
	require.NotNil(t, woolBook, "should find Wool in the parsed books")
	assert.Equal(t, "Wool (Wool, #1)", woolBook.Title)
	assert.Equal(t, 5.0, woolBook.MyRating)
	assert.Equal(t, 4.20, woolBook.AverageRating)
	assert.Equal(t, []string{"Hugh Howey"}, woolBook.Authors)
	assert.Equal(t, "read", woolBook.ExclusiveShelf)
}
