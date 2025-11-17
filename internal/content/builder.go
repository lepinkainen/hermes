// Package content provides markdown content generation from TMDB data.
package content

import (
	"fmt"
	"slices"
	"strings"
)

const (
	// TMDBDataStart is the HTML comment marker for the start of TMDB content
	TMDBDataStart = "<!-- TMDB_DATA_START -->"
	// TMDBDataEnd is the HTML comment marker for the end of TMDB content
	TMDBDataEnd = "<!-- TMDB_DATA_END -->"
)

// BuildCoverImageEmbed generates Obsidian embed syntax for a cover image.
// Returns: ![[filename|250]]
func BuildCoverImageEmbed(coverFilename string) string {
	if coverFilename == "" {
		return ""
	}
	return fmt.Sprintf("![[%s|250]]", coverFilename)
}

// BuildTMDBContent generates markdown content from TMDB details.
func BuildTMDBContent(details map[string]any, mediaType string, sections []string) string {
	if len(sections) == 0 {
		if mediaType == "tv" {
			sections = []string{"overview", "info", "seasons"}
		} else {
			sections = []string{"overview", "info"}
		}
	}

	var blocks []string
	for _, section := range sections {
		switch section {
		case "overview":
			if block := buildOverview(details); block != "" {
				blocks = append(blocks, block)
			}
		case "info":
			if block := buildInfo(details, mediaType); block != "" {
				blocks = append(blocks, block)
			}
		case "seasons":
			if mediaType == "tv" {
				if block := buildSeasons(details); block != "" {
					blocks = append(blocks, block)
				}
			}
		}
	}

	return strings.Join(blocks, "\n\n")
}

func buildOverview(details map[string]any) string {
	overview := stringVal(details, "overview")
	if strings.TrimSpace(overview) == "" {
		return ""
	}

	tagline := stringVal(details, "tagline")

	var builder strings.Builder
	builder.WriteString("## Overview\n\n")
	builder.WriteString(strings.TrimSpace(overview))
	builder.WriteString("\n")

	if tagline = strings.TrimSpace(tagline); tagline != "" {
		builder.WriteString("\n> _\"")
		builder.WriteString(tagline)
		builder.WriteString("\"_\n")
	}
	return builder.String()
}

