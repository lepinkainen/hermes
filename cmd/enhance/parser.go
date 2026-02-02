package enhance

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/lepinkainen/hermes/internal/content"
	"github.com/lepinkainen/hermes/internal/enrichment"
	"github.com/lepinkainen/hermes/internal/enrichment/omdb"
	fm "github.com/lepinkainen/hermes/internal/frontmatter"
	"github.com/lepinkainen/hermes/internal/importer/mediaids"
	"github.com/lepinkainen/hermes/internal/obsidian"
)

// Note represents a parsed markdown note with YAML frontmatter.
type Note struct {
	// Frontmatter fields (typed for convenience)
	Title        string
	Type         string // "movie", "tv", or "game"
	Year         int
	IMDBID       string
	TMDBID       int
	LetterboxdID string
	SteamAppID   int
	Seen         bool

	// Structured frontmatter and content using obsidian package
	Frontmatter *obsidian.Frontmatter
	Body        string
}

// parseNoteFile parses a markdown file and extracts frontmatter and content.
func parseNoteFile(filePath string) (*Note, error) {
	fileContent, err := readFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	note, err := parseNote(fileContent)
	if err != nil {
		return nil, err
	}

	// If title is missing, extract from filename
	if note.Title == "" {
		note.Title = extractTitleFromPath(filePath)
		note.Frontmatter.Set("title", note.Title)
	}

	return note, nil
}

// parseNote parses markdown content with YAML frontmatter.
func parseNote(fileContent string) (*Note, error) {
	// Use obsidian package for parsing
	obsNote, err := obsidian.ParseMarkdown([]byte(fileContent))
	if err != nil {
		return nil, fmt.Errorf("failed to parse markdown: %w", err)
	}

	note := &Note{
		Frontmatter: obsNote.Frontmatter,
		Body:        obsNote.Body,
	}

	// Extract typed fields
	note.Title = note.Frontmatter.GetString("title")

	// Get type from tmdb_type field or detect from tags
	// Convert frontmatter to map for DetectMediaType
	frontmatterMap := make(map[string]any)
	for _, key := range note.Frontmatter.Keys() {
		if val, ok := note.Frontmatter.Get(key); ok {
			frontmatterMap[key] = val
		}
	}
	note.Type = fm.DetectMediaType(frontmatterMap)

	note.Year = note.Frontmatter.GetInt("year")

	ids := mediaids.FromFrontmatter(frontmatterMap)
	note.IMDBID = ids.IMDBID
	note.TMDBID = ids.TMDBID
	note.LetterboxdID = ids.LetterboxdID

	note.Seen = note.Frontmatter.GetBool("seen")
	note.SteamAppID = note.Frontmatter.GetInt("steam_appid")

	return note, nil
}

// HasTMDBData checks if the note already has TMDB data in both frontmatter and body.
// Returns true only if both TMDB ID exists in frontmatter AND content markers exist in body.
func (n *Note) HasTMDBData() bool {
	return n.TMDBID != 0 && content.HasTMDBContentMarkers(n.Body)
}

// NeedsCover checks if the note needs a cover image.
// Returns true if the cover field is missing, empty, or the file doesn't exist.
func (n *Note) NeedsCover(noteDir string) bool {
	cover := n.Frontmatter.GetString("cover")
	if cover == "" {
		return true
	}

	// Check if the cover file actually exists
	coverPath := filepath.Join(noteDir, cover)
	if _, err := os.Stat(coverPath); os.IsNotExist(err) {
		return true
	}

	return false
}

// NeedsContent checks if the note needs TMDB content sections.
// Returns true if TMDB content markers are missing from the body.
func (n *Note) NeedsContent() bool {
	return !content.HasTMDBContentMarkers(n.Body)
}

// IsGame returns true if this note is detected as a game note.
func (n *Note) IsGame() bool {
	return n.Type == "game"
}

// HasSteamData checks if the note already has Steam data in both frontmatter and body.
// Returns true only if both Steam AppID exists in frontmatter AND content markers exist in body.
func (n *Note) HasSteamData() bool {
	return n.SteamAppID != 0 && content.HasSteamContentMarkers(n.Body)
}

// NeedsSteamContent checks if the note needs Steam content sections.
// Returns true if Steam content markers are missing from the body.
func (n *Note) NeedsSteamContent() bool {
	return !content.HasSteamContentMarkers(n.Body)
}

