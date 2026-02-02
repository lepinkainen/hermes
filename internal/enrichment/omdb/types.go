package omdb

// RatingsEnrichment contains ratings data from OMDB
type RatingsEnrichment struct {
	IMDbRating     float64 `json:"imdb_rating,omitempty"`
	RottenTomatoes string  `json:"rotten_tomatoes,omitempty"` // e.g., "94%"
	RTTomatometer  int     `json:"rt_tomatometer,omitempty"`  // parsed percentage (0-100)
	Metacritic     int     `json:"metacritic,omitempty"`      // score out of 100
}

// OMDBResponse represents the full response from the OMDB API
type OMDBResponse struct {
	Title      string   `json:"Title"`
	Year       string   `json:"Year"`
	Rated      string   `json:"Rated"`
	Released   string   `json:"Released"`
	Runtime    string   `json:"Runtime"`
	Genre      string   `json:"Genre"`
	Director   string   `json:"Director"`
	Writer     string   `json:"Writer"`
	Actors     string   `json:"Actors"`
	Plot       string   `json:"Plot"`
	Language   string   `json:"Language"`
	Country    string   `json:"Country"`
	Awards     string   `json:"Awards"`
	Poster     string   `json:"Poster"`
	Ratings    []Rating `json:"Ratings"`
	ImdbRating string   `json:"imdbRating"`
	ImdbVotes  string   `json:"imdbVotes"`
	ImdbID     string   `json:"imdbID"`
	Type       string   `json:"Type"`
	Response   string   `json:"Response"` // "True" or "False"
	Error      string   `json:"Error"`    // Present if Response is "False"
}

// Rating represents a rating from a specific source
type Rating struct {
	Source string `json:"Source"`
	Value  string `json:"Value"`
}
