package enrichment

import (
	"context"
	"log/slog"

	"github.com/lepinkainen/hermes/internal/content"
	"github.com/lepinkainen/hermes/internal/enrichment/book"
	"github.com/lepinkainen/hermes/internal/enrichment/book/enrichers"
	"github.com/lepinkainen/hermes/internal/fileutil"
)

// runBookEnrichers is a seam so tests can stub out the enricher pipeline.
var runBookEnrichers = book.RunEnrichers

// defaultBookContentSections are the content sections generated when
// BookEnrichmentOptions.ContentSections is empty.
var defaultBookContentSections = []string{"info", "description", "subjects"}

// EnrichFromBook enriches a book with data merged from the configured book
// enrichers (OpenLibrary, Google Books, BookBrainz, etc.). It resolves an
// ISBN via search when neither isbn nor isbn13 is provided, optionally shows
// a TUI for selection, downloads the cover, and generates content.
//
// Returns (nil, nil) when no ISBN could be resolved or no enricher returned
// data — this mirrors the not-found semantics of the underlying enrichers.
func EnrichFromBook(ctx context.Context, title, isbn, isbn13 string, opts BookEnrichmentOptions) (*BookEnrichment, error) {
	resolvedISBN, resolvedISBN13 := isbn, isbn13

	searchISBN := isbn13
	if searchISBN == "" {
		searchISBN = isbn
	}

	if searchISBN == "" {
		foundISBN, foundISBN13, err := resolveBookISBN(ctx, title, opts)
		if err != nil {
			return nil, err
		}
		if foundISBN == "" && foundISBN13 == "" {
			slog.Debug("No ISBN found for book", "title", title)
			return nil, nil
		}
		resolvedISBN, resolvedISBN13 = foundISBN, foundISBN13

		searchISBN = resolvedISBN13
		if searchISBN == "" {
			searchISBN = resolvedISBN
		}
	}

	merged, err := runBookEnrichers(ctx, searchISBN, enrichers.Default())
	if err != nil {
		return nil, err
	}
	if merged == nil {
		slog.Debug("No enrichment data found for book", "isbn", searchISBN, "title", title)
		return nil, nil
	}

	enrichment := mapBookEnrichmentData(merged, resolvedISBN, resolvedISBN13)

	if opts.DownloadCover && merged.CoverURL != nil && *merged.CoverURL != "" {
		result, coverErr := fileutil.DownloadCover(ctx, fileutil.CoverDownloadOptions{
			URL:          *merged.CoverURL,
			OutputDir:    opts.NoteDir,
			Filename:     fileutil.BuildCoverFilename(title),
			UpdateCovers: opts.Force,
		})
		if coverErr != nil {
			slog.Warn("Failed to download book cover", "title", title, "error", coverErr)
		} else if result != nil {
			enrichment.CoverPath = result.RelativePath
			enrichment.CoverFilename = result.Filename
		}
	}

	if opts.GenerateContent {
		sections := opts.ContentSections
		if len(sections) == 0 {
			sections = defaultBookContentSections
		}
		details := buildBookDetails(title, opts.BaseDetails, enrichment)
		enrichment.ContentMarkdown = content.BuildGoodreadsContent(details, sections)
	}

	return enrichment, nil
}

// mapBookEnrichmentData maps the merged enrichment data (from the book
// enrichers pipeline) onto a BookEnrichment, nil-checking pointer fields.
func mapBookEnrichmentData(merged *book.EnrichmentData, isbn, isbn13 string) *BookEnrichment {
	enrichment := &BookEnrichment{
		ISBN:          isbn,
		ISBN13:        isbn13,
		Subjects:      merged.Subjects,
		SubjectPeople: merged.SubjectPeople,
		Authors:       merged.Authors,
	}

	if merged.Description != nil {
		enrichment.Description = *merged.Description
	}
	if merged.Subtitle != nil {
		enrichment.Subtitle = *merged.Subtitle
	}
	if merged.Publisher != nil {
		enrichment.Publisher = *merged.Publisher
	}
	if merged.PublishDate != nil {
		enrichment.PublishDate = *merged.PublishDate
	}
	if merged.NumberOfPages != nil {
		enrichment.Pages = *merged.NumberOfPages
	}

	return enrichment
}

// buildBookDetails clones base (or starts from a zero value when nil) and
// fills only the fields that are still empty using data gathered from
// enrichment. Note-side data always wins over enrichment data.
func buildBookDetails(title string, base *content.GoodreadsBookDetails, enrichment *BookEnrichment) *content.GoodreadsBookDetails {
	var details content.GoodreadsBookDetails
	if base != nil {
		details = *base
	}

	if details.Title == "" {
		details.Title = title
	}
	if details.Subtitle == "" {
		details.Subtitle = enrichment.Subtitle
	}
	if details.Description == "" {
		details.Description = enrichment.Description
	}
	if details.Publisher == "" {
		details.Publisher = enrichment.Publisher
	}
	if details.Pages == 0 {
		details.Pages = enrichment.Pages
	}
	if len(details.Authors) == 0 {
		details.Authors = enrichment.Authors
	}
	if len(details.Subjects) == 0 {
		details.Subjects = enrichment.Subjects
	}
	if len(details.SubjectPeople) == 0 {
		details.SubjectPeople = enrichment.SubjectPeople
	}
	if details.ISBN == "" {
		details.ISBN = enrichment.ISBN
	}
	if details.ISBN13 == "" {
		details.ISBN13 = enrichment.ISBN13
	}

	return &details
}
