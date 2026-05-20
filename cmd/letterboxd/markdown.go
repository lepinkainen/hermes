package letterboxd

import (
	"fmt"
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
	fm := obsidian.NewFrontmatterWithTitle(fileutil.SanitizeFilename(movie.Name))

	// Add basic metadata
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
		fm.Set("duration", fileutil.FormatDuration(movie.Runtime))
	}

	// Add director if available
	if movie.Director != "" {
		fm.Set("directors", []string{movie.Director})
	}

	// Collect all tags using TagSet for deduplication and normalization
	tc := obsidian.NewTagSet()
	tc.Add("letterboxd/movie")

	tc.AddRatingTag(movie.Rating)
	tc.AddDecadeTag(movie.Year)
	tc.AddGenreTags(movie.Genres)

	// Add TMDB genre tags if available
	if movie.TMDBEnrichment != nil {
		for _, tmdbTag := range movie.TMDBEnrichment.GenreTags {
			tc.Add(tmdbTag)
		}
	}

	obsidian.ApplyTagSet(fm, tc)

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
		fmt.Fprintf(&body, "![[%s|%d]]\n\n", coverFilename, defaultCoverWidth)
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
	// Build markdown
	markdown, err := obsidian.BuildNoteMarkdown(fm, body.String())
	if err != nil {
		return fmt.Errorf("failed to build markdown: %w", err)
	}

	// Write content to file with logging
	return fileutil.WriteMarkdownFile(filePath, string(markdown), config.OverwriteFiles)
}
