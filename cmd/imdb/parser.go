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

	// Write outputs
	if err := writeMovieToJson(movies, outputJson); err != nil {
		log.Errorf("Error writing JSON: %v\n", err)
	}

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
	reader.FieldsPerRecord = 14

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
		"ImdbId": record[0],
	})

	// Parse rating
	rating, err := strconv.Atoi(record[1])
	if err != nil {
		return MovieSeen{}, fmt.Errorf("invalid rating: %v", err)
	}

	// Parse IMDb rating
	imdbRating, err := strconv.ParseFloat(record[7], 64)
	if err != nil {
		movieLogger.Warnf("Invalid IMDb rating: %v", err)
		imdbRating = 0
	}

	// Parse runtime
	runtimeMins, err := strconv.Atoi(record[8])
	if err != nil {
		if record[8] != "" {
			movieLogger.Warnf("Error parsing runtime %s: %v\n", record[8], err)
		}
		runtimeMins = 0
	}

	// Parse year
	year, err := strconv.Atoi(record[9])
	if err != nil {
		movieLogger.Warnf("Invalid year: %v", err)
		year = 0
	}

	// Parse number of votes
	numVotes, err := strconv.Atoi(record[11])
	if err != nil {
		movieLogger.Warnf("Invalid number of votes: %v", err)
		numVotes = 0
	}

	// Split genres into slice
	genres := strings.Split(record[10], ", ")

	// Split directors into slice
	directors := strings.Split(record[13], ", ")

	return MovieSeen{
		ImdbId:        record[0],
		MyRating:      rating,
		DateRated:     record[2],
		Title:         record[3],
		OriginalTitle: record[4],
		URL:           record[5],
		TitleType:     record[6],
		IMDbRating:    imdbRating,
		RuntimeMins:   runtimeMins,
		Year:          year,
		Genres:        genres,
		NumVotes:      numVotes,
		ReleaseDate:   record[12],
		Directors:     directors,
	}, nil
}
