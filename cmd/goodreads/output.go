package goodreads

import (
	"log/slog"

	"github.com/lepinkainen/hermes/internal/cmdutil"
)

func writeBooksToJSONIfEnabled(books []Book, writeJSON bool, jsonOutput string) {
	if !writeJSON {
		return
	}

	if err := writeBookToJson(books, jsonOutput); err != nil {
		slog.Error("Error writing books to JSON", "error", err)
	}
}

const goodreadsBooksSchema = `CREATE TABLE IF NOT EXISTS goodreads_books (
		id INTEGER PRIMARY KEY,
		title TEXT,
		authors TEXT,
		isbn TEXT,
		isbn13 TEXT,
		my_rating REAL,
		average_rating REAL,
		publisher TEXT,
		binding TEXT,
		number_of_pages INTEGER,
		year_published INTEGER,
		original_publication_year INTEGER,
		date_read TEXT,
		date_added TEXT,
		bookshelves TEXT,
		bookshelves_with_positions TEXT,
		exclusive_shelf TEXT,
		my_review TEXT,
		spoiler TEXT,
		private_notes TEXT,
		read_count INTEGER,
		owned_copies INTEGER,
		description TEXT,
		subjects TEXT,
		cover_id INTEGER,
		cover_url TEXT,
		subject_people TEXT,
		subtitle TEXT
	)`

func writeBooksToDatasetteIfEnabled(books []Book) error {
	return cmdutil.WriteToDatastore(books, goodreadsBooksSchema, "goodreads_books", "Goodreads books", bookToMap)
}
