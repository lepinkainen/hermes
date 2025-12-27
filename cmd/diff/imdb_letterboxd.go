package diff

import (
	"database/sql"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/lepinkainen/hermes/internal/fileutil"
	"github.com/lepinkainen/hermes/internal/obsidian"
	_ "modernc.org/sqlite"
)

type imdbMovie struct {
	ImdbID        string
	Title         string
	OriginalTitle string
	Year          int
	URL           string
	MyRating      int
}

type letterboxdMovie struct {
	Name          string
	Year          int
	LetterboxdID  string
	LetterboxdURI string
	ImdbID        string
	Rating        float64
}

type diffItem struct {
	Title            string
	Year             int
	ImdbID           string
	ImdbURL          string
	LetterboxdURI    string
	ImdbRating       int
	LetterboxdRating float64
	FuzzyMatches     []diffMatch
}

type diffMatch struct {
	Title            string
	Year             int
	ImdbID           string
	ImdbURL          string
	LetterboxdURI    string
	ImdbRating       int
	LetterboxdRating float64
}

// BuildDiffReport builds a diff report comparing IMDb and Letterboxd entries.
// The returned diffReport can be used to generate either markdown or HTML output.
func BuildDiffReport(mainDBPath, cacheDBPath string, now time.Time) (*diffReport, error) {
	if !fileutil.FileExists(mainDBPath) {
		return nil, fmt.Errorf("main database not found: %s", mainDBPath)
	}

	mainDB, err := sql.Open("sqlite", mainDBPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open main database: %w", err)
	}
	defer func() { _ = mainDB.Close() }()

	imdbMovies, err := loadImdbMovies(mainDB)
	if err != nil {
		return nil, err
	}

	letterboxdMovies, err := loadLetterboxdMovies(mainDB)
	if err != nil {
		return nil, err
	}

	mapping := map[string]string{}
	if cacheDBPath != "" && fileutil.FileExists(cacheDBPath) {
		mapping, err = loadLetterboxdMappings(cacheDBPath)
		if err != nil {
			return nil, err
		}
	} else if cacheDBPath != "" {
		slog.Warn("Cache database not found, continuing without letterboxd mappings", "path", cacheDBPath)
	}

	applyLetterboxdMappings(letterboxdMovies, mapping)

	imdbOnly, letterboxdOnly, resolvedTitleYear := diffIMDbLetterboxd(imdbMovies, letterboxdMovies)
	slog.Info(
		"Diff summary",
		"imdb_only", len(imdbOnly),
		"letterboxd_only", len(letterboxdOnly),
	)

	stats := diffStats{
		imdbOnlyCount:       len(imdbOnly),
		letterboxdOnlyCount: len(letterboxdOnly),
		resolvedTitleYear:   resolvedTitleYear,
	}
	for _, item := range imdbOnly {
		if len(item.FuzzyMatches) > 0 {
			stats.imdbOnlyWithFuzzy++
		}
	}
	for _, item := range letterboxdOnly {
		if len(item.FuzzyMatches) > 0 {
			stats.letterboxdOnlyWithFuzzy++
		}
	}

	return &diffReport{
		ImdbOnly:       imdbOnly,
		LetterboxdOnly: letterboxdOnly,
		Stats:          stats,
		GeneratedAt:    now,
		MainDBPath:     mainDBPath,
		CacheDBPath:    cacheDBPath,
	}, nil
}

// BuildIMDbLetterboxdReport builds an Obsidian markdown report comparing IMDb and Letterboxd entries.
func BuildIMDbLetterboxdReport(mainDBPath, cacheDBPath string, now time.Time) (*obsidian.Note, error) {
	report, err := BuildDiffReport(mainDBPath, cacheDBPath, now)
	if err != nil {
		return nil, err
	}
	return buildDiffNote(report.ImdbOnly, report.LetterboxdOnly, report.Stats, report.GeneratedAt, report.MainDBPath, report.CacheDBPath), nil
}

type diffStats struct {
	imdbOnlyCount           int
	letterboxdOnlyCount     int
	resolvedTitleYear       int
	imdbOnlyWithFuzzy       int
	letterboxdOnlyWithFuzzy int
}

// diffReport contains all data needed for diff output generation.
type diffReport struct {
	ImdbOnly       []diffItem
	LetterboxdOnly []diffItem
	Stats          diffStats
	GeneratedAt    time.Time
	MainDBPath     string
	CacheDBPath    string
}

