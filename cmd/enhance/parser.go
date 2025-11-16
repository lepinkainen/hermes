package enhance

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/lepinkainen/hermes/internal/enrichment"
	"gopkg.in/yaml.v3"
)

// Note represents a parsed markdown note with YAML frontmatter.
type Note struct {
	// Frontmatter fields
	Title  string `yaml:"title"`
	Type   string `yaml:"type"`
	Year   int    `yaml:"year"`
	IMDBID string `yaml:"imdb_id,omitempty"`
	TMDBID int    `yaml:"tmdb_id,omitempty"`

	// Raw frontmatter and content
	RawFrontmatter map[string]interface{}
	OriginalBody   string
}

// parseNoteFile parses a markdown file and extracts frontmatter and content.
func parseNoteFile(filePath string) (*Note, error) {
	content, err := readFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	note, err := parseNote(content)
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
func parseNote(content string) (*Note, error) {
	// Split frontmatter and body
	parts := strings.SplitN(content, "---", 3)
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

	return note, nil
}

// HasTMDBData checks if the note already has TMDB data.
func (n *Note) HasTMDBData() bool {
	return n.TMDBID != 0
}

// AddTMDBData adds TMDB enrichment data to the note's frontmatter.
func (n *Note) AddTMDBData(tmdbData *enrichment.TMDBEnrichment) {
	if tmdbData == nil {
		return
	}

	n.RawFrontmatter["tmdb_id"] = tmdbData.TMDBID
	n.RawFrontmatter["tmdb_type"] = tmdbData.TMDBType

	if tmdbData.RuntimeMins > 0 {
		n.RawFrontmatter["runtime_mins"] = tmdbData.RuntimeMins
	}

	if tmdbData.TotalEpisodes > 0 {
		n.RawFrontmatter["total_episodes"] = tmdbData.TotalEpisodes
	}

	if len(tmdbData.GenreTags) > 0 {
		// Merge with existing genres if present
		existingGenres := []string{}
		if genres, ok := n.RawFrontmatter["genres"].([]interface{}); ok {
			for _, g := range genres {
				if genreStr, ok := g.(string); ok {
					existingGenres = append(existingGenres, genreStr)
				}
			}
		} else if genres, ok := n.RawFrontmatter["genres"].([]string); ok {
			existingGenres = genres
		}

		// Combine and deduplicate
		genreSet := make(map[string]bool)
		for _, g := range existingGenres {
			genreSet[g] = true
		}
		for _, g := range tmdbData.GenreTags {
			genreSet[g] = true
		}

		genres := []string{}
		for g := range genreSet {
			genres = append(genres, g)
		}
		n.RawFrontmatter["genres"] = genres
	}

	if tmdbData.CoverPath != "" {
		n.RawFrontmatter["tmdb_cover"] = tmdbData.CoverPath
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

	// Write original body
	sb.WriteString(n.OriginalBody)

	// Append TMDB content if available and requested
	if tmdbData != nil && tmdbData.ContentMarkdown != "" {
		// Check if TMDB content already exists
		hasTMDBContent := strings.Contains(n.OriginalBody, "## TMDB") ||
			strings.Contains(n.OriginalBody, ">[!tmdb")

		if overwrite || !hasTMDBContent {
			sb.WriteString("\n\n")
			sb.WriteString(tmdbData.ContentMarkdown)
		}
	}

	return sb.String()
}

// readFile is a helper to read file content.
// This is separate for easier testing/mocking if needed.
func readFile(path string) (string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(content), nil
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
