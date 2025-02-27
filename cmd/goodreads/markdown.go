package goodreads

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/lepinkainen/hermes/internal/config"
	"github.com/lepinkainen/hermes/internal/fileutil"
	log "github.com/sirupsen/logrus"
)

func writeBookToMarkdown(book Book, directory string) error {
	// Ensure the directory exists
	if err := os.MkdirAll(directory, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", directory, err)
	}

	filePath := fileutil.GetMarkdownFilePath(book.Title, directory)

	var frontmatter strings.Builder
	frontmatter.WriteString("---\n")

	// Basic metadata
	frontmatter.WriteString("title: \"" + sanitizeGoodreadsTitle(book.Title) + "\"\n")
	frontmatter.WriteString("type: book\n")
	frontmatter.WriteString("goodreads_id: " + fmt.Sprintf("%d", book.ID) + "\n")

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

	// Additional metadata from OpenLibrary
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
		content.WriteString(fmt.Sprintf("> [!note]- Private Notes\n> %s\n", book.PrivateNotes))
	}

	// Write content to file with overwrite logic
	written, err := fileutil.WriteFileWithOverwrite(filePath, []byte(frontmatter.String()+content.String()), 0644, config.OverwriteFiles)
	if err != nil {
		return err
	}

	if !written {
		log.Debugf("Skipped existing file: %s", filePath)
	}

	return nil
}

// Helper function to sanitize Goodreads title
func sanitizeGoodreadsTitle(title string) string {
	return strings.ReplaceAll(title, ":", "")
}
