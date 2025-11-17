package letterboxd

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/lepinkainen/hermes/internal/config"
	"github.com/lepinkainen/hermes/internal/datastore"
	"github.com/lepinkainen/hermes/internal/enrichment"
	"github.com/lepinkainen/hermes/internal/errors"
	"github.com/spf13/viper"
)

// Convert Movie to map[string]any for database insertion
func movieToMap(movie Movie) map[string]any {
	return map[string]any{
		"date":           movie.Date,
		"name":           movie.Name,
		"year":           movie.Year,
		"letterboxd_id":  movie.LetterboxdID,
		"letterboxd_uri": movie.LetterboxdURI,
		"imdb_id":        movie.ImdbID,
		"director":       movie.Director,
		"cast":           strings.Join(movie.Cast, ","),
		"genres":         strings.Join(movie.Genres, ","),
		"runtime":        movie.Runtime,
		"rating":         movie.Rating,
		"poster_url":     movie.PosterURL,
		"description":    movie.Description,
	}
}

// ParseLetterboxd parses a Letterboxd CSV export file
func ParseLetterboxd() error {
	// Double check overwrite flag with global config
	if overwrite != config.OverwriteFiles {
		slog.Warn("Overwrite flag mismatch! Using global value",
			"local", overwrite,
			"global", config.OverwriteFiles)
		overwrite = config.OverwriteFiles
	}

	// Log the config at startup
	slog.Info("Starting Letterboxd parser", "overwrite", overwrite)

	// Open and process CSV file
	movies, err := processCSVFile(csvFile)
	if err != nil {
		slog.Error("Failed to process CSV", "error", err)
		return err
	}

	slog.Info("Found movies", "count", len(movies))

	// Write markdown files
	slog.Info("Writing markdown")
	if err := writeMoviesToMarkdown(movies, outputDir); err != nil {
		slog.Error("Error writing markdown", "error", err)
		return err
	}

	// Write to JSON if enabled
	if writeJSON {
		if err := writeMoviesToJSON(movies, jsonOutput); err != nil {
			slog.Error("Error writing movies to JSON", "error", err)
			return err
		}
	}

	// Datasette integration
	if viper.GetBool("datasette.enabled") {
		slog.Info("Writing Letterboxd movies to Datasette")
		mode := viper.GetString("datasette.mode")

		switch mode {
		case "local":
			store := datastore.NewSQLiteStore(viper.GetString("datasette.dbfile"))
			if err := store.Connect(); err != nil {
				slog.Error("Failed to connect to SQLite database", "error", err)
				return err
			}
			defer func() { _ = store.Close() }()

			schema := `CREATE TABLE IF NOT EXISTS letterboxd_movies (
				date TEXT,
				name TEXT,
				year INTEGER,
				letterboxd_id TEXT PRIMARY KEY,
				letterboxd_uri TEXT,
				imdb_id TEXT,
				director TEXT,
				cast TEXT,
				genres TEXT,
				runtime INTEGER,
				rating REAL,
				poster_url TEXT,
				description TEXT
			)`

			if err := store.CreateTable(schema); err != nil {
				slog.Error("Failed to create table", "error", err)
				return err
			}

			records := make([]map[string]any, len(movies))
			for i, movie := range movies {
				records[i] = movieToMap(movie)
			}

			if err := store.BatchInsert("hermes", "letterboxd_movies", records); err != nil {
				slog.Error("Failed to insert records", "error", err)
				return err
			}
			slog.Info("Successfully wrote movies to SQLite database", "count", len(movies))
		case "remote":
			client := datastore.NewDatasetteClient(
				viper.GetString("datasette.remote_url"),
				viper.GetString("datasette.api_token"),
			)
			if err := client.Connect(); err != nil {
				slog.Error("Failed to connect to remote Datasette", "error", err)
				return err
			}
			defer func() { _ = client.Close() }()

			records := make([]map[string]any, len(movies))
			for i, movie := range movies {
				records[i] = movieToMap(movie)
			}

			if err := client.BatchInsert("hermes", "letterboxd_movies", records); err != nil {
				slog.Error("Failed to insert records to remote Datasette", "error", err)
				return err
			}
			slog.Info("Successfully wrote movies to remote Datasette", "count", len(movies))
		default:
			slog.Error("Invalid Datasette mode", "mode", mode)
			return fmt.Errorf("invalid Datasette mode: %s", mode)
		}
	}

	slog.Info("Processed movies", "count", len(movies))
	return nil
}

// processCSVFile reads and parses the Letterboxd CSV file
func processCSVFile(filename string) ([]Movie, error) {
	csvFile, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open CSV file: %v", err)
	}
	defer func() { _ = csvFile.Close() }()

	// File existence check
	if fi, err := csvFile.Stat(); err != nil || fi.Size() == 0 {
		return nil, fmt.Errorf("CSV file is empty or cannot be read")
	}

	reader := csv.NewReader(csvFile)

	// Skip header
	if _, err := reader.Read(); err != nil {
		return nil, fmt.Errorf("failed to read header: %v", err)
	}

	var movies []Movie

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			slog.Warn("Error reading record", "error", err)
			continue
		}

		movie, err := parseMovieRecord(record)
		if err != nil {
			if skipInvalid {
				slog.Warn("Skipping invalid movie", "error", err)
				continue
			}
			return nil, fmt.Errorf("invalid movie: %v", err)
		}

		movies = append(movies, movie)
	}

	return movies, nil
}

