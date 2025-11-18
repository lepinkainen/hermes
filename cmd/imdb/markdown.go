package imdb

import (
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"time"

	"github.com/lepinkainen/hermes/internal/config"
	"github.com/lepinkainen/hermes/internal/content"
	"github.com/lepinkainen/hermes/internal/fileutil"
)

const defaultCoverWidth = 250

// writeMovieToMarkdown writes movie info to a markdown file
func writeMovieToMarkdown(movie MovieSeen, directory string) error {
	// Get the file path using the common utility
	filePath := fileutil.GetMarkdownFilePath(movie.Title, directory)

	// Use the MarkdownBuilder to construct the document
	mb := fileutil.NewMarkdownBuilder()

	// Handle titles - remove problematic characters and handle original titles
	movie.Title = fileutil.SanitizeFilename(movie.Title)
	mb.AddTitle(movie.Title)

	if movie.OriginalTitle != "" && movie.OriginalTitle != movie.Title {
		movie.OriginalTitle = fileutil.SanitizeFilename(movie.OriginalTitle)
		mb.AddField("original_title", movie.OriginalTitle)
	}

	// Add type-specific metadata
	mb.AddType(mapTypeToType(movie.TitleType))
	
	// Add seen flag if movie has a rating
	if movie.MyRating > 0 {
		mb.AddField("seen", true)
	}

	// Basic metadata
	mb.AddField("imdb_id", movie.ImdbId)
	mb.AddYear(movie.Year)
	mb.AddField("imdb_rating", movie.IMDbRating)
	mb.AddField("my_rating", movie.MyRating)

	// Format date in a more readable way
	if movie.DateRated != "" {
		mb.AddDate("date_rated", movie.DateRated)
	}

	// Add runtime and duration
	if movie.RuntimeMins > 0 {
		mb.AddField("runtime", movie.RuntimeMins)
		mb.AddDuration(movie.RuntimeMins)
	}

	// Add directors as an array
	if len(movie.Directors) > 0 {
		mb.AddStringArray("directors", movie.Directors)
	}

	// Collect all tags using a map for deduplication
	tagMap := make(map[string]bool)
	
	// Add type tag
	typeTag := mapTypeToTag(movie.TitleType)
	if typeTag != "UNKNOWN" {
		tagMap[typeTag] = true
	}

	// Add rating tag if available
	if movie.MyRating > 0 {
		ratingTag := fmt.Sprintf("rating/%d", movie.MyRating)
		tagMap[ratingTag] = true
	}

	// Add decade tag
	if movie.Year > 0 {
		decade := (movie.Year / 10) * 10
		decadeTag := fmt.Sprintf("year/%ds", decade)
		tagMap[decadeTag] = true
	}

	// Add genres as tags with genre/ prefix
	for _, genre := range movie.Genres {
		genreTag := fmt.Sprintf("genre/%s", genre)
		tagMap[genreTag] = true
	}

	// Add TMDB genre tags if available
	if movie.TMDBEnrichment != nil && len(movie.TMDBEnrichment.GenreTags) > 0 {
		for _, tmdbTag := range movie.TMDBEnrichment.GenreTags {
			tagMap[tmdbTag] = true
		}
	}

	// Convert map to slice and sort alphabetically
	tags := make([]string, 0, len(tagMap))
	for tag := range tagMap {
		tags = append(tags, tag)
	}
	sort.Strings(tags)

	mb.AddTags(tags...)

	// Add content rating if available
	if movie.ContentRated != "" {
		mb.AddField("content_rating", movie.ContentRated)
	}

	// Add awards if available
	if movie.Awards != "" {
		mb.AddField("awards", movie.Awards)
	}

	// Add poster image - download locally and use Obsidian syntax
	if movie.PosterURL != "" {
		coverFilename := fileutil.BuildCoverFilename(movie.Title)
		result, err := fileutil.DownloadCover(fileutil.CoverDownloadOptions{
			URL:          movie.PosterURL,
			OutputDir:    directory,
			Filename:     coverFilename,
			UpdateCovers: config.UpdateCovers,
		})
		if err != nil {
			slog.Warn("Failed to download cover", "title", movie.Title, "error", err)
			// Fall back to URL if download fails
			mb.AddField("cover", movie.PosterURL)
			mb.AddImage(movie.PosterURL)
		} else if result != nil {
			// Use local path in frontmatter
			mb.AddField("cover", result.RelativePath)
			mb.AddObsidianImage(result.Filename, defaultCoverWidth)
		}
	}

	// Add plot summary in a callout if available
	if movie.Plot != "" {
		mb.AddCallout("summary", "Plot", movie.Plot)
	}

	// Add awards in a callout if available
	if movie.Awards != "" {
		mb.AddCallout("award", "Awards", movie.Awards)
	}

	// Add IMDb link as info callout
	links := map[string]string{
		"View on IMDb": movie.URL,
	}
	mb.AddExternalLinksCallout("IMDb", links)

	// Add TMDB data if available
	if movie.TMDBEnrichment != nil {
		// Add TMDB metadata to frontmatter
		if movie.TMDBEnrichment.TMDBID > 0 {
			mb.AddField("tmdb_id", movie.TMDBEnrichment.TMDBID)
			mb.AddField("tmdb_type", movie.TMDBEnrichment.TMDBType)
		}

		// Add total episodes for TV shows
		if movie.TMDBEnrichment.TotalEpisodes > 0 {
			mb.AddField("total_episodes", movie.TMDBEnrichment.TotalEpisodes)
		}

		// Add TMDB content sections (includes cover embed if downloaded)
		// Wrap with markers for future updates
		if movie.TMDBEnrichment.ContentMarkdown != "" {
			wrappedContent := content.WrapWithMarkers(movie.TMDBEnrichment.ContentMarkdown)
			mb.AddParagraph(wrappedContent)
		}
	}

	// Write content to file with overwrite logic
	written, err := fileutil.WriteFileWithOverwrite(filePath, []byte(mb.Build()), 0644, config.OverwriteFiles)
	if err != nil {
		return err
	}

	if !written {
		slog.Debug("Skipped existing file", "path", filePath)
	} else {
		slog.Info("Wrote file", "path", filePath)
	}

	return nil
}

