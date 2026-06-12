package imdb

import "github.com/lepinkainen/hermes/internal/enrichment"

// MovieSeen represents a watched movie from IMDB export
type MovieSeen struct {
	Position      int      `json:"position,omitempty"`
	ImdbId        string   `json:"imdbId"`
	MyRating      int      `json:"myRating"`
	DateRated     string   `json:"dateRated"`
	Created       string   `json:"created,omitempty"`
	Modified      string   `json:"modified,omitempty"`
	Description   string   `json:"description,omitempty"`
	Title         string   `json:"title"`
	OriginalTitle string   `json:"originalTitle"`
	URL           string   `json:"url"`
	TitleType     string   `json:"titleType"`
	IMDbRating    float64  `json:"imdbRating"`
	RuntimeMins   int      `json:"runtimeMins"`
	Year          int      `json:"year"`
	Genres        []string `json:"genres"`
	NumVotes      int      `json:"numVotes"`
	ReleaseDate   string   `json:"releaseDate"`
	Directors     []string `json:"directors"`
	Plot          string   `json:"plot"`
	ContentRated  string   `json:"contentRated"`
	Awards        string   `json:"awards"`
	PosterURL     string   `json:"posterUrl"`
	// TMDB enrichment data
	TMDBEnrichment *enrichment.TMDBEnrichment `json:"tmdb,omitempty"`
}

// MovieWatchlist represents a movie in the watchlist
type MovieWatchlist struct {
	Const         string   `json:"imdbId"`
	Created       string   `json:"created"`
	Modified      string   `json:"modified"`
	Description   string   `json:"description"`
	Title         string   `json:"title"`
	OriginalTitle string   `json:"originalTitle"`
	URL           string   `json:"url"`
	TitleType     string   `json:"titleType"`
	IMDbRating    float64  `json:"imdbRating"`
	RuntimeMins   int      `json:"runtimeMins"`
	Year          int      `json:"year"`
	Genres        []string `json:"genres"`
	NumVotes      int      `json:"numVotes"`
	ReleaseDate   string   `json:"releaseDate"`
	Directors     []string `json:"directors"`
	YourRating    string   `json:"yourRating"`
	DateRated     string   `json:"dateRated"`
}
