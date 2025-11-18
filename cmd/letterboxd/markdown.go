package letterboxd

import (
	"fmt"
	"math"
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

	// Collect all tags using TagCollector for deduplication
	tc := fileutil.NewTagCollector()
	tc.Add("letterboxd/movie")

	// Add rating tag if available (rounded to integer)
	if movie.Rating > 0 {
		tc.AddFormat("rating/%d", int(math.Round(movie.Rating)))
	}

	// Add decade tag
	tc.Add(mb.GetDecadeTag(movie.Year))

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

	mb.AddTags(tc.GetSorted()...)

	// Add Letterboxd IDs
	mb.AddField("letterboxd_uri", movie.LetterboxdURI)
	mb.AddField("letterboxd_id", movie.LetterboxdID)

	// Add IMDb ID if available
	if movie.ImdbID != "" {
		mb.AddField("imdb_id", movie.ImdbID)
	}

	// Add poster - prefer TMDB cover (higher resolution), fall back to OMDB poster
	coverOpts := fileutil.AddCoverOptions{
		FallbackURL:  movie.PosterURL,
		Title:        movie.Name,
		Directory:    directory,
		Width:        defaultCoverWidth,
		UpdateCovers: config.UpdateCovers,
	}
	if movie.TMDBEnrichment != nil {
		coverOpts.TMDBCoverPath = movie.TMDBEnrichment.CoverPath
		coverOpts.TMDBCoverFilename = movie.TMDBEnrichment.CoverFilename
	}
	fileutil.AddCoverToMarkdown(mb, coverOpts)

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
		wrappedContent := ""
		if movie.TMDBEnrichment.ContentMarkdown != "" {
			wrappedContent = content.WrapWithMarkers(movie.TMDBEnrichment.ContentMarkdown)
		}
		mb.AddTMDBEnrichmentFields(
			movie.TMDBEnrichment.TMDBID,
			movie.TMDBEnrichment.TMDBType,
			movie.TMDBEnrichment.TotalEpisodes,
			wrappedContent,
		)
	}

	// Write content to file with logging
	return fileutil.WriteMarkdownFile(filePath, mb.Build(), config.OverwriteFiles)
}
