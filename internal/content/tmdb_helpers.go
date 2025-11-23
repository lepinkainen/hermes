package content

import (
	"fmt"
	"slices"
	"strings"
)

func stringVal(m map[string]any, key string) string {
	if val, ok := m[key]; ok {
		if s, ok := val.(string); ok {
			return s
		}
	}
	return ""
}

func intVal(m map[string]any, key string) (int, bool) {
	if val, ok := m[key]; ok {
		switch v := val.(type) {
		case float64:
			return int(v), true
		case int:
			return v, true
		case int64:
			return int(v), true
		case string:
			var parsed int
			if _, err := fmt.Sscanf(v, "%d", &parsed); err == nil {
				return parsed, true
			}
		}
	}
	return 0, false
}

func floatVal(m map[string]any, key string) (float64, bool) {
	if val, ok := m[key]; ok {
		switch v := val.(type) {
		case float64:
			return v, true
		case float32:
			return float64(v), true
		case int:
			return float64(v), true
		case int64:
			return float64(v), true
		}
	}
	return 0, false
}

func boolVal(m map[string]any, key string) bool {
	if val, ok := m[key]; ok {
		switch v := val.(type) {
		case bool:
			return v
		case string:
			return strings.EqualFold(v, "true")
		case float64:
			return v != 0
		case int:
			return v != 0
		}
	}
	return false
}

func stringSlice(m map[string]any, key string) []string {
	val, ok := m[key]
	if !ok {
		return nil
	}
	switch arr := val.(type) {
	case []any:
		out := make([]string, 0, len(arr))
		for _, item := range arr {
			if s, ok := item.(string); ok && s != "" {
				out = append(out, s)
			}
		}
		return out
	case []string:
		return append([]string(nil), arr...)
	default:
		return nil
	}
}

func nestedString(m map[string]any, key string, nestedKey string) string {
	raw, ok := m[key]
	if !ok {
		return ""
	}
	switch inner := raw.(type) {
	case map[string]any:
		return stringVal(inner, nestedKey)
	default:
		return ""
	}
}

func firstStringFromArray(m map[string]any, key string, nested string) string {
	raw, ok := m[key]
	if !ok {
		return ""
	}
	if arr, ok := raw.([]any); ok {
		for _, item := range arr {
			if obj, ok := item.(map[string]any); ok {
				if value := stringVal(obj, nested); value != "" {
					return value
				}
			}
		}
	}
	return ""
}

func usContentRating(details map[string]any) string {
	raw, ok := details["content_ratings"].(map[string]any)
	if !ok {
		return ""
	}
	results, ok := raw["results"].([]any)
	if !ok {
		return ""
	}
	for _, entry := range results {
		if obj, ok := entry.(map[string]any); ok {
			if strings.EqualFold(stringVal(obj, "iso_3166_1"), "US") {
				return stringVal(obj, "rating")
			}
		}
	}
	return ""
}

func friendlyHomepageName(url string) string {
	switch {
	case strings.Contains(url, "apple.com"):
		return "Apple TV+"
	case strings.Contains(url, "netflix.com"):
		return "Netflix"
	case strings.Contains(url, "hulu.com"):
		return "Hulu"
	case strings.Contains(url, "disneyplus.com"):
		return "Disney+"
	case strings.Contains(url, "primevideo.com") || strings.Contains(url, "amazon.com"):
		return "Prime Video"
	case strings.Contains(url, "hbo.com") || strings.Contains(url, "max.com"):
		return "Max"
	default:
		return "Official Website"
	}
}

func countryFlag(code string) string {
	flags := map[string]string{
		"GB": "🇬🇧",
		"US": "🇺🇸",
		"CA": "🇨🇦",
		"FR": "🇫🇷",
		"DE": "🇩🇪",
		"IT": "🇮🇹",
		"ES": "🇪🇸",
		"JP": "🇯🇵",
		"KR": "🇰🇷",
		"AU": "🇦🇺",
		"NZ": "🇳🇿",
		"IN": "🇮🇳",
		"BR": "🇧🇷",
		"MX": "🇲🇽",
		"SE": "🇸🇪",
		"NO": "🇳🇴",
		"DK": "🇩🇰",
		"FI": "🇫🇮",
		"NL": "🇳🇱",
		"BE": "🇧🇪",
		"CH": "🇨🇭",
		"AT": "🇦🇹",
		"IE": "🇮🇪",
		"PL": "🇵🇱",
		"CZ": "🇨🇿",
		"RU": "🇷🇺",
		"CN": "🇨🇳",
		"TW": "🇹🇼",
		"HK": "🇭🇰",
		"SG": "🇸🇬",
		"TH": "🇹🇭",
		"ID": "🇮🇩",
		"MY": "🇲🇾",
		"PH": "🇵🇭",
		"VN": "🇻🇳",
		"AR": "🇦🇷",
		"CL": "🇨🇱",
		"CO": "🇨🇴",
		"PE": "🇵🇪",
		"ZA": "🇿🇦",
		"EG": "🇪🇬",
		"IL": "🇮🇱",
		"TR": "🇹🇷",
		"GR": "🇬🇷",
		"PT": "🇵🇹",
		"RO": "🇷🇴",
		"HU": "🇭🇺",
		"UA": "🇺🇦",
	}
	code = strings.ToUpper(code)
	if flag, ok := flags[code]; ok {
		return flag
	}
	return "🌐"
}

func formatNumber(value int) string {
	if value == 0 {
		return "0"
	}
	part := fmt.Sprintf("%d", value)
	var result []string
	for len(part) > 3 {
		result = append(result, part[len(part)-3:])
		part = part[:len(part)-3]
	}
	result = append(result, part)
	slices.Reverse(result)
	return strings.Join(result, ",")
}
