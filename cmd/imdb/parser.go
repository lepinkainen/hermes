package imdb

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/lepinkainen/hermes/internal/cmdutil"
	"github.com/lepinkainen/hermes/internal/csvutil"
	"github.com/lepinkainen/hermes/internal/enrichment"
	"github.com/lepinkainen/hermes/internal/errors"
	"github.com/lepinkainen/hermes/internal/fileutil"
	"github.com/lepinkainen/hermes/internal/importer/enrich"
	"github.com/lepinkainen/hermes/internal/importer/mediaids"
	"github.com/lepinkainen/hermes/internal/omdb"
)

const IMDbMoviesSchema = `CREATE TABLE IF NOT EXISTS imdb_movies (
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

// Convert MovieSeen to map[string]any for database insertion
func movieToMap(movie MovieSeen) map[string]any {
	return cmdutil.StructToMap(movie, cmdutil.StructToMapOptions{
		JoinStringSlices: true,
		OmitFields: map[string]bool{
			"TMDBEnrichment": true,
		},
	})
}

func ParseImdb() error {
	// Create output directories once before processing
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}
	attachmentsDir := filepath.Join(outputDir, "attachments")
	if err := os.MkdirAll(attachmentsDir, 0755); err != nil {
		return fmt.Errorf("failed to create attachments directory: %w", err)
	}

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
	if err := cmdutil.WriteToDatastore(movies, IMDbMoviesSchema, "imdb_movies", "IMDB movies", movieToMap); err != nil {
		return err
	}

	slog.Info("Processed movies", "count", len(movies))
	return nil
}

func processCSVFile(filename string) ([]MovieSeen, error) {
	opts := csvutil.ProcessorOptions{
		FieldsPerRecord: 14, // IMDb export format has 14 fields
		SkipInvalid:     skipInvalid,
	}
	return csvutil.ProcessCSV(filename, parseMovieRecord, opts)
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
	_, err := enrich.Enrich(movie, enrich.Options[MovieSeen, *MovieSeen]{
		SkipOMDB: movie.Plot != "",
		FetchOMDB: func() (*MovieSeen, error) {
			movieData, _, err := omdb.GetCached(movie.ImdbId, func() (*MovieSeen, error) {
				return fetchMovieData(movie.ImdbId)
			})
			return movieData, err
		},
		ApplyOMDB: func(target *MovieSeen, omdbMovie *MovieSeen) {
			if omdbMovie == nil {
				return
			}
			target.Plot = omdbMovie.Plot
			target.PosterURL = omdbMovie.PosterURL
			target.ContentRated = omdbMovie.ContentRated
			target.Awards = omdbMovie.Awards

			if len(target.Genres) == 0 {
				target.Genres = omdbMovie.Genres
			}
			if len(target.Directors) == 0 {
				target.Directors = omdbMovie.Directors
			}
			if target.RuntimeMins == 0 {
				target.RuntimeMins = omdbMovie.RuntimeMins
			}
		},
		OnOMDBError: func(err error) {
			slog.Warn("Failed to enrich from OMDB", "title", movie.Title, "error", err)
		},
		OnOMDBRateLimit: func(error) {
			omdb.MarkRateLimitReached()
			slog.Warn("Skipping OMDB enrichment after rate limit", "title", movie.Title)
		},
		TMDBEnabled: tmdbEnabled,
		FetchTMDB: func() (*enrichment.TMDBEnrichment, error) {
			return enrichFromTMDB(movie)
		},
		ApplyTMDB: func(target *MovieSeen, tmdbEnrichment *enrichment.TMDBEnrichment) {
			target.TMDBEnrichment = tmdbEnrichment
		},
		OnTMDBError: func(err error) {
			slog.Warn("Failed to enrich from TMDB", "title", movie.Title, "error", err)
		},
	})

	return err
}

// enrichFromTMDB enriches a movie with TMDB data
func enrichFromTMDB(movie *MovieSeen) (*enrichment.TMDBEnrichment, error) {
	storedType := ""
	existingTMDBID := 0
	if movie.TMDBEnrichment != nil {
		storedType = movie.TMDBEnrichment.TMDBType
		existingTMDBID = movie.TMDBEnrichment.TMDBID
	}

	opts := enrichment.NewTMDBOptionsBuilder(outputDir).
		WithCover(tmdbDownloadCover).
		WithContent(tmdbGenerateContent, tmdbContentSections).
		WithInteractive(tmdbInteractive).
		WithStoredType(storedType).
		WithCoverCache(useTMDBCoverCache, tmdbCoverCachePath).
		Build()

	return enrichment.EnrichFromTMDB(context.Background(), movie.Title, movie.Year, movie.ImdbId, existingTMDBID, opts)
}

func loadExistingMediaIDs(movie *MovieSeen, directory string) {
	if movie == nil {
		return
	}

	filePath := fileutil.GetMarkdownFilePath(movie.Title, directory)

	ids, err := mediaids.FromFile(filePath)
	if err != nil {
		return
	}

	if movie.ImdbId == "" {
		movie.ImdbId = ids.IMDBID
	}

	if ids.TMDBID <= 0 {
		return
	}

	if movie.TMDBEnrichment == nil {
		movie.TMDBEnrichment = &enrichment.TMDBEnrichment{}
	}
	movie.TMDBEnrichment.TMDBID = ids.TMDBID
	movie.TMDBEnrichment.TMDBType = ids.TMDBType
}

func writeMoviesToMarkdown(movies []MovieSeen, directory string) error {
	for i := range movies {
		slog.Info("Processing movie", "title", movies[i].Title)

		loadExistingMediaIDs(&movies[i], directory)

		// Enrich with OMDB data
		if err := enrichMovieData(&movies[i]); err != nil {
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
