package diff

import "time"

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

type diffStats struct {
	imdbOnlyCount           int
	letterboxdOnlyCount     int
	resolvedTitleYear       int
	imdbOnlyWithFuzzy       int
	letterboxdOnlyWithFuzzy int
}

type diffReport struct {
	ImdbOnly       []diffItem
	LetterboxdOnly []diffItem
	Stats          diffStats
	GeneratedAt    time.Time
	MainDBPath     string
	CacheDBPath    string
}
