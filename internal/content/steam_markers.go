package content

import (
	"strings"
)

const (
	// SteamDataStart is the HTML comment marker for the start of Steam content
	SteamDataStart = "<!-- STEAM_DATA_START -->"
	// SteamDataEnd is the HTML comment marker for the end of Steam content
	SteamDataEnd = "<!-- STEAM_DATA_END -->"
)

// WrapWithSteamMarkers wraps content with Steam data markers.
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

// HasSteamContentMarkers returns true if the body contains both Steam markers.
func HasSteamContentMarkers(body string) bool {
	return strings.Contains(body, SteamDataStart) && strings.Contains(body, SteamDataEnd)
}

// GetSteamContent extracts content between Steam markers if they exist.
// Returns the content and true if markers found, empty string and false otherwise.
func GetSteamContent(body string) (string, bool) {
	if !HasSteamContentMarkers(body) {
		return "", false
	}
	startIdx := strings.Index(body, SteamDataStart)
	endIdx := strings.Index(body, SteamDataEnd)
	if startIdx == -1 || endIdx == -1 || endIdx <= startIdx {
		return "", false
	}
	content := body[startIdx+len(SteamDataStart) : endIdx]
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
