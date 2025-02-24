package cmd

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

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
	Description              string   `json:"Description"`
	Subjects                 []string `json:"Subjects"`
	CoverID                  int      `json:"Cover ID"`
	CoverURL                 string   `json:"Cover URL"`
	SubjectPeople            []string `json:"Subject People"`
	Subtitle                 string   `json:"Subtitle"`
}

// OpenLibraryBook struct represents the response from the OpenLibrary API
type OpenLibraryBook struct {
	Description   interface{}   `json:"description"`
	Covers        []int         `json:"covers"`
	Subjects      []interface{} `json:"subjects"`
	SubjectPeople []interface{} `json:"subject_people"`
	Title         string        `json:"title"`
	Subtitle      string        `json:"subtitle"`
	Publishers    []struct {
		Name string `json:"name"`
	} `json:"publishers"`
	Cover struct {
		Small  string `json:"small"`
		Medium string `json:"medium"`
		Large  string `json:"large"`
	} `json:"cover"`
}

// OpenLibraryEdition struct represents the response from the OpenLibrary API
type OpenLibraryEdition struct {
	Publishers      []string               `json:"publishers"`
	Number_of_pages int                    `json:"number_of_pages"`
	Cover           map[string]interface{} `json:"cover"`
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

func writeBookToMarkdown(book Book, directory string) error {
	// Sanitize book title for filename
	filename := sanitizeFilename(book.Title) + ".md"
	filePath := filepath.Join(directory, filename)

	var frontmatter strings.Builder
	frontmatter.WriteString("---\n")

	// Basic metadata
	frontmatter.WriteString("title: \"" + sanitizeGoodreadsTitle(book.Title) + "\"\n")
	frontmatter.WriteString("type: book\n")
	frontmatter.WriteString("goodreads_id: " + strconv.Itoa(book.ID) + "\n")

	if book.YearPublished > 0 {
		frontmatter.WriteString(fmt.Sprintf("year: %d\n", book.YearPublished))
	}
	if book.OriginalPublicationYear > 0 && book.OriginalPublicationYear != book.YearPublished {
		frontmatter.WriteString(fmt.Sprintf("original_year: %d\n", book.OriginalPublicationYear))
	}

	// Ratings
	if book.MyRating > 0 {
		frontmatter.WriteString(fmt.Sprintf("my_rating: %.1f\n", book.MyRating))
	}
	if book.AverageRating > 0 {
		frontmatter.WriteString(fmt.Sprintf("average_rating: %.1f\n", book.AverageRating))
	}

	// Dates
	if book.DateRead != "" {
		frontmatter.WriteString(fmt.Sprintf("date_read: %s\n", book.DateRead))
	}
	if book.DateAdded != "" {
		frontmatter.WriteString(fmt.Sprintf("date_added: %s\n", book.DateAdded))
	}

	// Book details
	if book.NumberOfPages > 0 {
		frontmatter.WriteString(fmt.Sprintf("pages: %d\n", book.NumberOfPages))
	}
	if book.Publisher != "" {
		frontmatter.WriteString(fmt.Sprintf("publisher: \"%s\"\n", book.Publisher))
	}
	if book.Binding != "" {
		frontmatter.WriteString(fmt.Sprintf("binding: \"%s\"\n", book.Binding))
	}

	// ISBNs
	if book.ISBN != "" {
		frontmatter.WriteString(fmt.Sprintf("isbn: \"%s\"\n", book.ISBN))
	}
	if book.ISBN13 != "" {
		frontmatter.WriteString(fmt.Sprintf("isbn13: \"%s\"\n", book.ISBN13))
	}

	// Authors as array
	if len(book.Authors) > 0 {
		frontmatter.WriteString("authors:\n")
		for _, author := range book.Authors {
			if author != "" {
				frontmatter.WriteString(fmt.Sprintf("  - \"%s\"\n", strings.TrimSpace(author)))
			}
		}
	}

	// Bookshelves as array
	if len(book.Bookshelves) > 0 {
		frontmatter.WriteString("bookshelves:\n")
		for _, shelf := range book.Bookshelves {
			if shelf != "" {
				frontmatter.WriteString(fmt.Sprintf("  - %s\n", strings.TrimSpace(shelf)))
			}
		}
	}

	// Tags
	tags := []string{
		"goodreads/book",
	}

	// Add rating tag
	if book.MyRating > 0 {
		tags = append(tags, fmt.Sprintf("rating/%.0f", book.MyRating))
	}

	// Add decade tag if we have a year
	if book.YearPublished > 0 {
		decade := (book.YearPublished / 10) * 10
		tags = append(tags, fmt.Sprintf("year/%ds", decade))
	}

	// Add shelf tag
	if book.ExclusiveShelf != "" {
		tags = append(tags, fmt.Sprintf("shelf/%s", book.ExclusiveShelf))
	}

	frontmatter.WriteString("tags:\n")
	for _, tag := range tags {
		frontmatter.WriteString(fmt.Sprintf("  - %s\n", tag))
	}

	// Add new frontmatter fields
	if book.Description != "" {
		frontmatter.WriteString(fmt.Sprintf("description: |\n  %s\n", book.Description))
	}

	if len(book.Subjects) > 0 {
		frontmatter.WriteString("subjects:\n")
		for _, subject := range book.Subjects {
			frontmatter.WriteString(fmt.Sprintf("  - \"%s\"\n", subject))
		}
	}

	if book.CoverURL != "" {
		frontmatter.WriteString(fmt.Sprintf("cover_url: \"%s\"\n", book.CoverURL))
	} else if book.CoverID > 0 {
		frontmatter.WriteString(fmt.Sprintf("cover_url: \"https://covers.openlibrary.org/b/id/%d-L.jpg\"\n", book.CoverID))
	}

	if book.Subtitle != "" {
		frontmatter.WriteString(fmt.Sprintf("subtitle: \"%s\"\n", book.Subtitle))
	}

	if len(book.SubjectPeople) > 0 {
		frontmatter.WriteString("subject_people:\n")
		for _, person := range book.SubjectPeople {
			frontmatter.WriteString(fmt.Sprintf("  - \"%s\"\n", person))
		}
	}

	frontmatter.WriteString("---\n\n")

	// Content section
	var content strings.Builder

	// Add review if exists
	if book.MyReview != "" {
		content.WriteString("## Review\n\n")
		// Replace HTML line breaks with newlines and clean up multiple newlines
		review := strings.ReplaceAll(book.MyReview, "<br/>", "\n")
		review = strings.ReplaceAll(review, "<br>", "\n")
		// Clean up multiple newlines
		multipleNewlines := regexp.MustCompile(`\n{3,}`)
		review = multipleNewlines.ReplaceAllString(review, "\n\n")
		content.WriteString(review + "\n\n")
	}

	// Add private notes in a callout if they exist
	if book.PrivateNotes != "" {
		content.WriteString(fmt.Sprintf(">[!note]- Private Notes\n> %s\n\n", book.PrivateNotes))
	}

	// Create directory if it doesn't exist
	if err := os.MkdirAll(directory, 0755); err != nil {
		return err
	}

	// Write content to file
	return os.WriteFile(filePath, []byte(frontmatter.String()+content.String()), 0644)
}

func sanitizeGoodreadsTitle(title string) string {
	return strings.ReplaceAll(title, ":", "")
}

func sanitizeFilename(filename string) string {
	// Replace invalid filename characters
	invalid := regexp.MustCompile(`[<>:"/\\|?*]`)
	safe := invalid.ReplaceAllString(filename, "-")
	// Remove multiple dashes
	multiDash := regexp.MustCompile(`-+`)
	safe = multiDash.ReplaceAllString(safe, "-")
	// Trim spaces and dashes from ends
	return strings.Trim(safe, " -")
}

// Add this helper function to handle the description field
func getDescription(desc interface{}) string {
	switch v := desc.(type) {
	case string:
		return v
	case map[string]interface{}:
		if value, ok := v["value"].(string); ok {
			return value
		}
	}
	return ""
}

// Add this helper function to handle subjects
func getSubjects(subjects []interface{}) []string {
	result := make([]string, 0)
	for _, subject := range subjects {
		switch v := subject.(type) {
		case string:
			result = append(result, v)
		case map[string]interface{}:
			if name, ok := v["name"].(string); ok {
				result = append(result, name)
			}
		}
	}
	return result
}

// Add helper function for subject people
func getSubjectPeople(subjects []interface{}) []string {
	result := make([]string, 0)
	for _, subject := range subjects {
		switch v := subject.(type) {
		case string:
			result = append(result, v)
		case map[string]interface{}:
			if name, ok := v["name"].(string); ok {
				result = append(result, name)
			}
		}
	}
	return result
}

func enrichBookFromOpenLibrary(book *Book) error {
	// Create an HTTP client with timeout
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	// Try ISBN13 first, then ISBN
	isbn := book.ISBN13
	if isbn == "" {
		isbn = book.ISBN
	}

	if isbn == "" {
		return fmt.Errorf("no ISBN available")
	}

	// Query OpenLibrary API
	url := fmt.Sprintf("https://openlibrary.org/api/books?bibkeys=ISBN:%s&format=json&jscmd=data", isbn)
	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var result map[string]OpenLibraryBook
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return err
	}

	// Check if we got an empty response
	if len(result) == 0 {
		return fmt.Errorf("no data found in OpenLibrary for ISBN: %s", isbn)
	}

	// Update book struct with OpenLibrary data
	if olBook, ok := result["ISBN:"+isbn]; ok {
		if olBook.Description != nil {
			book.Description = getDescription(olBook.Description)
		}
		if len(olBook.Subjects) > 0 {
			book.Subjects = getSubjects(olBook.Subjects)
		}
		if len(olBook.SubjectPeople) > 0 {
			book.SubjectPeople = getSubjectPeople(olBook.SubjectPeople)
		}
		if len(olBook.Covers) > 0 {
			book.CoverID = olBook.Covers[0]
		}
		if olBook.Cover.Large != "" {
			book.CoverURL = olBook.Cover.Large
		}
		if olBook.Subtitle != "" {
			book.Subtitle = olBook.Subtitle
		}
	}

	fmt.Printf("Enriched book %s with OpenLibrary data\n", book.Title)

	return nil
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
			ID:                       bookID,
			Title:                    record[1],
			Authors:                  authors,
			ISBN:                     isbn,
			ISBN13:                   isbn13,
			MyRating:                 myRating,
			AverageRating:            averageRating,
			Publisher:                record[10],
			Binding:                  record[11],
			NumberOfPages:            numberOfPages,
			YearPublished:            yearPublished,
			OriginalPublicationYear:  originalPublicationYear,
			DateRead:                 record[14],
			DateAdded:                record[15],
			Bookshelves:              splitString(record[16]),
			BookshelvesWithPositions: splitString(record[17]),
			ExclusiveShelf:           record[17],
			MyReview:                 record[18],
			Spoiler:                  record[19],
			PrivateNotes:             record[20],
			ReadCount:                readCount,
			OwnedCopies:              ownedCopies,
		}

		books = append(books, book)
	}

	// Create output directory
	outputDir := "markdown/goodreads"
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		fmt.Println(err)
		return
	}

	var processedCount int

	// Process each book individually
	for i := range books {
		// Try to enrich the book with OpenLibrary data
		if err := enrichBookFromOpenLibrary(&books[i]); err != nil {
			fmt.Printf("Warning: Could not enrich book %s: %v\n", books[i].Title, err)
		} else {
			// Add a small delay to be nice to the API, but only if enrichment succeeded
			time.Sleep(100 * time.Millisecond)
		}

		// Write the book to markdown immediately after processing
		if err := writeBookToMarkdown(books[i], outputDir); err != nil {
			fmt.Printf("Error writing markdown for book %s: %v\n", books[i].Title, err)
			continue
		}

		processedCount++
		if processedCount%10 == 0 {
			fmt.Printf("Processed %d books...\n", processedCount)
		}
	}

	fmt.Printf("Successfully processed %d books\n", processedCount)
}
