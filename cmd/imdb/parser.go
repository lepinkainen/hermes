package imdb

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

	"github.com/lepinkainen/hermes/internal/datastore"
	"github.com/lepinkainen/hermes/internal/enrichment"
	"github.com/lepinkainen/hermes/internal/errors"
	"github.com/spf13/viper"
)

// Convert MovieSeen to map[string]any for database insertion
func movieToMap(movie MovieSeen) map[string]any {
	return map[string]any{
		"position":       movie.Position,
		"imdb_id":        movie.ImdbId,
		"my_rating":      movie.MyRating,
		"date_rated":     movie.DateRated,
		"created":        movie.Created,
		"modified":       movie.Modified,
		"description":    movie.Description,
		"title":          movie.Title,
		"original_title": movie.OriginalTitle,
		"url":            movie.URL,
		"title_type":     movie.TitleType,
		"imdb_rating":    movie.IMDbRating,
		"runtime_mins":   movie.RuntimeMins,
		"year":           movie.Year,
		"genres":         strings.Join(movie.Genres, ","),
		"num_votes":      movie.NumVotes,
		"release_date":   movie.ReleaseDate,
		"directors":      strings.Join(movie.Directors, ","),
		"plot":           movie.Plot,
		"content_rated":  movie.ContentRated,
		"awards":         movie.Awards,
		"poster_url":     movie.PosterURL,
	}
}

