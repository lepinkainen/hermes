package imdb

// MovieSeen represents a watched movie from IMDB export
type MovieSeen struct {
	ImdbId        string   `json:"ImdbId"`
	MyRating      int      `json:"My Rating"`
	DateRated     string   `json:"Date Rated"`
	Title         string   `json:"Title"`
	OriginalTitle string   `json:"Original Title"`
	URL           string   `json:"URL"`
	TitleType     string   `json:"Title Type"`
	IMDbRating    float64  `json:"IMDb Rating"`
	RuntimeMins   int      `json:"Runtime (mins)"`
	Year          int      `json:"Year"`
	Genres        []string `json:"Genres"`
	NumVotes      int      `json:"Num Votes"`
	ReleaseDate   string   `json:"Release Date"`
	Directors     []string `json:"Directors"`
}

// MovieWatchlist represents a movie in the watchlist
type MovieWatchlist struct {
	Const         string  `json:"ImdbId"`
	Created       string  `json:"Created"`
	Modified      string  `json:"Modified"`
	Description   string  `json:"Description"`
	Title         string  `json:"Title"`
	OriginalTitle string  `json:"Original Title"`
	URL           string  `json:"URL"`
	TitleType     string  `json:"Title Type"`
	IMDbRating    float64 `json:"IMDb Rating"`
	RuntimeMins   int     `json:"Runtime (mins)"`
	Year          int     `json:"Year"`
	Genres        []string
	NumVotes      int    `json:"Num Votes"`
	ReleaseDate   string `json:"Release Date"`
	Directors     []string
	YourRating    string `json:"Your Rating"`
	DateRated     string `json:"Date Rated"`
}
