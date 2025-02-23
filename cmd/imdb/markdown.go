package imdb

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

// writeMovieToMarkdown writes movie info to a markdown file
func writeMovieToMarkdown(movie MovieSeen, directory string) error {
	// Sanitize movie title for filename
	filename := sanitizeFilename(movie.Title) + ".md"
	filePath := filepath.Join(directory, filename)

	// Create frontmatter content
	var frontmatter strings.Builder

	frontmatter.WriteString("---\n")

	// Handle titles - remove problematic characters and handle original titles
	movie.Title = sanitizeTitle(movie.Title)
	frontmatter.WriteString(fmt.Sprintf("title: \"%s\"\n", movie.Title))
	if movie.OriginalTitle != "" && movie.OriginalTitle != movie.Title {
		movie.OriginalTitle = sanitizeTitle(movie.OriginalTitle)
		frontmatter.WriteString(fmt.Sprintf("original_title: \"%s\"\n", movie.OriginalTitle))
	}

	// Add type-specific metadata
	frontmatter.WriteString(fmt.Sprintf("type: %s\n", mapTypeToType(movie.TitleType)))

	// Basic metadata
	frontmatter.WriteString(fmt.Sprintf("imdb_id: %s\n", movie.ImdbId))
	frontmatter.WriteString(fmt.Sprintf("year: %d\n", movie.Year))
	frontmatter.WriteString(fmt.Sprintf("imdb_rating: %.1f\n", movie.IMDbRating))
	frontmatter.WriteString(fmt.Sprintf("my_rating: %d\n", movie.MyRating))

	// Format date in a more readable way
	if date, err := time.Parse("2006-01-02", movie.DateRated); err == nil {
		frontmatter.WriteString(fmt.Sprintf("date_rated: %s\n", date.Format("2006-01-02")))
	}

	if movie.RuntimeMins > 0 {
		frontmatter.WriteString(fmt.Sprintf("runtime_mins: %d\n", movie.RuntimeMins))
		// Add human-readable duration
		hours := movie.RuntimeMins / 60
		mins := movie.RuntimeMins % 60
		if hours > 0 {
			frontmatter.WriteString(fmt.Sprintf("duration: %dh %dm\n", hours, mins))
		} else {
			frontmatter.WriteString(fmt.Sprintf("duration: %dm\n", mins))
		}
	}

	// Add genres as an array
	if len(movie.Genres) > 0 {
		frontmatter.WriteString("genres:\n")
		for _, genre := range movie.Genres {
			if genre != "" {
				frontmatter.WriteString(fmt.Sprintf("  - %s\n", strings.TrimSpace(genre)))
			}
		}
	}

	// Add directors as an array
	if len(movie.Directors) > 0 {
		frontmatter.WriteString("directors:\n")
		for _, director := range movie.Directors {
			if director != "" {
				frontmatter.WriteString(fmt.Sprintf("  - \"%s\"\n", strings.TrimSpace(director)))
			}
		}
	}

	// Add Obsidian-specific tags
	tags := []string{
		mapTypeToTag(movie.TitleType),            // e.g., #imdb/movie
		fmt.Sprintf("rating/%d", movie.MyRating), // e.g., #rating/8
	}

	// Add decade tag
	decade := (movie.Year / 10) * 10
	tags = append(tags, fmt.Sprintf("year/%ds", decade)) // e.g., #year/1990s

	frontmatter.WriteString("tags:\n")
	for _, tag := range tags {
		frontmatter.WriteString(fmt.Sprintf("  - %s\n", tag))
	}

	// Add content rating if available
	if movie.ContentRated != "" {
		frontmatter.WriteString(fmt.Sprintf("content_rating: \"%s\"\n", movie.ContentRated))
	}

	// Add awards if available
	if movie.Awards != "" {
		frontmatter.WriteString(fmt.Sprintf("awards: \"%s\"\n", movie.Awards))
	}

	frontmatter.WriteString("---\n\n")

	// Content section
	var content strings.Builder

	// Add poster image if available
	if movie.PosterURL != "" {
		content.WriteString(fmt.Sprintf("![[%s]]\n\n", movie.PosterURL))
	}

	// Add plot summary in a callout if available
	if movie.Plot != "" {
		content.WriteString(fmt.Sprintf(">[!summary]- Plot\n> %s\n\n", movie.Plot))
	}

	// Add awards in a callout if available
	if movie.Awards != "" {
		content.WriteString(fmt.Sprintf(">[!award]- Awards\n> %s\n\n", movie.Awards))
	}

	// Add IMDb link as a button (Obsidian feature)
	content.WriteString(fmt.Sprintf(">[!info]- IMDb\n> [View on IMDb](%s)\n\n", movie.URL))

	// Create directory if it doesn't exist
	if err := os.MkdirAll(directory, 0755); err != nil {
		return err
	}

	// Write content to file
	return os.WriteFile(filePath, []byte(frontmatter.String()+content.String()), 0644)
}

func sanitizeTitle(title string) string {
	return strings.ReplaceAll(title, ":", "")
}

func mapTypeToType(titleType string) string {
	switch titleType {
	case "Movie":
		return "movie"
	case "TV Series":
		return "tv-series"
	case "TV Mini Series":
		return "miniseries"
	// ... add other types as needed
	default:
		return strings.ToLower(titleType)
	}
}

// mapTypeToTag maps a imdb title type to a markdown tag
func mapTypeToTag(titleType string) string {
	switch titleType {
	case "Video Game":
		return "imdb/videogame"
	case "TV Series":
		return "imdb/tv-series"
	case "TV Special":
		return "imdb/tv-special"
	case "TV Mini Series":
		return "imdb/miniseries"
	case "TV Episode":
		return "imdb/tv-episode"
	case "TV Movie":
		return "imdb/tv-movie"
	case "TV Short":
		return "imdb/tv-short"
	case "Movie":
		return "imdb/movie"
	case "Video":
		return "imdb/video"
	case "Short":
		return "imdb/short-movie"
	case "Podcast Series":
		return "imdb/podcast"
	case "Podcast Episode":
		return "imdb/podcast-episode"
	default:
		log.Warnf("Unknown title type '%s'\n", titleType)
		return "UNKNOWN"
	}
}

func (m *MovieSeen) Validate() error {
	if m.ImdbId == "" {
		return fmt.Errorf("missing required field: ImdbId")
	}
	if m.Title == "" {
		return fmt.Errorf("missing required field: Title")
	}
	if m.Year < 1800 || m.Year > time.Now().Year()+5 {
		return fmt.Errorf("invalid year: %d", m.Year)
	}
	return nil
}

func sanitizeFilename(filename string) string {
	// Replace invalid filename characters
	invalid := regexp.MustCompile(`[<>:"/\\|?*]`)
	safe := invalid.ReplaceAllString(filename, "-")
	// Remove multiple dashes
	multiDash := regexp.MustCompile(`-+`)
	safe = multiDash.ReplaceAllString(safe, "-")
	// Trim spaces and dashes from ends
	return strings.Trim(safe, " -")
}
