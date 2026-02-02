package steam

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/lepinkainen/hermes/internal/config"
	"github.com/lepinkainen/hermes/internal/content"
	"github.com/lepinkainen/hermes/internal/fileutil"
	"github.com/lepinkainen/hermes/internal/obsidian"
)

const defaultCoverWidth = 500 // Vertical game cover, wider than movie posters

// CreateMarkdownFile generates a markdown file for a Steam game
func CreateMarkdownFile(game Game, details *GameDetails, directory string) error {
	// Use the common function for file path
	filename := fileutil.GetMarkdownFilePath(game.Name, directory)

	// Create frontmatter using obsidian.Note
	fm := obsidian.NewFrontmatterWithTitle(fileutil.SanitizeFilename(game.Name))

	// Add basic metadata
	fm.Set("type", "game")
	fm.Set("playtime", game.PlaytimeForever)

	// Add duration (formatted playtime)
	if game.PlaytimeForever > 0 {
		hours := game.PlaytimeForever / 60
		mins := game.PlaytimeForever % 60
		fm.Set("duration", fmt.Sprintf("%dh %dm", hours, mins))
	}

	// Add release date
	if details.ReleaseDate.Date != "" {
		fm.Set("release_date", details.ReleaseDate.Date)
	}

	// Add achievement metadata to frontmatter
	if len(details.Achievements) > 0 {
		achievementsTotal := len(details.Achievements)
		achievementsUnlocked := 0
		for _, ach := range details.Achievements {
			if ach.Achieved == 1 {
				achievementsUnlocked++
			}
		}
		fm.Set("achievements_total", achievementsTotal)
		fm.Set("achievements_unlocked", achievementsUnlocked)

		if achievementsTotal > 0 {
			percentage := (float64(achievementsUnlocked) / float64(achievementsTotal)) * 100
			fm.Set("achievements_percent", fmt.Sprintf("%.1f%%", percentage))
		}
	}

	// Handle cover image
	coverFilename := ""
	if details.HeaderImage != "" {
		coverResult, err := fileutil.DownloadCover(fileutil.CoverDownloadOptions{
			URL:          details.HeaderImage,
			OutputDir:    directory,
			Filename:     fileutil.BuildCoverFilename(game.Name),
			UpdateCovers: config.UpdateCovers,
		})
		if err == nil && coverResult != nil {
			coverFilename = coverResult.Filename
			fm.Set("cover", coverResult.RelativePath)
		}
	}

	// Add developers as an array
	if len(details.Developers) > 0 {
		fm.Set("developers", details.Developers)
	}

	// Add publishers as an array
	if len(details.Publishers) > 0 {
		fm.Set("publishers", details.Publishers)
	}

	// Add categories as an array
	if len(details.Categories) > 0 {
		categories := make([]string, len(details.Categories))
		for i, cat := range details.Categories {
			categories[i] = cat.Description
		}
		fm.Set("categories", categories)
	}

	// Add genres as an array
	if len(details.Genres) > 0 {
		genres := make([]string, len(details.Genres))
		for i, genre := range details.Genres {
			genres[i] = genre.Description
		}
		fm.Set("genres", genres)
	}

	// Collect tags
	tc := obsidian.NewTagSet()
	tc.Add("steam/game")
	obsidian.ApplyTagSet(fm, tc)

	// Add metacritic info if available
	if details.Metacritic.Score > 0 {
		fm.Set("metacritic_score", details.Metacritic.Score)
		if details.Metacritic.URL != "" {
			fm.Set("metacritic_url", details.Metacritic.URL)
		}
	}

	// Build body content
	var body strings.Builder

	// Add cover image using Obsidian syntax
	if coverFilename != "" {
		body.WriteString(fmt.Sprintf("![[%s|%d]]\n\n", coverFilename, defaultCoverWidth))
	}

	// Build Steam content sections using the existing steam_sections.go
	categories := make([]string, len(details.Categories))
	for i, cat := range details.Categories {
		categories[i] = cat.Description
	}
	genres := make([]string, len(details.Genres))
	for i, genre := range details.Genres {
		genres[i] = genre.Description
	}

	steamDetails := &content.SteamGameDetails{
		AppID:       game.AppID,
		Name:        game.Name,
		Description: details.Description,
		ShortDesc:   details.ShortDesc,
		HeaderImage: details.HeaderImage,
		Developers:  details.Developers,
		Publishers:  details.Publishers,
		ReleaseDate: details.ReleaseDate.Date,
		ComingSoon:  details.ReleaseDate.ComingSoon,
		Categories:  categories,
		Genres:      genres,
		Metacritic: struct {
			Score int
			URL   string
		}{
			Score: details.Metacritic.Score,
			URL:   details.Metacritic.URL,
		},
	}

	steamContent := content.BuildSteamContent(steamDetails, []string{"info", "description"}, "")
	if steamContent != "" {
		wrappedSteam := content.WrapWithSteamMarkers(steamContent)
		body.WriteString(wrappedSteam)
		body.WriteString("\n\n")
	}

	// Add achievements section (not in content markers since it's game-specific data)
	if len(details.Achievements) > 0 {
		achievementsSection := buildAchievementsSection(details.Achievements)
		if achievementsSection != "" {
			body.WriteString(achievementsSection)
			body.WriteString("\n\n")
		}
	}

	// Add screenshots section
	if len(details.Screenshots) > 0 {
		body.WriteString("## Screenshots\n\n")
		for _, screenshot := range details.Screenshots {
			body.WriteString(fmt.Sprintf("![%s](%s)\n\n", game.Name, screenshot.PathURL))
		}
	}

	// Create the note
	// Build markdown
	markdown, err := obsidian.BuildNoteMarkdown(fm, body.String())
	if err != nil {
		return fmt.Errorf("failed to build markdown: %w", err)
	}

	// Add trailing newlines to match expected format
	content := string(markdown) + "\n\n\n"

	// Write content to file with logging
	return fileutil.WriteMarkdownFile(filename, content, config.OverwriteFiles)
}

