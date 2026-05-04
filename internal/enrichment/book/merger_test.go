package book

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func strPtr(value string) *string { return &value }

func intPtr(value int) *int { return &value }

func TestPriorityMerger_MergeEmptyResults(t *testing.T) {
	merger := NewPriorityMerger()

	result := merger.Merge(nil)

	require.Nil(t, result)
}

func TestPriorityMerger_MergePriorityOrderWins(t *testing.T) {
	merger := NewPriorityMerger()
	results := []EnricherResult{
		{
			Priority: 20,
			Data: &EnrichmentData{
				Title:         strPtr("Low Priority Title"),
				Description:   strPtr("Low Priority Description"),
				Publisher:     strPtr("Low Priority Publisher"),
				NumberOfPages: intPtr(111),
				CoverURL:      strPtr("https://example.com/low.jpg"),
				PublishDate:   strPtr("1999"),
				Language:      strPtr("fi"),
				Authors:       []string{"Low Author"},
			},
		},
		{
			Priority: 10,
			Data: &EnrichmentData{
				Title:         strPtr("High Priority Title"),
				Description:   strPtr("High Priority Description"),
				Publisher:     strPtr("High Priority Publisher"),
				NumberOfPages: intPtr(222),
				CoverURL:      strPtr("https://example.com/high.jpg"),
				PublishDate:   strPtr("2001"),
				Language:      strPtr("en"),
				Authors:       []string{"High Author"},
			},
		},
	}

	result := merger.Merge(results)

	require.NotNil(t, result)
	require.Equal(t, "High Priority Title", *result.Title)
	require.Equal(t, "High Priority Description", *result.Description)
	require.Equal(t, "High Priority Publisher", *result.Publisher)
	require.Equal(t, 222, *result.NumberOfPages)
	require.Equal(t, "https://example.com/high.jpg", *result.CoverURL)
	require.Equal(t, "2001", *result.PublishDate)
	require.Equal(t, "en", *result.Language)
	require.Equal(t, []string{"High Author"}, result.Authors)
}

func TestPriorityMerger_MergeSkipsEmptyHighPriorityValues(t *testing.T) {
	merger := NewPriorityMerger()
	results := []EnricherResult{
		{
			Priority: 1,
			Data: &EnrichmentData{
				Title:         strPtr(""),
				Subtitle:      strPtr(""),
				Description:   strPtr(""),
				Publisher:     strPtr(""),
				NumberOfPages: intPtr(0),
				CoverURL:      strPtr(""),
				PublishDate:   strPtr(""),
				Language:      strPtr(""),
			},
		},
		{
			Priority: 2,
			Data: &EnrichmentData{
				Title:         strPtr("Fallback Title"),
				Subtitle:      strPtr("Fallback Subtitle"),
				Description:   strPtr("Fallback Description"),
				Publisher:     strPtr("Fallback Publisher"),
				NumberOfPages: intPtr(333),
				CoverURL:      strPtr("https://example.com/fallback.jpg"),
				PublishDate:   strPtr("2020"),
				Language:      strPtr("en"),
			},
		},
	}

	result := merger.Merge(results)

	require.NotNil(t, result)
	require.Equal(t, "Fallback Title", *result.Title)
	require.Equal(t, "Fallback Subtitle", *result.Subtitle)
	require.Equal(t, "Fallback Description", *result.Description)
	require.Equal(t, "Fallback Publisher", *result.Publisher)
	require.Equal(t, 333, *result.NumberOfPages)
	require.Equal(t, "https://example.com/fallback.jpg", *result.CoverURL)
	require.Equal(t, "2020", *result.PublishDate)
	require.Equal(t, "en", *result.Language)
}

func TestPriorityMerger_MergeSubjectsAndSubjectPeopleDedupedInPriorityOrder(t *testing.T) {
	merger := NewPriorityMerger()
	results := []EnricherResult{
		{
			Priority: 2,
			Data: &EnrichmentData{
				Subjects:      []string{"Fantasy", "Adventure", "Fantasy"},
				SubjectPeople: []string{"Alice", "Bob"},
			},
		},
		{
			Priority: 1,
			Data: &EnrichmentData{
				Subjects:      []string{"Science Fiction", "Fantasy"},
				SubjectPeople: []string{"Bob", "Charlie"},
			},
		},
	}

	result := merger.Merge(results)

	require.NotNil(t, result)
	require.Equal(t, []string{"Science Fiction", "Fantasy", "Adventure"}, result.Subjects)
	require.Equal(t, []string{"Bob", "Charlie", "Alice"}, result.SubjectPeople)
}

func TestPriorityMerger_MergeAuthorsFirstNonEmptyByPriority(t *testing.T) {
	merger := NewPriorityMerger()
	results := []EnricherResult{
		{Priority: 1, Data: &EnrichmentData{Authors: nil}},
		{Priority: 2, Data: &EnrichmentData{Authors: []string{"First Author", "Second Author"}}},
		{Priority: 3, Data: &EnrichmentData{Authors: []string{"Ignored Author"}}},
	}

	result := merger.Merge(results)

	require.NotNil(t, result)
	require.Equal(t, []string{"First Author", "Second Author"}, result.Authors)
}

func TestPriorityMerger_MergeIgnoresNilData(t *testing.T) {
	merger := NewPriorityMerger()
	results := []EnricherResult{
		{Priority: 1, Data: nil},
		{Priority: 2, Data: &EnrichmentData{Title: strPtr("Available Title")}},
	}

	result := merger.Merge(results)

	require.NotNil(t, result)
	require.Equal(t, "Available Title", *result.Title)
}
