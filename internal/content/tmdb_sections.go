package content

import (
	"fmt"
	"strings"
)

// BuildTMDBContent generates markdown content from TMDB details.
func BuildTMDBContent(details map[string]any, mediaType string, sections []string) string {
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
			if block := buildInfo(details, mediaType); block != "" {
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

func buildInfo(details map[string]any, mediaType string) string {
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
		builder.WriteString(fmt.Sprintf("| **Status** | %s (In Production) |\n", status))
	} else {
		builder.WriteString(fmt.Sprintf("| **Status** | %s |\n", status))
	}

	if mediaType == "tv" {
		seasons, _ := intVal(details, "number_of_seasons")
		episodes, _ := intVal(details, "number_of_episodes")
		builder.WriteString(fmt.Sprintf("| **Seasons** | %d (%d episodes) |\n", seasons, episodes))

		firstAir := stringVal(details, "first_air_date")
		lastAir := stringVal(details, "last_air_date")
		if firstAir != "" {
			airText := firstAir
			switch {
			case lastAir != "" && lastAir != firstAir:
				airText = fmt.Sprintf("%s → %s", firstAir, lastAir)
			case inProduction:
				airText = fmt.Sprintf("%s → Present", firstAir)
			}
			builder.WriteString(fmt.Sprintf("| **Aired** | %s |\n", airText))
		}
	} else {
		if runtime, ok := intVal(details, "runtime"); ok && runtime > 0 {
			builder.WriteString(fmt.Sprintf("| **Runtime** | %d min |\n", runtime))
		}
		release := stringVal(details, "release_date")
		if release != "" {
			builder.WriteString(fmt.Sprintf("| **Released** | %s |\n", release))
		}
	}

	if rating, ok := floatVal(details, "vote_average"); ok && rating > 0 {
		votes, _ := intVal(details, "vote_count")
		builder.WriteString(fmt.Sprintf("| **Rating** | ⭐ %.1f/10 (%s votes) |\n", rating, formatNumber(votes)))
	}

	if mediaType == "tv" {
		if networkName := firstStringFromArray(details, "networks", "name"); networkName != "" {
			builder.WriteString(fmt.Sprintf("| **Network** | %s |\n", networkName))
		}
	} else {
		if budget, ok := intVal(details, "budget"); ok && budget > 0 {
			builder.WriteString(fmt.Sprintf("| **Budget** | $%s |\n", formatNumber(budget)))
		}
		if revenue, ok := intVal(details, "revenue"); ok && revenue > 0 {
			builder.WriteString(fmt.Sprintf("| **Revenue** | $%s |\n", formatNumber(revenue)))
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
		builder.WriteString(fmt.Sprintf("| **Origin** | %s |\n", strings.Join(parts, " ")))
	}

	if mediaType == "tv" {
		if rating := usContentRating(details); rating != "" {
			builder.WriteString(fmt.Sprintf("| **Content Rating** | %s |\n", rating))
		}
	}

	if imdb := nestedString(details, "external_ids", "imdb_id"); imdb != "" {
		builder.WriteString(fmt.Sprintf("| **IMDB** | [imdb.com/title/%s](https://www.imdb.com/title/%s/) |\n", imdb, imdb))
	}
	if tvdb := nestedString(details, "external_ids", "tvdb_id"); tvdb != "" {
		builder.WriteString(fmt.Sprintf("| **TVDB** | [thetvdb.com/%s](https://thetvdb.com/series/%s) |\n", tvdb, tvdb))
	}

	if homepage := stringVal(details, "homepage"); homepage != "" {
		builder.WriteString(fmt.Sprintf("| **Homepage** | [%s](%s) |\n", friendlyHomepageName(homepage), homepage))
	}

	return strings.TrimRight(builder.String(), "\n")
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

		builder.WriteString(fmt.Sprintf("### %s (%s)", name, year))
		if vote > 0 {
			builder.WriteString(fmt.Sprintf(" • ⭐ %.1f/10", vote))
		}
		builder.WriteString("\n\n")

		if poster != "" {
			builder.WriteString(fmt.Sprintf("![%s](https://image.tmdb.org/t/p/w300%s)\n\n", name, poster))
		}

		if overview != "" {
			builder.WriteString(fmt.Sprintf("_%s_\n\n", overview))
		}

		builder.WriteString(fmt.Sprintf("**Episodes:** %d", episodeCount))

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
