package content

import (
	"fmt"
	"strings"
)

// IMDbMovieDetails holds IMDb-specific movie/TV details for content generation
type IMDbMovieDetails struct {
	Title         string
	OriginalTitle string
	Year          int
	TitleType     string // "Movie", "TV Series", etc.
	MyRating      int    // 1-10
	IMDbRating    float64
	DateRated     string
	Runtime       int
	Directors     []string
	Genres        []string
	ContentRating string
	Awards        string
	Plot          string
	IMDbID        string
	URL           string
}

// BuildIMDbContent generates IMDb content sections
// Sections: "info", "plot", "awards"
func BuildIMDbContent(details *IMDbMovieDetails, sections []string) string {
	if details == nil {
		return ""
	}

	var builder strings.Builder
	sectionMap := make(map[string]bool)
	for _, s := range sections {
		sectionMap[s] = true
	}

	// Info section (always first if present)
	if sectionMap["info"] {
		if info := buildIMDbInfoSection(details); info != "" {
			builder.WriteString(info)
			builder.WriteString("\n\n")
		}
	}

	// Plot section
	if sectionMap["plot"] && details.Plot != "" {
		if plot := buildIMDbPlotSection(details); plot != "" {
			builder.WriteString(plot)
			builder.WriteString("\n\n\n")
		}
	}

	// Awards section
	if sectionMap["awards"] && details.Awards != "" {
		if awards := buildIMDbAwardsSection(details); awards != "" {
			builder.WriteString(awards)
			builder.WriteString("\n\n")
		}
	}

	return strings.TrimSpace(builder.String())
}

// buildIMDbInfoSection creates an info table with key IMDb metadata
func buildIMDbInfoSection(details *IMDbMovieDetails) string {
	var builder strings.Builder

	builder.WriteString("## IMDb Details\n\n")
	builder.WriteString("| | |\n")
	builder.WriteString("|---|---|\n")

	// Title
	titleStr := fmt.Sprintf("%s (%d)", details.Title, details.Year)
	if details.OriginalTitle != "" && details.OriginalTitle != details.Title {
		titleStr = fmt.Sprintf("%s / %s (%d)", details.Title, details.OriginalTitle, details.Year)
	}
	builder.WriteString(fmt.Sprintf("| **Title** | %s |\n", titleStr))

	// My Rating
	if details.MyRating > 0 {
		builder.WriteString(fmt.Sprintf("| **My Rating** | %s (%d/10) |\n",
			buildStarRating10(details.MyRating), details.MyRating))
	}

	// IMDb Rating
	if details.IMDbRating > 0 {
		builder.WriteString(fmt.Sprintf("| **IMDb Rating** | %.1f/10 |\n", details.IMDbRating))
	}

	// Date Rated
	if details.DateRated != "" {
		builder.WriteString(fmt.Sprintf("| **Date Rated** | %s |\n", details.DateRated))
	}

	// Runtime
	if details.Runtime > 0 {
		builder.WriteString(fmt.Sprintf("| **Runtime** | %d min |\n", details.Runtime))
	}

	// Directors
	if len(details.Directors) > 0 {
		directorLabel := "Director"
		if len(details.Directors) > 1 {
			directorLabel = "Directors"
		}
		builder.WriteString(fmt.Sprintf("| **%s** | %s |\n",
			directorLabel, strings.Join(details.Directors, ", ")))
	}

	// Genres
	if len(details.Genres) > 0 {
		builder.WriteString(fmt.Sprintf("| **Genres** | %s |\n",
			strings.Join(details.Genres, ", ")))
	}

	// Content Rating
	if details.ContentRating != "" {
		builder.WriteString(fmt.Sprintf("| **Content Rating** | %s |\n", details.ContentRating))
	}

	// Awards (brief)
	if details.Awards != "" {
		builder.WriteString(fmt.Sprintf("| **Awards** | %s |\n", details.Awards))
	}

	// IMDb Link
	if details.IMDbID != "" {
		imdbURL := fmt.Sprintf("https://www.imdb.com/title/%s/", details.IMDbID)
		builder.WriteString(fmt.Sprintf("| **IMDb** | [%s](%s) |\n", details.IMDbID, imdbURL))
	}

	return builder.String()
}

// buildIMDbPlotSection creates a plot/description section
func buildIMDbPlotSection(details *IMDbMovieDetails) string {
	if details.Plot == "" {
		return ""
	}

	var builder strings.Builder
	builder.WriteString("## Plot\n\n")
	builder.WriteString(details.Plot)
	return builder.String()
}

// buildIMDbAwardsSection creates an awards section
func buildIMDbAwardsSection(details *IMDbMovieDetails) string {
	if details.Awards == "" {
		return ""
	}

	var builder strings.Builder
	builder.WriteString("## Awards\n\n")
	builder.WriteString(details.Awards)
	return builder.String()
}

// buildStarRating10 converts a 1-10 rating to star emojis
func buildStarRating10(rating int) string {
	if rating < 1 || rating > 10 {
		return ""
	}

	// Convert to 5-star scale (1-10 -> 0.5-5.0)
	fiveStarRating := float64(rating) / 2.0

	fullStars := int(fiveStarRating)
	hasHalf := (fiveStarRating - float64(fullStars)) >= 0.5

	var builder strings.Builder
	for i := 0; i < fullStars; i++ {
		builder.WriteString("⭐")
	}
	if hasHalf {
		builder.WriteString("½")
	}

	return builder.String()
}
