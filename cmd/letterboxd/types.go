package letterboxd

import "github.com/lepinkainen/hermes/internal/enrichment"

// Movie represents a movie from Letterboxd export
type Movie struct {
	Date          string `json:"date"`             // Date the movie was watched
	Name          string `json:"name"`             // Title of the movie
	Year          int    `json:"year"`             // Release year
	LetterboxdID  string `json:"letterboxdId"`     // Letterboxd ID extracted from URI
	LetterboxdURI string `json:"letterboxdUri"`    // Full Letterboxd URI
	ImdbID        string `json:"imdbId,omitempty"` // IMDB ID when available

	// Additional fields that might be enriched later
	Director        string   `json:"director,omitempty"`
	Cast            []string `json:"cast,omitempty"`
	Genres          []string `json:"genres,omitempty"`
	Runtime         int      `json:"runtime,omitempty"`
	Rating          float64  `json:"rating,omitempty"`          // User's personal Letterboxd rating (0.5-5 scale)
	CommunityRating float64  `json:"communityRating,omitempty"` // OMDB/TMDB community rating (0-10 scale)
	PosterURL       string   `json:"posterUrl,omitempty"`
	Description     string   `json:"description,omitempty"`

	// TMDB enrichment data
	TMDBEnrichment *enrichment.TMDBEnrichment `json:"tmdb,omitempty"`
}
