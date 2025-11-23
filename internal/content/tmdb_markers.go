package content

import (
	"strings"
)

const (
	// TMDBDataStart is the HTML comment marker for the start of TMDB content
	TMDBDataStart = "<!-- TMDB_DATA_START -->"
	// TMDBDataEnd is the HTML comment marker for the end of TMDB content
	TMDBDataEnd = "<!-- TMDB_DATA_END -->"
)

// WrapWithMarkers wraps content with TMDB data markers.
func WrapWithMarkers(content string) string {
	if content == "" {
		return ""
	}
	var builder strings.Builder
	builder.WriteString(TMDBDataStart)
	builder.WriteString("\n")
	builder.WriteString(strings.TrimSpace(content))
	builder.WriteString("\n")
	builder.WriteString(TMDBDataEnd)
	return builder.String()
}

// HasTMDBContentMarkers returns true if the body contains both TMDB markers.
func HasTMDBContentMarkers(body string) bool {
	return strings.Contains(body, TMDBDataStart) && strings.Contains(body, TMDBDataEnd)
}

// GetTMDBContent extracts content between TMDB markers if they exist.
// Returns the content and true if markers found, empty string and false otherwise.
func GetTMDBContent(body string) (string, bool) {
	if !HasTMDBContentMarkers(body) {
		return "", false
	}
	startIdx := strings.Index(body, TMDBDataStart)
	endIdx := strings.Index(body, TMDBDataEnd)
	if startIdx == -1 || endIdx == -1 || endIdx <= startIdx {
		return "", false
	}
	content := body[startIdx+len(TMDBDataStart) : endIdx]
	return strings.TrimSpace(content), true
}

// ReplaceTMDBContent replaces content between TMDB markers with new content.
// If markers don't exist, returns the original body unchanged.
func ReplaceTMDBContent(body string, newContent string) string {
	if !HasTMDBContentMarkers(body) {
		return body
	}
	startIdx := strings.Index(body, TMDBDataStart)
	endIdx := strings.Index(body, TMDBDataEnd)
	if startIdx == -1 || endIdx == -1 || endIdx <= startIdx {
		return body
	}

	before := strings.TrimSpace(body[:startIdx])
	after := strings.TrimSpace(body[endIdx+len(TMDBDataEnd):])

	var builder strings.Builder
	if before != "" {
		builder.WriteString(before)
		builder.WriteString("\n\n")
	}
	builder.WriteString(TMDBDataStart)
	builder.WriteString("\n")
	builder.WriteString(strings.TrimSpace(newContent))
	builder.WriteString("\n")
	builder.WriteString(TMDBDataEnd)
	if after != "" {
		builder.WriteString("\n")
		builder.WriteString(after)
	}
	return builder.String()
}
