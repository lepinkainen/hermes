package enhance

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/lepinkainen/hermes/internal/content"
	"github.com/lepinkainen/hermes/internal/enrichment"
	"gopkg.in/yaml.v3"
)

// Note represents a parsed markdown note with YAML frontmatter.
type Note struct {
	// Frontmatter fields
	Title        string `yaml:"title"`
	Type         string `yaml:"type"`
	Year         int    `yaml:"year"`
	IMDBID       string `yaml:"imdb_id,omitempty"`
	TMDBID       int    `yaml:"tmdb_id,omitempty"`
	LetterboxdID string `yaml:"letterboxd_id,omitempty"`
	Seen         bool   `yaml:"seen,omitempty"`

	// Raw frontmatter and content
	RawFrontmatter map[string]interface{}
	OriginalBody   string
}

// parseNoteFile parses a markdown file and extracts frontmatter and content.
func parseNoteFile(filePath string) (*Note, error) {
	fileContent, err := readFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	note, err := parseNote(fileContent)
	if err != nil {
		return nil, err
	}

	// If title is missing, extract from filename
	if note.Title == "" {
		note.Title = extractTitleFromPath(filePath)
	}

	return note, nil
}

// parseNote parses markdown content with YAML frontmatter.
func parseNote(fileContent string) (*Note, error) {
	// Split frontmatter and body
	parts := strings.SplitN(fileContent, "---", 3)
	if len(parts) < 3 {
		return nil, fmt.Errorf("invalid markdown format: missing frontmatter delimiters")
	}

	frontmatterStr := parts[1]
	body := parts[2]

	// Parse frontmatter
	var frontmatter map[string]interface{}
	if err := yaml.Unmarshal([]byte(frontmatterStr), &frontmatter); err != nil {
		return nil, fmt.Errorf("failed to parse frontmatter: %w", err)
	}

	note := &Note{
		RawFrontmatter: frontmatter,
		OriginalBody:   strings.TrimSpace(body),
	}

	// Extract typed fields
	if title, ok := frontmatter["title"].(string); ok {
		note.Title = title
	}

	// Try to get type from explicit field first
	if noteType, ok := frontmatter["type"].(string); ok {
		note.Type = noteType
	} else {
		// Fall back to detecting from tags array
		note.Type = detectTypeFromTags(frontmatter)
	}

	if year, ok := frontmatter["year"].(int); ok {
		note.Year = year
	}
	if imdbID, ok := frontmatter["imdb_id"].(string); ok {
		note.IMDBID = imdbID
	}
	if tmdbID, ok := frontmatter["tmdb_id"].(int); ok {
		note.TMDBID = tmdbID
	}
	if letterboxdID, ok := frontmatter["letterboxd_id"].(string); ok {
		note.LetterboxdID = letterboxdID
	}
	if seen, ok := frontmatter["seen"].(bool); ok {
		note.Seen = seen
	}

	return note, nil
}

// HasTMDBData checks if the note already has TMDB data in both frontmatter and body.
// Returns true only if both TMDB ID exists in frontmatter AND content markers exist in body.
func (n *Note) HasTMDBData() bool {
	return n.TMDBID != 0 && content.HasTMDBContentMarkers(n.OriginalBody)
}

// NeedsCover checks if the note needs a cover image.
// Returns true if the cover field is missing or empty.
func (n *Note) NeedsCover() bool {
	cover, ok := n.RawFrontmatter["cover"]
	if !ok {
		return true
	}

	coverStr, ok := cover.(string)
	return !ok || coverStr == ""
}

// NeedsMetadata checks if the note needs TMDB metadata fields.
// Returns true if TMDB ID or runtime/genres are missing.
func (n *Note) NeedsMetadata() bool {
	// If no TMDB ID, definitely needs metadata
	if n.TMDBID == 0 {
		return true
	}

	// Check if runtime is missing (for movies/TV shows)
	if n.Type == "movie" || n.Type == "tv" {
		if _, ok := n.RawFrontmatter["runtime"]; !ok {
			return true
		}
	}

	// Check if genres/tags are missing
	tags, ok := n.RawFrontmatter["tags"]
	if !ok {
		return true
	}

	// Check if tags array is empty
	switch v := tags.(type) {
	case []interface{}:
		return len(v) == 0
	case []string:
		return len(v) == 0
	default:
		return true
	}
}

