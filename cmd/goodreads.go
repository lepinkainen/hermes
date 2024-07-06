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
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

// Book struct represents a book entry in the CSV
type Book struct {
	ID                       int      `json:"Book Id"`
	Title                    string   `json:"Title"`
	Authors                  []string `json:"Authors"`
	ISBN                     string   `json:"ISBN"`
	ISBN13                   string   `json:"ISBN13"`
	MyRating                 float64  `json:"My Rating"`
	AverageRating            float64  `json:"Average Rating"`
	Publisher                string   `json:"Publisher"`
	Binding                  string   `json:"Binding"`
	NumberOfPages            int      `json:"Number of Pages"`
	YearPublished            int      `json:"Year Published"`
	OriginalPublicationYear  int      `json:"Original Publication Year"`
	DateRead                 string   `json:"Date Read"`
	DateAdded                string   `json:"Date Added"`
	Bookshelves              []string `json:"Bookshelves"`
	BookshelvesWithPositions []string `json:"Bookshelves with positions"`
	ExclusiveShelf           string   `json:"Exclusive Shelf"`
	MyReview                 string   `json:"My Review"`
	Spoiler                  string   `json:"Spoiler"`
	PrivateNotes             string   `json:"Private Notes"`
	ReadCount                int      `json:"Read Count"`
	OwnedCopies              int      `json:"Owned Copies"`
}

// goodreadsCmd represents the goodreads command
var goodreadsCmd = &cobra.Command{
	Use:   "goodreads",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("processing goodreads export...")
		parse_goodreads()
	},
}

func init() {
	importCmd.AddCommand(goodreadsCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// goodreadsCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// goodreadsCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

// Helper function to split comma-separated strings
func splitString(str string) []string {
	return strings.Split(str, ",")
}

func parse_goodreads() {
	// Open the CSV file
	csvFile, err := os.Open("goodreads_library_export.csv") // Replace "books.csv" with your actual filename
	if err != nil {
		fmt.Println(err)
		return
	}
	defer csvFile.Close()

	// Create a new CSV reader
	reader := csv.NewReader(csvFile)

	// Skip the header row (assuming the first row contains column names)
	_, err = reader.Read()
	if err != nil {
		fmt.Println(err)
		return
	}

	var books []Book

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

		// Convert string values to appropriate types
		bookID, err := strconv.Atoi(record[0])
		if err != nil {
			fmt.Println(err)
			continue
		}

		myRating, err := strconv.ParseFloat(record[8], 64)
		if err != nil {
			myRating = 0.0
		}

		averageRating, err := strconv.ParseFloat(record[9], 64)
		if err != nil {
			averageRating = 0.0
		}

		numberOfPages, err := strconv.Atoi(record[12])
		if err != nil {
			numberOfPages = 0
		}

		yearPublished, err := strconv.Atoi(record[13])
		if err != nil {
			yearPublished = 0
		}

		originalPublicationYear, err := strconv.Atoi(record[14])
		if err != nil {
			originalPublicationYear = 0
		}

		readCount, err := strconv.Atoi(record[20])
		if err != nil {
			readCount = 0
		}

		ownedCopies, err := strconv.Atoi(record[21])
		if err != nil {
			ownedCopies = 0
		}

		// Remove unnecessary quotes from ISBN and ISBN13 (if present)
		isbn := strings.TrimPrefix(strings.TrimSuffix(record[5], "\""), "=\"")
		isbn13 := strings.TrimPrefix(strings.TrimSuffix(record[6], "\""), "=\"")

		// Separate authors (assuming comma-separated)
		authors := splitString(record[2])

		// Create a new Book struct and append it to the slice
		book := Book{
			ID:                      bookID,
			Title:                   record[1],
			Authors:                 authors,
			ISBN:                    isbn,
			ISBN13:                  isbn13,
			MyRating:                myRating,
			AverageRating:           averageRating,
			Publisher:               record[10],
			Binding:                 record[11],
			NumberOfPages:           numberOfPages,
			YearPublished:           yearPublished,
			OriginalPublicationYear: originalPublicationYear,
			DateRead:                record[15],
			DateAdded:               record[16],
			Bookshelves:             splitString(record[17]),

			BookshelvesWithPositions: splitString(record[18]),
			ExclusiveShelf:           record[19],
			MyReview:                 record[20],
			Spoiler:                  record[21],
			PrivateNotes:             record[22],
			ReadCount:                readCount,
			OwnedCopies:              ownedCopies,
		}

		books = append(books, book)
	}

	// Convert the slice of books to JSON
	jsonData, err := json.Marshal(books)
	if err != nil {
		fmt.Println(err)
		return
	}

	// Open a new file for writing JSON data
	jsonFile, err := os.Create("goodreads.json") // Replace "books.json" with your desired output filename
	if err != nil {
		fmt.Println(err)
		return
	}
	defer jsonFile.Close()

	// Write the JSON data to the file
	jsonFile.Write(jsonData)

	fmt.Printf("Processed %d books\n", len(books))
}
