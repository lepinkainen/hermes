package enrichment

import (
	"sort"
)

// MergeTags combines two slices of tags, removing duplicates.
// Returns a sorted, deduplicated slice of tags.
//
// Deprecated: Use obsidian.MergeTags instead. This function will be removed in v2.0.0.
// Migration: Replace enrichment.MergeTags(a, b) with obsidian.MergeTags(a, b)
// Behavioral difference: obsidian.MergeTags normalizes tags according to Obsidian conventions
// (converts whitespace to hyphens, removes special characters, etc.), while this function
// preserves tags exactly as provided.
func MergeTags(existing, new []string) []string {
	tagSet := make(map[string]bool)

	for _, tag := range existing {
		if tag != "" {
			tagSet[tag] = true
		}
	}
	for _, tag := range new {
		if tag != "" {
			tagSet[tag] = true
		}
	}

	result := make([]string, 0, len(tagSet))
	for tag := range tagSet {
		result = append(result, tag)
	}

	sort.Strings(result)
	return result
}

// TagsFromAny extracts a string slice from a YAML value.
// Handles both []interface{} and []string types.
//
// Deprecated: Use obsidian.TagsFromAny instead. This function will be removed in v2.0.0.
// Migration: Replace enrichment.TagsFromAny(val) with obsidian.TagsFromAny(val)
// Behavioral difference: obsidian.TagsFromAny returns an empty slice []string{} for nil/invalid input,
// while this function returns nil. Update code that checks for nil to check for empty slices instead.
func TagsFromAny(val any) []string {
	var result []string

	switch v := val.(type) {
	case []interface{}:
		for _, item := range v {
			if s, ok := item.(string); ok && s != "" {
				result = append(result, s)
			}
		}
	case []string:
		for _, s := range v {
			if s != "" {
				result = append(result, s)
			}
		}
	}

	return result
}
