package steam

import (
	"fmt"
	"os"
	"strings"

	"github.com/lepinkainen/hermes/internal/config"
	"github.com/lepinkainen/hermes/internal/fileutil"
	log "github.com/sirupsen/logrus"
)

// CreateMarkdownFile generates a markdown file for a Steam game
func CreateMarkdownFile(game Game, details *GameDetails, directory string) error {
	// Ensure the directory exists
	if err := os.MkdirAll(directory, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", directory, err)
	}

	// Use the common function for file path
	filename := fileutil.GetMarkdownFilePath(game.Name, directory)

	// Create frontmatter content
	var frontmatter strings.Builder

	frontmatter.WriteString("---\n")
	fmt.Fprintf(&frontmatter, "title: \"%s\"\n", fileutil.SanitizeFilename(game.Name))
	frontmatter.WriteString("type: game\n")
	fmt.Fprintf(&frontmatter, "playtime: %d\n", game.PlaytimeForever)

	// Format playtime in a more readable way
	hours := game.PlaytimeForever / 60
	mins := game.PlaytimeForever % 60
	if hours > 0 {
		fmt.Fprintf(&frontmatter, "duration: %dh %dm\n", hours, mins)
	} else {
		fmt.Fprintf(&frontmatter, "duration: %dm\n", mins)
	}

	// Add release date
	fmt.Fprintf(&frontmatter, "release_date: \"%s\"\n", details.ReleaseDate.Date)

	// Add cover image
	fmt.Fprintf(&frontmatter, "cover: \"%s\"\n", details.HeaderImage)

	// Add developers as an array
	if len(details.Developers) > 0 {
		frontmatter.WriteString("developers:\n")
		for _, developer := range details.Developers {
			if developer != "" {
				fmt.Fprintf(&frontmatter, "  - \"%s\"\n", strings.TrimSpace(developer))
			}
		}
	}

	// Add publishers as an array
	if len(details.Publishers) > 0 {
		frontmatter.WriteString("publishers:\n")
		for _, publisher := range details.Publishers {
			if publisher != "" {
				fmt.Fprintf(&frontmatter, "  - \"%s\"\n", strings.TrimSpace(publisher))
			}
		}
	}

	// Add categories as an array
	if len(details.Categories) > 0 {
		frontmatter.WriteString("categories:\n")
		for _, category := range details.Categories {
			if category.Description != "" {
				fmt.Fprintf(&frontmatter, "  - \"%s\"\n", strings.TrimSpace(category.Description))
			}
		}
	}

	// Add genres as an array
	if len(details.Genres) > 0 {
		frontmatter.WriteString("genres:\n")
		for _, genre := range details.Genres {
			if genre.Description != "" {
				fmt.Fprintf(&frontmatter, "  - \"%s\"\n", strings.TrimSpace(genre.Description))
			}
		}
	}

	// Add tags
	frontmatter.WriteString("tags:\n")
	frontmatter.WriteString("  - steam/game\n")

	// Add metacritic info if available
	if details.Metacritic.Score > 0 {
		fmt.Fprintf(&frontmatter, "metacritic_score: %d\n", details.Metacritic.Score)
		fmt.Fprintf(&frontmatter, "metacritic_url: \"%s\"\n", details.Metacritic.URL)
	}

	frontmatter.WriteString("---\n\n")

	// Content section
	var content strings.Builder

	// Add title
	fmt.Fprintf(&content, "# %s\n\n", game.Name)

	// Add cover image if available
	if details.HeaderImage != "" {
		fmt.Fprintf(&content, "![](%s)\n\n", details.HeaderImage)
	}

	// Add description in a callout if available
	if details.Description != "" {
		content.WriteString("> [!summary]- Description\n> ")
		content.WriteString(details.Description)
		content.WriteString("\n\n")
	}

	// Add details in a callout
	content.WriteString("> [!info]- Game Details\n>\n")
	fmt.Fprintf(&content, "> - **Playtime**: %d minutes", game.PlaytimeForever)
	if hours > 0 {
		fmt.Fprintf(&content, " (%dh %dm)", hours, mins)
	}
	content.WriteString("\n")

	if len(details.Developers) > 0 {
		fmt.Fprintf(&content, "> - **Developers**: %s\n", strings.Join(details.Developers, ", "))
	}

	if len(details.Publishers) > 0 {
		fmt.Fprintf(&content, "> - **Publishers**: %s\n", strings.Join(details.Publishers, ", "))
	}

	fmt.Fprintf(&content, "> - **Release Date**: %s\n", details.ReleaseDate.Date)

	// Add categories and genres
	if len(details.Categories) > 0 {
		categories := make([]string, len(details.Categories))
		for i, cat := range details.Categories {
			categories[i] = cat.Description
		}
		fmt.Fprintf(&content, "> - **Categories**: %s\n", strings.Join(categories, ", "))
	}

	if len(details.Genres) > 0 {
		genres := make([]string, len(details.Genres))
		for i, genre := range details.Genres {
			genres[i] = genre.Description
		}
		fmt.Fprintf(&content, "> - **Genres**: %s\n", strings.Join(genres, ", "))
	}

	// Add metacritic info
	if details.Metacritic.Score > 0 {
		fmt.Fprintf(&content, "> - **Metacritic Score**: %d\n", details.Metacritic.Score)
		fmt.Fprintf(&content, "> - **Metacritic URL**: [View on Metacritic](%s)\n", details.Metacritic.URL)
	}

	content.WriteString("\n")

	// Add screenshots section
	if len(details.Screenshots) > 0 {
		content.WriteString("## Screenshots\n\n")
		for i, screenshot := range details.Screenshots {
			if i == len(details.Screenshots)-1 {
				// Last screenshot, don't add an extra newline
				fmt.Fprintf(&content, "![](%s)\n", screenshot.PathURL)
			} else {
				fmt.Fprintf(&content, "![](%s)\n\n", screenshot.PathURL)
			}
		}
	}

	// Write content to file with overwrite logic
	written, err := fileutil.WriteFileWithOverwrite(filename, []byte(frontmatter.String()+content.String()), 0644, config.OverwriteFiles)
	if err != nil {
		return err
	}

	if !written {
		log.Debugf("Skipped existing file: %s", filename)
	}

	return nil
}

// Helper functions are no longer needed as their functionality is integrated into the main function