func loadImdbMovies(db *sql.DB) ([]imdbMovie, error) {
	rows, err := db.Query(`
		SELECT imdb_id, title, original_title, year, url, my_rating
		FROM imdb_movies
		WHERE lower(title_type) = 'movie'
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to query imdb movies: %w", err)
	}
	defer func() { _ = rows.Close() }()

	movies := []imdbMovie{}
	for rows.Next() {
		var imdbID, title, originalTitle, url sql.NullString
		var year sql.NullInt64
		var myRating sql.NullInt64

		if err := rows.Scan(&imdbID, &title, &originalTitle, &year, &url, &myRating); err != nil {
			return nil, fmt.Errorf("failed to scan imdb movie: %w", err)
		}

		movies = append(movies, imdbMovie{
			ImdbID:        imdbID.String,
			Title:         title.String,
			OriginalTitle: originalTitle.String,
			Year:          int(year.Int64),
			URL:           url.String,
			MyRating:      int(myRating.Int64),
		})
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read imdb movies: %w", err)
	}

	slog.Info("Loaded IMDb movies", "count", len(movies))
	return movies, nil
}

func loadLetterboxdMovies(db *sql.DB) ([]letterboxdMovie, error) {
	rows, err := db.Query(`
		SELECT name, year, letterboxd_id, letterboxd_uri, imdb_id, rating
		FROM letterboxd_movies
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to query letterboxd movies: %w", err)
	}
	defer func() { _ = rows.Close() }()

	movies := []letterboxdMovie{}
	for rows.Next() {
		var name, letterboxdID, letterboxdURI, imdbID sql.NullString
		var year sql.NullInt64
		var rating sql.NullFloat64

		if err := rows.Scan(&name, &year, &letterboxdID, &letterboxdURI, &imdbID, &rating); err != nil {
			return nil, fmt.Errorf("failed to scan letterboxd movie: %w", err)
		}

		movies = append(movies, letterboxdMovie{
			Name:          name.String,
			Year:          int(year.Int64),
			LetterboxdID:  letterboxdID.String,
			LetterboxdURI: letterboxdURI.String,
			ImdbID:        imdbID.String,
			Rating:        rating.Float64,
		})
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read letterboxd movies: %w", err)
	}

	slog.Info("Loaded Letterboxd movies", "count", len(movies))
	return movies, nil
}

func loadLetterboxdMappings(cacheDBPath string) (map[string]string, error) {
	db, err := sql.Open("sqlite", cacheDBPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open cache database: %w", err)
	}
	defer func() { _ = db.Close() }()

	rows, err := db.Query(`
		SELECT letterboxd_uri, imdb_id
		FROM letterboxd_mapping_cache
		WHERE imdb_id IS NOT NULL AND imdb_id != ''
	`)
	if err != nil {
		if strings.Contains(err.Error(), "no such table") {
			slog.Warn("letterboxd_mapping_cache table missing in cache database")
			return map[string]string{}, nil
		}
		return nil, fmt.Errorf("failed to query letterboxd mapping cache: %w", err)
	}
	defer func() { _ = rows.Close() }()

	mapping := make(map[string]string)
	for rows.Next() {
		var uri, imdbID sql.NullString
		if err := rows.Scan(&uri, &imdbID); err != nil {
			return nil, fmt.Errorf("failed to scan letterboxd mapping: %w", err)
		}
		if uri.String != "" && imdbID.String != "" {
			mapping[uri.String] = imdbID.String
		}
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read letterboxd mapping cache: %w", err)
	}

	slog.Info("Loaded Letterboxd mappings", "count", len(mapping))
	return mapping, nil
}

func applyLetterboxdMappings(movies []letterboxdMovie, mapping map[string]string) {
	if len(mapping) == 0 {
		return
	}
	for i := range movies {
		if movies[i].ImdbID != "" {
			continue
		}
		if movies[i].LetterboxdURI == "" {
			continue
		}
		if imdbID, ok := mapping[movies[i].LetterboxdURI]; ok {
			movies[i].ImdbID = imdbID
		}
	}
}

