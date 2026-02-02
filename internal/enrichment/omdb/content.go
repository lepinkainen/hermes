package omdb

import (
	"fmt"
	"strings"
)

// BuildRatingsTable generates a markdown table with ratings data
func BuildRatingsTable(ratings *RatingsEnrichment) string {
	if ratings == nil {
		return ""
	}

	var rows []string

	// Add IMDb rating if available
	if ratings.IMDbRating > 0 {
		rows = append(rows, fmt.Sprintf("| IMDb | ⭐ %.1f/10 |", ratings.IMDbRating))
	}

	// Add Rotten Tomatoes rating if available
	if ratings.RottenTomatoes != "" {
		rows = append(rows, fmt.Sprintf("| Rotten Tomatoes | 🍅 %s |", ratings.RottenTomatoes))
	}

	// Add Metacritic rating if available
	if ratings.Metacritic > 0 {
		rows = append(rows, fmt.Sprintf("| Metacritic | 📊 %d/100 |", ratings.Metacritic))
	}

	// If no ratings available, return empty string
	if len(rows) == 0 {
		return ""
	}

	// Build the table
	var sb strings.Builder
	sb.WriteString("## Ratings\n\n")
	sb.WriteString("| Source | Score |\n")
	sb.WriteString("|--------|-------|\n")
	for _, row := range rows {
		sb.WriteString(row)
		sb.WriteString("\n")
	}

	return sb.String()
}
