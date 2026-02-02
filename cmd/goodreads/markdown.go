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

type coverContent struct {
	frontmatterValue string
	embed            string
}

func buildBookFrontmatter(book Book) *obsidian.Frontmatter {
	fm := obsidian.NewFrontmatterWithTitle(fileutil.SanitizeFilename(book.Title))

	// Basic metadata
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

	obsidian.ApplyTagSet(fm, tc)

	return fm
}

func buildCoverContent(book Book, directory string) coverContent {
	var coverURL string
	if book.CoverURL != "" {
		coverURL = book.CoverURL
	} else if book.CoverID > 0 {
		coverURL = fmt.Sprintf("https://covers.openlibrary.org/b/id/%d-L.jpg", book.CoverID)
	}

	if coverURL == "" {
		return coverContent{}
	}

	coverFilenameBase := fileutil.BuildCoverFilename(book.Title)
	result, err := fileutil.DownloadCover(fileutil.CoverDownloadOptions{
		URL:          coverURL,
		OutputDir:    directory,
		Filename:     coverFilenameBase,
		UpdateCovers: config.UpdateCovers,
	})
	if err != nil {
		slog.Warn("Failed to download cover", "title", book.Title, "error", err)
		return coverContent{
			frontmatterValue: coverURL,
			embed:            fmt.Sprintf("![](%s)\n\n", coverURL),
		}
	}

	if result == nil {
		return coverContent{}
	}

	return coverContent{
		frontmatterValue: result.RelativePath,
		embed:            fmt.Sprintf("![[%s|%d]]\n\n", result.Filename, defaultCoverWidth),
	}
}

func buildGoodreadsSection(book Book) string {
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
	if goodreadsContent == "" {
		return ""
	}

	wrappedGoodreads := content.WrapWithGoodreadsMarkers(goodreadsContent)
	return wrappedGoodreads + "\n\n"
}

func formatReview(review string) string {
	review = strings.ReplaceAll(review, "<br/>", "\n")
	review = strings.ReplaceAll(review, "<br>", "\n")

	multipleNewlines := regexp.MustCompile(`\n{3,}`)
	return multipleNewlines.ReplaceAllString(review, "\n\n")
}

func buildReviewSection(review string) string {
	if review == "" {
		return ""
	}

	cleaned := formatReview(review)
	return "## Review\n\n" + cleaned + "\n\n"
}

func buildPrivateNotesSection(notes string) string {
	if notes == "" {
		return ""
	}

	var body strings.Builder
	body.WriteString(">[!note]- Private Notes\n")

	for _, line := range strings.Split(notes, "\n") {
		body.WriteString("> ")
		body.WriteString(line)
		body.WriteString("\n")
	}

	return body.String()
}

func buildBookBody(book Book, directory string, fm *obsidian.Frontmatter) string {
	var body strings.Builder

	cover := buildCoverContent(book, directory)
	if cover.frontmatterValue != "" {
		fm.Set("cover", cover.frontmatterValue)
		body.WriteString(cover.embed)
	}

	body.WriteString(buildGoodreadsSection(book))
	body.WriteString(buildReviewSection(book.MyReview))
	body.WriteString(buildPrivateNotesSection(book.PrivateNotes))

	return strings.TrimSpace(body.String())
}

func writeBookToMarkdown(book Book, directory string) error {
	filePath := fileutil.GetMarkdownFilePath(book.Title, directory)

	fm := buildBookFrontmatter(book)
	body := buildBookBody(book, directory, fm)

	markdown, err := obsidian.BuildNoteMarkdown(fm, body)
	if err != nil {
		return fmt.Errorf("failed to build markdown: %w", err)
	}

	content := append(markdown, []byte("\n\n\n")...)

	written, err := fileutil.WriteFileWithOverwrite(filePath, content, 0644, config.OverwriteFiles)
	if err != nil {
		return err
	}

	fileutil.LogFileWriteResult(written, filePath)

	return nil
}
