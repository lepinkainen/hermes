package content

import (
	"strings"
)

const (
	// SteamDataStart is the start marker for Steam content
	SteamDataStart = "<!-- STEAM_DATA_START -->"
	// SteamDataEnd is the end marker for Steam content
	SteamDataEnd = "<!-- STEAM_DATA_END -->"
)

// WrapWithSteamMarkers wraps content with Steam markers
func WrapWithSteamMarkers(content string) string {
	if content == "" {
		return ""
	}

	var builder strings.Builder
	builder.WriteString(SteamDataStart)
	builder.WriteString("\n")
	builder.WriteString(strings.TrimSpace(content))
	builder.WriteString("\n")
	builder.WriteString(SteamDataEnd)
	return builder.String()
}

// HasSteamContentMarkers checks if note contains Steam content markers
func HasSteamContentMarkers(noteContent string) bool {
	return strings.Contains(noteContent, SteamDataStart) &&
		strings.Contains(noteContent, SteamDataEnd)
}

// GetSteamContent extracts content between Steam markers
func GetSteamContent(noteContent string) (string, bool) {
	startIndex := strings.Index(noteContent, SteamDataStart)
	endIndex := strings.Index(noteContent, SteamDataEnd)

	if startIndex == -1 || endIndex == -1 || endIndex <= startIndex {
		return "", false
	}

	// Extract content between markers
	start := startIndex + len(SteamDataStart)
	content := noteContent[start:endIndex]
	return strings.TrimSpace(content), true
}

// ReplaceSteamContent replaces content between Steam markers with new content.
// If markers don't exist, returns the original body unchanged.
func ReplaceSteamContent(body string, newContent string) string {
	if !HasSteamContentMarkers(body) {
		return body
	}
	startIdx := strings.Index(body, SteamDataStart)
	endIdx := strings.Index(body, SteamDataEnd)
	if startIdx == -1 || endIdx == -1 || endIdx <= startIdx {
		return body
	}

	before := strings.TrimSpace(body[:startIdx])
	after := strings.TrimSpace(body[endIdx+len(SteamDataEnd):])

	var builder strings.Builder
	if before != "" {
		builder.WriteString(before)
		builder.WriteString("\n\n")
	}
	builder.WriteString(SteamDataStart)
	builder.WriteString("\n")
	builder.WriteString(strings.TrimSpace(newContent))
	builder.WriteString("\n")
	builder.WriteString(SteamDataEnd)
	if after != "" {
		builder.WriteString("\n")
		builder.WriteString(after)
	}
	return builder.String()
}
