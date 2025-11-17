package letterboxd

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/lepinkainen/hermes/internal/content"
	"github.com/lepinkainen/hermes/internal/fileutil"
)

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

	// Add genres if available
	if len(movie.Genres) > 0 {
		mb.AddStringArray("tags", movie.Genres)
	}

	// Add standard tags
	tags := []string{"letterboxd/movie"}

	// Add rating tag if available
	if movie.Rating > 0 {
		tags = append(tags, fmt.Sprintf("rating/%.1f", movie.Rating))
	}

	// Add decade tag
	tags = append(tags, mb.GetDecadeTag(movie.Year))
	mb.AddTags(tags...)

	// Add Letterboxd IDs
	mb.AddField("letterboxd_uri", movie.LetterboxdURI)
	mb.AddField("letterboxd_id", movie.LetterboxdID)

	// Add IMDb ID if available
	if movie.ImdbID != "" {
		mb.AddField("imdb_id", movie.ImdbID)
	}

	// Add poster URL if available
	if movie.PosterURL != "" {
		mb.AddField("cover", movie.PosterURL)
	}

	// Add poster image if available
	if movie.PosterURL != "" {
		mb.AddImage(movie.PosterURL)
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

		// Add TMDB genre tags (merge with existing tags)
		if len(movie.TMDBEnrichment.GenreTags) > 0 {
			mb.AddTags(movie.TMDBEnrichment.GenreTags...)
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
