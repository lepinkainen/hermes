package content

import (
	"fmt"
	"strings"
)

// LetterboxdMovieDetails contains data needed to build Letterboxd content sections.
type LetterboxdMovieDetails struct {
	Title         string
	Year          int
	Rating        float64 // 0.5-5 scale
	DateWatched   string
	Runtime       int
	Director      string
	Genres        []string
	Cast          []string
	Description   string
	LetterboxdURI string
	LetterboxdID  string
	ImdbID        string
}

// BuildLetterboxdContent generates markdown content from Letterboxd details.
func BuildLetterboxdContent(details *LetterboxdMovieDetails, sections []string) string {
	if len(sections) == 0 {
		sections = []string{"info", "description", "cast"}
	}

	var blocks []string
	for _, section := range sections {
		switch section {
		case "info":
			if block := buildLetterboxdInfo(details); block != "" {
				blocks = append(blocks, block)
			}
		case "description":
			if block := buildLetterboxdDescription(details); block != "" {
				blocks = append(blocks, block)
			}
		case "cast":
			if block := buildLetterboxdCast(details); block != "" {
				blocks = append(blocks, block)
			}
		}
	}

	return strings.Join(blocks, "\n\n")
}

func buildLetterboxdInfo(details *LetterboxdMovieDetails) string {
	var builder strings.Builder
	builder.WriteString("## Letterboxd Details\n\n")
	builder.WriteString("| | |\n")
	builder.WriteString("|---|---|\n")

	// Title and Year
	if details.Title != "" {
		builder.WriteString(fmt.Sprintf("| **Title** | %s (%d) |\n", details.Title, details.Year))
	}

	// Rating (convert to stars)
	if details.Rating > 0 {
		stars := buildStarRating(details.Rating)
		builder.WriteString(fmt.Sprintf("| **My Rating** | %s (%.1f/5) |\n", stars, details.Rating))
	}

	// Date watched
	if details.DateWatched != "" {
		builder.WriteString(fmt.Sprintf("| **Date Watched** | %s |\n", details.DateWatched))
	}

	// Runtime
	if details.Runtime > 0 {
		builder.WriteString(fmt.Sprintf("| **Runtime** | %d min |\n", details.Runtime))
	}

	// Director
	if details.Director != "" {
		builder.WriteString(fmt.Sprintf("| **Director** | %s |\n", details.Director))
	}

	// Genres
	if len(details.Genres) > 0 {
		builder.WriteString(fmt.Sprintf("| **Genres** | %s |\n", strings.Join(details.Genres, ", ")))
	}

	// Letterboxd link
	if details.LetterboxdURI != "" {
		displayText := extractLetterboxdPath(details.LetterboxdURI)
		builder.WriteString(fmt.Sprintf("| **Letterboxd** | [%s](%s) |\n", displayText, details.LetterboxdURI))
	}

	// IMDb link
	if details.ImdbID != "" {
		builder.WriteString(fmt.Sprintf("| **IMDb** | [%s](https://www.imdb.com/title/%s/) |\n", details.ImdbID, details.ImdbID))
	}

	return strings.TrimRight(builder.String(), "\n")
}

func buildLetterboxdDescription(details *LetterboxdMovieDetails) string {
	if details.Description == "" {
		return ""
	}

	var builder strings.Builder
	builder.WriteString("## Description\n\n")
	builder.WriteString(strings.TrimSpace(details.Description))
	builder.WriteString("\n")

	return builder.String()
}

func buildLetterboxdCast(details *LetterboxdMovieDetails) string {
	if len(details.Cast) == 0 {
		return ""
	}

	var builder strings.Builder
	builder.WriteString("## Cast\n\n")

	for _, actor := range details.Cast {
		builder.WriteString(fmt.Sprintf("- %s\n", actor))
	}

	return strings.TrimRight(builder.String(), "\n")
}

// buildStarRating converts a rating (0.5-5 scale) to star emojis.
func buildStarRating(rating float64) string {
	fullStars := int(rating)
	hasHalfStar := (rating-float64(fullStars)) >= 0.25 && (rating-float64(fullStars)) < 0.75

	var builder strings.Builder
	for i := 0; i < fullStars; i++ {
		builder.WriteString("⭐")
	}
	if hasHalfStar {
		builder.WriteString("½")
	}

	return builder.String()
}

// extractLetterboxdPath extracts a clean display text from a Letterboxd URI.
// For full URLs like "https://letterboxd.com/film/the-godfather/", returns "film/the-godfather"
// For short URLs like "https://boxd.it/2bg8", returns "boxd.it/2bg8"
func extractLetterboxdPath(uri string) string {
	// Remove protocol
	display := strings.TrimPrefix(uri, "https://")
	display = strings.TrimPrefix(display, "http://")

	// For short URLs, keep as is: "boxd.it/xyz"
	if strings.HasPrefix(display, "boxd.it/") {
		return display
	}

	// For full URLs, extract the film path: "letterboxd.com/film/movie-name/" -> "film/movie-name"
	if strings.HasPrefix(display, "letterboxd.com/film/") {
		filmPath := strings.TrimPrefix(display, "letterboxd.com/")
		filmPath = strings.TrimSuffix(filmPath, "/")
		return filmPath
	}

	// Fallback: return as-is
	return display
}
