package obsidian

import "strings"

// DetectMediaType determines the media type from frontmatter.
// Checks tmdb_type field first, then falls back to detecting from tags.
// Returns "movie", "tv", "game", or empty string if type cannot be determined.
func DetectMediaType(fm *Frontmatter) string {
	if fm == nil {
		return ""
	}

	// Check tmdb_type field first
	if mediaType := fm.GetString("tmdb_type"); mediaType != "" {
		return mediaType
	}

	// Fall back to detecting from tags
	return detectTypeFromTags(fm)
}

// DetectMediaTypeFromTags determines the media type using only tag values.
// Returns "movie", "tv", "game", or empty string if no tag hints are present.
func DetectMediaTypeFromTags(fm *Frontmatter) string {
	if fm == nil {
		return ""
	}
	return detectTypeFromTags(fm)
}

// detectTypeFromTags attempts to determine media type from the tags array.
// Returns "movie", "tv", "game", or empty string if no type hints are present.
func detectTypeFromTags(fm *Frontmatter) string {
	moviePresent := false
	tvPresent := false
	gamePresent := false

	for _, tagStr := range fm.GetStringArray("tags") {
		tag := strings.ToLower(strings.TrimSpace(tagStr))
		switch {
		case tag == "movie" || strings.HasPrefix(tag, "movie/"):
			moviePresent = true
		case tag == "tv" || tag == "tv-show" || tag == "series" || strings.HasPrefix(tag, "tv/"):
			tvPresent = true
		case tag == "game" || strings.HasPrefix(tag, "game/") || strings.HasPrefix(tag, "steam/"):
			gamePresent = true
		}
	}

	switch {
	case gamePresent:
		return "game"
	case tvPresent:
		return "tv"
	case moviePresent:
		return "movie"
	default:
		return ""
	}
}