func mapTypeToType(titleType string) string {
	switch titleType {
	case "Movie":
		return "movie"
	case "TV Series":
		return "tv-series"
	case "TV Mini Series":
		return "miniseries"
	// ... add other types as needed
	default:
		return strings.ToLower(titleType)
	}
}

// mapTypeToTag maps a imdb title type to a markdown tag
func mapTypeToTag(titleType string) string {
	switch titleType {
	case "Video Game":
		return "imdb/videogame"
	case "TV Series":
		return "imdb/tv-series"
	case "TV Special":
		return "imdb/tv-special"
	case "TV Mini Series":
		return "imdb/miniseries"
	case "TV Episode":
		return "imdb/tv-episode"
	case "TV Movie":
		return "imdb/tv-movie"
	case "TV Short":
		return "imdb/tv-short"
	case "Movie":
		return "imdb/movie"
	case "Video":
		return "imdb/video"
	case "Short":
		return "imdb/short-movie"
	case "Podcast Series":
		return "imdb/podcast"
	case "Podcast Episode":
		return "imdb/podcast-episode"
	default:
		slog.Warn("Unknown title type", "type", titleType)
		return "UNKNOWN"
	}
}

func (m *MovieSeen) Validate() error {
	if m.ImdbId == "" {
		return fmt.Errorf("missing required field: ImdbId")
	}
	if m.Title == "" {
		return fmt.Errorf("missing required field: Title")
	}
	if m.Year < 1800 || m.Year > time.Now().Year()+5 {
		return fmt.Errorf("invalid year: %d", m.Year)
	}
	return nil
}