func ParseImdb() error {
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
		if _, isRateLimit := err.(*errors.RateLimitError); isRateLimit {
			slog.Error("Stopping import due to rate limit", "error", err)
			return err
		}
		slog.Error("Error writing markdown", "error", err)
		return err
	}

	// Write to JSON if enabled
	if writeJSON {
		if err := writeMovieToJson(movies, jsonOutput); err != nil {
			slog.Error("Error writing movies to JSON", "error", err)
		}
	}

	// Write to Datasette if enabled
	if viper.GetBool("datasette.enabled") {
		slog.Info("Writing to Datasette")
		mode := viper.GetString("datasette.mode")

		switch mode {
		case "local":
			// Initialize local SQLite store
			store := datastore.NewSQLiteStore(viper.GetString("datasette.dbfile"))
			if err := store.Connect(); err != nil {
				slog.Error("Failed to connect to SQLite database", "error", err)
				return err
			}
			defer func() { _ = store.Close() }()

			// Create table schema
			schema := `CREATE TABLE IF NOT EXISTS imdb_movies (
				position INTEGER,
				imdb_id TEXT PRIMARY KEY,
				my_rating INTEGER,
				date_rated TEXT,
				created TEXT,
				modified TEXT,
				description TEXT,
				title TEXT,
				original_title TEXT,
				url TEXT,
				title_type TEXT,
				imdb_rating REAL,
				runtime_mins INTEGER,
				year INTEGER,
				genres TEXT,
				num_votes INTEGER,
				release_date TEXT,
				directors TEXT,
				plot TEXT,
				content_rated TEXT,
				awards TEXT,
				poster_url TEXT
			)`

			if err := store.CreateTable(schema); err != nil {
				slog.Error("Failed to create table", "error", err)
				return err
			}

			// Convert movies to maps for insertion
			records := make([]map[string]any, len(movies))
			for i, movie := range movies {
				records[i] = movieToMap(movie)
			}

			// Insert records
			if err := store.BatchInsert("hermes", "imdb_movies", records); err != nil {
				slog.Error("Failed to insert records", "error", err)
				return err
			}
			slog.Info("Successfully wrote movies to SQLite database", "count", len(movies))

		case "remote":
			// Initialize remote Datasette client
			client := datastore.NewDatasetteClient(
				viper.GetString("datasette.remote_url"),
				viper.GetString("datasette.api_token"),
			)
			if err := client.Connect(); err != nil {
				slog.Error("Failed to connect to remote Datasette", "error", err)
				return err
			}
			defer func() { _ = client.Close() }()

			// Convert movies to maps for insertion
			records := make([]map[string]any, len(movies))
			for i, movie := range movies {
				records[i] = movieToMap(movie)
			}

			// Insert records
			if err := client.BatchInsert("hermes", "imdb_movies", records); err != nil {
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

func processCSVFile(filename string) ([]MovieSeen, error) {
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
	reader.FieldsPerRecord = 14 // New format has 14 fields

	// Skip header
	if _, err := reader.Read(); err != nil {
		return nil, fmt.Errorf("failed to read header: %v", err)
	}

	var movies []MovieSeen

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

// parseMovieRecord converts a CSV record into a MovieSeen struct
func parseMovieRecord(record []string) (MovieSeen, error) {
	// Create a logger attribute for context
	imdbID := record[0] // Const is now first column

	// Parse rating
	rating, err := strconv.Atoi(record[1]) // Your Rating is second column
	if err != nil {
		return MovieSeen{}, fmt.Errorf("invalid rating: %v", err)
	}

	// Parse IMDb rating
	var imdbRating float64
	if record[7] != "" && record[7] != "null" {
		imdbRating, err = strconv.ParseFloat(record[7], 64)
		if err != nil {
			slog.Warn("Invalid IMDb rating", "imdb_id", imdbID, "error", err)
		}
	}

	// Parse runtime
	var runtimeMins int
	if record[8] != "" {
		runtimeMins, err = strconv.Atoi(record[8])
		if err != nil {
			slog.Warn("Error parsing runtime", "imdb_id", imdbID, "runtime", record[8], "error", err)
		}
	}

	// Parse year
	var year int
	if record[9] != "" {
		year, err = strconv.Atoi(record[9])
		if err != nil {
			slog.Warn("Invalid year", "imdb_id", imdbID, "error", err)
		}
	}

	// Parse number of votes
	var numVotes int
	if record[11] != "" {
		numVotes, err = strconv.Atoi(record[11])
		if err != nil {
			slog.Warn("Invalid number of votes", "imdb_id", imdbID, "error", err)
		}
	}

	// Split genres into slice
	var genres []string
	if record[10] != "" {
		genres = strings.Split(record[10], ", ")
	}

	// Split directors into slice
	var directors []string
	if record[13] != "" {
		directors = strings.Split(record[13], ",")
	}

	return MovieSeen{
		ImdbId:        record[0],   // Const
		MyRating:      rating,      // Your Rating
		DateRated:     record[2],   // Date Rated
		Title:         record[3],   // Title
		OriginalTitle: record[4],   // Original Title
		URL:           record[5],   // URL
		TitleType:     record[6],   // Title Type
		IMDbRating:    imdbRating,  // IMDb Rating
		RuntimeMins:   runtimeMins, // Runtime (mins)
		Year:          year,        // Year
		Genres:        genres,      // Genres
		NumVotes:      numVotes,    // Num Votes
		ReleaseDate:   record[12],  // Release Date
		Directors:     directors,   // Directors
	}, nil
}

func enrichMovieData(movie *MovieSeen) error {
	// Skip if we already have enriched data
	if movie.Plot != "" {
		return nil
	}

	omdbMovie, err := getCachedMovie(movie.ImdbId)
	if err != nil {
		// Don't wrap rate limit errors
		if _, isRateLimit := err.(*errors.RateLimitError); isRateLimit {
			return err // Return the RateLimitError directly
		}
		return fmt.Errorf("failed to enrich movie data: %w", err)
	}

	// Preserve existing data but add OMDB enrichments
	movie.Plot = omdbMovie.Plot
	movie.PosterURL = omdbMovie.PosterURL
	movie.ContentRated = omdbMovie.ContentRated
	movie.Awards = omdbMovie.Awards

	// Only update these if they're empty
	if len(movie.Genres) == 0 {
		movie.Genres = omdbMovie.Genres
	}
	if len(movie.Directors) == 0 {
		movie.Directors = omdbMovie.Directors
	}
	if movie.RuntimeMins == 0 {
		movie.RuntimeMins = omdbMovie.RuntimeMins
	}

	// TMDB enrichment (if enabled)
	if tmdbEnabled {
		tmdbEnrichment, err := enrichFromTMDB(movie)
		if err != nil {
			slog.Warn("Failed to enrich from TMDB", "title", movie.Title, "error", err)
			// Don't fail the whole import if TMDB enrichment fails
		} else if tmdbEnrichment != nil {
			movie.TMDBEnrichment = tmdbEnrichment
		}
	}

	return nil
}

// enrichFromTMDB enriches a movie with TMDB data
func enrichFromTMDB(movie *MovieSeen) (*enrichment.TMDBEnrichment, error) {
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
	return enrichment.EnrichFromTMDB(ctx, movie.Title, movie.Year, movie.ImdbId, 0, opts)
}

func writeMoviesToMarkdown(movies []MovieSeen, directory string) error {
	for i := range movies {
		slog.Info("Processing movie", "title", movies[i].Title)

		// Enrich with OMDB data
		if err := enrichMovieData(&movies[i]); err != nil {
			// Check if it's a rate limit error
			if _, isRateLimit := err.(*errors.RateLimitError); isRateLimit {
				return err
			}
			slog.Warn("Failed to enrich movie", "title", movies[i].Title, "error", err)
			// Continue processing even if enrichment fails for other errors
			continue
		}

		err := writeMovieToMarkdown(movies[i], directory)
		if err != nil {
			return err
		}
		slog.Debug("Wrote movie to markdown file", "title", movies[i].Title, "directory", directory)
	}
	return nil
}
