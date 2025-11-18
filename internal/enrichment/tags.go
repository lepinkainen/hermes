package enrichment

import (
	"sort"
)

// MergeTags combines two slices of tags, removing duplicates.
// Returns a sorted, deduplicated slice of tags.
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
