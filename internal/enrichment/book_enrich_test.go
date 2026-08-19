package enrichment

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/lepinkainen/hermes/internal/content"
	"github.com/lepinkainen/hermes/internal/enrichment/book"
	"github.com/stretchr/testify/require"
)

func strPtr(s string) *string { return &s }
func intPtr(i int) *int       { return &i }

// stubRunBookEnrichers installs a stub for runBookEnrichers and restores the
// original after the test completes.
func stubRunBookEnrichers(t *testing.T, fn func(ctx context.Context, isbn string, list []book.Enricher) (*book.EnrichmentData, error)) {
	t.Helper()
	prev := runBookEnrichers
	runBookEnrichers = fn
	t.Cleanup(func() { runBookEnrichers = prev })
}

func TestEnrichFromBook_NoISBNAndNoSearchResults(t *testing.T) {
	withTestCacheDB(t)
	withOpenLibrarySearchServer(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"docs":[]}`))
	})

	called := false
	stubRunBookEnrichers(t, func(_ context.Context, _ string, _ []book.Enricher) (*book.EnrichmentData, error) {
		called = true
		return nil, nil
	})

	result, err := EnrichFromBook(t.Context(), "Nonexistent Book", "", "", BookEnrichmentOptions{})
	require.NoError(t, err)
	require.Nil(t, result)
	require.False(t, called, "enrichers should not run when no ISBN can be resolved")
}

func TestEnrichFromBook_NoMergedData(t *testing.T) {
	stubRunBookEnrichers(t, func(_ context.Context, isbn string, _ []book.Enricher) (*book.EnrichmentData, error) {
		require.Equal(t, "9780441172719", isbn)
		return nil, nil
	})

	result, err := EnrichFromBook(t.Context(), "Dune", "", "9780441172719", BookEnrichmentOptions{})
	require.NoError(t, err)
	require.Nil(t, result)
}

func TestEnrichFromBook_PrefersISBN13ForSearch(t *testing.T) {
	var seenISBN string
	stubRunBookEnrichers(t, func(_ context.Context, isbn string, _ []book.Enricher) (*book.EnrichmentData, error) {
		seenISBN = isbn
		return &book.EnrichmentData{Description: strPtr("desc")}, nil
	})

	result, err := EnrichFromBook(t.Context(), "Dune", "0441172717", "9780441172719", BookEnrichmentOptions{})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, "9780441172719", seenISBN)
	require.Equal(t, "0441172717", result.ISBN)
	require.Equal(t, "9780441172719", result.ISBN13)
}

func TestEnrichFromBook_MapsMergedFields(t *testing.T) {
	stubRunBookEnrichers(t, func(_ context.Context, _ string, _ []book.Enricher) (*book.EnrichmentData, error) {
		return &book.EnrichmentData{
			Description:   strPtr("A great book"),
			Subtitle:      strPtr("A subtitle"),
			Publisher:     strPtr("Ace Books"),
			PublishDate:   strPtr("1965"),
			NumberOfPages: intPtr(412),
			Subjects:      []string{"Science Fiction"},
			SubjectPeople: []string{"Paul Atreides"},
			Authors:       []string{"Frank Herbert"},
		}, nil
	})

	result, err := EnrichFromBook(t.Context(), "Dune", "", "9780441172719", BookEnrichmentOptions{})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, "A great book", result.Description)
	require.Equal(t, "A subtitle", result.Subtitle)
	require.Equal(t, "Ace Books", result.Publisher)
	require.Equal(t, "1965", result.PublishDate)
	require.Equal(t, 412, result.Pages)
	require.Equal(t, []string{"Science Fiction"}, result.Subjects)
	require.Equal(t, []string{"Paul Atreides"}, result.SubjectPeople)
	require.Equal(t, []string{"Frank Herbert"}, result.Authors)
}

func TestEnrichFromBook_GeneratesContentWithDefaultSections(t *testing.T) {
	stubRunBookEnrichers(t, func(_ context.Context, _ string, _ []book.Enricher) (*book.EnrichmentData, error) {
		return &book.EnrichmentData{
			Description: strPtr("A great book"),
			Subjects:    []string{"Science Fiction"},
		}, nil
	})

	result, err := EnrichFromBook(t.Context(), "Dune", "", "9780441172719", BookEnrichmentOptions{
		GenerateContent: true,
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Contains(t, result.ContentMarkdown, "## Book Info")
	require.Contains(t, result.ContentMarkdown, "## Description")
	require.Contains(t, result.ContentMarkdown, "## Subjects")
	require.Contains(t, result.ContentMarkdown, "A great book")
	// Content should be unwrapped -- no markers.
	require.False(t, strings.Contains(result.ContentMarkdown, "GOODREADS_DATA_START"))
}

func TestEnrichFromBook_ContentRespectsExplicitSections(t *testing.T) {
	stubRunBookEnrichers(t, func(_ context.Context, _ string, _ []book.Enricher) (*book.EnrichmentData, error) {
		return &book.EnrichmentData{
			Description: strPtr("A great book"),
			Subjects:    []string{"Science Fiction"},
		}, nil
	})

	result, err := EnrichFromBook(t.Context(), "Dune", "", "9780441172719", BookEnrichmentOptions{
		GenerateContent: true,
		ContentSections: []string{"description"},
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotContains(t, result.ContentMarkdown, "## Book Info")
	require.Contains(t, result.ContentMarkdown, "## Description")
	require.NotContains(t, result.ContentMarkdown, "## Subjects")
}

func TestBuildBookDetails_FillsOnlyEmptyFields(t *testing.T) {
	base := &content.GoodreadsBookDetails{
		Title:       "Note Title",
		Description: "Note description (should win)",
	}
	enrichment := &BookEnrichment{
		Description: "Enrichment description (should not overwrite)",
		Subtitle:    "Enrichment subtitle",
		Publisher:   "Enrichment publisher",
		Pages:       500,
		Authors:     []string{"Enrichment Author"},
		ISBN:        "0441172717",
		ISBN13:      "9780441172719",
	}

	details := buildBookDetails("Fallback Title", base, enrichment)

	require.Equal(t, "Note Title", details.Title, "existing title should win")
	require.Equal(t, "Note description (should win)", details.Description, "existing description should win")
	require.Equal(t, "Enrichment subtitle", details.Subtitle, "empty subtitle should be filled")
	require.Equal(t, "Enrichment publisher", details.Publisher, "empty publisher should be filled")
	require.Equal(t, 500, details.Pages, "zero pages should be filled")
	require.Equal(t, []string{"Enrichment Author"}, details.Authors, "empty authors should be filled")
	require.Equal(t, "0441172717", details.ISBN)
	require.Equal(t, "9780441172719", details.ISBN13)
}

func TestBuildBookDetails_NilBase(t *testing.T) {
	enrichment := &BookEnrichment{
		Description: "Enrichment description",
	}

	details := buildBookDetails("Fallback Title", nil, enrichment)

	require.Equal(t, "Fallback Title", details.Title)
	require.Equal(t, "Enrichment description", details.Description)
}
