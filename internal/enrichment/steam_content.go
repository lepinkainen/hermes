package enrichment

import (
	"github.com/lepinkainen/hermes/cmd/steam"
	"github.com/lepinkainen/hermes/internal/content"
	"github.com/lepinkainen/hermes/internal/obsidian"
)

// buildSteamContentMarkdown generates markdown content from Steam game details.
func buildSteamContentMarkdown(details *steam.GameDetails, coverFilename string, sections []string) string {
	if details == nil {
		return ""
	}

	// Convert GameDetails to SteamGameDetails for the content package
	gameDetails := &content.SteamGameDetails{
		AppID:       details.AppID,
		Name:        details.Name,
		Description: details.Description,
		ShortDesc:   details.ShortDesc,
		HeaderImage: details.HeaderImage,
		Developers:  details.Developers,
		Publishers:  details.Publishers,
		ReleaseDate: details.ReleaseDate.Date,
		ComingSoon:  details.ReleaseDate.ComingSoon,
	}

	// Convert categories
	categories := make([]string, len(details.Categories))
	for i, cat := range details.Categories {
		categories[i] = cat.Description
	}
	gameDetails.Categories = categories

	// Convert genres
	genres := make([]string, len(details.Genres))
	for i, genre := range details.Genres {
		genres[i] = genre.Description
	}
	gameDetails.Genres = genres

	// Set metacritic
	gameDetails.Metacritic.Score = details.Metacritic.Score
	gameDetails.Metacritic.URL = details.Metacritic.URL

	return content.BuildSteamContent(gameDetails, sections, coverFilename)
}

// extractGenreTags extracts genre tags from Steam game details.
func extractGenreTags(details *steam.GameDetails) []string {
	if details == nil {
		return nil
	}

	tags := make([]string, 0, len(details.Genres))
	for _, genre := range details.Genres {
		if genre.Description != "" {
			tags = append(tags, "genre/"+obsidian.NormalizeTag(genre.Description))
		}
	}
	return tags
}
