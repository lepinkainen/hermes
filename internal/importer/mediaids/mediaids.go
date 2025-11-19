package mediaids

import (
	"fmt"
	"os"
	"strings"

	"github.com/lepinkainen/hermes/internal/frontmatter"
)

// MediaIDs collects external identifiers from a markdown note.
type MediaIDs struct {
	TMDBID       int
	TMDBType     string
	IMDBID       string
	LetterboxdID string
}

// FromFrontmatter extracts all supported IDs from a parsed frontmatter map.
func FromFrontmatter(fm map[string]any) MediaIDs {
	if fm == nil {
		return MediaIDs{}
	}

	return MediaIDs{
		TMDBID:       frontmatter.IntFromAny(fm["tmdb_id"]),
		TMDBType:     frontmatter.StringFromAny(fm["tmdb_type"]),
		IMDBID:       frontmatter.StringFromAny(fm["imdb_id"]),
		LetterboxdID: frontmatter.StringFromAny(fm["letterboxd_id"]),
	}
}

// FromFile parses a markdown file and returns its external IDs.
func FromFile(path string) (MediaIDs, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return MediaIDs{}, err
	}

	note, err := frontmatter.ParseMarkdown(data)
	if err != nil {
		return MediaIDs{}, err
	}

	return FromFrontmatter(note.Frontmatter), nil
}

// HasAny reports whether the struct contains at least one identifier.
func (ids MediaIDs) HasAny() bool {
	return ids.TMDBID != 0 || ids.IMDBID != "" || ids.LetterboxdID != ""
}

// Summary renders a short, human-friendly description of all found IDs.
func (ids MediaIDs) Summary() string {
	var parts []string
	if ids.TMDBID != 0 {
		parts = append(parts, fmt.Sprintf("tmdb:%d", ids.TMDBID))
	}
	if ids.IMDBID != "" {
		parts = append(parts, fmt.Sprintf("imdb:%s", ids.IMDBID))
	}
	if ids.LetterboxdID != "" {
		parts = append(parts, fmt.Sprintf("letterboxd:%s", ids.LetterboxdID))
	}

	if len(parts) == 0 {
		return "no IDs"
	}

	return strings.Join(parts, ", ")
}
