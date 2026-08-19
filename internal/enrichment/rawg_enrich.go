package enrichment

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/lepinkainen/hermes/internal/config"
	"github.com/lepinkainen/hermes/internal/fileutil"
	"github.com/lepinkainen/hermes/internal/obsidian"
)

// EnrichFromRAWG enriches a game with RAWG (rawg.io) data. It is the
// fallback used by cmd/enhance when EnrichFromSteam finds nothing — Steam is
// PC-only, so console/handheld exclusives (Uncharted 1/2/4/Lost Legacy,
// Shadow of the Colossus, Nintendo titles, ...) never resolve there. RAWG's
// platform-agnostic database (500k+ games, all platforms) covers those.
//
// Returns (nil, nil) when no RAWG API key is configured (silent no-op, same
// as the ISBNdb enricher) or when nothing matches the search.
func EnrichFromRAWG(ctx context.Context, title string, opts SteamEnrichmentOptions) (*GameEnrichment, error) {
	apiKey := getRAWGAPIKey()
	if apiKey == "" {
		slog.Debug("RAWG API key not configured, skipping RAWG enrichment", "title", title)
		return nil, nil
	}

	match, err := searchRAWGGame(ctx, apiKey, title, opts)
	if err != nil {
		return nil, err
	}
	if match == nil {
		slog.Debug("No RAWG game found", "title", title)
		return nil, nil
	}

	// Fetch full details for description_raw + developers/publishers; search
	// rows only carry genres/released/metacritic/background_image.
	details, err := fetchRAWGDetails(ctx, apiKey, match.ID)
	if err != nil {
		slog.Warn("Failed to fetch RAWG game details", "rawg_id", match.ID, "error", err)
		return nil, nil
	}

	game := &GameEnrichment{
		RAWGID:      details.ID,
		Developers:  rawgCompanyNames(details.Developers),
		Publishers:  rawgCompanyNames(details.Publishers),
		ReleaseDate: details.Released,
		Description: strings.TrimSpace(details.DescriptionRaw),
	}

	if details.Metacritic != nil {
		game.MetacriticScore = *details.Metacritic
	}

	game.GenreTags = extractRAWGGenreTags(details.Genres)

	if opts.DownloadCover {
		coverFilename, coverPath := ensureRAWGCoverAssets(ctx, opts, title, details.BackgroundImage)
		game.CoverFilename = coverFilename
		game.CoverPath = coverPath
	}

	if opts.GenerateContent {
		game.ContentMarkdown = buildRAWGContentMarkdown(details, game)
	}

	return game, nil
}

// rawgCompanyNames extracts non-empty company names from RAWG developer or
// publisher entries.
func rawgCompanyNames(companies []RAWGCompany) []string {
	if len(companies) == 0 {
		return nil
	}
	names := make([]string, 0, len(companies))
	for _, c := range companies {
		if c.Name != "" {
			names = append(names, c.Name)
		}
	}
	return names
}

// extractRAWGGenreTags extracts genre tags from RAWG genres, using the same
// genre/<slug> convention as Steam for a uniform vault.
func extractRAWGGenreTags(genres []RAWGGenre) []string {
	if len(genres) == 0 {
		return nil
	}
	tags := make([]string, 0, len(genres))
	for _, g := range genres {
		if g.Name != "" {
			tags = append(tags, "genre/"+obsidian.NormalizeTag(g.Name))
		}
	}
	return tags
}

