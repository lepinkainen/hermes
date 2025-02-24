package imdb

// MovieSeen represents a watched movie from IMDB export
type MovieSeen struct {
	Position      int      `json:"Position,omitempty"`
	ImdbId        string   `json:"ImdbId"`
	MyRating      int      `json:"My Rating"`
	DateRated     string   `json:"Date Rated"`
	Created       string   `json:"Created,omitempty"`
	Modified      string   `json:"Modified,omitempty"`
	Description   string   `json:"Description,omitempty"`
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
	Plot          string   `json:"Plot"`
	ContentRated  string   `json:"Content Rated"`
	Awards        string   `json:"Awards"`
	PosterURL     string   `json:"Poster URL"`
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

type OMDbMovie struct {
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
}

type Rating struct {
	Source string `json:"Source"`
	Value  string `json:"Value"`
}
