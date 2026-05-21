package content

import (
	"fmt"
	"strings"
)

// BuildTMDBContent generates markdown content from TMDB details.
func BuildTMDBContent(details map[string]any, mediaType string, sections []string, letterboxdURI string) string {
	if len(sections) == 0 {
		if mediaType == "tv" {
			sections = []string{"overview", "info", "seasons"}
		} else {
			sections = []string{"overview", "info"}
		}
	}

	var blocks []string
	for _, section := range sections {
		switch section {
		case "overview":
			if block := buildOverview(details); block != "" {
				blocks = append(blocks, block)
			}
		case "info":
			if block := buildInfo(details, mediaType, letterboxdURI); block != "" {
				blocks = append(blocks, block)
			}
		case "seasons":
			if mediaType == "tv" {
				if block := buildSeasons(details); block != "" {
					blocks = append(blocks, block)
				}
			}
		}
	}

	return strings.Join(blocks, "\n\n")
}

func buildOverview(details map[string]any) string {
	overview := stringVal(details, "overview")
	if strings.TrimSpace(overview) == "" {
		return ""
	}

	tagline := stringVal(details, "tagline")

	var builder strings.Builder
	builder.WriteString("## Overview\n\n")
	builder.WriteString(strings.TrimSpace(overview))
	builder.WriteString("\n")

	if tagline = strings.TrimSpace(tagline); tagline != "" {
		builder.WriteString("\n> _\"")
		builder.WriteString(tagline)
		builder.WriteString("\"_\n")
	}
	return builder.String()
}

func buildInfo(details map[string]any, mediaType string, letterboxdURI string) string {
	var builder strings.Builder
	builder.WriteString("## ")
	if mediaType == "tv" {
		builder.WriteString("Series Info\n\n")
	} else {
		builder.WriteString("Movie Info\n\n")
	}

	builder.WriteString("| | |\n")
	builder.WriteString("|---|---|\n")

	status := stringVal(details, "status")
	inProduction := boolVal(details, "in_production")
	if status == "" {
		status = "Unknown"
	}
	if mediaType == "tv" && inProduction {
		fmt.Fprintf(&builder, "| **Status** | %s (In Production) |\n", status)
	} else {
		fmt.Fprintf(&builder, "| **Status** | %s |\n", status)
	}

	if mediaType == "tv" {
		writeTVAirInfo(&builder, details, inProduction)
	} else {
		writeMovieCreditsInfo(&builder, details)
	}

	if rating, ok := floatVal(details, "vote_average"); ok && rating > 0 {
		votes, _ := intVal(details, "vote_count")
		fmt.Fprintf(&builder, "| **Rating** | ⭐ %.1f/10 (%s votes) |\n", rating, formatNumber(votes))
	}

	if mediaType == "tv" {
		if networkName := firstStringFromArray(details, "networks", "name"); networkName != "" {
			fmt.Fprintf(&builder, "| **Network** | %s |\n", networkName)
		}
	} else {
		if budget, ok := intVal(details, "budget"); ok && budget > 0 {
			fmt.Fprintf(&builder, "| **Budget** | $%s |\n", formatNumber(budget))
		}
		if revenue, ok := intVal(details, "revenue"); ok && revenue > 0 {
			fmt.Fprintf(&builder, "| **Revenue** | $%s |\n", formatNumber(revenue))
		}
	}

	if countries := stringSlice(details, "origin_country"); len(countries) > 0 {
		parts := make([]string, 0, min(3, len(countries)))
		for i, code := range countries {
			if i >= 3 {
				break
			}
			parts = append(parts, fmt.Sprintf("%s %s", countryFlag(code), code))
		}
		fmt.Fprintf(&builder, "| **Origin** | %s |\n", strings.Join(parts, " "))
	}

	if mediaType == "tv" {
		if rating := usContentRating(details); rating != "" {
			fmt.Fprintf(&builder, "| **Content Rating** | %s |\n", rating)
		}
	}

	if imdb := nestedString(details, "external_ids", "imdb_id"); imdb != "" {
		fmt.Fprintf(&builder, "| **IMDB** | [imdb.com/title/%s](https://www.imdb.com/title/%s/) |\n", imdb, imdb)
	}
	if tvdb := nestedString(details, "external_ids", "tvdb_id"); tvdb != "" {
		fmt.Fprintf(&builder, "| **TVDB** | [thetvdb.com/%s](https://thetvdb.com/series/%s) |\n", tvdb, tvdb)
	}

	if letterboxdURI != "" {
		displayText := extractLetterboxdDisplayText(letterboxdURI)
		fmt.Fprintf(&builder, "| **Letterboxd** | [%s](%s) |\n", displayText, letterboxdURI)
	}

	if homepage := stringVal(details, "homepage"); homepage != "" {
		fmt.Fprintf(&builder, "| **Homepage** | [%s](%s) |\n", friendlyHomepageName(homepage), homepage)
	}

	return strings.TrimRight(builder.String(), "\n")
}

