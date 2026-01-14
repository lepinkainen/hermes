package diff

import (
	"database/sql"
	"fmt"
	"log/slog"
	"strings"

	_ "modernc.org/sqlite"
)

func loadImdbMovies(db *sql.DB) ([]imdbMovie, error) {
	rows, err := db.Query(`
		SELECT imdb_id, title, original_title, year, url, my_rating
		FROM imdb_movies
		WHERE lower(title_type) = 'movie'
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to query imdb movies: %w", err)
	}
	defer func() { _ = rows.Close() }()

	movies := []imdbMovie{}
	for rows.Next() {
		var imdbID, title, originalTitle, url sql.NullString
		var year sql.NullInt64
		var myRating sql.NullInt64

		if err := rows.Scan(&imdbID, &title, &originalTitle, &year, &url, &myRating); err != nil {
			return nil, fmt.Errorf("failed to scan imdb movie: %w", err)
		}

		movies = append(movies, imdbMovie{
			ImdbID:        imdbID.String,
			Title:         title.String,
			OriginalTitle: originalTitle.String,
			Year:          int(year.Int64),
			URL:           url.String,
			MyRating:      int(myRating.Int64),
		})
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read imdb movies: %w", err)
	}

	slog.Info("Loaded IMDb movies", "count", len(movies))
	return movies, nil
}

func loadLetterboxdMovies(db *sql.DB) ([]letterboxdMovie, error) {
	rows, err := db.Query(`
		SELECT name, year, letterboxd_id, letterboxd_uri, imdb_id, rating
		FROM letterboxd_movies
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to query letterboxd movies: %w", err)
	}
	defer func() { _ = rows.Close() }()

	movies := []letterboxdMovie{}
	for rows.Next() {
		var name, letterboxdID, letterboxdURI, imdbID sql.NullString
		var year sql.NullInt64
		var rating sql.NullFloat64

		if err := rows.Scan(&name, &year, &letterboxdID, &letterboxdURI, &imdbID, &rating); err != nil {
			return nil, fmt.Errorf("failed to scan letterboxd movie: %w", err)
		}

		movies = append(movies, letterboxdMovie{
			Name:          name.String,
			Year:          int(year.Int64),
			LetterboxdID:  letterboxdID.String,
			LetterboxdURI: letterboxdURI.String,
			ImdbID:        imdbID.String,
			Rating:        rating.Float64,
		})
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read letterboxd movies: %w", err)
	}

	slog.Info("Loaded Letterboxd movies", "count", len(movies))
	return movies, nil
}

func loadLetterboxdMappings(cacheDBPath string) (map[string]string, error) {
	db, err := sql.Open("sqlite", cacheDBPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open cache database: %w", err)
	}
	defer func() { _ = db.Close() }()

	rows, err := db.Query(`
		SELECT letterboxd_uri, imdb_id
		FROM letterboxd_mapping_cache
		WHERE imdb_id IS NOT NULL AND imdb_id != ''
	`)
	if err != nil {
		if strings.Contains(err.Error(), "no such table") {
			slog.Warn("letterboxd_mapping_cache table missing in cache database")
			return map[string]string{}, nil
		}
		return nil, fmt.Errorf("failed to query letterboxd mapping cache: %w", err)
	}
	defer func() { _ = rows.Close() }()

	mapping := make(map[string]string)
	for rows.Next() {
		var uri, imdbID sql.NullString
		if err := rows.Scan(&uri, &imdbID); err != nil {
			return nil, fmt.Errorf("failed to scan letterboxd mapping: %w", err)
		}
		if uri.String != "" && imdbID.String != "" {
			mapping[uri.String] = imdbID.String
		}
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read letterboxd mapping cache: %w", err)
	}

	slog.Info("Loaded Letterboxd mappings", "count", len(mapping))
	return mapping, nil
}

func applyLetterboxdMappings(movies []letterboxdMovie, mapping map[string]string) {
	if len(mapping) == 0 {
		return
	}
	for i := range movies {
		if movies[i].ImdbID != "" {
			continue
		}
		if movies[i].LetterboxdURI == "" {
			continue
		}
		if imdbID, ok := mapping[movies[i].LetterboxdURI]; ok {
			movies[i].ImdbID = imdbID
		}
	}
}