func diffIMDbLetterboxd(imdbMovies []imdbMovie, letterboxdMovies []letterboxdMovie) ([]diffItem, []diffItem, int) {
	imdbByID := make(map[string]imdbMovie)
	imdbByKey := make(map[string][]imdbMovie)
	for _, movie := range imdbMovies {
		if movie.ImdbID != "" {
			imdbByID[movie.ImdbID] = movie
		}
		addImdbIndex(imdbByKey, movie.Title, movie)
		if movie.OriginalTitle != "" && movie.OriginalTitle != movie.Title {
			addImdbIndex(imdbByKey, movie.OriginalTitle, movie)
		}
	}

	letterboxdByID := make(map[string]letterboxdMovie)
	letterboxdByKey := make(map[string][]letterboxdMovie)
	for _, movie := range letterboxdMovies {
		if movie.ImdbID != "" {
			letterboxdByID[movie.ImdbID] = movie
		}
		addLetterboxdIndex(letterboxdByKey, movie.Name, movie)
	}

	matchedImdb := map[string]bool{}
	matchedLetterboxd := map[string]bool{}

	for imdbID, imdbMovie := range imdbByID {
		letterboxdMovie, ok := letterboxdByID[imdbID]
		if !ok {
			continue
		}
		matchedImdb[imdbItemKey(imdbMovie)] = true
		matchedLetterboxd[letterboxdItemKey(letterboxdMovie)] = true
	}

	resolvedTitleYear := 0
	for key, imdbMatches := range imdbByKey {
		letterboxdMatches, ok := letterboxdByKey[key]
		if !ok {
			continue
		}
		imdbUnique := uniqueImdbMovies(imdbMatches)
		letterboxdUnique := uniqueLetterboxdMovies(letterboxdMatches)
		if len(imdbUnique) == 0 || len(letterboxdUnique) == 0 {
			continue
		}
		resolvedTitleYear++
		for _, movie := range imdbUnique {
			imdbKey := imdbItemKey(movie)
			if imdbKey != "" {
				matchedImdb[imdbKey] = true
			}
		}
		for _, movie := range letterboxdUnique {
			letterboxdKey := letterboxdItemKey(movie)
			if letterboxdKey != "" {
				matchedLetterboxd[letterboxdKey] = true
			}
		}
	}

	imdbOnly := []diffItem{}
	for _, movie := range imdbMovies {
		if matchedImdb[imdbItemKey(movie)] {
			continue
		}
		item := diffItem{
			Title:      displayTitle(movie.Title, movie.OriginalTitle, movie.ImdbID),
			Year:       movie.Year,
			ImdbID:     movie.ImdbID,
			ImdbURL:    imdbURL(movie.ImdbID, movie.URL),
			ImdbRating: movie.MyRating,
		}
		key := titleYearKey(movie.Title, movie.OriginalTitle, movie.Year)
		if key != "" {
			item.FuzzyMatches = buildLetterboxdMatches(letterboxdByKey[key])
		}
		imdbOnly = append(imdbOnly, item)
	}

	letterboxdOnly := []diffItem{}
	for _, movie := range letterboxdMovies {
		if matchedLetterboxd[letterboxdItemKey(movie)] {
			continue
		}
		item := diffItem{
			Title:            displayTitle(movie.Name, "", movie.LetterboxdID),
			Year:             movie.Year,
			ImdbID:           movie.ImdbID,
			ImdbURL:          imdbURL(movie.ImdbID, ""),
			LetterboxdURI:    movie.LetterboxdURI,
			LetterboxdRating: movie.Rating,
		}
		key := titleYearKey(movie.Name, "", movie.Year)
		if key != "" {
			item.FuzzyMatches = buildIMDbMatches(imdbByKey[key])
		}
		letterboxdOnly = append(letterboxdOnly, item)
	}

	sortDiffItems(imdbOnly)
	sortDiffItems(letterboxdOnly)
	return imdbOnly, letterboxdOnly, resolvedTitleYear
}

func imdbItemKey(movie imdbMovie) string {
	if movie.ImdbID != "" {
		return "imdb:" + movie.ImdbID
	}
	key := titleYearKey(movie.Title, movie.OriginalTitle, movie.Year)
	if key != "" {
		return "title:" + key
	}
	return ""
}

func letterboxdItemKey(movie letterboxdMovie) string {
	if movie.ImdbID != "" {
		return "imdb:" + movie.ImdbID
	}
	if movie.LetterboxdURI != "" {
		return "uri:" + movie.LetterboxdURI
	}
	if movie.LetterboxdID != "" {
		return "id:" + movie.LetterboxdID
	}
	key := titleYearKey(movie.Name, "", movie.Year)
	if key != "" {
		return "title:" + key
	}
	return ""
}

func uniqueImdbMovies(movies []imdbMovie) []imdbMovie {
	seen := map[string]bool{}
	result := make([]imdbMovie, 0, len(movies))
	for _, movie := range movies {
		key := imdbItemKey(movie)
		if key == "" || seen[key] {
			continue
		}
		seen[key] = true
		result = append(result, movie)
	}
	return result
}

