package letterboxd

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/lepinkainen/hermes/internal/cmdutil"
	"github.com/lepinkainen/hermes/internal/config"
	"github.com/lepinkainen/hermes/internal/csvutil"
	"github.com/lepinkainen/hermes/internal/enrichment"
	"github.com/lepinkainen/hermes/internal/errors"
	"github.com/lepinkainen/hermes/internal/fileutil"
	"github.com/lepinkainen/hermes/internal/importer/enrich"
	"github.com/lepinkainen/hermes/internal/importer/mediaids"
	"github.com/lepinkainen/hermes/internal/omdb"
)

const letterboxdMoviesSchema = `CREATE TABLE IF NOT EXISTS letterboxd_movies (
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
	if err := cmdutil.WriteToDatastore(movies, letterboxdMoviesSchema, "letterboxd_movies", "Letterboxd movies", movieToMap); err != nil {
		return err
	}

	slog.Info("Processed movies", "count", len(movies))
	return nil
}

// processCSVFile reads and parses the Letterboxd CSV file
func processCSVFile(filename string) ([]Movie, error) {
	opts := csvutil.ProcessorOptions{
		SkipInvalid: skipInvalid,
	}
	return csvutil.ProcessCSV(filename, parseMovieRecord, opts)
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
		slog.Warn("Invalid year", "title", movieName, "year", record[2])
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
	for i := range movies {
		// If we already have a TMDB ID in an existing note, use it directly to avoid searching
		loadExistingTMDBID(&movies[i], outputDir)

		// Enrich with OMDB/TMDB data unless explicitly skipped
		if skipEnrich {
			continue
		}

		if err := enrichMovieData(&movies[i]); err != nil {
			if errors.IsRateLimitError(err) {
				omdb.MarkRateLimitReached()
				slog.Warn("Skipping OMDB enrichment after rate limit", "title", movies[i].Name)
				continue
			}
			slog.Warn("Failed to enrich movie for JSON", "title", movies[i].Name, "error", err)
			// Continue processing even if enrichment fails for other errors
		}
	}

	return writeJSONFile(movies, jsonOutput)
}

// enrichMovieData fetches additional data from OMDB API
func enrichMovieData(movie *Movie) error {
	_, err := enrich.Enrich(movie, enrich.Options[Movie, *Movie]{
		SkipOMDB: movie.Description != "",
		FetchOMDB: func() (*Movie, error) {
			return getCachedMovie(movie.Name, movie.Year)
		},
		ApplyOMDB: func(target *Movie, enrichedMovie *Movie) {
			if enrichedMovie == nil {
				return
			}
			target.Description = enrichedMovie.Description
			target.PosterURL = enrichedMovie.PosterURL

			if target.Director == "" {
				target.Director = enrichedMovie.Director
			}
			if len(target.Cast) == 0 {
				target.Cast = enrichedMovie.Cast
			}
			if len(target.Genres) == 0 {
				target.Genres = enrichedMovie.Genres
			}
			if target.Runtime == 0 {
				target.Runtime = enrichedMovie.Runtime
			}
			if target.Rating == 0 {
				target.Rating = enrichedMovie.Rating
			}
		},
		OnOMDBError: func(err error) {
			slog.Warn("Failed to enrich from OMDB", "title", movie.Name, "error", err)
		},
		OnOMDBRateLimit: func(error) {
			omdb.MarkRateLimitReached()
			slog.Warn("Skipping OMDB enrichment after rate limit", "title", movie.Name)
		},
		TMDBEnabled: tmdbEnabled,
		FetchTMDB: func() (*enrichment.TMDBEnrichment, error) {
			return enrichFromTMDB(movie)
		},
		ApplyTMDB: func(target *Movie, tmdbEnrichment *enrichment.TMDBEnrichment) {
			target.TMDBEnrichment = tmdbEnrichment

			if tmdbEnrichment.RuntimeMins > 0 && target.Runtime == 0 {
				slog.Debug("Using TMDB runtime", "title", target.Name, "omdb_runtime", target.Runtime, "tmdb_runtime", tmdbEnrichment.RuntimeMins)
				target.Runtime = tmdbEnrichment.RuntimeMins
			}

			if tmdbEnrichment.CoverPath != "" || tmdbEnrichment.CoverFilename != "" {
				slog.Debug("Using TMDB cover (higher resolution)", "title", target.Name)
			}
		},
		OnTMDBError: func(err error) {
			slog.Warn("Failed to enrich from TMDB", "title", movie.Name, "error", err)
		},
	})

	return err
}

// enrichFromTMDB enriches a movie with TMDB data
func enrichFromTMDB(movie *Movie) (*enrichment.TMDBEnrichment, error) {
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
		WithMoviesOnly(true). // Letterboxd only catalogs movies, not TV shows
		WithStoredType(storedType).
		WithCoverCache(useTMDBCoverCache, tmdbCoverCachePath).
		Build()

	return enrichment.EnrichFromTMDB(context.Background(), movie.Name, movie.Year, movie.ImdbID, existingTMDBID, opts)
}

// writeMoviesToMarkdown writes each movie to a markdown file
func writeMoviesToMarkdown(movies []Movie, directory string) error {
	for i := range movies {
		slog.Info("Processing movie", "title", movies[i].Name)

		// First, check if we have a cached mapping for this Letterboxd URI
		if mapping, err := GetLetterboxdMapping(movies[i].LetterboxdURI); err != nil {
			slog.Warn("Failed to load Letterboxd mapping from cache", "error", err)
		} else if mapping != nil {
			// Use cached mapping
			if mapping.TMDBID != 0 && movies[i].TMDBEnrichment == nil {
				// Initialize TMDB enrichment from cached mapping
				movies[i].TMDBEnrichment = &enrichment.TMDBEnrichment{
					TMDBID:   mapping.TMDBID,
					TMDBType: mapping.TMDBType,
				}
				slog.Debug("Loaded TMDB ID from Letterboxd mapping cache",
					"title", movies[i].Name,
					"letterboxd_uri", movies[i].LetterboxdURI,
					"tmdb_id", mapping.TMDBID)
			}
			if mapping.ImdbID != "" && movies[i].ImdbID == "" {
				movies[i].ImdbID = mapping.ImdbID
				slog.Debug("Loaded IMDB ID from Letterboxd mapping cache",
					"title", movies[i].Name,
					"imdb_id", mapping.ImdbID)
			}
		}

		// If a TMDB ID already exists in the note, reuse it instead of searching
		loadExistingTMDBID(&movies[i], directory)

		// Enrich with OMDB data if not skipped
		if !skipEnrich {
			if err := enrichMovieData(&movies[i]); err != nil {
				if errors.IsStopProcessingError(err) {
					return err
				}
				if errors.IsRateLimitError(err) {
					omdb.MarkRateLimitReached()
					slog.Warn("Skipping OMDB enrichment after rate limit", "title", movies[i].Name)
					// Continue writing without enrichment
				} else {
					slog.Warn("Failed to enrich movie", "title", movies[i].Name, "error", err)
					// Continue processing even if enrichment fails for other errors
				}
			}
		}

		// After enrichment, save the mapping to cache if we have new data
		if movies[i].LetterboxdURI != "" {
			mapping := LetterboxdMapping{
				LetterboxdURI: movies[i].LetterboxdURI,
				ImdbID:        movies[i].ImdbID,
			}
			if movies[i].TMDBEnrichment != nil {
				mapping.TMDBID = movies[i].TMDBEnrichment.TMDBID
				mapping.TMDBType = movies[i].TMDBEnrichment.TMDBType
			}
			if err := SetLetterboxdMapping(mapping); err != nil {
				slog.Warn("Failed to save Letterboxd mapping to cache", "error", err)
			}
		}

		err := writeMovieToMarkdown(movies[i], directory)
		if err != nil {
			slog.Error("Failed to write markdown", "title", movies[i].Name, "error", err)
			// Continue with other movies on error
			continue
		}
		slog.Debug("Wrote movie to markdown file", "title", movies[i].Name)
	}
	return nil
}

// loadExistingTMDBID reads the existing markdown (if any) and initializes TMDB ID/type
// so enrichment can fetch directly without searching again.
func loadExistingTMDBID(movie *Movie, directory string) {
	if movie == nil {
		return
	}

	title := fmt.Sprintf("%s (%d)", movie.Name, movie.Year)
	filePath := fileutil.GetMarkdownFilePath(title, directory)

	ids, err := mediaids.FromFile(filePath)
	if err != nil {
		return
	}

	// Prefer stored IMDb ID when CSV is missing it
	if movie.ImdbID == "" {
		movie.ImdbID = ids.IMDBID
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
