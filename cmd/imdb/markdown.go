package imdb

import (
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/lepinkainen/hermes/internal/config"
	"github.com/lepinkainen/hermes/internal/content"
	"github.com/lepinkainen/hermes/internal/fileutil"
	"github.com/lepinkainen/hermes/internal/obsidian"
)

const defaultCoverWidth = 250

// writeMovieToMarkdown writes movie info to a markdown file
func writeMovieToMarkdown(movie MovieSeen, directory string) error {
	// Get the file path using the common utility
	filePath := fileutil.GetMarkdownFilePath(movie.Title, directory)

	// Handle titles - remove problematic characters and handle original titles
	movie.Title = fileutil.SanitizeFilename(movie.Title)
	fm := obsidian.NewFrontmatterWithTitle(movie.Title)

	if movie.OriginalTitle != "" && movie.OriginalTitle != movie.Title {
		movie.OriginalTitle = fileutil.SanitizeFilename(movie.OriginalTitle)
		fm.Set("original_title", movie.OriginalTitle)
	}

	// Add type-specific metadata
	fm.Set("type", mapTypeToType(movie.TitleType))

	// Add seen flag if movie has a rating
	if movie.MyRating > 0 {
		fm.Set("seen", true)
	}

	// Basic metadata
	fm.Set("imdb_id", movie.ImdbId)
	if movie.Year > 0 {
		fm.Set("year", movie.Year)
	}
	if movie.IMDbRating > 0 {
		fm.Set("imdb_rating", movie.IMDbRating)
	}
	if movie.MyRating > 0 {
		fm.Set("my_rating", movie.MyRating)
	}

	// Format date in a more readable way
	if movie.DateRated != "" {
		fm.Set("date_rated", movie.DateRated)
	}

	// Add runtime and duration
	if movie.RuntimeMins > 0 {
		fm.Set("runtime", movie.RuntimeMins)
		hours := movie.RuntimeMins / 60
		mins := movie.RuntimeMins % 60
		fm.Set("duration", fmt.Sprintf("%dh %dm", hours, mins))
	}

	// Add directors as an array
	if len(movie.Directors) > 0 {
		fm.Set("directors", movie.Directors)
	}

	// Collect all tags using TagSet for deduplication and normalization
	tc := obsidian.NewTagSet()

	// Add type tag
	typeTag := mapTypeToTag(movie.TitleType)
	tc.AddIf(typeTag != "UNKNOWN", typeTag)

	// Add rating tag if available
	if movie.MyRating > 0 {
		tc.AddFormat("rating/%d", movie.MyRating)
	}

	// Add decade tag
	if movie.Year > 0 {
		decade := (movie.Year / 10) * 10
		tc.AddFormat("year/%ds", decade)
	}

	// Add genres as tags with genre/ prefix
	for _, genre := range movie.Genres {
		tc.AddFormat("genre/%s", genre)
	}

	// Add TMDB genre tags if available
	if movie.TMDBEnrichment != nil {
		for _, tmdbTag := range movie.TMDBEnrichment.GenreTags {
			tc.Add(tmdbTag)
		}
	}

	obsidian.ApplyTagSet(fm, tc)

	// Add content rating if available
	if movie.ContentRated != "" {
		fm.Set("content_rating", movie.ContentRated)
	}

	// Add awards if available
	if movie.Awards != "" {
		fm.Set("awards", movie.Awards)
	}

	// Handle cover image
	coverFilename := ""
	if movie.TMDBEnrichment != nil && movie.TMDBEnrichment.CoverFilename != "" {
		coverFilename = movie.TMDBEnrichment.CoverFilename
		fm.Set("cover", movie.TMDBEnrichment.CoverPath)
	} else if movie.PosterURL != "" {
		// Download cover from poster URL
		coverOpts := fileutil.CoverDownloadOptions{
			URL:          movie.PosterURL,
			OutputDir:    directory,
			Filename:     fileutil.BuildCoverFilename(movie.Title),
			UpdateCovers: config.UpdateCovers,
		}
		result, err := fileutil.DownloadCover(coverOpts)
		if err == nil && result != nil {
			coverFilename = result.Filename
			fm.Set("cover", result.RelativePath)
		}
	}

	// Add TMDB frontmatter fields if available
	if movie.TMDBEnrichment != nil {
		if movie.TMDBEnrichment.TMDBID > 0 {
			fm.Set("tmdb_id", movie.TMDBEnrichment.TMDBID)
			fm.Set("tmdb_type", movie.TMDBEnrichment.TMDBType)
		}
		if movie.TMDBEnrichment.TotalEpisodes > 0 {
			fm.Set("total_episodes", movie.TMDBEnrichment.TotalEpisodes)
		}
	}

	// Build body content
	var body strings.Builder

	// Add cover image using Obsidian syntax
	if coverFilename != "" {
		body.WriteString(fmt.Sprintf("![[%s|%d]]\n\n", coverFilename, defaultCoverWidth))
	}

	// Build IMDb content sections
	imdbDetails := &content.IMDbMovieDetails{
		Title:         movie.Title,
		OriginalTitle: movie.OriginalTitle,
		Year:          movie.Year,
		TitleType:     movie.TitleType,
		MyRating:      movie.MyRating,
		IMDbRating:    movie.IMDbRating,
		DateRated:     movie.DateRated,
		Runtime:       movie.RuntimeMins,
		Directors:     movie.Directors,
		Genres:        movie.Genres,
		ContentRating: movie.ContentRated,
		Awards:        movie.Awards,
		Plot:          movie.Plot,
		IMDbID:        movie.ImdbId,
		URL:           movie.URL,
	}

	imdbContent := content.BuildIMDbContent(imdbDetails, []string{"info", "plot", "awards"})
	if imdbContent != "" {
		wrappedIMDb := content.WrapWithIMDbMarkers(imdbContent)
		body.WriteString(wrappedIMDb)
		body.WriteString("\n\n")
	}

	// Add TMDB content if available
	if movie.TMDBEnrichment != nil && movie.TMDBEnrichment.ContentMarkdown != "" {
		wrappedTMDB := content.WrapWithMarkers(movie.TMDBEnrichment.ContentMarkdown)
		body.WriteString(wrappedTMDB)
		body.WriteString("\n")
	}

	// Create the note
	// Build markdown
	markdown, err := obsidian.BuildNoteMarkdown(fm, body.String())
	if err != nil {
		return fmt.Errorf("failed to build markdown: %w", err)
	}

	// Write content to file with logging
	return fileutil.WriteMarkdownFile(filePath, string(markdown), config.OverwriteFiles)
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
