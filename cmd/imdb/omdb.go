package imdb

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/spf13/viper"
)

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

func fetchMovieData(imdbID string) (*MovieSeen, error) {
	apiKey := viper.GetString("imdb.omdb_api_key")
	if apiKey == "" {
		return nil, fmt.Errorf("imdb.omdb_api_key not set in config")
	}

	url := fmt.Sprintf("http://www.omdbapi.com/?i=%s&apikey=%s", imdbID, apiKey)

	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch data: %w", err)
	}
	defer resp.Body.Close()

	var omdbMovie OMDbMovie
	if err := json.NewDecoder(resp.Body).Decode(&omdbMovie); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Enhanced conversion
	movie := &MovieSeen{
		Title:        omdbMovie.Title,
		ImdbId:       omdbMovie.ImdbID,
		TitleType:    omdbMovie.Type,
		IMDbRating:   parseFloat(omdbMovie.ImdbRating),
		Plot:         omdbMovie.Plot,
		PosterURL:    omdbMovie.Poster,
		ContentRated: omdbMovie.Rated,
		Awards:       omdbMovie.Awards,
		// Parse genres from comma-separated string
		Genres: strings.Split(omdbMovie.Genre, ", "),
		// Parse directors from comma-separated string
		Directors: strings.Split(omdbMovie.Director, ", "),
		// Parse runtime to minutes
		RuntimeMins: parseRuntime(omdbMovie.Runtime),
	}

	return movie, nil
}

// Helper functions to parse OMDB data
func parseFloat(s string) float64 {
	f, _ := strconv.ParseFloat(s, 64)
	return f
}

func parseRuntime(runtime string) int {
	// Convert "123 min" to 123
	mins := strings.TrimSuffix(runtime, " min")
	val, _ := strconv.Atoi(mins)
	return val
}
