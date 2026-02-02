// Package book provides interfaces and utilities for enriching book metadata
// from multiple external sources.
package book

import (
	"context"
)

// Enricher defines the interface for fetching book information from external sources.
// Each implementation should handle its own authentication, rate limiting, and data
// transformation to the common EnrichmentData format.
type Enricher interface {
	// Name returns the human-readable name of the source (e.g., "OpenLibrary").
	Name() string

	// Priority returns the priority when merging data. Lower values indicate
	// higher priority. This helps determine which source's data to prefer
	// when merging conflicting information.
	Priority() int

	// Ping tests the connection to the source and returns an error if it
	// cannot be reached for whatever reason.
	Ping(ctx context.Context) error

	// Enrich retrieves book information using the provided ISBN.
	// Implementations should attempt to fetch as much data as possible.
	// Returns nil, nil if book not found (allows other enrichers to try).
	// Returns nil, error for actual errors (network issues, rate limits, etc.)
	Enrich(ctx context.Context, isbn string) (*EnrichmentData, error)
}

// EnrichmentData contains book metadata extracted from an external source.
// Pointer fields distinguish "not set" from "empty string".
type EnrichmentData struct {
	// Title is the main title of the book.
	Title *string

	// Subtitle is the secondary title or tagline.
	Subtitle *string

	// Description is the book's summary or blurb.
	Description *string

	// Publisher is the publishing company name.
	Publisher *string

	// NumberOfPages is the page count.
	NumberOfPages *int

	// CoverURL is the URL to the cover image.
	CoverURL *string

	// PublishDate is the publication date (format varies by source).
	PublishDate *string

	// Language is the language code (e.g., "en", "fi").
	Language *string

	// Subjects are topic/category tags.
	Subjects []string

	// SubjectPeople are people mentioned as subjects (biographies, etc.)
	SubjectPeople []string

	// Authors are the book's author names.
	Authors []string
}
