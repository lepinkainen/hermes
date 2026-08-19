package enhance

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/lepinkainen/hermes/internal/content"
	"github.com/lepinkainen/hermes/internal/enrichment"
	"github.com/lepinkainen/hermes/internal/enrichment/omdb"
	"github.com/lepinkainen/hermes/internal/importer/mediaids"
	"github.com/lepinkainen/hermes/internal/obsidian"
)

// Note represents a parsed markdown note with YAML frontmatter.
type Note struct {
	// Frontmatter fields (typed for convenience)
	Title        string
	Type         string // "movie", "tv", "game", or "book"
	Year         int
	IMDBID       string
	TMDBID       int
	LetterboxdID string
	SteamAppID   int
	RAWGID       int
	Seen         bool
	ISBN         string
	ISBN13       string
	GoodreadsID  string

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
	note.Type = obsidian.DetectMediaType(note.Frontmatter)

	note.Year = note.Frontmatter.GetInt("year")

	ids := mediaids.FromFrontmatter(note.Frontmatter)
	note.IMDBID = ids.IMDBID
	note.TMDBID = ids.TMDBID
	note.LetterboxdID = ids.LetterboxdID

	note.Seen = note.Frontmatter.GetBool("seen")
	note.SteamAppID = note.Frontmatter.GetInt("steam_appid")
	note.RAWGID = note.Frontmatter.GetInt("rawg_id")

	note.ISBN = note.Frontmatter.GetString("isbn")
	note.ISBN13 = note.Frontmatter.GetString("isbn13")
	note.GoodreadsID = note.Frontmatter.GetString("goodreads_id")

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

// IsBook returns true if this note is detected as a book note.
func (n *Note) IsBook() bool {
	return n.Type == "book"
}

// NeedsGoodreadsContent checks if the note needs Goodreads-style book content
// sections. Returns true if the content markers are missing from the body.
func (n *Note) NeedsGoodreadsContent() bool {
	return !content.HasGoodreadsContentMarkers(n.Body)
}

// HasSteamData checks if the note already has Steam data in both frontmatter and body.
// Returns true only if both Steam AppID exists in frontmatter AND content markers exist in body.
func (n *Note) HasSteamData() bool {
	return n.SteamAppID != 0 && content.HasSteamContentMarkers(n.Body)
}

// HasOMDBData checks if the note has OMDB ratings in frontmatter.
func (n *Note) HasOMDBData() bool {
	// Check if any OMDB rating field exists
	if _, ok := n.Frontmatter.Get("imdb_rating"); ok {
		return true
	}
	if n.Frontmatter.GetString("rt_score") != "" {
		return true
	}
	if n.Frontmatter.GetInt("metacritic_score") > 0 {
		return true
	}
	return false
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

	if tmdbData.IMDBID != "" {
		n.Frontmatter.Set("imdb_id", tmdbData.IMDBID)
		n.IMDBID = tmdbData.IMDBID
	}

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
		// Replace tool-managed genre/* tags rather than union-merging, so
		// stale genres from a prior match don't accumulate on regenerate.
		existingTags := stripGenreTags(n.Frontmatter.GetStringArray("tags"))
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

// AddGameData adds RAWG game enrichment data to the note's frontmatter.
// This is the fallback path used when Steam has no listing for a title
// (console/handheld exclusives); it writes rawg_id instead of steam_appid.
func (n *Note) AddGameData(gameData *enrichment.GameEnrichment) {
	if gameData == nil {
		return
	}

	// RAWG-sourced note: this title isn't on Steam, so drop any stale
	// steam_appid a prior (mis)match may have written — otherwise a bogus
	// Steam id lingers next to the correct rawg_id.
	n.Frontmatter.Delete("steam_appid")
	n.SteamAppID = 0

	n.Frontmatter.Set("rawg_id", gameData.RAWGID)
	n.RAWGID = gameData.RAWGID

	if len(gameData.GenreTags) > 0 {
		// Replace tool-managed genre/* tags with the authoritative source's
		// genres; union-merging would let stale genres from a prior wrong
		// match accumulate. Non-genre tags are preserved.
		existingTags := stripGenreTags(n.Frontmatter.GetStringArray("tags"))
		mergedTags := obsidian.MergeTags(existingTags, gameData.GenreTags)
		n.Frontmatter.Set("tags", mergedTags)
	}

	if gameData.CoverPath != "" {
		n.Frontmatter.Set("cover", gameData.CoverPath)
	}

	if len(gameData.Developers) > 0 {
		n.Frontmatter.Set("developers", gameData.Developers)
	}

	if len(gameData.Publishers) > 0 {
		n.Frontmatter.Set("publishers", gameData.Publishers)
	}

	if gameData.ReleaseDate != "" {
		n.Frontmatter.Set("release_date", gameData.ReleaseDate)
	}

	if gameData.MetacriticScore > 0 {
		n.Frontmatter.Set("metacritic_score", gameData.MetacriticScore)
	}
}

// stripGenreTags drops tool-managed "genre/*" tags, preserving all others.
// Genre tags are owned by the enrichment source, so on (re)enrichment they
// are replaced wholesale rather than union-merged (which would let stale
// genres from a prior, possibly wrong, match linger).
func stripGenreTags(tags []string) []string {
	kept := make([]string, 0, len(tags))
	for _, t := range tags {
		if strings.HasPrefix(t, "genre/") {
			continue
		}
		kept = append(kept, t)
	}
	return kept
}

// AddBookData adds book enrichment data to the note's frontmatter.
// Unlike AddTMDBData/AddSteamData, fields are only filled in when currently
// empty/missing: Goodreads-exported data is treated as authoritative, and
// enrichment only fills gaps left by the original import.
func (n *Note) AddBookData(d *enrichment.BookEnrichment) {
	if d == nil {
		return
	}

	if n.Frontmatter.GetString("isbn") == "" && d.ISBN != "" {
		n.Frontmatter.Set("isbn", d.ISBN)
		n.ISBN = d.ISBN
	}

	if n.Frontmatter.GetString("isbn13") == "" && d.ISBN13 != "" {
		n.Frontmatter.Set("isbn13", d.ISBN13)
		n.ISBN13 = d.ISBN13
	}

	if n.Frontmatter.GetString("cover") == "" && d.CoverPath != "" {
		n.Frontmatter.Set("cover", d.CoverPath)
	}

	if n.Frontmatter.GetInt("pages") == 0 && d.Pages > 0 {
		n.Frontmatter.Set("pages", d.Pages)
	}

	if n.Frontmatter.GetString("publisher") == "" && d.Publisher != "" {
		n.Frontmatter.Set("publisher", d.Publisher)
	}

	if n.Frontmatter.GetString("description") == "" && d.Description != "" {
		n.Frontmatter.Set("description", d.Description)
	}

	if n.Frontmatter.GetString("subtitle") == "" && d.Subtitle != "" {
		n.Frontmatter.Set("subtitle", d.Subtitle)
	}

	if len(n.Frontmatter.GetStringArray("authors")) == 0 && len(d.Authors) > 0 {
		n.Frontmatter.Set("authors", d.Authors)
	}

	if len(n.Frontmatter.GetStringArray("subjects")) == 0 && len(d.Subjects) > 0 {
		n.Frontmatter.Set("subjects", d.Subjects)
	}

	if len(n.Frontmatter.GetStringArray("subject_people")) == 0 && len(d.SubjectPeople) > 0 {
		n.Frontmatter.Set("subject_people", d.SubjectPeople)
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

// BuildMarkdownForGame builds the complete markdown content with updated
// frontmatter and RAWG game content. It reuses the Steam content markers
// (content.WrapWithSteamMarkers/ReplaceSteamContent) so NeedsSteamContent's
// skip logic works uniformly regardless of whether a game note was
// enriched from Steam or RAWG.
func (n *Note) BuildMarkdownForGame(originalContent string, gameData *enrichment.GameEnrichment, overwrite bool) string {
	body := n.Body
	if gameData != nil && gameData.ContentMarkdown != "" {
		if content.HasSteamContentMarkers(body) {
			// Replace existing content between markers
			if overwrite {
				body = content.ReplaceSteamContent(body, gameData.ContentMarkdown)
			}
		} else {
			// No markers exist - append wrapped content
			wrappedContent := content.WrapWithSteamMarkers(gameData.ContentMarkdown)
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

// BuildMarkdownForBook builds the complete markdown content with updated
// frontmatter and Goodreads-style book content.
func (n *Note) BuildMarkdownForBook(originalContent string, d *enrichment.BookEnrichment, regenerate bool) string {
	// Handle book content with marker-based replacement
	body := n.Body
	if d != nil && d.ContentMarkdown != "" {
		if content.HasGoodreadsContentMarkers(body) {
			// Replace existing content between markers
			if regenerate {
				body = content.ReplaceGoodreadsContent(body, d.ContentMarkdown)
			}
		} else {
			// No markers exist - append wrapped content
			wrappedContent := content.WrapWithGoodreadsMarkers(d.ContentMarkdown)
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

// buildGoodreadsBaseDetails builds a GoodreadsBookDetails from the note's
// existing frontmatter, to be used as BookEnrichmentOptions.BaseDetails.
// Enrichment only fills fields left empty here.
func (n *Note) buildGoodreadsBaseDetails() *content.GoodreadsBookDetails {
	fm := n.Frontmatter
	return &content.GoodreadsBookDetails{
		Title:                   n.Title,
		Subtitle:                fm.GetString("subtitle"),
		Authors:                 fm.GetStringArray("authors"),
		Publisher:               fm.GetString("publisher"),
		Pages:                   fm.GetInt("pages"),
		YearPublished:           n.Year,
		OriginalPublicationYear: fm.GetInt("original_year"),
		MyRating:                floatFromFrontmatter(fm, "my_rating"),
		AverageRating:           floatFromFrontmatter(fm, "average_rating"),
		ISBN:                    n.ISBN,
		ISBN13:                  n.ISBN13,
		Binding:                 fm.GetString("binding"),
		GoodreadsID:             n.GoodreadsID,
		Description:             fm.GetString("description"),
		Subjects:                fm.GetStringArray("subjects"),
		SubjectPeople:           fm.GetStringArray("subject_people"),
	}
}

// floatFromFrontmatter reads a numeric frontmatter field as a float64.
// YAML unmarshaling can produce int, int64, float64, or string for numeric
// fields (e.g. my_rating may be an int like "5" or a float like "4.5");
// all are handled. Returns 0 if the field is missing or not convertible.
func floatFromFrontmatter(fm *obsidian.Frontmatter, key string) float64 {
	val, ok := fm.Get(key)
	if !ok {
		return 0
	}
	switch v := val.(type) {
	case float64:
		return v
	case float32:
		return float64(v)
	case int:
		return float64(v)
	case int64:
		return float64(v)
	case string:
		if f, err := strconv.ParseFloat(strings.TrimSpace(v), 64); err == nil {
			return f
		}
	}
	return 0
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

// parseTitleYearFromTitle extracts a year from titles like "Legend (2015)".
// Returns the cleaned title, parsed year, and true if a year was found.
func parseTitleYearFromTitle(title string) (string, int, bool) {
	trimmed := strings.TrimSpace(title)
	if !strings.HasSuffix(trimmed, ")") {
		return "", 0, false
	}

	start := strings.LastIndex(trimmed, " (")
	if start == -1 {
		return "", 0, false
	}

	yearStr := trimmed[start+2 : len(trimmed)-1]
	if len(yearStr) != 4 {
		return "", 0, false
	}

	year, err := strconv.Atoi(yearStr)
	if err != nil {
		return "", 0, false
	}

	cleanTitle := strings.TrimSpace(trimmed[:start])
	if cleanTitle == "" {
		return "", 0, false
	}

	return cleanTitle, year, true
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