func uniqueLetterboxdMovies(movies []letterboxdMovie) []letterboxdMovie {
	seen := map[string]bool{}
	result := make([]letterboxdMovie, 0, len(movies))
	for _, movie := range movies {
		key := letterboxdItemKey(movie)
		if key == "" || seen[key] {
			continue
		}
		seen[key] = true
		result = append(result, movie)
	}
	return result
}

func addImdbIndex(index map[string][]imdbMovie, title string, movie imdbMovie) {
	key := titleYearKey(title, "", movie.Year)
	if key == "" {
		return
	}
	index[key] = append(index[key], movie)
}

func addLetterboxdIndex(index map[string][]letterboxdMovie, title string, movie letterboxdMovie) {
	key := titleYearKey(title, "", movie.Year)
	if key == "" {
		return
	}
	index[key] = append(index[key], movie)
}

func titleYearKey(title, fallback string, year int) string {
	candidate := strings.TrimSpace(title)
	if candidate == "" {
		candidate = strings.TrimSpace(fallback)
	}
	if candidate == "" || year <= 0 {
		return ""
	}
	normalized := normalizeTitle(candidate)
	if normalized == "" {
		return ""
	}
	return fmt.Sprintf("%s|%d", normalized, year)
}

func normalizeTitle(title string) string {
	lower := strings.ToLower(strings.TrimSpace(title))
	if lower == "" {
		return ""
	}
	var b strings.Builder
	b.Grow(len(lower))
	lastSpace := false
	for _, r := range lower {
		if unicode.IsLetter(r) || unicode.IsNumber(r) {
			b.WriteRune(r)
			lastSpace = false
			continue
		}
		if !lastSpace {
			b.WriteRune(' ')
			lastSpace = true
		}
	}
	return strings.TrimSpace(b.String())
}

func displayTitle(primary, fallback, identifier string) string {
	primary = strings.TrimSpace(primary)
	if primary != "" {
		return primary
	}
	fallback = strings.TrimSpace(fallback)
	if fallback != "" {
		return fallback
	}
	if identifier != "" {
		return identifier
	}
	return "Untitled"
}

func imdbURL(imdbID, existing string) string {
	if existing != "" {
		return existing
	}
	if imdbID == "" {
		return ""
	}
	return fmt.Sprintf("https://www.imdb.com/title/%s/", imdbID)
}

func buildLetterboxdMatches(matches []letterboxdMovie) []diffMatch {
	if len(matches) == 0 {
		return nil
	}
	seen := map[string]bool{}
	result := make([]diffMatch, 0, len(matches))
	for _, match := range matches {
		key := match.LetterboxdURI
		if key == "" {
			key = match.LetterboxdID
		}
		if key != "" && seen[key] {
			continue
		}
		if key != "" {
			seen[key] = true
		}
		result = append(result, diffMatch{
			Title:            displayTitle(match.Name, "", match.LetterboxdID),
			Year:             match.Year,
			ImdbID:           match.ImdbID,
			ImdbURL:          imdbURL(match.ImdbID, ""),
			LetterboxdURI:    match.LetterboxdURI,
			LetterboxdRating: match.Rating,
		})
	}
	sortMatches(result)
	return result
}

func buildIMDbMatches(matches []imdbMovie) []diffMatch {
	if len(matches) == 0 {
		return nil
	}
	seen := map[string]bool{}
	result := make([]diffMatch, 0, len(matches))
	for _, match := range matches {
		key := match.ImdbID
		if key != "" && seen[key] {
			continue
		}
		if key != "" {
			seen[key] = true
		}
		result = append(result, diffMatch{
			Title:      displayTitle(match.Title, match.OriginalTitle, match.ImdbID),
			Year:       match.Year,
			ImdbID:     match.ImdbID,
			ImdbURL:    imdbURL(match.ImdbID, match.URL),
			ImdbRating: match.MyRating,
		})
	}
	sortMatches(result)
	return result
}

func sortDiffItems(items []diffItem) {
	sort.Slice(items, func(i, j int) bool {
		iTitle := strings.ToLower(items[i].Title)
		jTitle := strings.ToLower(items[j].Title)
		if iTitle == jTitle {
			return items[i].Year < items[j].Year
		}
		return iTitle < jTitle
	})
}

func sortMatches(matches []diffMatch) {
	sort.Slice(matches, func(i, j int) bool {
		iTitle := strings.ToLower(matches[i].Title)
		jTitle := strings.ToLower(matches[j].Title)
		if iTitle == jTitle {
			return matches[i].Year < matches[j].Year
		}
		return iTitle < jTitle
	})
}

