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

	// Use the MarkdownBuilder to construct the document
	mb := fileutil.NewMarkdownBuilder()

	// Add basic metadata
	mb.AddTitle(fileutil.SanitizeFilename(game.Name))
	mb.AddType("game")
	mb.AddField("playtime", game.PlaytimeForever)

	// Add duration (formatted playtime)
	if game.PlaytimeForever > 0 {
		mb.AddDuration(game.PlaytimeForever)
	}

	// Add release date
	mb.AddField("release_date", details.ReleaseDate.Date)

	// Add cover image
	mb.AddField("cover", details.HeaderImage)

	// Add developers as an array
	if len(details.Developers) > 0 {
		mb.AddStringArray("developers", details.Developers)
	}

	// Add publishers as an array
	if len(details.Publishers) > 0 {
		mb.AddStringArray("publishers", details.Publishers)
	}

	// Add categories as an array
	if len(details.Categories) > 0 {
		categories := make([]string, len(details.Categories))
		for i, cat := range details.Categories {
			categories[i] = cat.Description
		}
		mb.AddStringArray("categories", categories)
	}

	// Add genres as an array
	if len(details.Genres) > 0 {
		genres := make([]string, len(details.Genres))
		for i, genre := range details.Genres {
			genres[i] = genre.Description
		}
		mb.AddStringArray("genres", genres)
	}

	// Add tags
	mb.AddTags("steam/game")

	// Add metacritic info if available
	if details.Metacritic.Score > 0 {
		mb.AddField("metacritic_score", details.Metacritic.Score)
		mb.AddField("metacritic_url", details.Metacritic.URL)
	}

	// Add title as heading
	mb.AddParagraph(fmt.Sprintf("# %s", game.Name))

	// Add cover image if available
	if details.HeaderImage != "" {
		mb.AddImage(details.HeaderImage)
	}

	// Add description in a callout if available
	if details.Description != "" {
		mb.AddCallout("summary", "Description", details.Description)
	}

	// Add game details in a callout
	var detailsContent strings.Builder

	// Playtime
	fmt.Fprintf(&detailsContent, "- **Playtime**: %d minutes", game.PlaytimeForever)
	if hours := game.PlaytimeForever / 60; hours > 0 {
		fmt.Fprintf(&detailsContent, " (%dh %dm)", hours, game.PlaytimeForever%60)
	}
	detailsContent.WriteString("\n")

	// Developers
	if len(details.Developers) > 0 {
		fmt.Fprintf(&detailsContent, "- **Developers**: %s\n", strings.Join(details.Developers, ", "))
	}

	// Publishers
	if len(details.Publishers) > 0 {
		fmt.Fprintf(&detailsContent, "- **Publishers**: %s\n", strings.Join(details.Publishers, ", "))
	}

	// Release date
	fmt.Fprintf(&detailsContent, "- **Release Date**: %s\n", details.ReleaseDate.Date)

	// Categories
	if len(details.Categories) > 0 {
		categories := make([]string, len(details.Categories))
		for i, cat := range details.Categories {
			categories[i] = cat.Description
		}
		fmt.Fprintf(&detailsContent, "- **Categories**: %s\n", strings.Join(categories, ", "))
	}

	// Genres
	if len(details.Genres) > 0 {
		genres := make([]string, len(details.Genres))
		for i, genre := range details.Genres {
			genres[i] = genre.Description
		}
		fmt.Fprintf(&detailsContent, "- **Genres**: %s\n", strings.Join(genres, ", "))
	}

	// Metacritic
	if details.Metacritic.Score > 0 {
		fmt.Fprintf(&detailsContent, "- **Metacritic Score**: %d\n", details.Metacritic.Score)
		fmt.Fprintf(&detailsContent, "- **Metacritic URL**: [View on Metacritic](%s)\n", details.Metacritic.URL)
	}

	mb.AddCallout("info", "Game Details", detailsContent.String())

	// Add screenshots section
	if len(details.Screenshots) > 0 {
		mb.AddParagraph("## Screenshots")

		for _, screenshot := range details.Screenshots {
			mb.AddImage(screenshot.PathURL)
		}
	}

	// Write content to file with overwrite logic
	content := mb.Build()

	// Trim trailing newlines to match expected format in tests
	content = strings.TrimRight(content, "\n") + "\n"

	written, err := fileutil.WriteFileWithOverwrite(filename, []byte(content), 0644, config.OverwriteFiles)
	if err != nil {
		return err
	}

	if !written {
		log.Debugf("Skipped existing file: %s", filename)
	} else {
		log.Infof("Wrote %s", filename)
	}

	return nil
}

// Helper functions are no longer needed as their functionality is integrated into the main function