// NeedsContent checks if the note needs TMDB content sections.
// Returns true if TMDB content markers are missing from the body.
func (n *Note) NeedsContent() bool {
	return !content.HasTMDBContentMarkers(n.OriginalBody)
}

// AddTMDBData adds TMDB enrichment data to the note's frontmatter.
func (n *Note) AddTMDBData(tmdbData *enrichment.TMDBEnrichment) {
	if tmdbData == nil {
		return
	}

	n.RawFrontmatter["tmdb_id"] = tmdbData.TMDBID
	n.RawFrontmatter["tmdb_type"] = tmdbData.TMDBType

	if tmdbData.RuntimeMins > 0 {
		n.RawFrontmatter["runtime"] = tmdbData.RuntimeMins
	}

	if tmdbData.TotalEpisodes > 0 {
		n.RawFrontmatter["total_episodes"] = tmdbData.TotalEpisodes
	}

	if len(tmdbData.GenreTags) > 0 {
		// Merge with existing tags if present
		existingTags := []string{}
		if tags, ok := n.RawFrontmatter["tags"].([]interface{}); ok {
			for _, g := range tags {
				if tagStr, ok := g.(string); ok {
					existingTags = append(existingTags, tagStr)
				}
			}
		} else if tags, ok := n.RawFrontmatter["tags"].([]string); ok {
			existingTags = tags
		}

		// Combine and deduplicate
		tagSet := make(map[string]bool)
		for _, g := range existingTags {
			tagSet[g] = true
		}
		for _, g := range tmdbData.GenreTags {
			tagSet[g] = true
		}

		tags := []string{}
		for g := range tagSet {
			tags = append(tags, g)
		}
		n.RawFrontmatter["tags"] = tags
	}

	if tmdbData.CoverPath != "" {
		n.RawFrontmatter["cover"] = tmdbData.CoverPath
	}

	// Set seen flag if movie has any rating but seen field is not already set
	if !n.hasSeenField() && n.hasAnyRating() {
		n.RawFrontmatter["seen"] = true
	}
}

// BuildMarkdown builds the complete markdown content with updated frontmatter and content.
func (n *Note) BuildMarkdown(originalContent string, tmdbData *enrichment.TMDBEnrichment, overwrite bool) string {
	var sb strings.Builder

	// Write frontmatter
	sb.WriteString("---\n")
	frontmatterBytes, err := yaml.Marshal(n.RawFrontmatter)
	if err != nil {
		// Fallback to original if marshaling fails
		return originalContent
	}
	sb.Write(frontmatterBytes)
	sb.WriteString("---\n\n")

	// Handle TMDB content with marker-based replacement
	body := n.OriginalBody
	if tmdbData != nil && tmdbData.ContentMarkdown != "" {
		if content.HasTMDBContentMarkers(body) {
			// Replace existing TMDB content between markers
			if overwrite {
				body = content.ReplaceTMDBContent(body, tmdbData.ContentMarkdown)
			}
		} else {
			// No markers exist - append wrapped content
			wrappedContent := content.WrapWithMarkers(tmdbData.ContentMarkdown)
			body = strings.TrimRight(body, "\n")
			if body != "" {
				body += "\n\n"
			}
			body += wrappedContent
		}
	}

	sb.WriteString(body)
	return sb.String()
}

// readFile is a helper to read file content.
// This is separate for easier testing/mocking if needed.
func readFile(path string) (string, error) {
	fileContent, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(fileContent), nil
}

// extractTitleFromPath extracts a title from the file path.
// For example: "/path/to/Lilo & Stitch.md" -> "Lilo & Stitch"
func extractTitleFromPath(filePath string) string {
	// Get the base filename
	filename := filepath.Base(filePath)
	// Remove the .md extension
	title := strings.TrimSuffix(filename, filepath.Ext(filename))
	return title
}

