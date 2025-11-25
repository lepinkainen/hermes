package goodreads

import "strings"

// Helper function to split comma-separated strings
func splitString(str string) []string {
	if str == "" {
		return nil
	}
	var splitStrings = strings.Split(str, ",")
	for i, s := range splitStrings {
		splitStrings[i] = strings.TrimSpace(s)
	}
	return splitStrings
}

// Helper function to handle the description field
func getDescription(desc any) string {
	switch v := desc.(type) {
	case string:
		return v
	case map[string]any:
		if value, ok := v["value"].(string); ok {
			return value
		}
	}
	return ""
}

// Helper function to handle subjects
func getSubjects(subjects []any) []string {
	result := make([]string, 0)
	for _, subject := range subjects {
		switch v := subject.(type) {
		case string:
			result = append(result, v)
		case map[string]any:
			if name, ok := v["name"].(string); ok {
				result = append(result, name)
			}
		}
	}
	return result
}
