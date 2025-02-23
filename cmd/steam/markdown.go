package steam

import (
	"fmt"
	"os"
	"strings"
)

// CreateMarkdownFile generates a markdown file for a Steam game
func CreateMarkdownFile(game Game, details *GameDetails, directory string) error {
	// Ensure the directory exists
	if err := os.MkdirAll(directory, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", directory, err)
	}

	// Use the common function for file path
	filename := getGameFilePath(game.Name, directory)

	// Extract category descriptions
	categories := make([]string, len(details.Categories))
	for i, cat := range details.Categories {
		categories[i] = cat.Description
	}

	// Extract genre descriptions
	genres := make([]string, len(details.Genres))
	for i, genre := range details.Genres {
		genres[i] = genre.Description
	}

	content := fmt.Sprintf(`---
title: %s
playtime: %d
developers: %s
publishers: %s
release_date: %s
cover: %s
---

# %s

## Description
%s

## Details
- **Playtime**: %d minutes
- **Developers**: %s
- **Publishers**: %s
- **Release Date**: %s
- **Categories**: %s
- **Genres**: %s
%s

## Screenshots
%s
`,
		sanitizeFilename(game.Name),
		game.PlaytimeForever,
		strings.Join(details.Developers, ", "),
		strings.Join(details.Publishers, ", "),
		details.ReleaseDate.Date,
		details.HeaderImage,
		game.Name,
		details.Description,
		game.PlaytimeForever,
		strings.Join(details.Developers, ", "),
		strings.Join(details.Publishers, ", "),
		details.ReleaseDate.Date,
		strings.Join(categories, ", "),
		strings.Join(genres, ", "),
		generateMetacriticSection(details),
		generateScreenshotsSection(details),
	)

	return os.WriteFile(filename, []byte(content), 0644)
}

// Helper function to generate the metacritic section
func generateMetacriticSection(details *GameDetails) string {
	if details.Metacritic.Score > 0 {
		return fmt.Sprintf("- **Metacritic Score**: %d\n- **Metacritic URL**: %s",
			details.Metacritic.Score,
			details.Metacritic.URL)
	}
	return ""
}

// Helper function to generate the screenshots section
func generateScreenshotsSection(details *GameDetails) string {
	var sb strings.Builder
	for _, screenshot := range details.Screenshots {
		sb.WriteString(fmt.Sprintf("![](%s)\n", screenshot.PathURL))
	}
	return sb.String()
}
