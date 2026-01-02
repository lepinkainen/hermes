package content

import (
	"strings"
)

const (
	// LetterboxdDataStart is the HTML comment marker for the start of Letterboxd content
	LetterboxdDataStart = "<!-- LETTERBOXD_DATA_START -->"
	// LetterboxdDataEnd is the HTML comment marker for the end of Letterboxd content
	LetterboxdDataEnd = "<!-- LETTERBOXD_DATA_END -->"
)

// WrapWithLetterboxdMarkers wraps content with Letterboxd data markers.
func WrapWithLetterboxdMarkers(content string) string {
	if content == "" {
		return ""
	}
	var builder strings.Builder
	builder.WriteString(LetterboxdDataStart)
	builder.WriteString("\n")
	builder.WriteString(strings.TrimSpace(content))
	builder.WriteString("\n")
	builder.WriteString(LetterboxdDataEnd)
	return builder.String()
}

// HasLetterboxdContentMarkers returns true if the body contains both Letterboxd markers.
func HasLetterboxdContentMarkers(body string) bool {
	return strings.Contains(body, LetterboxdDataStart) && strings.Contains(body, LetterboxdDataEnd)
}

// GetLetterboxdContent extracts content between Letterboxd markers if they exist.
// Returns the content and true if markers found, empty string and false otherwise.
func GetLetterboxdContent(body string) (string, bool) {
	if !HasLetterboxdContentMarkers(body) {
		return "", false
	}
	startIdx := strings.Index(body, LetterboxdDataStart)
	endIdx := strings.Index(body, LetterboxdDataEnd)
	if startIdx == -1 || endIdx == -1 || endIdx <= startIdx {
		return "", false
	}
	content := body[startIdx+len(LetterboxdDataStart) : endIdx]
	return strings.TrimSpace(content), true
}

// ReplaceLetterboxdContent replaces content between Letterboxd markers with new content.
// If markers don't exist, returns the original body unchanged.
func ReplaceLetterboxdContent(body string, newContent string) string {
	if !HasLetterboxdContentMarkers(body) {
		return body
	}
	startIdx := strings.Index(body, LetterboxdDataStart)
	endIdx := strings.Index(body, LetterboxdDataEnd)
	if startIdx == -1 || endIdx == -1 || endIdx <= startIdx {
		return body
	}

	before := strings.TrimSpace(body[:startIdx])
	after := strings.TrimSpace(body[endIdx+len(LetterboxdDataEnd):])

	var builder strings.Builder
	if before != "" {
		builder.WriteString(before)
		builder.WriteString("\n\n")
	}
	builder.WriteString(LetterboxdDataStart)
	builder.WriteString("\n")
	builder.WriteString(strings.TrimSpace(newContent))
	builder.WriteString("\n")
	builder.WriteString(LetterboxdDataEnd)
	if after != "" {
		builder.WriteString("\n")
		builder.WriteString(after)
	}
	return builder.String()
}
