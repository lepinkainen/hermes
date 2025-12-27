package letterboxd

import "github.com/lepinkainen/hermes/internal/enrichment"

// Movie represents a movie from Letterboxd export
type Movie struct {
	Date          string `json:"Date"`             // Date the movie was watched
	Name          string `json:"Name"`             // Title of the movie
	Year          int    `json:"Year"`             // Release year
	LetterboxdID  string `json:"LetterboxdID"`     // Letterboxd ID extracted from URI
	LetterboxdURI string `json:"LetterboxdURI"`    // Full Letterboxd URI
	ImdbID        string `json:"ImdbID,omitempty"` // IMDB ID when available

	// Additional fields that might be enriched later
	Director        string   `json:"Director,omitempty"`
	Cast            []string `json:"Cast,omitempty"`
	Genres          []string `json:"Genres,omitempty"`
	Runtime         int      `json:"Runtime,omitempty"`
	Rating          float64  `json:"Rating,omitempty"`          // User's personal Letterboxd rating (0.5-5 scale)
	CommunityRating float64  `json:"CommunityRating,omitempty"` // OMDB/TMDB community rating (0-10 scale)
	PosterURL       string   `json:"PosterURL,omitempty"`
	Description     string   `json:"Description,omitempty"`

	// TMDB enrichment data
	TMDBEnrichment *enrichment.TMDBEnrichment `json:"TMDBEnrichment,omitempty"`
}
