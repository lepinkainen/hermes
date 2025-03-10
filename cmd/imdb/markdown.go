package imdb

import (
	"fmt"
	"strings"
	"time"

	"github.com/lepinkainen/hermes/internal/config"
	"github.com/lepinkainen/hermes/internal/fileutil"
	log "github.com/sirupsen/logrus"
)

// writeMovieToMarkdown writes movie info to a markdown file
func writeMovieToMarkdown(movie MovieSeen, directory string) error {
	// Get the file path using the common utility
	filePath := fileutil.GetMarkdownFilePath(movie.Title, directory)

	// Use the MarkdownBuilder to construct the document
	mb := fileutil.NewMarkdownBuilder()

	// Handle titles - remove problematic characters and handle original titles
	movie.Title = sanitizeTitle(movie.Title)
	mb.AddTitle(movie.Title)

	if movie.OriginalTitle != "" && movie.OriginalTitle != movie.Title {
		movie.OriginalTitle = sanitizeTitle(movie.OriginalTitle)
		mb.AddField("original_title", movie.OriginalTitle)
	}

	// Add type-specific metadata
	mb.AddType(mapTypeToType(movie.TitleType))

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
		mb.AddField("runtime_mins", movie.RuntimeMins)
		mb.AddDuration(movie.RuntimeMins)
	}

	// Add genres as an array
	if len(movie.Genres) > 0 {
		mb.AddStringArray("genres", movie.Genres)
	}

	// Add directors as an array
	if len(movie.Directors) > 0 {
		mb.AddStringArray("directors", movie.Directors)
	}

	// Add Obsidian-specific tags
	tags := []string{
		mapTypeToTag(movie.TitleType),            // e.g., #imdb/movie
		fmt.Sprintf("rating/%d", movie.MyRating), // e.g., #rating/8
	}

	// Add decade tag
	decade := (movie.Year / 10) * 10
	tags = append(tags, fmt.Sprintf("year/%ds", decade)) // e.g., #year/1990s

	mb.AddTags(tags...)

	// Add content rating if available
	if movie.ContentRated != "" {
		mb.AddField("content_rating", movie.ContentRated)
	}

	// Add awards if available
	if movie.Awards != "" {
		mb.AddField("awards", movie.Awards)
	}

	// Add poster image if available
	if movie.PosterURL != "" {
		mb.AddImage(movie.PosterURL)
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

	// Write content to file with overwrite logic
	written, err := fileutil.WriteFileWithOverwrite(filePath, []byte(mb.Build()), 0644, config.OverwriteFiles)
	if err != nil {
		return err
	}

	if !written {
		log.Debugf("Skipped existing file: %s", filePath)
	} else {
		log.Infof("Wrote %s", filePath)
	}

	return nil
}

func sanitizeTitle(title string) string {
	return strings.ReplaceAll(title, ":", "")
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
		log.Warnf("Unknown title type '%s'\n", titleType)
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