// ensureRAWGCoverAssets downloads the RAWG background image as the note's
// cover, reusing fileutil.DownloadCover exactly as ensureSteamCoverAssets
// does for Steam header images.
func ensureRAWGCoverAssets(ctx context.Context, opts SteamEnrichmentOptions, title string, backgroundImageURL string) (string, string) {
	if backgroundImageURL == "" {
		slog.Debug("No RAWG background image URL provided", "title", title)
		return "", ""
	}

	coverFilename := fileutil.SanitizeFilename(title) + " - cover.jpg"
	coverPath := filepath.Join(opts.AttachmentsDir, coverFilename)

	// Check if cover already exists
	if _, err := os.Stat(coverPath); err == nil && !config.UpdateCovers {
		slog.Debug("RAWG cover already exists, skipping download", "path", coverPath)
		return coverFilename, relativeCoverPath(opts.NoteDir, coverPath)
	}

	result, err := fileutil.DownloadCover(ctx, fileutil.CoverDownloadOptions{
		URL:          backgroundImageURL,
		OutputDir:    filepath.Dir(opts.AttachmentsDir), // Parent of attachments
		Filename:     coverFilename,
		UpdateCovers: config.UpdateCovers,
	})

	if err != nil {
		slog.Warn("Failed to download RAWG cover", "error", err, "url", backgroundImageURL)
		return "", ""
	}

	if result == nil {
		return "", ""
	}

	slog.Info("Downloaded RAWG cover", "path", result.LocalPath)
	return coverFilename, result.RelativePath
}

// buildRAWGContentMarkdown generates a small markdown section (info table +
// description) from RAWG game details, to be wrapped with the Steam content
// markers by cmd/enhance so NeedsSteamContent's skip logic works uniformly
// for RAWG-enriched notes.
func buildRAWGContentMarkdown(details *RAWGGameResult, game *GameEnrichment) string {
	if details == nil {
		return ""
	}

	var blocks []string

	if game.CoverFilename != "" {
		blocks = append(blocks, fmt.Sprintf("![[%s|500]]", game.CoverFilename))
	}

	if info := buildRAWGInfo(details, game); info != "" {
		blocks = append(blocks, info)
	}

	if desc := strings.TrimSpace(details.DescriptionRaw); desc != "" {
		var d strings.Builder
		d.WriteString("## Description\n\n")
		d.WriteString(desc)
		blocks = append(blocks, d.String())
	}

	return strings.Join(blocks, "\n\n")
}

func buildRAWGInfo(details *RAWGGameResult, game *GameEnrichment) string {
	var builder strings.Builder
	builder.WriteString("## Game Info\n\n")
	builder.WriteString("| | |\n")
	builder.WriteString("|---|---|\n")

	if len(game.Developers) > 0 {
		fmt.Fprintf(&builder, "| **Developer** | %s |\n", strings.Join(game.Developers, ", "))
	}

	if len(game.Publishers) > 0 {
		fmt.Fprintf(&builder, "| **Publisher** | %s |\n", strings.Join(game.Publishers, ", "))
	}

	if game.ReleaseDate != "" {
		fmt.Fprintf(&builder, "| **Release Date** | %s |\n", game.ReleaseDate)
	}

	genres := rawgGenreNames(details.Genres)
	if len(genres) > 0 {
		fmt.Fprintf(&builder, "| **Genres** | %s |\n", strings.Join(genres, ", "))
	}

	rawgURL := rawgGameURL(details)

	if game.MetacriticScore > 0 {
		if rawgURL != "" {
			fmt.Fprintf(&builder, "| **Metacritic** | [%d/100](%s) |\n", game.MetacriticScore, rawgURL)
		} else {
			fmt.Fprintf(&builder, "| **Metacritic** | %d/100 |\n", game.MetacriticScore)
		}
	}

	if rawgURL != "" {
		fmt.Fprintf(&builder, "| **RAWG** | [%s](%s) |\n", rawgURL, rawgURL)
	}

	return strings.TrimRight(builder.String(), "\n")
}

func rawgGenreNames(genres []RAWGGenre) []string {
	if len(genres) == 0 {
		return nil
	}
	names := make([]string, 0, len(genres))
	for _, g := range genres {
		if g.Name != "" {
			names = append(names, g.Name)
		}
	}
	return names
}

func rawgGameURL(details *RAWGGameResult) string {
	if details.Slug == "" {
		return ""
	}
	return "https://rawg.io/games/" + details.Slug
}
