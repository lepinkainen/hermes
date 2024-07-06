/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

type MovieSeen struct {
	ImdbId        string   `json:"ImdbId"`
	MyRating      int      `json:"My Rating"`
	DateRated     string   `json:"Date Rated"`
	Title         string   `json:"Title"`
	OriginalTitle string   `json:"Original Title"`
	URL           string   `json:"URL"`
	TitleType     string   `json:"Title Type"`
	IMDbRating    float64  `json:"IMDb Rating"`
	RuntimeMins   int      `json:"Runtime (mins)"`
	Year          int      `json:"Year"`
	Genres        []string `json:"Genres"`
	NumVotes      int      `json:"Num Votes"`
	ReleaseDate   string   `json:"Release Date"`
	Directors     []string `json:"Directors"`
}

// Movie struct represents a movie entry in the CSV
type MovieWatchlist struct {
	Const         string  `json:"ImdbId"`
	Created       string  `json:"Created"`
	Modified      string  `json:"Modified"`
	Description   string  `json:"Description"`
	Title         string  `json:"Title"`
	OriginalTitle string  `json:"Original Title"`
	URL           string  `json:"URL"`
	TitleType     string  `json:"Title Type"`
	IMDbRating    float64 `json:"IMDb Rating"`
	RuntimeMins   int     `json:"Runtime (mins)"`
	Year          int     `json:"Year"`
	Genres        []string
	NumVotes      int    `json:"Num Votes"`
	ReleaseDate   string `json:"Release Date"`
	Directors     []string
	YourRating    string `json:"Your Rating"`
	DateRated     string `json:"Date Rated"`
}

// imdbCmd represents the imdb command
var imdbCmd = &cobra.Command{
	Use:   "imdb",
	Short: "Parse IMDB export",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Processing imdb export...")
		parse_imdb()
	},
}

func init() {
	importCmd.AddCommand(imdbCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// imdbCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// imdbCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

func parse_imdb() {
	// Open the CSV file
	csvFile, err := os.Open("imdb_export.csv") // Replace "movies.csv" with your actual filename
	if err != nil {
		fmt.Println(err)
		return
	}
	defer csvFile.Close()

	// Create a new CSV reader
	reader := csv.NewReader(csvFile)
	reader.FieldsPerRecord = 14 // Imdb watched export has exactly 14 fields

	// Skip the header row (assuming the first row contains column names)
	_, err = reader.Read()
	if err != nil {
		fmt.Println(err)
		return
	}

	var movies []MovieSeen

	// Read each record from the CSV file
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			fmt.Println(err)
			continue
		}

		imdbRating, err := strconv.ParseFloat(record[7], 64)
		if err != nil {
			fmt.Printf("%s: Error parsing imdbRating %s: %v\n", record[0], record[7], err)
			imdbRating = 0.0
		}

		myRating, err := strconv.Atoi(record[1])
		if err != nil {
			fmt.Printf("Error parsing myRating: %v\n", err)
			myRating = 0
		}

		runtimeMins, err := strconv.Atoi(record[8])
		if err != nil {
			if record[8] != "" {
				fmt.Printf("%s: Error parsing runtime %s: %v\n", record[0], record[8], err)
			}
			runtimeMins = 0
		}

		year, err := strconv.Atoi(record[9])
		if err != nil {
			year = 0
			fmt.Printf("Error parsing year: %v\n", err)
		}

		numVotes, err := strconv.Atoi(record[11])
		if err != nil {
			fmt.Printf("Error parsing votes: %v\n", err)
			numVotes = 0
		}

		// Separate genres (assuming comma-separated)
		genres := strings.Split(record[10], ",")

		// Separate directors (assuming comma-separated)
		directors := strings.Split(record[13], ",")

		// Create a new Movie struct and append it to the slice
		movie := MovieSeen{
			ImdbId:        record[0],
			MyRating:      myRating,
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
		}

		// debug fmt.Printf("%v\n", movie)

		movies = append(movies, movie)
	}

	writeMovieToJson(movies)
	err = writeMoviesToMarkdown(movies, "markdown/imdb/")
	if err != nil {
		fmt.Printf("Error writing markdown: %v", err)
	}

	fmt.Printf("Processed %d movies\n", len(movies))
}

func writeMovieToJson(movies []MovieSeen) {
	// Convert the slice of movies to JSON
	jsonData, err := json.Marshal(movies)
	if err != nil {
		fmt.Println(err)
		return
	}

	// Open a new file for writing JSON data
	jsonFile, err := os.Create("movies.json") // Replace "movies.json" with your desired filename
	if err != nil {
		fmt.Println(err)
		return
	}
	defer jsonFile.Close()

	// Write the JSON data to the file
	_, err = jsonFile.Write(jsonData)
	if err != nil {
		fmt.Println(err)
	}
}

// writeMovieToMarkdown writes movie info to a markdown file
func writeMovieToMarkdown(movie MovieSeen, directory string) error {
	// Sanitize movie title for filename
	filename := sanitizeFilename(movie.Title) + ".md"
	filePath := filepath.Join(directory, filename)

	// Create markdown content
	var title string
	if movie.Title == movie.OriginalTitle {
		movie.Title = sanitizeTitle(movie.Title)
		title = fmt.Sprintf("title: %s\n", movie.Title)
	} else {
		movie.Title = sanitizeTitle(movie.Title)
		movie.OriginalTitle = sanitizeTitle(movie.OriginalTitle)
		title = fmt.Sprintf("title: %s\noriginal_title: %s\n", movie.Title, movie.OriginalTitle)
	}

	tags := []string{}
	tags = append(tags, mapTypeToTag(movie.TitleType))

	genreList := strings.Join(movie.Genres, "\n  - ")
	tagList := strings.Join(tags, "\n  - ")

	content := fmt.Sprintf("---\n%surl: %s\nyear: %d\nimdb_rating: %.2f\nmy_rating: %d\ndate_rated: %s\nruntime (min): %d\ngenres:\n  - %s\ntags:\n  - %s\n---\n\n",
		title, movie.URL, movie.Year, movie.IMDbRating, movie.MyRating, movie.DateRated, movie.RuntimeMins, genreList, tagList)

	// Create directory if it doesn't exist
	if err := os.MkdirAll(directory, 0755); err != nil {
		return err
	}

	// Write content to file
	return os.WriteFile(filePath, []byte(content), 0644)
}

func sanitizeTitle(title string) string {
	return strings.ReplaceAll(title, ":", "")
}

// writeMoviesToMarkdown writes a list of movies to markdown files
func writeMoviesToMarkdown(movies []MovieSeen, directory string) error {
	for _, movie := range movies {
		err := writeMovieToMarkdown(movie, directory)
		if err != nil {
			return err
		}
	}
	return nil
}

// mapTypeToTag maps a imdb title type to a markdown tag
func mapTypeToTag(titleType string) string {
	switch titleType {
	case "Video Game":
		return "imdb/videogame"
	case "TV Series":
		return "imdb/tv-series"
	case "TV Special":
		return "imdb/tv-special"
	case "TV Mini Series":
		return "imdb/miniseries"
	case "TV Episode":
		return "imdb/tv-episode"
	case "TV Movie":
		return "imdb/tv-movie"
	case "TV Short":
		return "imdb/tv-short"
	case "Movie":
		return "imdb/movie"
	case "Video":
		return "imdb/video"
	case "Short":
		return "imdb/short-movie"
	case "Podcast Series":
		return "imdb/podcast"
	case "Podcast Episode":
		return "imdb/podcast-episode"
	default:
		fmt.Printf("Unknown title type '%s'\n", titleType)
		return "UNKNOWN"
	}
}