func buildDiffNote(imdbOnly, letterboxdOnly []diffItem, stats diffStats, now time.Time, mainDBPath, cacheDBPath string) *obsidian.Note {
	fm := obsidian.NewFrontmatter()
	fm.Set("title", "IMDb vs Letterboxd Diff")
	fm.Set("date", now.Format("2006-01-02"))
	fm.Set("imdb_only", stats.imdbOnlyCount)
	fm.Set("letterboxd_only", stats.letterboxdOnlyCount)
	fm.Set("resolved_title_year", stats.resolvedTitleYear)
	fm.Set("imdb_only_with_fuzzy", stats.imdbOnlyWithFuzzy)
	fm.Set("letterboxd_only_with_fuzzy", stats.letterboxdOnlyWithFuzzy)

	tags := obsidian.NewTagSet()
	tags.Add("diff/imdb-letterboxd")
	tags.Add("report")
	tags.Add("movies")
	fm.Set("tags", tags.GetSorted())

	body := buildDiffBody(imdbOnly, letterboxdOnly, stats, now, mainDBPath, cacheDBPath)

	return &obsidian.Note{
		Frontmatter: fm,
		Body:        body,
	}
}

func buildDiffBody(imdbOnly, letterboxdOnly []diffItem, stats diffStats, now time.Time, mainDBPath, cacheDBPath string) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("# IMDb vs Letterboxd Diff (%s)\n\n", now.Format("2006-01-02")))
	b.WriteString("This report compares IMDb and Letterboxd movie imports in SQLite. TV titles are excluded.\n\n")
	b.WriteString(fmt.Sprintf("- Main DB: `%s`\n", mainDBPath))
	if cacheDBPath != "" {
		b.WriteString(fmt.Sprintf("- Cache DB: `%s`\n", cacheDBPath))
	}
	b.WriteString("- Matching: IMDb ID, then auto-resolved title+year, then fuzzy suggestions\n")
	b.WriteString(fmt.Sprintf("- Auto-resolved title+year matches: %d\n\n", stats.resolvedTitleYear))

	writeDiffSection(&b, "IMDb-only (missing from Letterboxd)", imdbOnly)
	writeDiffSection(&b, "Letterboxd-only (missing from IMDb)", letterboxdOnly)

	return b.String()
}

func writeDiffSection(b *strings.Builder, title string, items []diffItem) {
	b.WriteString("## ")
	b.WriteString(title)
	b.WriteString("\n\n")
	if len(items) == 0 {
		b.WriteString("_None found._\n\n")
		return
	}
	for _, item := range items {
		b.WriteString("- [ ] ")
		b.WriteString(formatDiffLine(item))
		b.WriteString("\n")
		if len(item.FuzzyMatches) > 0 {
			b.WriteString("  - Possible matches (title + year):\n")
			for _, match := range item.FuzzyMatches {
				b.WriteString("    - ")
				b.WriteString(formatMatchLine(match))
				b.WriteString("\n")
			}
		}
	}
	b.WriteString("\n")
}

func formatDiffLine(item diffItem) string {
	parts := []string{formatTitleYear(item.Title, item.Year)}
	if item.ImdbID != "" {
		parts = append(parts, fmt.Sprintf("IMDb %s", item.ImdbID))
	}
	if item.ImdbURL != "" {
		parts = append(parts, item.ImdbURL)
	}
	if item.LetterboxdURI != "" {
		parts = append(parts, item.LetterboxdURI)
	}
	if item.ImdbRating > 0 {
		parts = append(parts, fmt.Sprintf("IMDb rating %d/10", item.ImdbRating))
	}
	if item.LetterboxdRating > 0 {
		parts = append(parts, fmt.Sprintf("Letterboxd rating %.1f/5", item.LetterboxdRating))
	}
	return strings.Join(parts, " — ")
}

func formatMatchLine(match diffMatch) string {
	parts := []string{formatTitleYear(match.Title, match.Year)}
	if match.ImdbID != "" {
		parts = append(parts, fmt.Sprintf("IMDb %s", match.ImdbID))
	}
	if match.ImdbURL != "" {
		parts = append(parts, match.ImdbURL)
	}
	if match.LetterboxdURI != "" {
		parts = append(parts, match.LetterboxdURI)
	}
	if match.ImdbRating > 0 {
		parts = append(parts, fmt.Sprintf("IMDb rating %d/10", match.ImdbRating))
	}
	if match.LetterboxdRating > 0 {
		parts = append(parts, fmt.Sprintf("Letterboxd rating %.1f/5", match.LetterboxdRating))
	}
	return strings.Join(parts, " — ")
}

func formatTitleYear(title string, year int) string {
	if year > 0 {
		return fmt.Sprintf("%s (%d)", title, year)
	}
	return title
}
