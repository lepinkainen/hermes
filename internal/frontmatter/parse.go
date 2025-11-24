package frontmatter

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// ParsedNote represents a parsed markdown note with YAML frontmatter.
type ParsedNote struct {
	// Frontmatter is the raw YAML frontmatter as a map
	Frontmatter map[string]any
	// Body is the content after the frontmatter
	Body string
}

// ParseMarkdown parses markdown content with YAML frontmatter.
// Returns the parsed frontmatter and body, or an error if the format is invalid.
func ParseMarkdown(content []byte) (*ParsedNote, error) {
	trimmed := bytes.TrimSpace(content)
	if !bytes.HasPrefix(trimmed, []byte("---")) {
		return nil, fmt.Errorf("invalid markdown format: missing opening frontmatter delimiter")
	}

	// Split frontmatter section
	parts := bytes.SplitN(trimmed, []byte("---"), 3)
	if len(parts) < 3 {
		return nil, fmt.Errorf("invalid markdown format: missing closing frontmatter delimiter")
	}

	var fm map[string]any
	if err := yaml.Unmarshal(parts[1], &fm); err != nil {
		return nil, fmt.Errorf("failed to parse frontmatter: %w", err)
	}

	return &ParsedNote{
		Frontmatter: fm,
		Body:        strings.TrimSpace(string(parts[2])),
	}, nil
}

// IntFromAny converts various types to int.
// Handles int, int64, float64, and string types.
// Returns 0 if conversion fails.
func IntFromAny(val any) int {
	switch v := val.(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	case string:
		if n, err := strconv.Atoi(strings.TrimSpace(v)); err == nil {
			return n
		}
	}
	return 0
}

// StringFromAny extracts a string from any type.
// Returns empty string if not a string type.
func StringFromAny(val any) string {
	if s, ok := val.(string); ok {
		return strings.TrimSpace(s)
	}
	return ""
}

// GetInt retrieves an integer value from frontmatter by key.
// Returns 0 if key doesn't exist or value is not convertible to int.
func (p *ParsedNote) GetInt(key string) int {
	val, ok := p.Frontmatter[key]
	if !ok {
		return 0
	}
	return IntFromAny(val)
}

// GetString retrieves a string value from frontmatter by key.
// Returns empty string if key doesn't exist or value is not a string.
func (p *ParsedNote) GetString(key string) string {
	val, ok := p.Frontmatter[key]
	if !ok {
		return ""
	}
	return StringFromAny(val)
}

// DetectMediaType determines the media type from frontmatter.
// Checks tmdb_type field first, then falls back to detecting from tags.
// Returns "movie", "tv", or empty string if type cannot be determined.
func DetectMediaType(fm map[string]any) string {
	if fm == nil {
		return ""
	}

	// Check tmdb_type field first
	if mediaType := StringFromAny(fm["tmdb_type"]); mediaType != "" {
		return mediaType
	}

	// Fall back to detecting from tags
	return detectTypeFromTags(fm)
}

// DetectMediaTypeFromTags determines the media type using only tag values.
// Returns "movie", "tv", or empty string if no tag hints are present.
func DetectMediaTypeFromTags(fm map[string]any) string {
	if fm == nil {
		return ""
	}
	return detectTypeFromTags(fm)
}

// detectTypeFromTags attempts to determine media type from the tags array.
func detectTypeFromTags(fm map[string]any) string {
	tags, ok := fm["tags"]
	if !ok {
		return ""
	}

	moviePresent := false
	tvPresent := false

	// helper checks a single tag string
	checkTag := func(tagStr string) {
		tag := strings.ToLower(strings.TrimSpace(tagStr))
		switch {
		case tag == "movie" || strings.HasPrefix(tag, "movie/"):
			moviePresent = true
		case tag == "tv" || tag == "tv-show" || tag == "series" || strings.HasPrefix(tag, "tv/"):
			tvPresent = true
		}
	}

	// Handle []any (common YAML array representation)
	if tagSlice, ok := tags.([]any); ok {
		for _, tag := range tagSlice {
			if tagStr, ok := tag.(string); ok {
				checkTag(tagStr)
			}
		}
	}

	// Handle []string
	if tagSlice, ok := tags.([]string); ok {
		for _, tag := range tagSlice {
			checkTag(tag)
		}
	}

	switch {
	case tvPresent:
		return "tv"
	case moviePresent:
		return "movie"
	default:
		return ""
	}
}
