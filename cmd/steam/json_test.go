package steam

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/lepinkainen/hermes/internal/config"
	"github.com/lepinkainen/hermes/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriteGameToJson(t *testing.T) {
	env := testutil.NewTestEnv(t)
	testutil.SetTestConfig(t)

	// Create test data
	testTime := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)
	games := []GameDetails{
		{
			Game: Game{
				AppID:           12345,
				Name:            "Test Game One",
				PlaytimeForever: 120,
				PlaytimeRecent:  30,
				LastPlayed:      testTime,
				DetailsFetched:  true,
			},
			Description: "First test game",
			ShortDesc:   "Game 1",
			HeaderImage: "https://example.com/game1.jpg",
			Developers:  []string{"Dev1"},
			Publishers:  []string{"Pub1"},
			ReleaseDate: struct {
				ComingSoon bool   `json:"coming_soon"`
				Date       string `json:"date"`
			}{
				ComingSoon: false,
				Date:       "2023-01-15",
			},
			Categories: []Category{{ID: 1, Description: "Single-player"}},
			Genres:     []Genre{{ID: "1", Description: "Action"}},
			Metacritic: MetacriticData{Score: 85, URL: "https://example.com/metacritic1"},
		},
		{
			Game: Game{
				AppID:           67890,
				Name:            "Test Game Two",
				PlaytimeForever: 0,
				PlaytimeRecent:  0,
				LastPlayed:      time.Time{},
				DetailsFetched:  true,
			},
			Description: "Second test game",
			ShortDesc:   "Game 2",
			HeaderImage: "https://example.com/game2.jpg",
			Developers:  []string{"Dev2", "Dev3"},
			Publishers:  []string{"Pub2"},
			ReleaseDate: struct {
				ComingSoon bool   `json:"coming_soon"`
				Date       string `json:"date"`
			}{
				ComingSoon: false,
				Date:       "2024-03-20",
			},
			Categories: []Category{
				{ID: 1, Description: "Single-player"},
				{ID: 2, Description: "Multi-player"},
			},
			Genres: []Genre{
				{ID: "23", Description: "Indie"},
				{ID: "9", Description: "Strategy"},
			},
			Metacritic: MetacriticData{Score: 78, URL: "https://example.com/metacritic2"},
		},
	}

	// Write to JSON
	jsonPath := filepath.Join(env.RootDir(), "steam_games.json")
	config.OverwriteFiles = true // Set global config
	err := writeGameToJson(games, jsonPath)
	require.NoError(t, err)

	// Verify file exists
	_, err = os.Stat(jsonPath)
	require.NoError(t, err, "JSON file should exist")

	// Read and verify JSON content
	data, err := os.ReadFile(jsonPath)
	require.NoError(t, err)

	var readGames []GameDetails
	err = json.Unmarshal(data, &readGames)
	require.NoError(t, err)

	// Verify we got the same number of games
	require.Len(t, readGames, 2)

	// Verify first game
	assert.Equal(t, 12345, readGames[0].AppID)
	assert.Equal(t, "Test Game One", readGames[0].Name)
	assert.Equal(t, 120, readGames[0].PlaytimeForever)
	assert.Equal(t, "First test game", readGames[0].Description)
	assert.Equal(t, []string{"Dev1"}, readGames[0].Developers)

	// Verify second game
	assert.Equal(t, 67890, readGames[1].AppID)
	assert.Equal(t, "Test Game Two", readGames[1].Name)
	assert.Equal(t, 0, readGames[1].PlaytimeForever)
	assert.Equal(t, "Second test game", readGames[1].Description)
	assert.Equal(t, []string{"Dev2", "Dev3"}, readGames[1].Developers)
}

func TestWriteGameToJson_EmptyList(t *testing.T) {
	env := testutil.NewTestEnv(t)
	testutil.SetTestConfig(t)

	// Write empty list
	jsonPath := filepath.Join(env.RootDir(), "empty_games.json")
	config.OverwriteFiles = true
	err := writeGameToJson([]GameDetails{}, jsonPath)
	require.NoError(t, err)

	// Read and verify
	data, err := os.ReadFile(jsonPath)
	require.NoError(t, err)

	var readGames []GameDetails
	err = json.Unmarshal(data, &readGames)
	require.NoError(t, err)

	assert.Len(t, readGames, 0)
}

func TestWriteGameToJson_OverwriteProtection(t *testing.T) {
	env := testutil.NewTestEnv(t)
	testutil.SetTestConfig(t)

	jsonPath := filepath.Join(env.RootDir(), "test.json")

	// Create initial file
	initialData := []GameDetails{
		{
			Game: Game{AppID: 111, Name: "Initial Game"},
		},
	}
	config.OverwriteFiles = true
	err := writeGameToJson(initialData, jsonPath)
	require.NoError(t, err)

	// Try to write again without overwrite
	newData := []GameDetails{
		{
			Game: Game{AppID: 222, Name: "New Game"},
		},
	}
	config.OverwriteFiles = false
	err = writeGameToJson(newData, jsonPath)
	// WriteJSONFile returns nil error when file exists and overwrite is false (it just skips)
	require.NoError(t, err)

	// Verify original data is still there (file was not overwritten)
	data, err := os.ReadFile(jsonPath)
	require.NoError(t, err)

	var readGames []GameDetails
	err = json.Unmarshal(data, &readGames)
	require.NoError(t, err)
	require.Len(t, readGames, 1)
	assert.Equal(t, 111, readGames[0].AppID)
	assert.Equal(t, "Initial Game", readGames[0].Name)
}
