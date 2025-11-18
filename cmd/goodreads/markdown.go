package goodreads

import (
	"fmt"
	"log/slog"
	"os"
	"regexp"
	"strings"

	"github.com/lepinkainen/hermes/internal/config"
	"github.com/lepinkainen/hermes/internal/fileutil"
)

const defaultCoverWidth = 250

func writeBookToMarkdown(book Book, directory string) error {
	// Ensure the directory exists
	if err := os.MkdirAll(directory, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", directory, err)
	}

	filePath := fileutil.GetMarkdownFilePath(book.Title, directory)

	// Use the MarkdownBuilder to construct the document
	mb := fileutil.NewMarkdownBuilder()

	// Basic metadata
	mb.AddTitle(fileutil.SanitizeFilename(book.Title))
	mb.AddType("book")
	mb.AddField("goodreads_id", book.ID)

	if book.YearPublished > 0 {
		mb.AddYear(book.YearPublished)
	}
	if book.OriginalPublicationYear > 0 && book.OriginalPublicationYear != book.YearPublished {
		mb.AddField("original_year", book.OriginalPublicationYear)
	}

	// Ratings
	if book.MyRating > 0 {
		mb.AddField("my_rating", book.MyRating)
	}
	if book.AverageRating > 0 {
		mb.AddField("average_rating", book.AverageRating)
	}

	// Dates
	if book.DateRead != "" {
		mb.AddDate("date_read", book.DateRead)
	}
	if book.DateAdded != "" {
		mb.AddDate("date_added", book.DateAdded)
	}

	// Book details
	if book.NumberOfPages > 0 {
		mb.AddField("pages", book.NumberOfPages)
	}
	if book.Publisher != "" {
		mb.AddField("publisher", book.Publisher)
	}
	if book.Binding != "" {
		mb.AddField("binding", book.Binding)
	}

	// ISBNs
	if book.ISBN != "" {
		mb.AddField("isbn", book.ISBN)
	}
	if book.ISBN13 != "" {
		mb.AddField("isbn13", book.ISBN13)
	}

	// Authors as array
	if len(book.Authors) > 0 {
		mb.AddStringArray("authors", book.Authors)
	}

	// Bookshelves as array
	if len(book.Bookshelves) > 0 {
		mb.AddStringArray("bookshelves", book.Bookshelves)
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

	mb.AddTags(tags...)

	// Additional metadata from OpenLibrary
	if book.Description != "" {
		mb.AddField("description", "|\n  "+book.Description)
	}

	if len(book.Subjects) > 0 {
		mb.AddStringArray("subjects", book.Subjects)
	}

	// Handle cover - download locally and use Obsidian syntax
	var coverURL string
	if book.CoverURL != "" {
		coverURL = book.CoverURL
	} else if book.CoverID > 0 {
		coverURL = fmt.Sprintf("https://covers.openlibrary.org/b/id/%d-L.jpg", book.CoverID)
	}

	if coverURL != "" {
		coverFilename := fileutil.BuildCoverFilename(book.Title)
		result, err := fileutil.DownloadCover(fileutil.CoverDownloadOptions{
			URL:          coverURL,
			OutputDir:    directory,
			Filename:     coverFilename,
			UpdateCovers: config.UpdateCovers,
		})
		if err != nil {
			slog.Warn("Failed to download cover", "title", book.Title, "error", err)
			// Fall back to URL if download fails
			mb.AddField("cover", coverURL)
			mb.AddImage(coverURL)
		} else if result != nil {
			// Use local path in frontmatter
			mb.AddField("cover", result.RelativePath)
			mb.AddObsidianImage(result.Filename, defaultCoverWidth)
		}
	}

	if book.Subtitle != "" {
		mb.AddField("subtitle", book.Subtitle)
	}

	if len(book.SubjectPeople) > 0 {
		mb.AddStringArray("subject_people", book.SubjectPeople)
	}

	// Add review if exists
	if book.MyReview != "" {
		// Replace HTML line breaks with newlines and clean up multiple newlines
		review := strings.ReplaceAll(book.MyReview, "<br/>", "\n")
		review = strings.ReplaceAll(review, "<br>", "\n")
		// Clean up multiple newlines
		multipleNewlines := regexp.MustCompile(`\n{3,}`)
		review = multipleNewlines.ReplaceAllString(review, "\n\n")

		mb.AddParagraph("## Review")
		mb.AddParagraph(review)
	}

	// Add private notes in a callout if they exist
	if book.PrivateNotes != "" {
		mb.AddCallout("note", "Private Notes", book.PrivateNotes)
	}

	// Write content to file with overwrite logic
	written, err := fileutil.WriteFileWithOverwrite(filePath, []byte(mb.Build()), 0644, config.OverwriteFiles)
	if err != nil {
		return err
	}

	if !written {
		slog.Debug("Skipped existing file", "path", filePath)
	} else {
		slog.Info("Wrote file", "path", filePath)
	}

	return nil
}