// AddTMDBData adds TMDB enrichment data to the note's frontmatter.
func (n *Note) AddTMDBData(tmdbData *enrichment.TMDBEnrichment) {
	if tmdbData == nil {
		return
	}

	n.Frontmatter.Set("tmdb_id", tmdbData.TMDBID)
	n.Frontmatter.Set("tmdb_type", tmdbData.TMDBType)
	n.TMDBID = tmdbData.TMDBID

	if tmdbData.RuntimeMins > 0 {
		n.Frontmatter.Set("runtime", tmdbData.RuntimeMins)
	}

	if tmdbData.TotalEpisodes > 0 {
		n.Frontmatter.Set("total_episodes", tmdbData.TotalEpisodes)
	}

	if len(tmdbData.GenreTags) > 0 {
		// Merge with existing tags using obsidian utility
		existingTags := n.Frontmatter.GetStringArray("tags")
		mergedTags := obsidian.MergeTags(existingTags, tmdbData.GenreTags)
		n.Frontmatter.Set("tags", mergedTags)
	}

	if tmdbData.CoverPath != "" {
		n.Frontmatter.Set("cover", tmdbData.CoverPath)
	}

	// Set finished flag for TV shows based on TMDB status
	if tmdbData.Finished != nil {
		n.Frontmatter.Set("finished", *tmdbData.Finished)
	}

	// Set seen flag if movie has any rating but seen field is not already set
	if !n.hasSeenField() && n.hasAnyRating() {
		n.Frontmatter.Set("seen", true)
		n.Seen = true
	}
}

// AddSteamData adds Steam enrichment data to the note's frontmatter.
func (n *Note) AddSteamData(steamData *enrichment.SteamEnrichment) {
	if steamData == nil {
		return
	}

	n.Frontmatter.Set("steam_appid", steamData.SteamAppID)
	n.SteamAppID = steamData.SteamAppID

	if len(steamData.GenreTags) > 0 {
		// Merge with existing tags using obsidian utility
		existingTags := n.Frontmatter.GetStringArray("tags")
		mergedTags := obsidian.MergeTags(existingTags, steamData.GenreTags)
		n.Frontmatter.Set("tags", mergedTags)
	}

	if steamData.CoverPath != "" {
		n.Frontmatter.Set("cover", steamData.CoverPath)
	}

	if len(steamData.Developers) > 0 {
		n.Frontmatter.Set("developers", steamData.Developers)
	}

	if len(steamData.Publishers) > 0 {
		n.Frontmatter.Set("publishers", steamData.Publishers)
	}

	if steamData.ReleaseDate != "" {
		n.Frontmatter.Set("release_date", steamData.ReleaseDate)
	}

	if steamData.MetacriticScore > 0 {
		n.Frontmatter.Set("metacritic_score", steamData.MetacriticScore)
	}
}

// AddOMDBData adds OMDB ratings enrichment data to the note's frontmatter.
func (n *Note) AddOMDBData(omdbData *omdb.RatingsEnrichment) {
	if omdbData == nil {
		return
	}

	if omdbData.IMDbRating > 0 {
		n.Frontmatter.Set("imdb_rating", omdbData.IMDbRating)
	}

	if omdbData.RottenTomatoes != "" {
		n.Frontmatter.Set("rt_score", omdbData.RottenTomatoes)
	}

	if omdbData.RTTomatometer > 0 {
		n.Frontmatter.Set("rt_tomatometer", omdbData.RTTomatometer)
	}

	if omdbData.Metacritic > 0 {
		n.Frontmatter.Set("metacritic_score", omdbData.Metacritic)
	}
}

// BuildMarkdown builds the complete markdown content with updated frontmatter and content.
func (n *Note) BuildMarkdown(originalContent string, tmdbData *enrichment.TMDBEnrichment, omdbRatings *omdb.RatingsEnrichment, overwrite bool) string {
	// Handle TMDB content with marker-based replacement
	body := n.Body

	// Prepare content to insert: start with OMDB ratings table if available
	var contentToInsert strings.Builder
	if omdbRatings != nil {
		ratingsTable := omdb.BuildRatingsTable(omdbRatings)
		if ratingsTable != "" {
			contentToInsert.WriteString(ratingsTable)
			contentToInsert.WriteString("\n")
		}
	}

	// Add TMDB content after ratings
	if tmdbData != nil && tmdbData.ContentMarkdown != "" {
		contentToInsert.WriteString(tmdbData.ContentMarkdown)
	}

	// Only modify body if we have content to insert
	if contentToInsert.Len() > 0 {
		finalContent := contentToInsert.String()
		if content.HasTMDBContentMarkers(body) {
			// Replace existing TMDB content between markers
			if overwrite {
				body = content.ReplaceTMDBContent(body, finalContent)
			}
		} else {
			// No markers exist - append wrapped content
			wrappedContent := content.WrapWithMarkers(finalContent)
			body = strings.TrimRight(body, "\n")
			if body != "" {
				body += "\n\n"
			}
			body += wrappedContent
		}
	}

	// Build using obsidian package
	obsNote := &obsidian.Note{
		Frontmatter: n.Frontmatter,
		Body:        body,
	}

	result, err := obsNote.Build()
	if err != nil {
		// Fallback to original if building fails
		return originalContent
	}

	return string(result)
}