// parseMovieRecord converts a CSV record into a Movie struct
func parseMovieRecord(record []string) (Movie, error) {
	if len(record) < 4 {
		return Movie{}, fmt.Errorf("record does not have enough fields: got %d, expected at least 4", len(record))
	}

	// Movie name is the second column
	movieName := record[1]

	// Extract Letterboxd ID from URI
	letterboxdID := ""
	uri := record[3]
	parts := strings.Split(uri, "/")
	if len(parts) > 0 {
		letterboxdID = parts[len(parts)-1]
	}

	// Parse year
	year, err := strconv.Atoi(record[2])
	if err != nil {
		slog.Warn("Invalid year", "movie", movieName, "year", record[2])
		if !skipInvalid {
			return Movie{}, fmt.Errorf("invalid year: %s", record[2])
		}
		// Set default year if invalid
		year = 0
	}

	movie := Movie{
		Date:          record[0],
		Name:          record[1],
		Year:          year,
		LetterboxdURI: uri,
		LetterboxdID:  letterboxdID,
	}

	return movie, nil
}

// writeMoviesToJSON writes the movies to a JSON file
func writeMoviesToJSON(movies []Movie, jsonOutput string) error {
	// Enrich with OMDB data if not skipped
	if !skipEnrich {
		for i := range movies {
			if err := enrichMovieData(&movies[i]); err != nil {
				// Check if it's a rate limit error
				if _, isRateLimit := err.(*errors.RateLimitError); isRateLimit {
					return err
				}
				slog.Warn("Failed to enrich movie for JSON", "movie", movies[i].Name, "error", err)
				// Continue processing even if enrichment fails for other errors
			}
		}
	}

	return writeJSONFile(movies, jsonOutput)
}

// enrichMovieData fetches additional data from OMDB API
func enrichMovieData(movie *Movie) error {
	// Skip if we already have enriched data
	if movie.Description != "" {
		return nil
	}

	enrichedMovie, err := getCachedMovie(movie.Name, movie.Year)
	if err != nil {
		// Don't wrap rate limit errors
		if _, isRateLimit := err.(*errors.RateLimitError); isRateLimit {
			return err // Return the RateLimitError directly
		}
		return fmt.Errorf("failed to enrich movie data: %w", err)
	}

	// Preserve existing data but add OMDB enrichments
	movie.Description = enrichedMovie.Description
	movie.PosterURL = enrichedMovie.PosterURL

	// Only update these if they're empty
	if movie.Director == "" {
		movie.Director = enrichedMovie.Director
	}
	if len(movie.Cast) == 0 {
		movie.Cast = enrichedMovie.Cast
	}
	if len(movie.Genres) == 0 {
		movie.Genres = enrichedMovie.Genres
	}
	if movie.Runtime == 0 {
		movie.Runtime = enrichedMovie.Runtime
	}
	if movie.Rating == 0 {
		movie.Rating = enrichedMovie.Rating
	}

	// TMDB enrichment (if enabled)
	if tmdbEnabled {
		tmdbEnrichment, err := enrichFromTMDB(movie)
		if err != nil {
			slog.Warn("Failed to enrich from TMDB", "movie", movie.Name, "error", err)
			// Don't fail the whole import if TMDB enrichment fails
		} else if tmdbEnrichment != nil {
			movie.TMDBEnrichment = tmdbEnrichment
		}
	}

	return nil
}

// enrichFromTMDB enriches a movie with TMDB data
func enrichFromTMDB(movie *Movie) (*enrichment.TMDBEnrichment, error) {
	// Prepare attachments directory
	attachmentsDir := filepath.Join(outputDir, "attachments")
	if err := os.MkdirAll(attachmentsDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create attachments directory: %w", err)
	}

	opts := enrichment.TMDBEnrichmentOptions{
		DownloadCover:   tmdbDownloadCover,
		GenerateContent: tmdbGenerateContent,
		ContentSections: tmdbContentSections,
		AttachmentsDir:  attachmentsDir,
		NoteDir:         outputDir,
		Interactive:     tmdbInteractive,
	}

	// Use context.Background() for enrichment
	ctx := context.Background()
	return enrichment.EnrichFromTMDB(ctx, movie.Name, movie.Year, movie.ImdbID, 0, opts)
}

// writeMoviesToMarkdown writes each movie to a markdown file
func writeMoviesToMarkdown(movies []Movie, directory string) error {
	for i := range movies {
		slog.Info("Processing movie", "name", movies[i].Name)

		// Enrich with OMDB data if not skipped
		if !skipEnrich {
			if err := enrichMovieData(&movies[i]); err != nil {
				// Check if it's a rate limit error
				if _, isRateLimit := err.(*errors.RateLimitError); isRateLimit {
					return err
				}
				slog.Warn("Failed to enrich movie", "movie", movies[i].Name, "error", err)
				// Continue processing even if enrichment fails for other errors
			}
		}

		err := writeMovieToMarkdown(movies[i], directory)
		if err != nil {
			slog.Error("Failed to write markdown", "movie", movies[i].Name, "error", err)
			// Continue with other movies on error
			continue
		}
		slog.Debug("Wrote movie to markdown file", "movie", movies[i].Name)
	}
	return nil
}
