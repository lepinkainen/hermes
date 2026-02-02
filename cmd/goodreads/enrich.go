package goodreads

import (
	"context"
	"log/slog"
	"sync"

	"github.com/lepinkainen/hermes/cmd/goodreads/enrichers"
	bookpkg "github.com/lepinkainen/hermes/internal/enrichment/book"
)

var (
	defaultEnrichers []bookpkg.Enricher
	enrichersOnce    sync.Once
	defaultMerger    bookpkg.Merger
)

// getDefaultEnrichers returns the list of configured book enrichers.
// ISBNdb is included only if the API key is configured.
func getDefaultEnrichers() []bookpkg.Enricher {
	enrichersOnce.Do(func() {
		defaultEnrichers = []bookpkg.Enricher{
			enrichers.NewISBNdbEnricher(),      // Priority 0 - highest (skips if no API key)
			enrichers.NewOpenLibraryEnricher(), // Priority 1
			enrichers.NewGoogleBooksEnricher(), // Priority 2
		}
		defaultMerger = bookpkg.NewPriorityMerger()
	})
	return defaultEnrichers
}

// enrichBookWithEnrichers uses the new enricher system to enrich a book.
// It runs all configured enrichers and merges the results by priority.
func enrichBookWithEnrichers(ctx context.Context, book *Book) error {
	// Get ISBN to search with
	searchISBN := book.ISBN13
	if searchISBN == "" {
		searchISBN = book.ISBN
	}

	if searchISBN == "" {
		return nil // No ISBN available
	}

	enricherList := getDefaultEnrichers()
	results := make([]bookpkg.EnricherResult, 0, len(enricherList))

	// Run all enrichers and collect results
	for _, e := range enricherList {
		data, err := e.Enrich(ctx, searchISBN)
		if err != nil {
			slog.Debug("Enricher failed", "enricher", e.Name(), "isbn", searchISBN, "error", err)
			continue
		}

		if data != nil {
			slog.Debug("Enricher returned data", "enricher", e.Name(), "isbn", searchISBN)
			results = append(results, bookpkg.EnricherResult{
				Data:     data,
				Source:   e.Name(),
				Priority: e.Priority(),
			})
		}
	}

	if len(results) == 0 {
		slog.Debug("No enrichment data found", "isbn", searchISBN)
		return nil
	}

	// Merge results by priority
	merged := defaultMerger.Merge(results)
	if merged == nil {
		return nil
	}

	// Apply merged data to book (only fill empty fields)
	applyEnrichmentData(book, merged)

	return nil
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
