// Package parseutil provides small, dependency-free string-munging helpers
// shared across importers (OMDB, OpenLibrary, Google Books, BookBrainz, ISBNdb).
package parseutil

import (
	"strconv"
	"strings"
)

// NormalizeISBN strips hyphens and spaces from an ISBN string.
func NormalizeISBN(isbn string) string {
	normalized := strings.ReplaceAll(isbn, "-", "")
	normalized = strings.ReplaceAll(normalized, " ", "")
	return normalized
}

// ParseFloat parses a float, returning 0 on error.
func ParseFloat(s string) float64 {
	f, _ := strconv.ParseFloat(s, 64)
	return f
}

// ParseRuntime parses an OMDB-style runtime string like "123 min" and returns
// the integer minute value. Returns 0 if the input does not match.
func ParseRuntime(runtime string) int {
	mins := strings.TrimSuffix(runtime, " min")
	val, _ := strconv.Atoi(mins)
	return val
}

// ParseYear parses a year string. Ranges like "2019-2022" or "2019–2022"
// return the first year. Returns 0 on error.
func ParseYear(year string) int {
	if strings.Contains(year, "–") || strings.Contains(year, "-") {
		parts := strings.FieldsFunc(year, func(r rune) bool {
			return r == '–' || r == '-'
		})
		if len(parts) > 0 {
			val, _ := strconv.Atoi(parts[0])
			return val
		}
	}
	val, _ := strconv.Atoi(year)
	return val
}

// ParseCommaList splits a comma-separated string into trimmed, non-empty
// entries. Returns nil for empty input or the OMDB sentinel "N/A".
func ParseCommaList(list string) []string {
	if list == "" || list == "N/A" {
		return nil
	}
	parts := strings.Split(list, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if t := strings.TrimSpace(p); t != "" {
			out = append(out, t)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