// buildAchievementsSection creates the achievements section with progress and checklist
func buildAchievementsSection(achievements []Achievement) string {
	if len(achievements) == 0 {
		return ""
	}

	var builder strings.Builder
	builder.WriteString("## Achievements\n\n")

	// Sort: unlocked first (by unlock time desc), then locked (alphabetically)
	achievementsCopy := make([]Achievement, len(achievements))
	copy(achievementsCopy, achievements)

	sort.SliceStable(achievementsCopy, func(i, j int) bool {
		if achievementsCopy[i].Achieved != achievementsCopy[j].Achieved {
			return achievementsCopy[i].Achieved > achievementsCopy[j].Achieved
		}
		if achievementsCopy[i].Achieved == 1 {
			return achievementsCopy[i].UnlockTime > achievementsCopy[j].UnlockTime
		}
		return achievementsCopy[i].Name < achievementsCopy[j].Name
	})

	// Calculate stats
	achievementsTotal := len(achievementsCopy)
	achievementsUnlocked := 0
	for _, ach := range achievementsCopy {
		if ach.Achieved == 1 {
			achievementsUnlocked++
		}
	}

	// Progress summary
	builder.WriteString(fmt.Sprintf("**Progress**: %d / %d (%.1f%%)\n\n",
		achievementsUnlocked,
		achievementsTotal,
		(float64(achievementsUnlocked)/float64(achievementsTotal))*100))

	// Checklist
	for _, ach := range achievementsCopy {
		checkbox := "[ ]"
		timestamp := ""
		if ach.Achieved == 1 {
			checkbox = "[x]"
			if ach.UnlockTime > 0 {
				unlockDate := time.Unix(ach.UnlockTime, 0).Format("2006-01-02")
				timestamp = fmt.Sprintf(" *(unlocked %s)*", unlockDate)
			}
		}

		builder.WriteString(fmt.Sprintf("- %s **%s**", checkbox, ach.Name))
		if ach.Description != "" {
			builder.WriteString(fmt.Sprintf(": %s", ach.Description))
		}
		builder.WriteString(timestamp)
		builder.WriteString("\n")
	}

	return builder.String()
}
