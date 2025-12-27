package diff

import (
	"bytes"
	_ "embed"
	"fmt"
	"html/template"
	"math"
	"net/url"
)

// imdbToLetterboxdMap maps IMDb ratings (1-10) to Letterboxd stars (0.5-5).
// This reflects non-linear usage patterns: IMDb 1-2 are rarely used,
// while Letterboxd's 5-star scale is more evenly distributed.
var imdbToLetterboxdMap = map[int]float64{
	1:  0.5,
	2:  0.5,
	3:  1.0,
	4:  1.5,
	5:  2.0,
	6:  3.0,
	7:  3.5,
	8:  4.0,
	9:  4.5,
	10: 5.0,
}

// letterboxdToImdbMap maps Letterboxd stars (0.5-5) to IMDb ratings (1-10).
var letterboxdToImdbMap = map[float64]int{
	0.5: 2,
	1.0: 3,
	1.5: 4,
	2.0: 5,
	2.5: 5,
	3.0: 6,
	3.5: 7,
	4.0: 8,
	4.5: 9,
	5.0: 10,
}

// imdbToLB converts an IMDb rating (1-10) to Letterboxd stars (0.5-5).
func imdbToLB(imdb int) float64 {
	if imdb < 1 {
		imdb = 1
	}
	if imdb > 10 {
		imdb = 10
	}
	return imdbToLetterboxdMap[imdb]
}

// lbToImdb converts Letterboxd stars (0.5-5) to an IMDb rating (1-10).
func lbToImdb(stars float64) int {
	// Round to nearest 0.5
	stars = math.Round(stars*2) / 2
	if stars < 0.5 {
		stars = 0.5
	}
	if stars > 5.0 {
		stars = 5.0
	}
	if v, ok := letterboxdToImdbMap[stars]; ok {
		return v
	}
	return 5 // fallback (should never hit if table is complete)
}

//go:embed template.html
var htmlTemplate string

// htmlTemplateData is the data passed to the HTML template.
type htmlTemplateData struct {
	ImdbOnly       []diffItem
	LetterboxdOnly []diffItem
	Stats          htmlStats
	GeneratedAt    interface{ Format(string) string }
	MainDBPath     string
	CacheDBPath    string
}

// htmlStats wraps diffStats with exported fields for template access.
type htmlStats struct {
	ImdbOnlyCount           int
	LetterboxdOnlyCount     int
	ResolvedTitleYear       int
	ImdbOnlyWithFuzzy       int
	LetterboxdOnlyWithFuzzy int
}

// renderDiffHTML renders the diff report as an HTML page.
func renderDiffHTML(report *diffReport) ([]byte, error) {
	funcMap := template.FuncMap{
		"letterboxdSearchURL": letterboxdSearchURL,
		"imdbSearchURL":       imdbSearchURL,
		// hasUserRating returns true only for user's personal Letterboxd ratings (0.5-5 scale)
		// Ratings > 5 are TMDB enrichment averages, not user ratings
		"hasUserRating": func(r float64) bool { return r > 0 && r <= 5 },
		"hasIntRating":  func(r int) bool { return r > 0 },
		// Convert IMDb rating (1-10) to Letterboxd scale (0.5-5) using non-linear mapping
		"imdbToLetterboxd": imdbToLB,
		// Convert Letterboxd user rating (0.5-5) to IMDb scale (1-10) using non-linear mapping
		"letterboxdToImdb": lbToImdb,
	}

	tmpl, err := template.New("diff").Funcs(funcMap).Parse(htmlTemplate)
	if err != nil {
		return nil, fmt.Errorf("failed to parse template: %w", err)
	}

	data := htmlTemplateData{
		ImdbOnly:       report.ImdbOnly,
		LetterboxdOnly: report.LetterboxdOnly,
		Stats: htmlStats{
			ImdbOnlyCount:           report.Stats.imdbOnlyCount,
			LetterboxdOnlyCount:     report.Stats.letterboxdOnlyCount,
			ResolvedTitleYear:       report.Stats.resolvedTitleYear,
			ImdbOnlyWithFuzzy:       report.Stats.imdbOnlyWithFuzzy,
			LetterboxdOnlyWithFuzzy: report.Stats.letterboxdOnlyWithFuzzy,
		},
		GeneratedAt: report.GeneratedAt,
		MainDBPath:  report.MainDBPath,
		CacheDBPath: report.CacheDBPath,
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.Bytes(), nil
}

// letterboxdSearchURL builds a Letterboxd search URL for a movie.
func letterboxdSearchURL(title string, year int) string {
	query := fmt.Sprintf("%s %d", title, year)
	return fmt.Sprintf("https://letterboxd.com/search/%s/", url.PathEscape(query))
}

// imdbSearchURL builds an IMDb search URL for a movie.
func imdbSearchURL(title string, year int) string {
	query := fmt.Sprintf("%s %d", title, year)
	return fmt.Sprintf("https://www.imdb.com/find/?q=%s", url.QueryEscape(query))
}
