package letterboxd

import (
	"fmt"
	"math"
	"strings"

	"github.com/lepinkainen/hermes/internal/config"
	"github.com/lepinkainen/hermes/internal/content"
	"github.com/lepinkainen/hermes/internal/fileutil"
	"github.com/lepinkainen/hermes/internal/obsidian"
)

const defaultCoverWidth = 250

// writeMovieToMarkdown writes a single movie to a markdown file
func writeMovieToMarkdown(movie Movie, directory string) error {
	// Get the file path using the common utility
	// For Letterboxd, we want to include the year in the filename
	title := fmt.Sprintf("%s (%d)", movie.Name, movie.Year)
	filePath := fileutil.GetMarkdownFilePath(title, directory)

	// Create frontmatter using obsidian.Note
	fm := obsidian.NewFrontmatter()

	// Add basic metadata
	fm.Set("title", fileutil.SanitizeFilename(movie.Name))
	fm.Set("type", "movie")
	if movie.Year > 0 {
		fm.Set("year", movie.Year)
	}

	// Add date watched if available
	if movie.Date != "" {
		fm.Set("date_watched", movie.Date)
	}

	// Add rating if available
	if movie.Rating > 0 {
		fm.Set("letterboxd_rating", movie.Rating)
		fm.Set("seen", true)
	}

	// Add duration if available
	if movie.Runtime > 0 {
		fm.Set("runtime", movie.Runtime)
		hours := movie.Runtime / 60
		mins := movie.Runtime % 60
		fm.Set("duration", fmt.Sprintf("%dh %dm", hours, mins))
	}

	// Add director if available
	if movie.Director != "" {
		fm.Set("directors", []string{movie.Director})
	}

	// Collect all tags using TagSet for deduplication and normalization
	tc := obsidian.NewTagSet()
	tc.Add("letterboxd/movie")

	// Add rating tag if available (rounded to integer)
	if movie.Rating > 0 {
		tc.AddFormat("rating/%d", int(math.Round(movie.Rating)))
	}

	// Add decade tag
	tc.Add(getDecadeTag(movie.Year))

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

	fm.Set("tags", tc.GetSorted())

	// Add Letterboxd IDs
	fm.Set("letterboxd_uri", movie.LetterboxdURI)
	fm.Set("letterboxd_id", movie.LetterboxdID)

	// Add IMDb ID if available
	if movie.ImdbID != "" {
		fm.Set("imdb_id", movie.ImdbID)
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
			Filename:     fileutil.BuildCoverFilename(movie.Name),
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

	// Build Letterboxd content sections
	letterboxdDetails := &content.LetterboxdMovieDetails{
		Title:         movie.Name,
		Year:          movie.Year,
		Rating:        movie.Rating,
		DateWatched:   movie.Date,
		Runtime:       movie.Runtime,
		Director:      movie.Director,
		Genres:        movie.Genres,
		Cast:          movie.Cast,
		Description:   movie.Description,
		LetterboxdURI: movie.LetterboxdURI,
		LetterboxdID:  movie.LetterboxdID,
		ImdbID:        movie.ImdbID,
	}

	letterboxdContent := content.BuildLetterboxdContent(letterboxdDetails, []string{"info", "description", "cast"})
	if letterboxdContent != "" {
		wrappedLetterboxd := content.WrapWithLetterboxdMarkers(letterboxdContent)
		body.WriteString(wrappedLetterboxd)
		body.WriteString("\n\n")
	}

	// Add TMDB content if available
	if movie.TMDBEnrichment != nil && movie.TMDBEnrichment.ContentMarkdown != "" {
		wrappedTMDB := content.WrapWithMarkers(movie.TMDBEnrichment.ContentMarkdown)
		body.WriteString(wrappedTMDB)
		body.WriteString("\n")
	}

	// Create the note
	note := &obsidian.Note{
		Frontmatter: fm,
		Body:        strings.TrimSpace(body.String()),
	}

	// Build markdown
	markdown, err := note.Build()
	if err != nil {
		return fmt.Errorf("failed to build markdown: %w", err)
	}

	// Write content to file with logging
	return fileutil.WriteMarkdownFile(filePath, string(markdown), config.OverwriteFiles)
}

// getDecadeTag returns a decade tag based on the year
func getDecadeTag(year int) string {
	switch {
	case year >= 2020:
		return "year/2020s"
	case year >= 2010:
		return "year/2010s"
	case year >= 2000:
		return "year/2000s"
	case year >= 1990:
		return "year/1990s"
	case year >= 1980:
		return "year/1980s"
	case year >= 1970:
		return "year/1970s"
	case year >= 1960:
		return "year/1960s"
	case year >= 1950:
		return "year/1950s"
	default:
		return "year/pre-1950s"
	}
}
