package letterboxd

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/lepinkainen/hermes/internal/config"
	"github.com/lepinkainen/hermes/internal/errors"
	log "github.com/sirupsen/logrus"
)

// ParseLetterboxd parses a Letterboxd CSV export file
func ParseLetterboxd() error {
	// Double check overwrite flag with global config
	if overwrite != config.OverwriteFiles {
		log.Warnf("Overwrite flag mismatch! Local=%v, Global=%v. Using global value.",
			overwrite, config.OverwriteFiles)
		overwrite = config.OverwriteFiles
	}

	// Log the config at startup
	log.Infof("Starting Letterboxd parser with overwrite=%v", overwrite)

	// Open and process CSV file
	movies, err := processCSVFile(csvFile)
	if err != nil {
		log.Fatalf("Failed to process CSV: %v", err)
		return err
	}

	log.Infof("Found %d movies\n", len(movies))

	// Write markdown files
	log.Infof("Writing markdown\n")
	if err := writeMoviesToMarkdown(movies, outputDir); err != nil {
		log.Errorf("Error writing markdown: %v\n", err)
		return err
	}

	// Write to JSON if enabled
	if writeJSON {
		if err := writeMoviesToJSON(movies, jsonOutput); err != nil {
			log.Errorf("Error writing movies to JSON: %v\n", err)
			return err
		}
	}

	log.Infof("Processed %d movies\n", len(movies))
	return nil
}

// processCSVFile reads and parses the Letterboxd CSV file
func processCSVFile(filename string) ([]Movie, error) {
	csvFile, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open CSV file: %v", err)
	}
	defer csvFile.Close()

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
			log.Warnf("Error reading record: %v", err)
			continue
		}

		movie, err := parseMovieRecord(record)
		if err != nil {
			if skipInvalid {
				log.Warnf("Skipping invalid movie: %v", err)
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

	// Create logger with context
	movieLogger := log.WithFields(log.Fields{
		"Name": record[1], // Movie name is the second column
	})

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
		movieLogger.Warnf("Invalid year: %s", record[2])
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
				log.Warnf("Failed to enrich movie %s for JSON: %v", movies[i].Name, err)
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

	return nil
}

// writeMoviesToMarkdown writes each movie to a markdown file
func writeMoviesToMarkdown(movies []Movie, directory string) error {
	for i := range movies {
		log.Infof("Processing: %s\n", movies[i].Name)

		// Enrich with OMDB data if not skipped
		if !skipEnrich {
			if err := enrichMovieData(&movies[i]); err != nil {
				// Check if it's a rate limit error
				if _, isRateLimit := err.(*errors.RateLimitError); isRateLimit {
					return err
				}
				log.Warnf("Failed to enrich movie %s: %v", movies[i].Name, err)
				// Continue processing even if enrichment fails for other errors
			}
		}

		err := writeMovieToMarkdown(movies[i], directory)
		if err != nil {
			log.Errorf("Failed to write markdown for %s: %v", movies[i].Name, err)
			// Continue with other movies on error
			continue
		}
		log.Debugf("Wrote movie %s to markdown file", movies[i].Name)
	}
	return nil
}
