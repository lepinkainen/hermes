package content

import (
	"regexp"
	"strings"
)

const (
	// IMDbDataStart is the start marker for IMDb content
	IMDbDataStart = "<!-- IMDB_DATA_START -->"
	// IMDbDataEnd is the end marker for IMDb content
	IMDbDataEnd = "<!-- IMDB_DATA_END -->"
)

// WrapWithIMDbMarkers wraps content with IMDb markers
func WrapWithIMDbMarkers(content string) string {
	if content == "" {
		return ""
	}

	var builder strings.Builder
	builder.WriteString(IMDbDataStart)
	builder.WriteString("\n")
	builder.WriteString(strings.TrimSpace(content))
	builder.WriteString("\n")
	builder.WriteString(IMDbDataEnd)
	return builder.String()
}

// HasIMDbContentMarkers checks if note contains IMDb content markers
func HasIMDbContentMarkers(noteContent string) bool {
	return strings.Contains(noteContent, IMDbDataStart) &&
		strings.Contains(noteContent, IMDbDataEnd)
}

// GetIMDbContent extracts content between IMDb markers
func GetIMDbContent(noteContent string) string {
	startIndex := strings.Index(noteContent, IMDbDataStart)
	endIndex := strings.Index(noteContent, IMDbDataEnd)

	if startIndex == -1 || endIndex == -1 || endIndex <= startIndex {
		return ""
	}

	// Extract content between markers
	start := startIndex + len(IMDbDataStart)
	content := noteContent[start:endIndex]
	return strings.TrimSpace(content)
}

// ReplaceIMDbContent replaces content between IMDb markers with new content
func ReplaceIMDbContent(noteContent string, newContent string) string {
	// Pattern to match everything between (and including) the IMDb markers
	pattern := regexp.MustCompile(`(?s)` + regexp.QuoteMeta(IMDbDataStart) + `.*?` + regexp.QuoteMeta(IMDbDataEnd))

	// Create new wrapped content
	wrappedContent := WrapWithIMDbMarkers(newContent)

	// Replace the matched content with the new wrapped content
	return pattern.ReplaceAllString(noteContent, wrappedContent)
}
