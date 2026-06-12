package fileutil

import "fmt"

// FormatDuration formats minutes into human-readable duration (e.g. "2h 30m")
func FormatDuration(minutes int) string {
	hours := minutes / 60
	mins := minutes % 60
	return fmt.Sprintf("%dh %dm", hours, mins)
}
