package book

import (
	"testing"

	"github.com/stretchr/testify/require"
)

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
				Title:         new("Low Priority Title"),
				Description:   new("Low Priority Description"),
				Publisher:     new("Low Priority Publisher"),
				NumberOfPages: new(111),
				CoverURL:      new("https://example.com/low.jpg"),
				PublishDate:   new("1999"),
				Language:      new("fi"),
				Authors:       []string{"Low Author"},
			},
		},
		{
			Priority: 10,
			Data: &EnrichmentData{
				Title:         new("High Priority Title"),
				Description:   new("High Priority Description"),
				Publisher:     new("High Priority Publisher"),
				NumberOfPages: new(222),
				CoverURL:      new("https://example.com/high.jpg"),
				PublishDate:   new("2001"),
				Language:      new("en"),
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
				Title:         new(""),
				Subtitle:      new(""),
				Description:   new(""),
				Publisher:     new(""),
				NumberOfPages: new(0),
				CoverURL:      new(""),
				PublishDate:   new(""),
				Language:      new(""),
			},
		},
		{
			Priority: 2,
			Data: &EnrichmentData{
				Title:         new("Fallback Title"),
				Subtitle:      new("Fallback Subtitle"),
				Description:   new("Fallback Description"),
				Publisher:     new("Fallback Publisher"),
				NumberOfPages: new(333),
				CoverURL:      new("https://example.com/fallback.jpg"),
				PublishDate:   new("2020"),
				Language:      new("en"),
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
		{Priority: 2, Data: &EnrichmentData{Title: new("Available Title")}},
	}

	result := merger.Merge(results)

	require.NotNil(t, result)
	require.Equal(t, "Available Title", *result.Title)
}
