package cmd

import "strings"

// sanitizeFilename replaces invalid characters in the filename with underscores
func sanitizeFilename(filename string) string {

	filename = strings.ReplaceAll(filename, ":", "")

	return strings.ReplaceAll(filename, "[^a-zA-Z0-9\\s:]+", "_")
}
