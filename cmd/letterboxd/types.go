package letterboxd

// Movie represents a movie from Letterboxd export
type Movie struct {
	Date          string `json:"Date"`             // Date the movie was watched
	Name          string `json:"Name"`             // Title of the movie
	Year          int    `json:"Year"`             // Release year
	LetterboxdID  string `json:"LetterboxdID"`     // Letterboxd ID extracted from URI
	LetterboxdURI string `json:"LetterboxdURI"`    // Full Letterboxd URI
	ImdbID        string `json:"ImdbID,omitempty"` // IMDB ID when available

	// Additional fields that might be enriched later
	Director    string   `json:"Director,omitempty"`
	Cast        []string `json:"Cast,omitempty"`
	Genres      []string `json:"Genres,omitempty"`
	Runtime     int      `json:"Runtime,omitempty"`
	Rating      float64  `json:"Rating,omitempty"`
	PosterURL   string   `json:"PosterURL,omitempty"`
	Description string   `json:"Description,omitempty"`
}
