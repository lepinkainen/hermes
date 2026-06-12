package obsidian

import (
	"strconv"
	"strings"
)

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

// StringFromAny extracts a trimmed string from any type.
// Returns empty string if not a string type.
func StringFromAny(val any) string {
	if s, ok := val.(string); ok {
		return strings.TrimSpace(s)
	}
	return ""
}