func writeTVAirInfo(builder *strings.Builder, details map[string]any, inProduction bool) {
	seasons, _ := intVal(details, "number_of_seasons")
	episodes, _ := intVal(details, "number_of_episodes")
	fmt.Fprintf(builder, "| **Seasons** | %d (%d episodes) |\n", seasons, episodes)

	firstAir := stringVal(details, "first_air_date")
	if firstAir == "" {
		return
	}
	lastAir := stringVal(details, "last_air_date")
	airText := firstAir
	switch {
	case lastAir != "" && lastAir != firstAir:
		airText = fmt.Sprintf("%s → %s", firstAir, lastAir)
	case inProduction:
		airText = fmt.Sprintf("%s → Present", firstAir)
	}
	fmt.Fprintf(builder, "| **Aired** | %s |\n", airText)
}

func writeMovieCreditsInfo(builder *strings.Builder, details map[string]any) {
	if runtime, ok := intVal(details, "runtime"); ok && runtime > 0 {
		fmt.Fprintf(builder, "| **Runtime** | %d min |\n", runtime)
	}
	if release := stringVal(details, "release_date"); release != "" {
		fmt.Fprintf(builder, "| **Released** | %s |\n", release)
	}
	if directors := getDirectors(details); len(directors) > 0 {
		fmt.Fprintf(builder, "| **Director** | %s |\n", strings.Join(directors, ", "))
	}
	if writers := getWriters(details); len(writers) > 0 {
		fmt.Fprintf(builder, "| **Writer** | %s |\n", strings.Join(writers, ", "))
	}
	if actors := getTopActors(details); len(actors) > 0 {
		fmt.Fprintf(builder, "| **Cast** | %s |\n", strings.Join(actors, ", "))
	}
}

// extractLetterboxdDisplayText extracts a clean display text from a Letterboxd URI.
// For short URLs like "https://boxd.it/2bg8", returns "boxd.it/2bg8"
// For full URLs like "https://letterboxd.com/film/the-godfather/", returns "film/the-godfather"
// For search URLs like "https://letterboxd.com/search/wildcat/", returns "Search: wildcat"
func extractLetterboxdDisplayText(uri string) string {
	// Remove protocol
	display := strings.TrimPrefix(uri, "https://")
	display = strings.TrimPrefix(display, "http://")

	// Handle search URLs specially
	if strings.Contains(display, "letterboxd.com/search/") {
		// Extract search term
		searchTerm := strings.TrimPrefix(display, "letterboxd.com/search/")
		searchTerm = strings.TrimSuffix(searchTerm, "/")
		return fmt.Sprintf("Search: %s", searchTerm)
	}

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

	return display
}

func buildSeasons(details map[string]any) string {
	raw, ok := details["seasons"].([]any)
	if !ok || len(raw) == 0 {
		return ""
	}

	var builder strings.Builder
	builder.WriteString("## Seasons\n\n")

	for idx, season := range raw {
		s, ok := season.(map[string]any)
		if !ok {
			continue
		}
		name := stringVal(s, "name")
		if name == "" {
			if num, ok := intVal(s, "season_number"); ok {
				name = fmt.Sprintf("Season %d", num)
			} else {
				name = "Season"
			}
		}

		airDate := stringVal(s, "air_date")
		year := "TBA"
		if len(airDate) >= 4 {
			year = airDate[:4]
		}
		vote, _ := floatVal(s, "vote_average")
		overview := strings.TrimSpace(stringVal(s, "overview"))
		episodeCount, _ := intVal(s, "episode_count")
		poster := stringVal(s, "poster_path")

		fmt.Fprintf(&builder, "### %s (%s)", name, year)
		if vote > 0 {
			fmt.Fprintf(&builder, " • ⭐ %.1f/10", vote)
		}
		builder.WriteString("\n\n")

		if poster != "" {
			fmt.Fprintf(&builder, "![%s](https://image.tmdb.org/t/p/w300%s)\n\n", name, poster)
		}

		if overview != "" {
			fmt.Fprintf(&builder, "_%s_\n\n", overview)
		}

		fmt.Fprintf(&builder, "**Episodes:** %d", episodeCount)

		inProduction := boolVal(details, "in_production")
		isLatest := idx == len(raw)-1

		if isLatest && inProduction {
			builder.WriteString(" • **Status:** Currently Airing\n\n")
		} else {
			builder.WriteString(" • **Status:** ✅ Complete\n\n")
		}

		builder.WriteString("---\n\n")
	}

	out := strings.TrimRight(builder.String(), "\n")
	if !strings.HasSuffix(out, "\n") {
		out += "\n"
	}
	return out
}