func buildInfo(details map[string]any, mediaType string) string {
	var builder strings.Builder
	builder.WriteString("## ")
	if mediaType == "tv" {
		builder.WriteString("Series Info\n\n")
	} else {
		builder.WriteString("Movie Info\n\n")
	}

	builder.WriteString("| | |\n")
	builder.WriteString("|---|---|\n")

	status := stringVal(details, "status")
	inProduction := boolVal(details, "in_production")
	if status == "" {
		status = "Unknown"
	}
	if mediaType == "tv" && inProduction {
		builder.WriteString(fmt.Sprintf("| **Status** | %s (In Production) |\n", status))
	} else {
		builder.WriteString(fmt.Sprintf("| **Status** | %s |\n", status))
	}

	if mediaType == "tv" {
		seasons, _ := intVal(details, "number_of_seasons")
		episodes, _ := intVal(details, "number_of_episodes")
		builder.WriteString(fmt.Sprintf("| **Seasons** | %d (%d episodes) |\n", seasons, episodes))

		firstAir := stringVal(details, "first_air_date")
		lastAir := stringVal(details, "last_air_date")
		if firstAir != "" {
			airText := firstAir
			switch {
			case lastAir != "" && lastAir != firstAir:
				airText = fmt.Sprintf("%s → %s", firstAir, lastAir)
			case inProduction:
				airText = fmt.Sprintf("%s → Present", firstAir)
			}
			builder.WriteString(fmt.Sprintf("| **Aired** | %s |\n", airText))
		}
	} else {
		if runtime, ok := intVal(details, "runtime"); ok && runtime > 0 {
			builder.WriteString(fmt.Sprintf("| **Runtime** | %d min |\n", runtime))
		}
		release := stringVal(details, "release_date")
		if release != "" {
			builder.WriteString(fmt.Sprintf("| **Released** | %s |\n", release))
		}
	}

	if rating, ok := floatVal(details, "vote_average"); ok && rating > 0 {
		votes, _ := intVal(details, "vote_count")
		builder.WriteString(fmt.Sprintf("| **Rating** | ⭐ %.1f/10 (%s votes) |\n", rating, formatNumber(votes)))
	}

	if mediaType == "tv" {
		if networkName := firstStringFromArray(details, "networks", "name"); networkName != "" {
			builder.WriteString(fmt.Sprintf("| **Network** | %s |\n", networkName))
		}
	} else {
		if budget, ok := intVal(details, "budget"); ok && budget > 0 {
			builder.WriteString(fmt.Sprintf("| **Budget** | $%s |\n", formatNumber(budget)))
		}
		if revenue, ok := intVal(details, "revenue"); ok && revenue > 0 {
			builder.WriteString(fmt.Sprintf("| **Revenue** | $%s |\n", formatNumber(revenue)))
		}
	}

	if countries := stringSlice(details, "origin_country"); len(countries) > 0 {
		parts := make([]string, 0, min(3, len(countries)))
		for i, code := range countries {
			if i >= 3 {
				break
			}
			parts = append(parts, fmt.Sprintf("%s %s", countryFlag(code), code))
		}
		builder.WriteString(fmt.Sprintf("| **Origin** | %s |\n", strings.Join(parts, " ")))
	}

	if mediaType == "tv" {
		if rating := usContentRating(details); rating != "" {
			builder.WriteString(fmt.Sprintf("| **Content Rating** | %s |\n", rating))
		}
	}

	if imdb := nestedString(details, "external_ids", "imdb_id"); imdb != "" {
		builder.WriteString(fmt.Sprintf("| **IMDB** | [imdb.com/title/%s](https://www.imdb.com/title/%s/) |\n", imdb, imdb))
	}
	if tvdb := nestedString(details, "external_ids", "tvdb_id"); tvdb != "" {
		builder.WriteString(fmt.Sprintf("| **TVDB** | [thetvdb.com/%s](https://thetvdb.com/series/%s) |\n", tvdb, tvdb))
	}

	if homepage := stringVal(details, "homepage"); homepage != "" {
		builder.WriteString(fmt.Sprintf("| **Homepage** | [%s](%s) |\n", friendlyHomepageName(homepage), homepage))
	}

	return strings.TrimRight(builder.String(), "\n")
}

func buildSeasons(details map[string]any) string {
	raw, ok := details["seasons"].([]any)
	if !ok || len(raw) == 0 {
		return ""
	}

	var builder strings.Builder
	builder.WriteString("## Seasons\n\n")

	for idx, season := range raw {
		s, ok := season.(map[string]any)
		if !ok {
			continue
		}
		name := stringVal(s, "name")
		if name == "" {
			if num, ok := intVal(s, "season_number"); ok {
				name = fmt.Sprintf("Season %d", num)
			} else {
				name = "Season"
			}
		}

		airDate := stringVal(s, "air_date")
		year := "TBA"
		if len(airDate) >= 4 {
			year = airDate[:4]
		}
		vote, _ := floatVal(s, "vote_average")
		overview := strings.TrimSpace(stringVal(s, "overview"))
		episodeCount, _ := intVal(s, "episode_count")
		poster := stringVal(s, "poster_path")

		builder.WriteString(fmt.Sprintf("### %s (%s)", name, year))
		if vote > 0 {
			builder.WriteString(fmt.Sprintf(" • ⭐ %.1f/10", vote))
		}
		builder.WriteString("\n\n")

		if poster != "" {
			builder.WriteString(fmt.Sprintf("![%s](https://image.tmdb.org/t/p/w300%s)\n\n", name, poster))
		}

		if overview != "" {
			builder.WriteString(fmt.Sprintf("_%s_\n\n", overview))
		}

		builder.WriteString(fmt.Sprintf("**Episodes:** %d", episodeCount))

		inProduction := boolVal(details, "in_production")
		isLatest := idx == len(raw)-1

		if isLatest && inProduction {
			builder.WriteString(" • **Status:** Currently Airing\n\n")
		} else {
			builder.WriteString(" • **Status:** ✅ Complete\n\n")
		}

		builder.WriteString("---\n\n")
	}

	out := strings.TrimRight(builder.String(), "\n")
	if !strings.HasSuffix(out, "\n") {
		out += "\n"
	}
	return out
}

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
