package diff

import (
	"fmt"
	"strings"
	"time"

	"github.com/lepinkainen/hermes/internal/obsidian"
)

func buildDiffNote(imdbOnly, letterboxdOnly []diffItem, stats diffStats, now time.Time, mainDBPath, cacheDBPath string) *obsidian.Note {
	fm := obsidian.NewFrontmatter()
	fm.Set("title", "IMDb vs Letterboxd Diff")
	fm.Set("date", now.Format("2006-01-02"))
	fm.Set("imdb_only", stats.imdbOnlyCount)
	fm.Set("letterboxd_only", stats.letterboxdOnlyCount)
	fm.Set("resolved_title_year", stats.resolvedTitleYear)
	fm.Set("imdb_only_with_fuzzy", stats.imdbOnlyWithFuzzy)
	fm.Set("letterboxd_only_with_fuzzy", stats.letterboxdOnlyWithFuzzy)

	tags := obsidian.NewTagSet()
	tags.Add("diff/imdb-letterboxd")
	tags.Add("report")
	tags.Add("movies")
	fm.Set("tags", tags.GetSorted())

	body := buildDiffBody(imdbOnly, letterboxdOnly, stats, now, mainDBPath, cacheDBPath)

	return &obsidian.Note{
		Frontmatter: fm,
		Body:        body,
	}
}

func buildDiffBody(imdbOnly, letterboxdOnly []diffItem, stats diffStats, now time.Time, mainDBPath, cacheDBPath string) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("# IMDb vs Letterboxd Diff (%s)\n\n", now.Format("2006-01-02")))
	b.WriteString("This report compares IMDb and Letterboxd movie imports in SQLite. TV titles are excluded.\n\n")
	b.WriteString(fmt.Sprintf("- Main DB: `%s`\n", mainDBPath))
	if cacheDBPath != "" {
		b.WriteString(fmt.Sprintf("- Cache DB: `%s`\n", cacheDBPath))
	}
	b.WriteString("- Matching: IMDb ID, then auto-resolved title+year, then fuzzy suggestions\n")
	b.WriteString(fmt.Sprintf("- Auto-resolved title+year matches: %d\n\n", stats.resolvedTitleYear))

	writeDiffSection(&b, "IMDb-only (missing from Letterboxd)", imdbOnly)
	writeDiffSection(&b, "Letterboxd-only (missing from IMDb)", letterboxdOnly)

	return b.String()
}

func writeDiffSection(b *strings.Builder, title string, items []diffItem) {
	b.WriteString("## ")
	b.WriteString(title)
	b.WriteString("\n\n")
	if len(items) == 0 {
		b.WriteString("_None found._\n\n")
		return
	}
	for _, item := range items {
		b.WriteString("- [ ] ")
		b.WriteString(formatDiffLine(item))
		b.WriteString("\n")
		if len(item.FuzzyMatches) > 0 {
			b.WriteString("  - Possible matches (title + year):\n")
			for _, match := range item.FuzzyMatches {
				b.WriteString("    - ")
				b.WriteString(formatMatchLine(match))
				b.WriteString("\n")
			}
		}
	}
	b.WriteString("\n")
}

func formatDiffLine(item diffItem) string {
	parts := []string{formatTitleYear(item.Title, item.Year)}
	if item.ImdbID != "" {
		parts = append(parts, fmt.Sprintf("IMDb %s", item.ImdbID))
	}
	if item.ImdbURL != "" {
		parts = append(parts, item.ImdbURL)
	}
	if item.LetterboxdURI != "" {
		parts = append(parts, item.LetterboxdURI)
	}
	if item.ImdbRating > 0 {
		parts = append(parts, fmt.Sprintf("IMDb rating %d/10", item.ImdbRating))
	}
	if item.LetterboxdRating > 0 {
		parts = append(parts, fmt.Sprintf("Letterboxd rating %.1f/5", item.LetterboxdRating))
	}
	return strings.Join(parts, " — ")
}

func formatMatchLine(match diffMatch) string {
	parts := []string{formatTitleYear(match.Title, match.Year)}
	if match.ImdbID != "" {
		parts = append(parts, fmt.Sprintf("IMDb %s", match.ImdbID))
	}
	if match.ImdbURL != "" {
		parts = append(parts, match.ImdbURL)
	}
	if match.LetterboxdURI != "" {
		parts = append(parts, match.LetterboxdURI)
	}
	if match.ImdbRating > 0 {
		parts = append(parts, fmt.Sprintf("IMDb rating %d/10", match.ImdbRating))
	}
	if match.LetterboxdRating > 0 {
		parts = append(parts, fmt.Sprintf("Letterboxd rating %.1f/5", match.LetterboxdRating))
	}
	return strings.Join(parts, " — ")
}

func formatTitleYear(title string, year int) string {
	if year > 0 {
		return fmt.Sprintf("%s (%d)", title, year)
	}
	return title
}