// detectTypeFromTags attempts to detect if this is a movie or TV show from tags.
func detectTypeFromTags(frontmatter map[string]interface{}) string {
	tags, ok := frontmatter["tags"]
	if !ok {
		return ""
	}

	// Handle []interface{} (common YAML array representation)
	if tagSlice, ok := tags.([]interface{}); ok {
		for _, tag := range tagSlice {
			if tagStr, ok := tag.(string); ok {
				if tagStr == "movie" {
					return "movie"
				}
				if tagStr == "tv" || tagStr == "tv-show" || tagStr == "series" {
					return "tv"
				}
			}
		}
	}

	// Handle []string
	if tagSlice, ok := tags.([]string); ok {
		for _, tag := range tagSlice {
			if tag == "movie" {
				return "movie"
			}
			if tag == "tv" || tag == "tv-show" || tag == "series" {
				return "tv"
			}
		}
	}

	return ""
}

// MediaIDs represents all external IDs found in the frontmatter.
type MediaIDs struct {
	TMDBID       int    `yaml:"tmdb_id,omitempty"`
	IMDBID       string `yaml:"imdb_id,omitempty"`
	LetterboxdID string `yaml:"letterboxd_id,omitempty"`
}

// GetMediaIDs extracts all external media IDs from the frontmatter.
// Returns a struct containing any TMDB, IMDB, or Letterboxd IDs found.
func (n *Note) GetMediaIDs() MediaIDs {
	ids := MediaIDs{
		TMDBID:       n.TMDBID,
		IMDBID:       n.IMDBID,
		LetterboxdID: n.LetterboxdID,
	}
	return ids
}

// HasAnyID checks if the note has any external ID (TMDB, IMDB, or Letterboxd).
// Returns true if at least one ID is present and non-empty.
func (n *Note) HasAnyID() bool {
	ids := n.GetMediaIDs()
	return ids.TMDBID != 0 || ids.IMDBID != "" || ids.LetterboxdID != ""
}

// GetIDSummary returns a formatted string summary of all available IDs.
// Useful for logging and debugging.
func (n *Note) GetIDSummary() string {
	ids := n.GetMediaIDs()
	var summary []string
	if ids.TMDBID != 0 {
		summary = append(summary, fmt.Sprintf("tmdb:%d", ids.TMDBID))
	}
	if ids.IMDBID != "" {
		summary = append(summary, fmt.Sprintf("imdb:%s", ids.IMDBID))
	}
	if ids.LetterboxdID != "" {
		summary = append(summary, fmt.Sprintf("letterboxd:%s", ids.LetterboxdID))
	}
	if len(summary) == 0 {
		return "no IDs"
	}
	return strings.Join(summary, ", ")
}

// hasSeenField checks if the note already has a seen field in frontmatter.
func (n *Note) hasSeenField() bool {
	_, exists := n.RawFrontmatter["seen"]
	return exists
}

// hasAnyRating checks if the note has any rating field (imdb_rating, my_rating, or letterboxd_rating).
func (n *Note) hasAnyRating() bool {
	// Check for IMDb rating
	if imdbRating, ok := n.RawFrontmatter["imdb_rating"]; ok {
		if rating, isFloat := imdbRating.(float64); isFloat && rating > 0 {
			return true
		}
		if rating, isInt := imdbRating.(int); isInt && rating > 0 {
			return true
		}
	}
	
	// Check for my_rating
	if myRating, ok := n.RawFrontmatter["my_rating"]; ok {
		if rating, isInt := myRating.(int); isInt && rating > 0 {
			return true
		}
		if rating, isFloat := myRating.(float64); isFloat && rating > 0 {
			return true
		}
	}
	
	// Check for letterboxd_rating
	if letterboxdRating, ok := n.RawFrontmatter["letterboxd_rating"]; ok {
		if rating, isFloat := letterboxdRating.(float64); isFloat && rating > 0 {
			return true
		}
		if rating, isInt := letterboxdRating.(int); isInt && rating > 0 {
			return true
		}
	}
	
	return false
}
