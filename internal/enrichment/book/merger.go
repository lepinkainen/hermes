package book

import (
	"sort"
)

// Merger defines the interface for merging book information from multiple sources.
type Merger interface {
	// Merge combines multiple EnricherResults into a single EnrichmentData.
	// Results are merged by priority (lower priority number = higher precedence).
	Merge(results []EnricherResult) *EnrichmentData
}

// PriorityMerger implements Merger using priority-based field selection.
// For each field, it uses the first non-empty value from the sorted results.
type PriorityMerger struct{}

// NewPriorityMerger creates a new PriorityMerger.
func NewPriorityMerger() *PriorityMerger {
	return &PriorityMerger{}
}

// Merge combines multiple EnricherResults into a single EnrichmentData.
// Results are sorted by priority (lower = higher precedence) and each field
// takes the first non-empty value.
func (m *PriorityMerger) Merge(results []EnricherResult) *EnrichmentData {
	if len(results) == 0 {
		return nil
	}

	// Sort by priority (lower = higher precedence)
	sort.Slice(results, func(i, j int) bool {
		return results[i].Priority < results[j].Priority
	})

	merged := &EnrichmentData{}

	for _, result := range results {
		if result.Data == nil {
			continue
		}

		// Title
		if merged.Title == nil && result.Data.Title != nil && *result.Data.Title != "" {
			merged.Title = result.Data.Title
		}

		// Subtitle
		if merged.Subtitle == nil && result.Data.Subtitle != nil && *result.Data.Subtitle != "" {
			merged.Subtitle = result.Data.Subtitle
		}

		// Description
		if merged.Description == nil && result.Data.Description != nil && *result.Data.Description != "" {
			merged.Description = result.Data.Description
		}

		// Publisher
		if merged.Publisher == nil && result.Data.Publisher != nil && *result.Data.Publisher != "" {
			merged.Publisher = result.Data.Publisher
		}

		// NumberOfPages
		if merged.NumberOfPages == nil && result.Data.NumberOfPages != nil && *result.Data.NumberOfPages > 0 {
			merged.NumberOfPages = result.Data.NumberOfPages
		}

		// CoverURL
		if merged.CoverURL == nil && result.Data.CoverURL != nil && *result.Data.CoverURL != "" {
			merged.CoverURL = result.Data.CoverURL
		}

		// PublishDate
		if merged.PublishDate == nil && result.Data.PublishDate != nil && *result.Data.PublishDate != "" {
			merged.PublishDate = result.Data.PublishDate
		}

		// Language
		if merged.Language == nil && result.Data.Language != nil && *result.Data.Language != "" {
			merged.Language = result.Data.Language
		}

		// Subjects - merge all unique values
		if len(result.Data.Subjects) > 0 {
			merged.Subjects = mergeStringSlices(merged.Subjects, result.Data.Subjects)
		}

		// SubjectPeople - merge all unique values
		if len(result.Data.SubjectPeople) > 0 {
			merged.SubjectPeople = mergeStringSlices(merged.SubjectPeople, result.Data.SubjectPeople)
		}

		// Authors - prefer first non-empty list
		if len(merged.Authors) == 0 && len(result.Data.Authors) > 0 {
			merged.Authors = result.Data.Authors
		}
	}

	return merged
}

// mergeStringSlices merges two string slices, removing duplicates.
func mergeStringSlices(a, b []string) []string {
	seen := make(map[string]bool)
	result := make([]string, 0, len(a)+len(b))

	for _, s := range a {
		if !seen[s] {
			seen[s] = true
			result = append(result, s)
		}
	}

	for _, s := range b {
		if !seen[s] {
			seen[s] = true
			result = append(result, s)
		}
	}

	return result
}
