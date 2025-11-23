package goodreads

import (
	"log/slog"

	"github.com/lepinkainen/hermes/internal/datastore"
	"github.com/spf13/viper"
)

func writeBooksToJSONIfEnabled(books []Book) {
	if !writeJSON {
		return
	}

	if err := writeBookToJson(books, jsonOutput); err != nil {
		slog.Error("Error writing books to JSON", "error", err)
	}
}

func writeBooksToDatasetteIfEnabled(books []Book) error {
	if !viper.GetBool("datasette.enabled") {
		return nil
	}

	slog.Info("Writing Goodreads books to Datasette")

	store := datastore.NewSQLiteStore(viper.GetString("datasette.dbfile"))
	if err := store.Connect(); err != nil {
		slog.Error("Failed to connect to SQLite database", "error", err)
		return err
	}
	defer func() { _ = store.Close() }()

	schema := `CREATE TABLE IF NOT EXISTS goodreads_books (
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

	if err := store.CreateTable(schema); err != nil {
		slog.Error("Failed to create table", "error", err)
		return err
	}

	records := make([]map[string]any, len(books))
	for i, book := range books {
		records[i] = bookToMap(book)
	}

	if err := store.BatchInsert("hermes", "goodreads_books", records); err != nil {
		slog.Error("Failed to insert records", "error", err)
		return err
	}
	slog.Info("Successfully wrote books to SQLite database", "count", len(books))

	return nil
}
