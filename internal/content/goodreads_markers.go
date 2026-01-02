package content

import (
	"strings"
)

const (
	// GoodreadsDataStart is the start marker for Goodreads content
	GoodreadsDataStart = "<!-- GOODREADS_DATA_START -->"
	// GoodreadsDataEnd is the end marker for Goodreads content
	GoodreadsDataEnd = "<!-- GOODREADS_DATA_END -->"
)

// WrapWithGoodreadsMarkers wraps content with Goodreads markers
func WrapWithGoodreadsMarkers(content string) string {
	if content == "" {
		return ""
	}

	var builder strings.Builder
	builder.WriteString(GoodreadsDataStart)
	builder.WriteString("\n")
	builder.WriteString(strings.TrimSpace(content))
	builder.WriteString("\n")
	builder.WriteString(GoodreadsDataEnd)
	return builder.String()
}

// HasGoodreadsContentMarkers checks if note contains Goodreads content markers
func HasGoodreadsContentMarkers(noteContent string) bool {
	return strings.Contains(noteContent, GoodreadsDataStart) &&
		strings.Contains(noteContent, GoodreadsDataEnd)
}

// GetGoodreadsContent extracts content between Goodreads markers
func GetGoodreadsContent(noteContent string) (string, bool) {
	startIndex := strings.Index(noteContent, GoodreadsDataStart)
	endIndex := strings.Index(noteContent, GoodreadsDataEnd)

	if startIndex == -1 || endIndex == -1 || endIndex <= startIndex {
		return "", false
	}

	// Extract content between markers
	start := startIndex + len(GoodreadsDataStart)
	content := noteContent[start:endIndex]
	return strings.TrimSpace(content), true
}

// ReplaceGoodreadsContent replaces content between Goodreads markers with new content.
// If markers don't exist, returns the original body unchanged.
func ReplaceGoodreadsContent(body string, newContent string) string {
	if !HasGoodreadsContentMarkers(body) {
		return body
	}
	startIdx := strings.Index(body, GoodreadsDataStart)
	endIdx := strings.Index(body, GoodreadsDataEnd)
	if startIdx == -1 || endIdx == -1 || endIdx <= startIdx {
		return body
	}

	before := strings.TrimSpace(body[:startIdx])
	after := strings.TrimSpace(body[endIdx+len(GoodreadsDataEnd):])

	var builder strings.Builder
	if before != "" {
		builder.WriteString(before)
		builder.WriteString("\n\n")
	}
	builder.WriteString(GoodreadsDataStart)
	builder.WriteString("\n")
	builder.WriteString(strings.TrimSpace(newContent))
	builder.WriteString("\n")
	builder.WriteString(GoodreadsDataEnd)
	if after != "" {
		builder.WriteString("\n")
		builder.WriteString(after)
	}
	return builder.String()
}
