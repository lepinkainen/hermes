package letterboxd

import (
	"fmt"
	"log/slog"
	"math"
	"sort"
	"strings"

	"github.com/lepinkainen/hermes/internal/config"
	"github.com/lepinkainen/hermes/internal/content"
	"github.com/lepinkainen/hermes/internal/fileutil"
)

const defaultCoverWidth = 250

// writeMovieToMarkdown writes a single movie to a markdown file
func writeMovieToMarkdown(movie Movie, directory string) error {
	// Get the file path using the common utility
	// For Letterboxd, we want to include the year in the filename
	title := fmt.Sprintf("%s (%d)", movie.Name, movie.Year)
	filePath := fileutil.GetMarkdownFilePath(title, directory)

	// Use the MarkdownBuilder to construct the document
	mb := fileutil.NewMarkdownBuilder()

	// Add basic metadata
	mb.AddTitle(fileutil.SanitizeFilename(movie.Name))
	mb.AddType("movie")
	mb.AddYear(movie.Year)

	// Add date watched if available
	if movie.Date != "" {
		mb.AddDate("date_watched", movie.Date)
	}

	// Add rating if available
	if movie.Rating > 0 {
		mb.AddField("letterboxd_rating", movie.Rating)
		mb.AddField("seen", true)
	}

	// Add duration if available
	if movie.Runtime > 0 {
		mb.AddField("runtime", movie.Runtime)
		mb.AddDuration(movie.Runtime)
	}

	// Add director if available
	if movie.Director != "" {
		mb.AddStringArray("directors", []string{movie.Director})
	}

	// Collect all tags using a map for deduplication
	tagMap := make(map[string]bool)
	tagMap["letterboxd/movie"] = true

	// Add rating tag if available (rounded to integer)
	if movie.Rating > 0 {
		ratingTag := fmt.Sprintf("rating/%d", int(math.Round(movie.Rating)))
		tagMap[ratingTag] = true
	}

	// Add decade tag
	decadeTag := mb.GetDecadeTag(movie.Year)
	tagMap[decadeTag] = true

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

	// Add Letterboxd IDs
	mb.AddField("letterboxd_uri", movie.LetterboxdURI)
	mb.AddField("letterboxd_id", movie.LetterboxdID)

	// Add IMDb ID if available
	if movie.ImdbID != "" {
		mb.AddField("imdb_id", movie.ImdbID)
	}

	// Add poster - prefer TMDB cover (higher resolution), fall back to OMDB poster
	coverAdded := false

	// First check if TMDB cover is available (downloaded during enrichment)
	if movie.TMDBEnrichment != nil && movie.TMDBEnrichment.CoverPath != "" {
		mb.AddField("cover", movie.TMDBEnrichment.CoverPath)
		mb.AddObsidianImage(movie.TMDBEnrichment.CoverFilename, defaultCoverWidth)
		coverAdded = true
	}

	// Fall back to OMDB poster if no TMDB cover
	if !coverAdded && movie.PosterURL != "" {
		coverFilename := fileutil.BuildCoverFilename(movie.Name)
		result, err := fileutil.DownloadCover(fileutil.CoverDownloadOptions{
			URL:          movie.PosterURL,
			OutputDir:    directory,
			Filename:     coverFilename,
			UpdateCovers: config.UpdateCovers,
		})
		if err != nil {
			slog.Warn("Failed to download cover", "title", movie.Name, "error", err)
			// Fall back to URL if download fails
			mb.AddField("cover", movie.PosterURL)
			mb.AddImage(movie.PosterURL)
		} else if result != nil {
			// Use local path in frontmatter
			mb.AddField("cover", result.RelativePath)
			mb.AddObsidianImage(result.Filename, defaultCoverWidth)
		}
	}

	// Add description/plot if available
	if movie.Description != "" {
		mb.AddCallout("summary", "Plot", movie.Description)
	}

	// Add cast if available
	if len(movie.Cast) > 0 {
		var castContent strings.Builder
		for _, actor := range movie.Cast {
			fmt.Fprintf(&castContent, "- %s\n", actor)
		}
		mb.AddCallout("cast", "Cast", castContent.String())
	}

	// Add external links callout with deterministic ordering
	var linksContent strings.Builder
	fmt.Fprintf(&linksContent, "[View on Letterboxd](%s)", movie.LetterboxdURI)

	if movie.ImdbID != "" {
		fmt.Fprintf(&linksContent, "\n[View on IMDb](%s)", fmt.Sprintf("https://www.imdb.com/title/%s", movie.ImdbID))
	}

	mb.AddCallout("info", "Letterboxd", linksContent.String())

	// Add TMDB data if available
	if movie.TMDBEnrichment != nil {
		// Add TMDB metadata to frontmatter
		if movie.TMDBEnrichment.TMDBID > 0 {
			mb.AddField("tmdb_id", movie.TMDBEnrichment.TMDBID)
			mb.AddField("tmdb_type", movie.TMDBEnrichment.TMDBType)
		}

		// Add total episodes for TV shows (shouldn't happen for Letterboxd, but good to have)
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

	// Write the content to file with the common utility that respects overwrite settings
	written, err := fileutil.WriteFileWithOverwrite(filePath, []byte(mb.Build()), 0644, overwrite)
	if err != nil {
		return err
	}

	if written {
		slog.Info("Wrote file", "path", filePath)
	} else {
		slog.Info("Skipped existing file", "path", filePath)
	}

	return nil
}
