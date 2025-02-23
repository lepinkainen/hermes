package imdb

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

func parse_imdb() {
	// Set log level
	level, err := log.ParseLevel(logLevel)
	if err != nil {
		log.Fatalf("Invalid log level '%s': %v", logLevel, err)
		return
	}
	log.SetLevel(level)

	// Open and process CSV file
	movies, err := processCSVFile(inputFile)
	if err != nil {
		log.Fatalf("Failed to process CSV: %v", err)
		return
	}

	log.Infof("Found %d movies\n", len(movies))

	// Write outputs
	log.Infof("Writing JSON\n")
	if err := writeMovieToJson(movies, outputJson); err != nil {
		log.Errorf("Error writing JSON: %v\n", err)
	}

	log.Infof("Writing markdown\n")

	if err := writeMoviesToMarkdown(movies, filepath.Join(viper.GetString("MarkdownOutputDir"), "imdb")); err != nil {
		log.Errorf("Error writing markdown: %v\n", err)
	}

	log.Infof("Processed %d movies\n", len(movies))
}

func processCSVFile(filename string) ([]MovieSeen, error) {
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
	reader.FieldsPerRecord = 18

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

// parseMovieRecord converts a CSV record into a MovieSeen struct
func parseMovieRecord(record []string) (MovieSeen, error) {
	// Create logger with context
	movieLogger := log.WithFields(log.Fields{
		"ImdbId": record[1],
	})

	// Parse position
	position, err := strconv.Atoi(record[0])
	if err != nil {
		movieLogger.Warnf("Invalid position: %v", err)
		position = 0
	}

	// Parse rating (if exists)
	var rating int
	if record[16] != "" {
		rating, err = strconv.Atoi(record[16])
		if err != nil {
			return MovieSeen{}, fmt.Errorf("invalid rating: %v", err)
		}
	}

	// Parse IMDb rating
	var imdbRating float64
	if record[9] != "" {
		imdbRating, err = strconv.ParseFloat(record[9], 64)
		if err != nil {
			movieLogger.Warnf("Invalid IMDb rating: %v", err)
		}
	}

	// Parse runtime
	var runtimeMins int
	if record[10] != "" {
		runtimeMins, err = strconv.Atoi(record[10])
		if err != nil {
			movieLogger.Warnf("Error parsing runtime %s: %v\n", record[10], err)
		}
	}

	// Parse year
	var year int
	if record[11] != "" {
		year, err = strconv.Atoi(record[11])
		if err != nil {
			movieLogger.Warnf("Invalid year: %v", err)
		}
	}

	// Parse number of votes
	var numVotes int
	if record[13] != "" {
		numVotes, err = strconv.Atoi(record[13])
		if err != nil {
			movieLogger.Warnf("Invalid number of votes: %v", err)
		}
	}

	// Split genres into slice (handle empty case)
	var genres []string
	if record[12] != "" {
		genres = strings.Split(record[12], ", ")
	}

	// Split directors into slice (handle empty case)
	var directors []string
	if record[15] != "" {
		directors = strings.Split(record[15], ", ")
	}

	return MovieSeen{
		Position:      position,
		ImdbId:        record[1],
		MyRating:      rating,
		DateRated:     record[17],
		Created:       record[2],
		Modified:      record[3],
		Description:   record[4],
		Title:         record[5],
		OriginalTitle: record[6],
		URL:           record[7],
		TitleType:     record[8],
		IMDbRating:    imdbRating,
		RuntimeMins:   runtimeMins,
		Year:          year,
		Genres:        genres,
		NumVotes:      numVotes,
		ReleaseDate:   record[14],
		Directors:     directors,
	}, nil
}

func enrichMovieData(movie *MovieSeen) error {
	// Skip if we already have enriched data
	if movie.Plot != "" {
		return nil
	}

	omdbMovie, err := getCachedMovie(movie.ImdbId)
	if err != nil {
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

	return nil
}

func writeMoviesToMarkdown(movies []MovieSeen, directory string) error {
	for i := range movies {
		log.Infof("Processing movie %d of %d (%s)\n", i+1, len(movies), movies[i].Title)
		// Enrich with OMDB data
		if err := enrichMovieData(&movies[i]); err != nil {
			log.Warnf("Failed to enrich movie %s: %v", movies[i].Title, err)
			// Continue processing even if enrichment fails
		}

		err := writeMovieToMarkdown(movies[i], directory)
		if err != nil {
			return err
		}
	}
	return nil
}
