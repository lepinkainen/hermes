package content

import (
	"fmt"
	"strings"
)

// SteamGameDetails contains the fields needed for building Steam content.
// This matches the structure from cmd/steam/types.go.
type SteamGameDetails struct {
	AppID       int
	Name        string
	Description string
	ShortDesc   string
	HeaderImage string
	Developers  []string
	Publishers  []string
	ReleaseDate string
	ComingSoon  bool
	Categories  []string
	Genres      []string
	Metacritic  struct {
		Score int
		URL   string
	}
}

// BuildSteamContent generates markdown content from Steam game details.
func BuildSteamContent(details *SteamGameDetails, sections []string, coverFilename string) string {
	if details == nil {
		return ""
	}

	if len(sections) == 0 {
		sections = []string{"info", "description"}
	}

	var blocks []string

	// Add cover image first if available
	// Width set to 500 because it's a vertical image compared to the horizontal
	// images for movies and tv shows.
	if coverFilename != "" {
		blocks = append(blocks, fmt.Sprintf("![[%s|500]]", coverFilename))
	}

	for _, section := range sections {
		switch section {
		case "info":
			if block := buildSteamInfo(details); block != "" {
				blocks = append(blocks, block)
			}
		case "description":
			if block := buildSteamDescription(details); block != "" {
				blocks = append(blocks, block)
			}
		}
	}

	return strings.Join(blocks, "\n\n")
}

func buildSteamInfo(details *SteamGameDetails) string {
	var builder strings.Builder
	builder.WriteString("## Game Info\n\n")
	builder.WriteString("| | |\n")
	builder.WriteString("|---|---|\n")

	// Developers
	if len(details.Developers) > 0 {
		builder.WriteString(fmt.Sprintf("| **Developer** | %s |\n", strings.Join(details.Developers, ", ")))
	}

	// Publishers
	if len(details.Publishers) > 0 {
		builder.WriteString(fmt.Sprintf("| **Publisher** | %s |\n", strings.Join(details.Publishers, ", ")))
	}

	// Release Date
	if details.ReleaseDate != "" {
		if details.ComingSoon {
			builder.WriteString(fmt.Sprintf("| **Release Date** | %s (Coming Soon) |\n", details.ReleaseDate))
		} else {
			builder.WriteString(fmt.Sprintf("| **Release Date** | %s |\n", details.ReleaseDate))
		}
	}

	// Genres
	if len(details.Genres) > 0 {
		builder.WriteString(fmt.Sprintf("| **Genres** | %s |\n", strings.Join(details.Genres, ", ")))
	}

	// Categories (e.g., Single-player, Multi-player, Steam Achievements)
	if len(details.Categories) > 0 {
		// Limit to most important categories
		cats := details.Categories
		if len(cats) > 5 {
			cats = cats[:5]
		}
		builder.WriteString(fmt.Sprintf("| **Features** | %s |\n", strings.Join(cats, ", ")))
	}

	// Metacritic
	if details.Metacritic.Score > 0 {
		if details.Metacritic.URL != "" {
			builder.WriteString(fmt.Sprintf("| **Metacritic** | [%d/100](%s) |\n", details.Metacritic.Score, details.Metacritic.URL))
		} else {
			builder.WriteString(fmt.Sprintf("| **Metacritic** | %d/100 |\n", details.Metacritic.Score))
		}
	}

	// Steam Store link
	if details.AppID > 0 {
		builder.WriteString(fmt.Sprintf("| **Steam** | [store.steampowered.com/app/%d](https://store.steampowered.com/app/%d) |\n", details.AppID, details.AppID))
	}

	return strings.TrimRight(builder.String(), "\n")
}

func buildSteamDescription(details *SteamGameDetails) string {
	// Prefer short description, fall back to long description
	desc := details.ShortDesc
	if desc == "" {
		desc = details.Description
	}
	desc = strings.TrimSpace(desc)
	if desc == "" {
		return ""
	}

	// Clean up HTML tags that may be in the description
	desc = cleanHTMLTags(desc)

	var builder strings.Builder
	builder.WriteString("## Description\n\n")
	builder.WriteString(desc)
	builder.WriteString("\n")

	return builder.String()
}

// cleanHTMLTags removes common HTML tags from Steam descriptions.
func cleanHTMLTags(s string) string {
	// Replace <br> and <br/> with newlines
	s = strings.ReplaceAll(s, "<br>", "\n")
	s = strings.ReplaceAll(s, "<br/>", "\n")
	s = strings.ReplaceAll(s, "<br />", "\n")

	// Remove other common HTML tags
	replacements := []string{
		"<p>", "",
		"</p>", "\n",
		"<b>", "**",
		"</b>", "**",
		"<strong>", "**",
		"</strong>", "**",
		"<i>", "_",
		"</i>", "_",
		"<em>", "_",
		"</em>", "_",
		"<ul>", "",
		"</ul>", "",
		"<li>", "- ",
		"</li>", "\n",
	}

	for i := 0; i < len(replacements); i += 2 {
		s = strings.ReplaceAll(s, replacements[i], replacements[i+1])
	}

	// Collapse multiple newlines
	for strings.Contains(s, "\n\n\n") {
		s = strings.ReplaceAll(s, "\n\n\n", "\n\n")
	}

	return strings.TrimSpace(s)
}
