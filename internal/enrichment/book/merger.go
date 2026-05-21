package book

import (
	"cmp"
	"slices"
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
	slices.SortFunc(results, func(a, b EnricherResult) int {
		return cmp.Compare(a.Priority, b.Priority)
	})

	merged := &EnrichmentData{}

	for _, result := range results {
		if result.Data == nil {
			continue
		}
		mergeScalarFields(merged, result.Data)
		mergeSliceFields(merged, result.Data)
	}

	return merged
}

func mergeScalarFields(merged, src *EnrichmentData) {
	if merged.Title == nil && src.Title != nil && *src.Title != "" {
		merged.Title = src.Title
	}
	if merged.Subtitle == nil && src.Subtitle != nil && *src.Subtitle != "" {
		merged.Subtitle = src.Subtitle
	}
	if merged.Description == nil && src.Description != nil && *src.Description != "" {
		merged.Description = src.Description
	}
	if merged.Publisher == nil && src.Publisher != nil && *src.Publisher != "" {
		merged.Publisher = src.Publisher
	}
	if merged.NumberOfPages == nil && src.NumberOfPages != nil && *src.NumberOfPages > 0 {
		merged.NumberOfPages = src.NumberOfPages
	}
	if merged.CoverURL == nil && src.CoverURL != nil && *src.CoverURL != "" {
		merged.CoverURL = src.CoverURL
	}
	if merged.PublishDate == nil && src.PublishDate != nil && *src.PublishDate != "" {
		merged.PublishDate = src.PublishDate
	}
	if merged.Language == nil && src.Language != nil && *src.Language != "" {
		merged.Language = src.Language
	}
}

func mergeSliceFields(merged, src *EnrichmentData) {
	if len(src.Subjects) > 0 {
		merged.Subjects = mergeStringSlices(merged.Subjects, src.Subjects)
	}
	if len(src.SubjectPeople) > 0 {
		merged.SubjectPeople = mergeStringSlices(merged.SubjectPeople, src.SubjectPeople)
	}
	if len(merged.Authors) == 0 && len(src.Authors) > 0 {
		merged.Authors = src.Authors
	}
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
