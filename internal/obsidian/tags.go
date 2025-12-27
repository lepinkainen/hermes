package obsidian

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

// NormalizeTag normalizes a tag according to Obsidian conventions.
// Normalization steps:
// 1. Preserve case (no lowercasing)
// 2. Strip leading # if present
// 3. Trim leading/trailing whitespace
// 4. Convert all whitespace to hyphens (\s+ → -)
// 5. Strip leading/trailing hyphens
// 6. Collapse repeated hyphens (-{2,} → -)
// 7. Remove special characters (&, #, etc.) but preserve / for hierarchy
// 8. Return empty string if result is empty after normalization
func NormalizeTag(tag string) string {
	// Strip leading # and trim whitespace
	tag = strings.TrimSpace(tag)
	tag = strings.TrimPrefix(tag, "#")
	tag = strings.TrimSpace(tag)

	// Return early if empty
	if tag == "" {
		return ""
	}

	// Replace & with "and" before other processing
	tag = strings.ReplaceAll(tag, "&", "and")
	// Remove # symbols
	tag = strings.ReplaceAll(tag, "#", "")

	// Convert whitespace to hyphens
	wsRegex := regexp.MustCompile(`\s+`)
	tag = wsRegex.ReplaceAllString(tag, "-")

	// Collapse repeated hyphens
	hyphenRegex := regexp.MustCompile(`-+`)
	tag = hyphenRegex.ReplaceAllString(tag, "-")

	// Trim leading/trailing hyphens
	tag = strings.Trim(tag, "-")

	return tag
}

// NormalizeTags normalizes a slice of tags, removing empty results.
// Returns a sorted, deduplicated slice.
func NormalizeTags(tags []string) []string {
	seen := make(map[string]bool)
	result := make([]string, 0, len(tags))

	for _, tag := range tags {
		normalized := NormalizeTag(tag)
		if normalized != "" && !seen[normalized] {
			seen[normalized] = true
			result = append(result, normalized)
		}
	}

	sort.Strings(result)
	return result
}

// TagSet provides tag collection with automatic normalization and deduplication.
type TagSet struct {
	tags map[string]bool
}

// NewTagSet creates a new TagSet for collecting tags.
func NewTagSet() *TagSet {
	return &TagSet{
		tags: make(map[string]bool),
	}
}

// Add adds a tag to the set after normalization.
// Empty tags and duplicates are automatically filtered.
func (ts *TagSet) Add(tag string) {
	normalized := NormalizeTag(tag)
	if normalized != "" {
		ts.tags[normalized] = true
	}
}

// AddIf conditionally adds a tag if the condition is true.
func (ts *TagSet) AddIf(condition bool, tag string) {
	if condition {
		ts.Add(tag)
	}
}

// AddFormat adds a formatted tag (like fmt.Sprintf).
func (ts *TagSet) AddFormat(format string, args ...interface{}) {
	tag := fmt.Sprintf(format, args...)
	ts.Add(tag)
}

// GetSorted returns all tags as a sorted slice.
func (ts *TagSet) GetSorted() []string {
	result := make([]string, 0, len(ts.tags))
	for tag := range ts.tags {
		result = append(result, tag)
	}
	sort.Strings(result)
	return result
}

// MergeTags combines two tag slices, normalizes them, and returns a sorted, deduplicated result.
func MergeTags(existing, new []string) []string {
	seen := make(map[string]bool)

	// Add existing tags
	for _, tag := range existing {
		normalized := NormalizeTag(tag)
		if normalized != "" {
			seen[normalized] = true
		}
	}

	// Add new tags
	for _, tag := range new {
		normalized := NormalizeTag(tag)
		if normalized != "" {
			seen[normalized] = true
		}
	}

	// Convert to sorted slice
	result := make([]string, 0, len(seen))
	for tag := range seen {
		result = append(result, tag)
	}
	sort.Strings(result)
	return result
}

// TagsFromAny safely extracts a string slice from a polymorphic YAML value.
// YAML unmarshaling can produce []interface{} or []string, this handles both.
func TagsFromAny(val any) []string {
	if val == nil {
		return []string{}
	}

	// Handle []string directly
	if strSlice, ok := val.([]string); ok {
		result := make([]string, 0, len(strSlice))
		for _, s := range strSlice {
			if s != "" {
				result = append(result, s)
			}
		}
		return result
	}

	// Handle []interface{} from YAML
	if ifaceSlice, ok := val.([]interface{}); ok {
		result := make([]string, 0, len(ifaceSlice))
		for _, item := range ifaceSlice {
			if str, ok := item.(string); ok && str != "" {
				result = append(result, str)
			}
		}
		return result
	}

	return []string{}
}
