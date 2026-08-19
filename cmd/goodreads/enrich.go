package goodreads

import (
	"context"

	bookpkg "github.com/lepinkainen/hermes/internal/enrichment/book"
	"github.com/lepinkainen/hermes/internal/enrichment/book/enrichers"
)

// enrichBookWithEnrichers uses the new enricher system to enrich a book.
// It runs all configured enrichers and merges the results by priority.
func enrichBookWithEnrichers(ctx context.Context, book *Book) {
	searchISBN := book.ISBN13
	if searchISBN == "" {
		searchISBN = book.ISBN
	}

	if searchISBN == "" {
		return
	}

	merged, err := bookpkg.RunEnrichers(ctx, searchISBN, enrichers.Default())
	if err != nil || merged == nil {
		return
	}

	applyEnrichmentData(book, merged)
}

// applyEnrichmentData applies enrichment data to a book, only filling empty fields.
func applyEnrichmentData(book *Book, data *bookpkg.EnrichmentData) {
	if data == nil {
		return
	}

	if book.Description == "" && data.Description != nil {
		book.Description = *data.Description
	}

	if book.Subtitle == "" && data.Subtitle != nil {
		book.Subtitle = *data.Subtitle
	}

	if book.Publisher == "" && data.Publisher != nil {
		book.Publisher = *data.Publisher
	}

	if book.NumberOfPages == 0 && data.NumberOfPages != nil {
		book.NumberOfPages = *data.NumberOfPages
	}

	if book.CoverURL == "" && data.CoverURL != nil {
		book.CoverURL = *data.CoverURL
	}

	if len(book.Subjects) == 0 && len(data.Subjects) > 0 {
		book.Subjects = data.Subjects
	}

	if len(book.SubjectPeople) == 0 && len(data.SubjectPeople) > 0 {
		book.SubjectPeople = data.SubjectPeople
	}

	if len(book.Authors) == 0 && len(data.Authors) > 0 {
		book.Authors = data.Authors
	}
}
