package goodreads

import (
	"fmt"
	"log/slog"
	"regexp"
	"strings"

	"github.com/lepinkainen/hermes/internal/config"
	"github.com/lepinkainen/hermes/internal/content"
	"github.com/lepinkainen/hermes/internal/fileutil"
	"github.com/lepinkainen/hermes/internal/obsidian"
)

const defaultCoverWidth = 250

func writeBookToMarkdown(book Book, directory string) error {
	filePath := fileutil.GetMarkdownFilePath(book.Title, directory)

	// Create frontmatter using obsidian.Frontmatter
	fm := obsidian.NewFrontmatter()

	// Basic metadata
	fm.Set("title", fileutil.SanitizeFilename(book.Title))
	fm.Set("type", "book")
	fm.Set("goodreads_id", fmt.Sprintf("%d", book.ID))

	if book.YearPublished > 0 {
		fm.Set("year", book.YearPublished)
	}
	if book.OriginalPublicationYear > 0 && book.OriginalPublicationYear != book.YearPublished {
		fm.Set("original_year", book.OriginalPublicationYear)
	}

	// Ratings
	if book.MyRating > 0 {
		fm.Set("my_rating", book.MyRating)
	}
	if book.AverageRating > 0 {
		fm.Set("average_rating", book.AverageRating)
	}

	// Dates
	if book.DateRead != "" {
		fm.Set("date_read", book.DateRead)
	}
	if book.DateAdded != "" {
		fm.Set("date_added", book.DateAdded)
	}

	// Book details
	if book.NumberOfPages > 0 {
		fm.Set("pages", book.NumberOfPages)
	}
	if book.Publisher != "" {
		fm.Set("publisher", book.Publisher)
	}
	if book.Binding != "" {
		fm.Set("binding", book.Binding)
	}

	// ISBNs
	if book.ISBN != "" {
		fm.Set("isbn", book.ISBN)
	}
	if book.ISBN13 != "" {
		fm.Set("isbn13", book.ISBN13)
	}

	// Authors as array
	if len(book.Authors) > 0 {
		fm.Set("authors", book.Authors)
	}

	// Bookshelves as array
	if len(book.Bookshelves) > 0 {
		fm.Set("bookshelves", book.Bookshelves)
	}

	// Additional metadata from OpenLibrary
	if book.Description != "" {
		fm.Set("description", book.Description)
	}

	if len(book.Subjects) > 0 {
		fm.Set("subjects", book.Subjects)
	}

	if book.Subtitle != "" {
		fm.Set("subtitle", book.Subtitle)
	}

	if len(book.SubjectPeople) > 0 {
		fm.Set("subject_people", book.SubjectPeople)
	}

	// Collect all tags using TagSet for deduplication and normalization
	tc := obsidian.NewTagSet()
	tc.Add("goodreads/book")

	// Add rating tag
	tc.AddIf(book.MyRating > 0, fmt.Sprintf("rating/%.0f", book.MyRating))

	// Add decade tag if we have a year
	if book.YearPublished > 0 {
		decade := (book.YearPublished / 10) * 10
		tc.AddFormat("year/%ds", decade)
	}

	// Add shelf tag
	tc.AddIf(book.ExclusiveShelf != "", fmt.Sprintf("shelf/%s", book.ExclusiveShelf))

	fm.Set("tags", tc.GetSorted())

	// Build body content
	var body strings.Builder
	coverFilename := ""

	// Handle cover - download locally and use Obsidian syntax
	var coverURL string
	if book.CoverURL != "" {
		coverURL = book.CoverURL
	} else if book.CoverID > 0 {
		coverURL = fmt.Sprintf("https://covers.openlibrary.org/b/id/%d-L.jpg", book.CoverID)
	}

	if coverURL != "" {
		coverFilenameBase := fileutil.BuildCoverFilename(book.Title)
		result, err := fileutil.DownloadCover(fileutil.CoverDownloadOptions{
			URL:          coverURL,
			OutputDir:    directory,
			Filename:     coverFilenameBase,
			UpdateCovers: config.UpdateCovers,
		})
		if err != nil {
			slog.Warn("Failed to download cover", "title", book.Title, "error", err)
			// Fall back to URL if download fails
			fm.Set("cover", coverURL)
			body.WriteString(fmt.Sprintf("![](%s)\n\n", coverURL))
		} else if result != nil {
			// Use local path in frontmatter
			fm.Set("cover", result.RelativePath)
			coverFilename = result.Filename
			body.WriteString(fmt.Sprintf("![[%s|%d]]\n\n", coverFilename, defaultCoverWidth))
		}
	}

	// Build Goodreads content sections wrapped in markers
	goodreadsDetails := &content.GoodreadsBookDetails{
		Title:                   book.Title,
		Subtitle:                book.Subtitle,
		Authors:                 book.Authors,
		Publisher:               book.Publisher,
		Pages:                   book.NumberOfPages,
		YearPublished:           book.YearPublished,
		OriginalPublicationYear: book.OriginalPublicationYear,
		MyRating:                book.MyRating,
		AverageRating:           book.AverageRating,
		ISBN:                    book.ISBN,
		ISBN13:                  book.ISBN13,
		Binding:                 book.Binding,
		GoodreadsID:             fmt.Sprintf("%d", book.ID),
		Description:             book.Description,
		Subjects:                book.Subjects,
		SubjectPeople:           book.SubjectPeople,
	}

	goodreadsContent := content.BuildGoodreadsContent(goodreadsDetails, []string{"info", "description", "subjects"})
	if goodreadsContent != "" {
		wrappedGoodreads := content.WrapWithGoodreadsMarkers(goodreadsContent)
		body.WriteString(wrappedGoodreads)
		body.WriteString("\n\n")
	}

	// Add review if exists (outside markers - user content)
	if book.MyReview != "" {
		// Replace HTML line breaks with newlines and clean up multiple newlines
		review := strings.ReplaceAll(book.MyReview, "<br/>", "\n")
		review = strings.ReplaceAll(review, "<br>", "\n")
		// Clean up multiple newlines
		multipleNewlines := regexp.MustCompile(`\n{3,}`)
		review = multipleNewlines.ReplaceAllString(review, "\n\n")

		body.WriteString("## Review\n\n")
		body.WriteString(review)
		body.WriteString("\n\n")
	}

	// Add private notes in a callout if they exist (outside markers - user content)
	if book.PrivateNotes != "" {
		body.WriteString(">[!note]- Private Notes\n")
		// Split by lines and add "> " prefix to each
		noteLines := strings.Split(book.PrivateNotes, "\n")
		for _, line := range noteLines {
			body.WriteString("> ")
			body.WriteString(line)
			body.WriteString("\n")
		}
	}

	// Create the note
	note := &obsidian.Note{
		Frontmatter: fm,
		Body:        strings.TrimSpace(body.String()),
	}

	// Build markdown
	markdown, err := note.Build()
	if err != nil {
		return fmt.Errorf("failed to build markdown: %w", err)
	}

	// Add trailing newlines to match expected format
	content := append(markdown, []byte("\n\n\n")...)

	// Write content to file with overwrite logic
	written, err := fileutil.WriteFileWithOverwrite(filePath, content, 0644, config.OverwriteFiles)
	if err != nil {
		return err
	}

	fileutil.LogFileWriteResult(written, filePath)

	return nil
}
