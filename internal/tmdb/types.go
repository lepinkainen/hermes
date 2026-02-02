package tmdb

import (
	"strconv"
)

// SearchResult represents a single search result from TMDB.
type SearchResult struct {
	ID           int
	MediaType    string
	Title        string
	Name         string
	PosterPath   string
	Overview     string
	ReleaseDate  string
	FirstAirDate string
	VoteAverage  float64
	VoteCount    int     `json:"vote_count"`
	Popularity   float64 `json:"popularity"`
	Runtime      int     `json:"runtime"`
	OriginalLang string  `json:"original_language"`
}

// DisplayTitle returns the appropriate title for the search result.
func (r SearchResult) DisplayTitle() string {
	if r.Title != "" {
		return r.Title
	}
	return r.Name
}

// YearInt returns the release year for movies or first air year for TV shows as int.
func (r SearchResult) YearInt() int {
	dateStr := r.ReleaseDate
	if r.MediaType == "tv" {
		dateStr = r.FirstAirDate
	}
	if len(dateStr) >= 4 {
		if year, err := strconv.Atoi(dateStr[:4]); err == nil {
			return year
		}
	}
	return 0
}

// Year extracts the year from the release or air date.
func (r SearchResult) Year() string {
	source := r.ReleaseDate
	if r.MediaType == "tv" {
		source = r.FirstAirDate
	}
	if source == "" {
		return "Unknown"
	}
	if len(source) >= 4 {
		return source[:4]
	}
	return source
}

// Metadata holds TMDB metadata for a movie or TV show.
type Metadata struct {
	TMDBID        int
	TMDBType      string
	IMDBID        string // IMDb ID from external_ids (e.g., "tt1234567")
	Runtime       *int
	TotalEpisodes *int
	GenreTags     []string
	Status        string // TV show status: "Ended", "Canceled", "Returning Series", etc.
}