// BuildMarkdownForSteam builds the complete markdown content with updated frontmatter and Steam content.
func (n *Note) BuildMarkdownForSteam(originalContent string, steamData *enrichment.SteamEnrichment, overwrite bool) string {
	// Handle Steam content with marker-based replacement
	body := n.Body
	if steamData != nil && steamData.ContentMarkdown != "" {
		if content.HasSteamContentMarkers(body) {
			// Replace existing Steam content between markers
			if overwrite {
				body = content.ReplaceSteamContent(body, steamData.ContentMarkdown)
			}
		} else {
			// No markers exist - append wrapped content
			wrappedContent := content.WrapWithSteamMarkers(steamData.ContentMarkdown)
			body = strings.TrimRight(body, "\n")
			if body != "" {
				body += "\n\n"
			}
			body += wrappedContent
		}
	}

	// Build using obsidian package
	obsNote := &obsidian.Note{
		Frontmatter: n.Frontmatter,
		Body:        body,
	}

	result, err := obsNote.Build()
	if err != nil {
		// Fallback to original if building fails
		return originalContent
	}

	return string(result)
}

// readFile is a helper to read file content.
// This is separate for easier testing/mocking if needed.
func readFile(path string) (string, error) {
	fileContent, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(fileContent), nil
}

// extractTitleFromPath extracts a title from the file path.
// For example: "/path/to/Lilo & Stitch.md" -> "Lilo & Stitch"
func extractTitleFromPath(filePath string) string {
	// Get the base filename
	filename := filepath.Base(filePath)
	// Remove the .md extension
	title := strings.TrimSuffix(filename, filepath.Ext(filename))
	return title
}

// GetMediaIDs extracts all external media IDs from the frontmatter.
// Returns a struct containing any TMDB, IMDB, or Letterboxd IDs found.
func (n *Note) GetMediaIDs() mediaids.MediaIDs {
	return mediaids.MediaIDs{
		TMDBID:       n.TMDBID,
		IMDBID:       n.IMDBID,
		LetterboxdID: n.LetterboxdID,
	}
}

// HasAnyID checks if the note has any external ID (TMDB, IMDB, or Letterboxd).
// Returns true if at least one ID is present and non-empty.
func (n *Note) HasAnyID() bool {
	return n.GetMediaIDs().HasAny()
}

// GetIDSummary returns a formatted string summary of all available IDs.
// Useful for logging and debugging.
func (n *Note) GetIDSummary() string {
	return n.GetMediaIDs().Summary()
}

// hasSeenField checks if the note already has a seen field in frontmatter.
func (n *Note) hasSeenField() bool {
	_, exists := n.Frontmatter.Get("seen")
	return exists
}

// hasAnyRating checks if the note has any rating field (imdb_rating, my_rating, or letterboxd_rating).
func (n *Note) hasAnyRating() bool {
	// Check for IMDb rating
	if imdbRating, ok := n.Frontmatter.Get("imdb_rating"); ok {
		if rating, isFloat := imdbRating.(float64); isFloat && rating > 0 {
			return true
		}
		if rating, isInt := imdbRating.(int); isInt && rating > 0 {
			return true
		}
	}

	// Check for my_rating
	if myRating, ok := n.Frontmatter.Get("my_rating"); ok {
		if rating, isInt := myRating.(int); isInt && rating > 0 {
			return true
		}
		if rating, isFloat := myRating.(float64); isFloat && rating > 0 {
			return true
		}
	}

	// Check for letterboxd_rating
	if letterboxdRating, ok := n.Frontmatter.Get("letterboxd_rating"); ok {
		if rating, isFloat := letterboxdRating.(float64); isFloat && rating > 0 {
			return true
		}
		if rating, isInt := letterboxdRating.(int); isInt && rating > 0 {
			return true
		}
	}

	return false
}
