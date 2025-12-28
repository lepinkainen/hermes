package steam

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestGameDetailsToMap(t *testing.T) {
	// Create a test GameDetails with all fields populated
	testTime := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)

	details := GameDetails{
		Game: Game{
			AppID:           12345,
			Name:            "Test Game",
			PlaytimeForever: 120,
			PlaytimeRecent:  30,
			LastPlayed:      testTime,
			DetailsFetched:  true,
		},
		Description: "This is a detailed description of the test game.",
		ShortDesc:   "Short description",
		HeaderImage: "https://example.com/header.jpg",
		Screenshots: []Screenshot{
			{ID: 1, PathURL: "https://example.com/screenshot1.jpg"},
			{ID: 2, PathURL: "https://example.com/screenshot2.jpg"},
		},
		Developers: []string{"Developer One", "Developer Two"},
		Publishers: []string{"Publisher One", "Publisher Two"},
		ReleaseDate: struct {
			ComingSoon bool   `json:"coming_soon"`
			Date       string `json:"date"`
		}{
			ComingSoon: false,
			Date:       "15 Jan, 2023",
		},
		Categories: []Category{
			{ID: 1, Description: "Single-player"},
			{ID: 2, Description: "Multi-player"},
		},
		Genres: []Genre{
			{ID: "1", Description: "Action"},
			{ID: "2", Description: "Adventure"},
		},
		Metacritic: MetacriticData{
			Score: 85,
			URL:   "https://www.metacritic.com/game/test-game",
		},
	}

	result := gameDetailsToMap(details)

	// Verify all fields are correctly mapped
	assert.Equal(t, 12345, result["appid"])
	assert.Equal(t, "Test Game", result["name"])
	assert.Equal(t, 120, result["playtime_forever"])
	assert.Equal(t, 30, result["playtime_2weeks"])
	assert.Equal(t, testTime.String(), result["last_played"])
	assert.Equal(t, true, result["details_fetched"])
	assert.Equal(t, "This is a detailed description of the test game.", result["detailed_description"])
	assert.Equal(t, "Short description", result["short_description"])
	assert.Equal(t, "https://example.com/header.jpg", result["header_image"])
	assert.Equal(t, "", result["screenshots"]) // Screenshots are empty in current implementation
	assert.Equal(t, "Developer One,Developer Two", result["developers"])
	assert.Equal(t, "Publisher One,Publisher Two", result["publishers"])
	assert.Equal(t, "15 Jan, 2023", result["release_date"])
	assert.Equal(t, false, result["coming_soon"])
	assert.Equal(t, "", result["categories"]) // Categories are empty in current implementation
	assert.Equal(t, "", result["genres"])     // Genres are empty in current implementation
	assert.Equal(t, 85, result["metacritic_score"])
	assert.Equal(t, "https://www.metacritic.com/game/test-game", result["metacritic_url"])
}

func TestGameDetailsToMap_MinimalData(t *testing.T) {
	// Test with minimal data (no optional fields)
	details := GameDetails{
		Game: Game{
			AppID:           999,
			Name:            "Minimal Game",
			PlaytimeForever: 0,
			PlaytimeRecent:  0,
			LastPlayed:      time.Time{},
			DetailsFetched:  false,
		},
		Description: "",
		ShortDesc:   "",
		HeaderImage: "",
		Developers:  []string{},
		Publishers:  []string{},
		ReleaseDate: struct {
			ComingSoon bool   `json:"coming_soon"`
			Date       string `json:"date"`
		}{
			ComingSoon: false,
			Date:       "",
		},
		Metacritic: MetacriticData{
			Score: 0,
			URL:   "",
		},
	}

	result := gameDetailsToMap(details)

	// Verify minimal fields are correctly mapped
	assert.Equal(t, 999, result["appid"])
	assert.Equal(t, "Minimal Game", result["name"])
	assert.Equal(t, 0, result["playtime_forever"])
	assert.Equal(t, 0, result["playtime_2weeks"])
	assert.Equal(t, false, result["details_fetched"])
	assert.Equal(t, "", result["detailed_description"])
	assert.Equal(t, "", result["developers"])
	assert.Equal(t, "", result["publishers"])
	assert.Equal(t, 0, result["metacritic_score"])
}
